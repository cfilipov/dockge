package compose

import (
	"os"
	"path/filepath"
)

// GlobalEnvArgs returns --env-file flags to prepend to compose args
// when global.env exists in the stacks directory. If the stack also has
// a local .env, it is re-added explicitly (--env-file overrides the
// default .env loading). Returns nil when no global.env exists.
func GlobalEnvArgs(stacksDir, stackName string) []string {
	globalPath := filepath.Join(stacksDir, "global.env")
	if _, err := os.Stat(globalPath); err != nil {
		return nil
	}
	args := []string{"--env-file", "../global.env"}
	localEnv := filepath.Join(stacksDir, stackName, ".env")
	if _, err := os.Stat(localEnv); err == nil {
		args = append(args, "--env-file", "./.env")
	}
	return args
}
