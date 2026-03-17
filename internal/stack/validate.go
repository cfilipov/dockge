package stack

import (
	"errors"
	"fmt"
	"path/filepath"
)

// ValidateStackName checks that a stack name is safe for use as a
// directory name and Docker Compose project name.
// Only lowercase alphanumeric, hyphens, and underscores are allowed.
func ValidateStackName(name string) error {
	if name == "" {
		return errors.New("stack name must not be empty")
	}
	if len(name) > 255 {
		return errors.New("stack name must not exceed 255 characters")
	}
	// Only allow: lowercase alphanumeric, hyphens, underscores.
	// This blocks dots, slashes, spaces, shell metacharacters, null bytes.
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("stack name contains invalid character: %q", r)
		}
	}
	// Reject names that start with a hyphen (invalid project name)
	if name[0] == '-' {
		return errors.New("stack name must not start with a hyphen")
	}
	// Defense-in-depth: reject path traversal patterns
	if !filepath.IsLocal(name) {
		return errors.New("stack name contains path traversal")
	}
	return nil
}
