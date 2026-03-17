package stack

import (
	"strings"
	"testing"
)

func TestValidateStackName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// Valid names
		{"my-stack", false},
		{"web_app", false},
		{"stack123", false},
		{"a", false},
		{"test-alpine", false},
		{"web-app", false},
		{"monitoring", false},
		{"blog", false},

		// Empty
		{"", true},

		// Path traversal
		{"../evil", true},
		{"..%2fevil", true},

		// Shell injection
		{"stack;id", true},
		{"stack$(cmd)", true},
		{"stack`cmd`", true},
		{"stack|cat", true},
		{"stack&bg", true},

		// Dots (hidden files, traversal building blocks)
		{".hidden", true},
		{"stack.name", true},

		// Uppercase
		{"UPPERCASE", true},
		{"Mixed", true},

		// Spaces and special chars
		{"has space", true},
		{"tab\there", true},
		{"slash/path", true},
		{"back\\slash", true},

		// Null byte
		{"null\x00byte", true},

		// Leading hyphen
		{"-leading", true},

		// Too long (256 chars)
		{strings.Repeat("a", 256), true},

		// Max valid length (255 chars)
		{strings.Repeat("a", 255), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStackName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStackName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
