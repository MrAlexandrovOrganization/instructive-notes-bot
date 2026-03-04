package state_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
)

func TestStateManager_DefaultStateIsIdle(t *testing.T) {
	t.Parallel()

	mgr := state.NewManager()
	ctx := mgr.Get(12345)
	assert.Equal(t, state.StateIdle, ctx.State)
}

func TestStateManager_SetAndGet(t *testing.T) {
	t.Parallel()

	mgr := state.NewManager()
	mgr.Set(42, &state.UserContext{
		State:       state.StateWritingNoteText,
		PendingData: "participant-id",
	})

	ctx := mgr.Get(42)
	assert.Equal(t, state.StateWritingNoteText, ctx.State)
	assert.Equal(t, "participant-id", ctx.PendingData)
}

func TestStateManager_Reset(t *testing.T) {
	t.Parallel()

	mgr := state.NewManager()
	mgr.Set(99, &state.UserContext{State: state.StateUploadingPhoto, PendingData: "some-id"})
	mgr.Reset(99)

	ctx := mgr.Get(99)
	assert.Equal(t, state.StateIdle, ctx.State)
	assert.Empty(t, ctx.PendingData)
}

func TestStateManager_SetState(t *testing.T) {
	t.Parallel()

	mgr := state.NewManager()
	mgr.SetState(77, state.StateAddingGroupName)

	ctx := mgr.Get(77)
	assert.Equal(t, state.StateAddingGroupName, ctx.State)
}

func TestStateManager_Concurrent(t *testing.T) {
	t.Parallel()

	mgr := state.NewManager()
	done := make(chan struct{}, 10)

	for i := 0; i < 10; i++ {
		go func(id int64) {
			mgr.Set(id, &state.UserContext{State: state.StateWritingNoteText})
			_ = mgr.Get(id)
			mgr.Reset(id)
			done <- struct{}{}
		}(int64(i))
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
