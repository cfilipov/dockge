package stack

import (
    "os"
    "path/filepath"
    "testing"
)

func TestStatusConvert(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name   string
        input  string
        expect int
    }{
        {"running single", "running(2)", RUNNING},
        {"exited single", "exited(1)", EXITED},
        {"mixed running and exited", "running(2), exited(1)", RUNNING_AND_EXITED},
        {"created", "created(1)", CREATED_STACK},
        {"empty string", "", UNKNOWN},
        {"running zero count", "running(0)", RUNNING},   // falls through to strings.Contains
        {"exited zero count", "exited(0)", EXITED},     // falls through to strings.Contains
        {"running no parens", "running", RUNNING},
        {"exited no parens", "exited", EXITED},
        {"created prefix", "created(3), running(1)", CREATED_STACK},
        {"malformed no closing paren", "running(2", RUNNING}, // falls through to strings.Contains
        {"unknown status", "paused(1)", UNKNOWN},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            got := StatusConvert(tt.input)
            if got != tt.expect {
                t.Errorf("StatusConvert(%q) = %d, want %d", tt.input, got, tt.expect)
            }
        })
    }
}

func TestParseComposeLs(t *testing.T) {
    t.Parallel()

    t.Run("valid JSON", func(t *testing.T) {
        t.Parallel()
        data := []byte(`[{"Name":"web-app","Status":"running(2)","ConfigFiles":"/opt/stacks/web-app/compose.yaml"}]`)
        entries, err := ParseComposeLs(data)
        if err != nil {
            t.Fatal(err)
        }
        if len(entries) != 1 {
            t.Fatalf("expected 1 entry, got %d", len(entries))
        }
        if entries[0].Name != "web-app" {
            t.Errorf("Name = %q", entries[0].Name)
        }
        if entries[0].Status != "running(2)" {
            t.Errorf("Status = %q", entries[0].Status)
        }
    })

    t.Run("empty input", func(t *testing.T) {
        t.Parallel()
        _, err := ParseComposeLs([]byte{})
        if err == nil {
            t.Error("expected error for empty input")
        }
    })

    t.Run("malformed JSON", func(t *testing.T) {
        t.Parallel()
        _, err := ParseComposeLs([]byte(`not json`))
        if err == nil {
            t.Error("expected error for malformed JSON")
        }
    })

    t.Run("empty array", func(t *testing.T) {
        t.Parallel()
        entries, err := ParseComposeLs([]byte(`[]`))
        if err != nil {
            t.Fatal(err)
        }
        if len(entries) != 0 {
            t.Errorf("expected 0 entries, got %d", len(entries))
        }
    })
}

func TestIsStarted(t *testing.T) {
    t.Parallel()

    tests := []struct {
        status int
        want   bool
    }{
        {UNKNOWN, false},
        {CREATED_FILE, false},
        {CREATED_STACK, false},
        {RUNNING, true},
        {EXITED, false},
        {RUNNING_AND_EXITED, true},
        {UNHEALTHY, true},
    }

    for _, tt := range tests {
        s := &Stack{Status: tt.status}
        if got := s.IsStarted(); got != tt.want {
            t.Errorf("IsStarted() with status %d = %v, want %v", tt.status, got, tt.want)
        }
    }
}

func TestToSimpleJSON(t *testing.T) {
    t.Parallel()

    s := &Stack{
        Name:              "my-stack",
        Status:            RUNNING,
        IsManagedByDockge: true,
        ComposeFileName:   "compose.yaml",
    }

    result := s.ToSimpleJSON("", true, false)

    if result.Name != "my-stack" {
        t.Errorf("Name = %v", result.Name)
    }
    if result.Status != RUNNING {
        t.Errorf("Status = %v", result.Status)
    }
    if result.Started != true {
        t.Errorf("Started = %v", result.Started)
    }
    if result.IsManagedByDockge != true {
        t.Errorf("IsManagedByDockge = %v", result.IsManagedByDockge)
    }
    if result.ImageUpdatesAvailable != true {
        t.Errorf("ImageUpdatesAvailable = %v", result.ImageUpdatesAvailable)
    }
    if result.RecreateNecessary != false {
        t.Errorf("RecreateNecessary = %v", result.RecreateNecessary)
    }
    if result.Endpoint != "" {
        t.Errorf("Endpoint = %v", result.Endpoint)
    }
}

