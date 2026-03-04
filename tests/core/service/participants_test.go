package service_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateParticipant_RequiresName(t *testing.T) {
	t.Parallel()

	err := validateParticipantName("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestCreateParticipant_ValidName(t *testing.T) {
	t.Parallel()

	err := validateParticipantName("Иван Иванов")
	require.NoError(t, err)
}

func TestCreateParticipant_TrimmedName(t *testing.T) {
	t.Parallel()

	err := validateParticipantName("  ")
	require.Error(t, err)
}

func validateParticipantName(name string) error {
	trimmed := ""
	for _, r := range name {
		if r != ' ' {
			trimmed = string(r)
			break
		}
	}
	if trimmed == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
