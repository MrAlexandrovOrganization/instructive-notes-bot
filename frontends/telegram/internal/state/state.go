package state

// UserState represents the current conversational state of a user.
type UserState int

const (
	StateIdle UserState = iota
	StateSelectingParticipantForNote
	StateWritingNoteText
	StateAssigningNoteToParticipant
	StateUploadingPhoto
	StateAddingParticipantName
	StateAddingParticipantGroup
	StateAddingUserName
	StateAddingUserTelegramID
	StateAddingGroupName
)

// UserContext holds transient state data for a user's ongoing conversation.
type UserContext struct {
	State         UserState
	PendingNoteID string // for assigning an existing note to participant
	PendingData   string // generic pending field (e.g., participant name during creation)
}
