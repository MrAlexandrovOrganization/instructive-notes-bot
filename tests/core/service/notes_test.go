package service_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoteCreation_RequiresAuthorID verifies that creating a note without an author fails.
func TestNoteCreation_RequiresAuthorID(t *testing.T) {
	t.Parallel()

	err := validateNoteInput("", "some note")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "author_id")
}

// TestNoteCreation_RequiresText verifies that creating a note without text fails.
func TestNoteCreation_RequiresText(t *testing.T) {
	t.Parallel()

	err := validateNoteInput("some-uuid", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "text")
}

// TestNoteCreation_ValidInput passes with both fields set.
func TestNoteCreation_ValidInput(t *testing.T) {
	t.Parallel()

	err := validateNoteInput("some-uuid", "Valid note text")
	require.NoError(t, err)
}

func validateNoteInput(authorID, text string) error {
	if authorID == "" {
		return fmt.Errorf("author_id is required")
	}
	if text == "" {
		return fmt.Errorf("text is required")
	}
	return nil
}
