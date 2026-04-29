package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
)

func TestPageInfoFields(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		total, hasNext := pageInfoFields(nil)
		assert.Equal(t, int32(0), total)
		assert.False(t, hasNext)
	})

	t.Run("has_next", func(t *testing.T) {
		pi := &commonv1.PageInfo{Total: 25, HasNext: true}
		total, hasNext := pageInfoFields(pi)
		assert.Equal(t, int32(25), total)
		assert.True(t, hasNext)
	})

	t.Run("no_next", func(t *testing.T) {
		pi := &commonv1.PageInfo{Total: 5, HasNext: false}
		total, hasNext := pageInfoFields(pi)
		assert.Equal(t, int32(5), total)
		assert.False(t, hasNext)
	})
}

func TestNotesTitle(t *testing.T) {
	t.Parallel()
	h := &NotesHandler{}

	tests := []struct {
		ctx      state.NotesContext
		expected string
	}{
		{state.NotesCtxMy, "📋 *Мои заметки*"},
		{state.NotesCtxAll, "📊 *Все заметки*"},
		{state.NotesCtxUnassigned, "📄 *Заметки без участника*"},
		{state.NotesCtxParticipant, "📋 *Заметки по участнику*"},
	}
	for _, tt := range tests {
		uc := &state.UserContext{NotesCtx: tt.ctx}
		assert.Equal(t, tt.expected, h.notesTitle(uc), "context: %s", tt.ctx)
	}
}

func TestNotesBackTo(t *testing.T) {
	t.Parallel()
	h := &NotesHandler{}

	tests := []struct {
		ctx      state.NotesContext
		data     string
		expected string
	}{
		{state.NotesCtxMy, "", "back:menu"},
		{state.NotesCtxAll, "", "back:menu"},
		{state.NotesCtxUnassigned, "", "back:notes"},
		{state.NotesCtxParticipant, "p-123", "back:participant:p-123"},
	}
	for _, tt := range tests {
		uc := &state.UserContext{NotesCtx: tt.ctx, PendingData: tt.data}
		assert.Equal(t, tt.expected, h.notesBackTo(uc), "context: %s", tt.ctx)
	}
}

func TestBuildListRequest_MyNotes(t *testing.T) {
	t.Parallel()
	h := &NotesHandler{}
	user := &usersv1.User{Id: "user-1", Role: usersv1.Role_ROLE_ORGANIZER}

	req := h.buildListRequest(user, state.NotesCtxMy, "", 0)
	assert.Equal(t, "user-1", req.AuthorId)
	assert.False(t, req.AllNotes)
	assert.Equal(t, int32(0), req.Pagination.Offset)
}

func TestBuildListRequest_AllNotes(t *testing.T) {
	t.Parallel()
	h := &NotesHandler{}
	user := &usersv1.User{Id: "user-1", Role: usersv1.Role_ROLE_ADMIN}

	req := h.buildListRequest(user, state.NotesCtxAll, "", 16)
	assert.Empty(t, req.AuthorId)
	assert.True(t, req.AllNotes)
	assert.Equal(t, int32(16), req.Pagination.Offset)
}

func TestBuildListRequest_UnassignedAsOrganizer(t *testing.T) {
	t.Parallel()
	h := &NotesHandler{}
	user := &usersv1.User{Id: "user-1", Role: usersv1.Role_ROLE_ORGANIZER}

	req := h.buildListRequest(user, state.NotesCtxUnassigned, "", 0)
	assert.Equal(t, "user-1", req.AuthorId)
	assert.True(t, req.UnassignedOnly)
	assert.False(t, req.AllNotes)
}

func TestBuildListRequest_UnassignedAsAdmin(t *testing.T) {
	t.Parallel()
	h := &NotesHandler{}
	user := &usersv1.User{Id: "admin-1", Role: usersv1.Role_ROLE_ADMIN}

	req := h.buildListRequest(user, state.NotesCtxUnassigned, "", 0)
	assert.Empty(t, req.AuthorId)
	assert.True(t, req.UnassignedOnly)
	assert.True(t, req.AllNotes)
}

func TestBuildListRequest_ParticipantAsOrganizer(t *testing.T) {
	t.Parallel()
	h := &NotesHandler{}
	user := &usersv1.User{Id: "user-1", Role: usersv1.Role_ROLE_ORGANIZER}

	req := h.buildListRequest(user, state.NotesCtxParticipant, "p-123", 0)
	assert.Equal(t, "user-1", req.AuthorId)
	assert.Equal(t, "p-123", req.ParticipantId)
	assert.False(t, req.AllNotes)
}

func TestBuildListRequest_ParticipantAsRoot(t *testing.T) {
	t.Parallel()
	h := &NotesHandler{}
	user := &usersv1.User{Id: "root-1", Role: usersv1.Role_ROLE_ROOT}

	req := h.buildListRequest(user, state.NotesCtxParticipant, "p-123", 0)
	assert.Empty(t, req.AuthorId)
	assert.Equal(t, "p-123", req.ParticipantId)
	assert.True(t, req.AllNotes)
}

func TestEscapeMarkdown(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello", EscapeMarkdown("hello"))
	assert.Equal(t, "a\\.b", EscapeMarkdown("a.b"))
	assert.Equal(t, "a\\*b\\*c", EscapeMarkdown("a*b*c"))
	assert.Equal(t, "test\\_under", EscapeMarkdown("test_under"))
}
