package state

// UserState represents the current conversational state of a user.
type UserState int

const (
	StateIdle UserState = iota
	StateWritingNoteText
	StateAssigningNoteToParticipant
	StateUploadingPhoto
	StateAddingParticipantName
	StateAddingParticipantGroup
	StateAddingUserName
	StateAddingUserTelegramID
	StateAddingUserRole
	StateAddingGroupName
)

// NotesContext describes which notes list the user is currently viewing.
type NotesContext string

const (
	NotesCtxMy          NotesContext = "my"
	NotesCtxAll         NotesContext = "all"
	NotesCtxUnassigned  NotesContext = "unassigned"
	NotesCtxParticipant NotesContext = "participant" // PendingData holds participant ID
)

// UserContext holds transient state data for a user's ongoing conversation.
type UserContext struct {
	State         UserState
	PendingNoteID string // for assigning an existing note to participant
	PendingData   string // primary pending field (e.g., name during creation)
	PendingData2  string // secondary pending field (e.g., telegramID during user creation)

	// Pagination state for notes lists.
	NotesCtx    NotesContext // which notes list is active
	PageCursors []string     // stack of cursors for each visited page (index 0 = page 0 cursor "")
}
