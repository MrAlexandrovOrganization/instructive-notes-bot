package keyboards_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/keyboards"
)

func callbacksFrom(t *testing.T, role usersv1.Role) []string {
	t.Helper()
	kb := keyboards.MainMenu(role)
	var cbs []string
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			cbs = append(cbs, *btn.CallbackData)
		}
	}
	return cbs
}

func TestMainMenu_AdminHasGroups(t *testing.T) {
	t.Parallel()
	cbs := callbacksFrom(t, usersv1.Role_ROLE_ADMIN)
	assert.Contains(t, cbs, "menu:groups")
	assert.Contains(t, cbs, "menu:all_notes")
	assert.Contains(t, cbs, "menu:admin")
	assert.Contains(t, cbs, "menu:participants")
}

func TestMainMenu_RootSameAsAdmin(t *testing.T) {
	t.Parallel()
	admin := callbacksFrom(t, usersv1.Role_ROLE_ADMIN)
	root := callbacksFrom(t, usersv1.Role_ROLE_ROOT)
	assert.Equal(t, admin, root)
}

func TestMainMenu_CuratorHasMyGroupAndGroups(t *testing.T) {
	t.Parallel()
	cbs := callbacksFrom(t, usersv1.Role_ROLE_CURATOR)
	assert.Contains(t, cbs, "menu:my_group")
	assert.Contains(t, cbs, "menu:groups")
	assert.Contains(t, cbs, "menu:notes")
	assert.NotContains(t, cbs, "menu:admin")
}

func TestMainMenu_OrganizerHasGroups(t *testing.T) {
	t.Parallel()
	cbs := callbacksFrom(t, usersv1.Role_ROLE_ORGANIZER)
	assert.Contains(t, cbs, "menu:participants")
	assert.Contains(t, cbs, "menu:groups")
	assert.Contains(t, cbs, "menu:notes")
	assert.NotContains(t, cbs, "menu:admin")
}

func TestAdminPanel_HasUsersAndGroups(t *testing.T) {
	t.Parallel()
	kb := keyboards.AdminPanel()
	var cbs []string
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			cbs = append(cbs, *btn.CallbackData)
		}
	}
	assert.Contains(t, cbs, "admin:users")
	assert.Contains(t, cbs, "admin:groups")
	assert.Contains(t, cbs, "back:menu")
}

func TestCancelInline(t *testing.T) {
	t.Parallel()
	kb := keyboards.CancelInline()
	assert.Len(t, kb.InlineKeyboard, 1)
	assert.Equal(t, "cancel", *kb.InlineKeyboard[0][0].CallbackData)
}
