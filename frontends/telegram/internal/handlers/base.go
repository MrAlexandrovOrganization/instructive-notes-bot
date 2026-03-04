package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/client"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/state"
)

// Base holds shared dependencies for all handlers.
type Base struct {
	Bot     *tgbotapi.BotAPI
	Clients *client.Clients
	States  *state.Manager
}

// NewBase creates a new Base with shared dependencies.
func NewBase(bot *tgbotapi.BotAPI, clients *client.Clients, states *state.Manager) *Base {
	return &Base{
		Bot:     bot,
		Clients: clients,
		States:  states,
	}
}
