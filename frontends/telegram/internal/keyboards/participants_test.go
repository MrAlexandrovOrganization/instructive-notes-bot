package keyboards_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
)

func TestParticipantsList_ShowsGroupName(t *testing.T) {
	t.Parallel()
	participants := []*participantsv1.Participant{
		{Id: "p1", Name: "Иванов", GroupName: "Отряд 1", NotesCount: 2},
		{Id: "p2", Name: "Петров", GroupName: "", NotesCount: 0},
	}
	kb := keyboards.ParticipantsList(keyboards.ParticipantsListOpts{
		Participants: participants,
		Role:         usersv1.Role_ROLE_ORGANIZER,
		BackTo:       "back:menu",
		PageSize:     10,
	})

	assert.Contains(t, kb.InlineKeyboard[0][0].Text, "Иванов")
	assert.Contains(t, kb.InlineKeyboard[0][0].Text, "Отряд 1")
	assert.Contains(t, kb.InlineKeyboard[0][0].Text, "📝")

	assert.Contains(t, kb.InlineKeyboard[1][0].Text, "Петров")
	assert.NotContains(t, kb.InlineKeyboard[1][0].Text, "(")
	assert.NotContains(t, kb.InlineKeyboard[1][0].Text, "📝")
}

func TestParticipantsList_AddButtonForAllRoles(t *testing.T) {
	t.Parallel()
	roles := []usersv1.Role{
		usersv1.Role_ROLE_ORGANIZER,
		usersv1.Role_ROLE_CURATOR,
		usersv1.Role_ROLE_ADMIN,
		usersv1.Role_ROLE_ROOT,
	}
	for _, role := range roles {
		kb := keyboards.ParticipantsList(keyboards.ParticipantsListOpts{
			Role:     role,
			BackTo:   "back:menu",
			PageSize: 10,
		})
		found := false
		for _, row := range kb.InlineKeyboard {
			for _, btn := range row {
				if *btn.CallbackData == "participant:add" {
					found = true
				}
			}
		}
		assert.True(t, found, "role %v should have add button", role)
	}
}

func TestParticipantsList_BackButtonUsesBackTo(t *testing.T) {
	t.Parallel()
	kb := keyboards.ParticipantsList(keyboards.ParticipantsListOpts{
		Role:     usersv1.Role_ROLE_ORGANIZER,
		BackTo:   "back:groups",
		PageSize: 10,
	})
	lastRow := kb.InlineKeyboard[len(kb.InlineKeyboard)-1]
	assert.Equal(t, "back:groups", *lastRow[0].CallbackData)
}

func TestParticipantsList_Pagination(t *testing.T) {
	t.Parallel()
	participants := make([]*participantsv1.Participant, 10)
	for i := range participants {
		participants[i] = &participantsv1.Participant{Id: "p", Name: "Name"}
	}

	t.Run("first page with next", func(t *testing.T) {
		kb := keyboards.ParticipantsList(keyboards.ParticipantsListOpts{
			Participants: participants,
			Role:         usersv1.Role_ROLE_ORGANIZER,
			BackTo:       "back:menu",
			Offset:       0,
			HasNext:      true,
			PageSize:     10,
		})
		// Find nav row — should have only "Далее".
		navRow := kb.InlineKeyboard[10]
		require.Len(t, navRow, 1)
		assert.Equal(t, "page:participants:10", *navRow[0].CallbackData)
	})

	t.Run("middle page both buttons", func(t *testing.T) {
		kb := keyboards.ParticipantsList(keyboards.ParticipantsListOpts{
			Participants: participants,
			Role:         usersv1.Role_ROLE_ORGANIZER,
			BackTo:       "back:menu",
			Offset:       10,
			HasNext:      true,
			PageSize:     10,
		})
		navRow := kb.InlineKeyboard[10]
		require.Len(t, navRow, 2)
		assert.Equal(t, "page:participants:0", *navRow[0].CallbackData)
		assert.Equal(t, "page:participants:20", *navRow[1].CallbackData)
	})

	t.Run("last page only prev", func(t *testing.T) {
		p := []*participantsv1.Participant{{Id: "p", Name: "Last"}}
		kb := keyboards.ParticipantsList(keyboards.ParticipantsListOpts{
			Participants: p,
			Role:         usersv1.Role_ROLE_ORGANIZER,
			BackTo:       "back:menu",
			Offset:       10,
			HasNext:      false,
			PageSize:     10,
		})
		navRow := kb.InlineKeyboard[1]
		require.Len(t, navRow, 1)
		assert.Equal(t, "page:participants:0", *navRow[0].CallbackData)
	})
}

