package middleware

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/client"
)

// AuthMiddleware resolves a Telegram user to a system user via gRPC.
type AuthMiddleware struct {
	clients        *client.Clients
	rootTelegramID int64
}

// NewAuthMiddleware creates a new AuthMiddleware.
func NewAuthMiddleware(clients *client.Clients, rootTelegramID int64) *AuthMiddleware {
	return &AuthMiddleware{
		clients:        clients,
		rootTelegramID: rootTelegramID,
	}
}

// ResolveUser looks up or creates a user from a Telegram update.
// Returns nil if the user has no access (not registered and not ROOT).
func (m *AuthMiddleware) ResolveUser(ctx context.Context, from *tgbotapi.User) (*usersv1.User, error) {
	telegramID := from.ID

	// Determine the role to assign on first creation.
	defaultRole := usersv1.Role_ROLE_ORGANIZER
	if telegramID == m.rootTelegramID {
		defaultRole = usersv1.Role_ROLE_ROOT
	}

	resp, err := m.clients.Users.GetOrCreateUser(ctx, &usersv1.GetOrCreateUserRequest{
		TelegramId: telegramID,
		Name:       from.FirstName + " " + from.LastName,
		Username:   from.UserName,
	})
	if err != nil {
		return nil, err
	}

	user := resp.User

	// If user was just created and is root, update role.
	if resp.Created && defaultRole == usersv1.Role_ROLE_ROOT {
		updatedUser, err := m.clients.Users.UpdateUserRole(ctx, &usersv1.UpdateUserRoleRequest{
			Id:   user.Id,
			Role: usersv1.Role_ROLE_ROOT,
		})
		if err != nil {
			slog.Error("update root role", "error", err)
			return user, nil
		}
		return updatedUser, nil
	}

	// Non-root users who were just created have no access — they're "organizer" by default
	// but in this system every registered user has access. Only truly unregistered users
	// (new organizers who haven't been added by admin) should be blocked.
	// The bootstrap approach: ROOT auto-creates on /start. Others must be added by admin.
	// Since GetOrCreate always creates, we check if the new user is NOT root and was just created.
	if resp.Created && defaultRole != usersv1.Role_ROLE_ROOT {
		// New user, not root — delete them and return nil (no access).
		_, delErr := m.clients.Users.DeleteUser(ctx, &usersv1.DeleteUserRequest{Id: user.Id})
		if delErr != nil {
			slog.Error("delete unauthorized user", "error", delErr)
		}
		return nil, nil
	}

	return user, nil
}

// IsKnownUser checks if a user already exists without creating them.
func (m *AuthMiddleware) IsKnownUser(ctx context.Context, telegramID int64) (*usersv1.User, error) {
	u, err := m.clients.Users.GetUserByTelegramID(ctx, &usersv1.GetUserByTelegramIDRequest{
		TelegramId: telegramID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}