func TestToJSON(t *testing.T) {
    t.Parallel()

    s := &Stack{
        Name:                "my-stack",
        Status:              EXITED,
        ComposeYAML:         "services:\n  web:\n    image: nginx\n",
        ComposeENV:          "FOO=bar",
        ComposeOverrideYAML: "services:\n  web:\n    ports:\n      - 80:80\n",
    }

    result := s.ToJSON("ep1", "example.com", false, true)

    if result.ComposeYAML != s.ComposeYAML {
        t.Errorf("ComposeYAML mismatch")
    }
    if result.ComposeENV != s.ComposeENV {
        t.Errorf("ComposeENV mismatch")
    }
    if result.ComposeOverrideYAML != s.ComposeOverrideYAML {
        t.Errorf("ComposeOverrideYAML mismatch")
    }
    if result.PrimaryHostname != "example.com" {
        t.Errorf("PrimaryHostname = %v", result.PrimaryHostname)
    }
    if result.Endpoint != "ep1" {
        t.Errorf("Endpoint = %v", result.Endpoint)
    }
    if result.RecreateNecessary != true {
        t.Errorf("RecreateNecessary = %v", result.RecreateNecessary)
    }
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    original := &Stack{
        Name:                "round-trip",
        ComposeYAML:         "services:\n  app:\n    image: alpine:3.19\n",
        ComposeENV:          "KEY=value\nSECRET=hunter2",
        ComposeOverrideYAML: "services:\n  app:\n    ports:\n      - 8080:80\n",
    }

    if err := original.SaveToDisk(dir); err != nil {
        t.Fatal(err)
    }

    loaded := &Stack{Name: "round-trip"}
    if err := loaded.LoadFromDisk(dir); err != nil {
        t.Fatal(err)
    }

    if loaded.ComposeYAML != original.ComposeYAML {
        t.Errorf("ComposeYAML mismatch:\ngot:  %q\nwant: %q", loaded.ComposeYAML, original.ComposeYAML)
    }
    if loaded.ComposeENV != original.ComposeENV {
        t.Errorf("ComposeENV mismatch:\ngot:  %q\nwant: %q", loaded.ComposeENV, original.ComposeENV)
    }
    if loaded.ComposeOverrideYAML != original.ComposeOverrideYAML {
        t.Errorf("ComposeOverrideYAML mismatch:\ngot:  %q\nwant: %q", loaded.ComposeOverrideYAML, original.ComposeOverrideYAML)
    }
    if loaded.ComposeFileName != "compose.yaml" {
        t.Errorf("ComposeFileName = %q, want compose.yaml", loaded.ComposeFileName)
    }
    if loaded.ComposeOverrideFileName != "compose.override.yaml" {
        t.Errorf("ComposeOverrideFileName = %q, want compose.override.yaml", loaded.ComposeOverrideFileName)
    }
}

func TestSaveRemovesEmptyEnv(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()

    // First save with .env content
    s := &Stack{
        Name:        "env-test",
        ComposeYAML: "services:\n  app:\n    image: alpine\n",
        ComposeENV:  "KEY=value",
    }
    if err := s.SaveToDisk(dir); err != nil {
        t.Fatal(err)
    }

    envPath := filepath.Join(dir, "env-test", ".env")
    if _, err := os.Stat(envPath); err != nil {
        t.Fatal("expected .env to exist after first save")
    }

    // Save again with empty .env
    s.ComposeENV = ""
    if err := s.SaveToDisk(dir); err != nil {
        t.Fatal(err)
    }

    if _, err := os.Stat(envPath); !os.IsNotExist(err) {
        t.Error("expected .env to be removed when ComposeENV is empty")
    }
}

func TestComposeFileExists(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()

    // No compose file
    stackDir := filepath.Join(dir, "empty-stack")
    os.MkdirAll(stackDir, 0755)
    if ComposeFileExists(dir, "empty-stack") {
        t.Error("expected false for empty stack dir")
    }

    // compose.yaml
    os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte("services: {}"), 0644)
    if !ComposeFileExists(dir, "empty-stack") {
        t.Error("expected true with compose.yaml")
    }

    // docker-compose.yml variant
    stackDir2 := filepath.Join(dir, "alt-stack")
    os.MkdirAll(stackDir2, 0755)
    os.WriteFile(filepath.Join(stackDir2, "docker-compose.yml"), []byte("services: {}"), 0644)
    if !ComposeFileExists(dir, "alt-stack") {
        t.Error("expected true with docker-compose.yml")
    }

    // Nonexistent stack dir
    if ComposeFileExists(dir, "nonexistent") {
        t.Error("expected false for nonexistent stack")
    }
}

func TestSaveDefaultsComposeFileName(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    s := &Stack{
        Name:        "default-name",
        ComposeYAML: "services: {}\n",
    }

    if err := s.SaveToDisk(dir); err != nil {
        t.Fatal(err)
    }

    if s.ComposeFileName != "compose.yaml" {
        t.Errorf("ComposeFileName = %q, want compose.yaml", s.ComposeFileName)
    }

    composePath := filepath.Join(dir, "default-name", "compose.yaml")
    if _, err := os.Stat(composePath); err != nil {
        t.Error("expected compose.yaml to exist on disk")
    }
}
