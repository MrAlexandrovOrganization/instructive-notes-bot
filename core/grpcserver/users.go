package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	usersv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/users/v1"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/service"
)

type usersServer struct {
	usersv1.UnimplementedUsersServiceServer
	svc *service.UsersService
}

func newUsersServer(svc *service.UsersService) *usersServer {
	return &usersServer{svc: svc}
}

func (s *usersServer) GetOrCreateUser(ctx context.Context, req *usersv1.GetOrCreateUserRequest) (*usersv1.GetOrCreateUserResponse, error) {
	u, created, err := s.svc.GetOrCreate(ctx, req.TelegramId, req.Name, req.Username, "organizer")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get or create user: %v", err)
	}
	return &usersv1.GetOrCreateUserResponse{
		User:    repoUserToProto(u),
		Created: created,
	}, nil
}

func (s *usersServer) GetUser(ctx context.Context, req *usersv1.GetUserRequest) (*usersv1.User, error) {
	u, err := s.svc.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}
	return repoUserToProto(u), nil
}

func (s *usersServer) GetUserByTelegramID(ctx context.Context, req *usersv1.GetUserByTelegramIDRequest) (*usersv1.User, error) {
	u, err := s.svc.GetByTelegramID(ctx, req.TelegramId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}
	return repoUserToProto(u), nil
}

func (s *usersServer) ListUsers(ctx context.Context, req *usersv1.ListUsersRequest) (*usersv1.ListUsersResponse, error) {
	limit := 20
	offset := 0
	roleFilter := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = int(req.Pagination.Limit)
		}
		offset = int(req.Pagination.Offset)
	}
	if req.RoleFilter != usersv1.Role_ROLE_UNSPECIFIED {
		roleFilter = protoRoleToString(req.RoleFilter)
	}

	users, err := s.svc.List(ctx, roleFilter, limit+1, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list users: %v", err)
	}

	hasNext := len(users) > limit
	if hasNext {
		users = users[:limit]
	}

	protoUsers := make([]*usersv1.User, 0, len(users))
	for _, u := range users {
		protoUsers = append(protoUsers, repoUserToProto(u))
	}

	resp := &usersv1.ListUsersResponse{
		Users: protoUsers,
		PageInfo: &commonv1.PageInfo{
			HasNext: hasNext,
		},
	}
	return resp, nil
}

func (s *usersServer) UpdateUserRole(ctx context.Context, req *usersv1.UpdateUserRoleRequest) (*usersv1.User, error) {
	u, err := s.svc.UpdateRole(ctx, req.Id, protoRoleToString(req.Role))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "update role: %v", err)
	}
	return repoUserToProto(u), nil
}

func (s *usersServer) AssignCuratorGroup(ctx context.Context, req *usersv1.AssignCuratorGroupRequest) (*usersv1.User, error) {
	u, err := s.svc.AssignGroup(ctx, req.UserId, req.GroupId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "assign group: %v", err)
	}
	return repoUserToProto(u), nil
}

func (s *usersServer) DeleteUser(ctx context.Context, req *usersv1.DeleteUserRequest) (*commonv1.SuccessResponse, error) {
	if err := s.svc.Delete(ctx, req.Id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "delete user: %v", err)
	}
	return &commonv1.SuccessResponse{Success: true}, nil
}

func repoUserToProto(u *repository.User) *usersv1.User {
	proto := &usersv1.User{
		Id:         u.ID,
		TelegramId: u.TelegramID,
		Name:       u.Name,
		Username:   u.Username,
		Role:       stringRoleToProto(u.Role),
		CreatedAt:  u.CreatedAt.String(),
		UpdatedAt:  u.UpdatedAt.String(),
	}
	if u.GroupID != nil {
		proto.GroupId = *u.GroupID
	}
	return proto
}

func protoRoleToString(r usersv1.Role) string {
	switch r {
	case usersv1.Role_ROLE_ORGANIZER:
		return "organizer"
	case usersv1.Role_ROLE_CURATOR:
		return "curator"
	case usersv1.Role_ROLE_ADMIN:
		return "admin"
	case usersv1.Role_ROLE_ROOT:
		return "root"
	default:
		return "organizer"
	}
}

func stringRoleToProto(r string) usersv1.Role {
	switch r {
	case "organizer":
		return usersv1.Role_ROLE_ORGANIZER
	case "curator":
		return usersv1.Role_ROLE_CURATOR
	case "admin":
		return usersv1.Role_ROLE_ADMIN
	case "root":
		return usersv1.Role_ROLE_ROOT
	default:
		return usersv1.Role_ROLE_UNSPECIFIED
	}
}
