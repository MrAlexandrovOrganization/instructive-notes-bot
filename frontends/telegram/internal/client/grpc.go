package client

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	mediav1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/media/v1"
	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
)

// Clients holds all gRPC client stubs.
type Clients struct {
	conn         *grpc.ClientConn
	Users        usersv1.UsersServiceClient
	Groups       groupsv1.GroupsServiceClient
	Participants participantsv1.ParticipantsServiceClient
	Notes        notesv1.NotesServiceClient
	Media        mediav1.MediaServiceClient
}

// New creates gRPC client connections to the core service.
func New(addr string) (*Clients, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(52428800),
			grpc.MaxCallSendMsgSize(52428800),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("dial grpc: %w", err)
	}
	return &Clients{
		conn:         conn,
		Users:        usersv1.NewUsersServiceClient(conn),
		Groups:       groupsv1.NewGroupsServiceClient(conn),
		Participants: participantsv1.NewParticipantsServiceClient(conn),
		Notes:        notesv1.NewNotesServiceClient(conn),
		Media:        mediav1.NewMediaServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Clients) Close() error {
	return c.conn.Close()
}
