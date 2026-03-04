package grpcserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	mediav1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/media/v1"
	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/service"
)

// Server wraps the gRPC server with all registered services.
type Server struct {
	grpc *grpc.Server
	port int
}

// New creates and configures a gRPC server with all services registered.
func New(
	port int,
	maxMsgSize int,
	usersSvc *service.UsersService,
	groupsSvc *service.GroupsService,
	participantsSvc *service.ParticipantsService,
	notesSvc *service.NotesService,
	mediaSvc *service.MediaService,
) *Server {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
		grpc.ChainUnaryInterceptor(
			loggingInterceptor,
			recoveryInterceptor,
		),
	}

	grpcSrv := grpc.NewServer(opts...)

	usersv1.RegisterUsersServiceServer(grpcSrv, newUsersServer(usersSvc))
	groupsv1.RegisterGroupsServiceServer(grpcSrv, newGroupsServer(groupsSvc))
	participantsv1.RegisterParticipantsServiceServer(grpcSrv, newParticipantsServer(participantsSvc))
	notesv1.RegisterNotesServiceServer(grpcSrv, newNotesServer(notesSvc))
	mediav1.RegisterMediaServiceServer(grpcSrv, newMediaServer(mediaSvc))

	reflection.Register(grpcSrv)

	return &Server{grpc: grpcSrv, port: port}
}

// Run starts the gRPC server and blocks until it stops.
func (s *Server) Run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	slog.Info("gRPC server starting", "port", s.port)
	return s.grpc.Serve(lis)
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	s.grpc.GracefulStop()
}

func loggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		slog.Error("gRPC error", "method", info.FullMethod, "error", err)
	}
	return resp, err
}

func recoveryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in gRPC handler", "method", info.FullMethod, "panic", r)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}
