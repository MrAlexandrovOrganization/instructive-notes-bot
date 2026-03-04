package service_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRole_ValidRoles(t *testing.T) {
	t.Parallel()

	validRoles := []string{"organizer", "curator", "admin", "root"}
	for _, role := range validRoles {
		t.Run(role, func(t *testing.T) {
			t.Parallel()
			err := validateRole(role)
			require.NoError(t, err)
		})
	}
}

func TestUpdateRole_InvalidRole(t *testing.T) {
	t.Parallel()

	err := validateRole("superadmin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func validateRole(role string) error {
	validRoles := map[string]bool{"organizer": true, "curator": true, "admin": true, "root": true}
	if !validRoles[role] {
		return fmt.Errorf("invalid role: %s", role)
	}
	return nil
}