func TestParticipantView_HasNotesAndPhotoButtons(t *testing.T) {
	t.Parallel()
	kb := keyboards.ParticipantView("p1", usersv1.Role_ROLE_ORGANIZER)

	var callbacks []string
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			callbacks = append(callbacks, *btn.CallbackData)
		}
	}
	assert.Contains(t, callbacks, "notes:participant:p1")
	assert.Contains(t, callbacks, "participant:photo:p1")
	assert.Contains(t, callbacks, "back:participants_list")
	for _, cb := range callbacks {
		assert.NotContains(t, cb, "participant:edit:")
		assert.NotContains(t, cb, "participant:delete:")
	}
}

func TestGroupsListForAssign_AllParticipantsFirst(t *testing.T) {
	t.Parallel()
	groups := []*groupsv1.Group{
		{Id: "g1", Name: "Отряд 1"},
		{Id: "g2", Name: "Отряд 2"},
	}
	kb := keyboards.GroupsListForAssign(groups, "g2", "note-1")

	assert.Equal(t, "group:for_note:all", *kb.InlineKeyboard[0][0].CallbackData)
	assert.Contains(t, kb.InlineKeyboard[1][0].Text, "⭐")
	assert.Equal(t, "group:for_note:g2", *kb.InlineKeyboard[1][0].CallbackData)
	assert.Equal(t, "group:for_note:g1", *kb.InlineKeyboard[2][0].CallbackData)
	lastRow := kb.InlineKeyboard[len(kb.InlineKeyboard)-1]
	assert.Equal(t, "note:view:note-1", *lastRow[0].CallbackData)
}

func TestSelectParticipantForNote_BackGoesToAssign(t *testing.T) {
	t.Parallel()
	participants := []*participantsv1.Participant{
		{Id: "p1", Name: "Иванов", GroupName: "Отряд 1"},
	}
	kb := keyboards.SelectParticipantForNote(keyboards.SelectParticipantOpts{
		Participants: participants,
		NoteID:       "note-1",
		PageSize:     20,
	})

	assert.Contains(t, kb.InlineKeyboard[0][0].Text, "Иванов")
	assert.Contains(t, kb.InlineKeyboard[0][0].Text, "Отряд 1")
	lastRow := kb.InlineKeyboard[len(kb.InlineKeyboard)-1]
	assert.Equal(t, "note:assign:note-1", *lastRow[0].CallbackData)
}

func TestSelectParticipantForNote_Pagination(t *testing.T) {
	t.Parallel()
	participants := make([]*participantsv1.Participant, 20)
	for i := range participants {
		participants[i] = &participantsv1.Participant{Id: "p", Name: "Name"}
	}
	kb := keyboards.SelectParticipantForNote(keyboards.SelectParticipantOpts{
		Participants: participants,
		NoteID:       "note-1",
		Offset:       20,
		HasNext:      true,
		PageSize:     20,
	})
	// Find nav row (after 20 participant rows).
	navRow := kb.InlineKeyboard[20]
	require.Len(t, navRow, 2)
	assert.Equal(t, "page:assign_participant:0", *navRow[0].CallbackData)
	assert.Equal(t, "page:assign_participant:40", *navRow[1].CallbackData)
}

func TestGroupsListForBrowse_ClickableGroups(t *testing.T) {
	t.Parallel()
	groups := []*groupsv1.Group{
		{Id: "g1", Name: "Отряд 1", Description: "Младшие"},
		{Id: "g2", Name: "Отряд 2"},
	}
	kb := keyboards.GroupsListForBrowse(groups)

	assert.Equal(t, "group:view:g1", *kb.InlineKeyboard[0][0].CallbackData)
	assert.Contains(t, kb.InlineKeyboard[0][0].Text, "Младшие")
	assert.Equal(t, "group:view:g2", *kb.InlineKeyboard[1][0].CallbackData)
	require.Len(t, kb.InlineKeyboard, 4)
	assert.Equal(t, "admin:add_group", *kb.InlineKeyboard[2][0].CallbackData)
	assert.Equal(t, "back:admin", *kb.InlineKeyboard[3][0].CallbackData)
}
