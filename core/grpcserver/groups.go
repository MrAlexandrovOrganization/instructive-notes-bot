package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	groupsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/groups/v1"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/service"
)

type groupsServer struct {
	groupsv1.UnimplementedGroupsServiceServer
	svc *service.GroupsService
}

func newGroupsServer(svc *service.GroupsService) *groupsServer {
	return &groupsServer{svc: svc}
}

func (s *groupsServer) CreateGroup(ctx context.Context, req *groupsv1.CreateGroupRequest) (*groupsv1.Group, error) {
	g, err := s.svc.Create(ctx, req.Name, req.Description)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "create group: %v", err)
	}
	return repoGroupToProto(g), nil
}

func (s *groupsServer) GetGroup(ctx context.Context, req *groupsv1.GetGroupRequest) (*groupsv1.Group, error) {
	g, err := s.svc.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "group not found")
		}
		return nil, status.Errorf(codes.Internal, "get group: %v", err)
	}
	return repoGroupToProto(g), nil
}

func (s *groupsServer) ListGroups(ctx context.Context, req *groupsv1.ListGroupsRequest) (*groupsv1.ListGroupsResponse, error) {
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = int(req.Pagination.Limit)
		}
		cursor = req.Pagination.Cursor
	}

	groups, err := s.svc.List(ctx, limit, cursor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list groups: %v", err)
	}

	protoGroups := make([]*groupsv1.Group, 0, len(groups))
	for _, g := range groups {
		protoGroups = append(protoGroups, repoGroupToProto(g))
	}

	resp := &groupsv1.ListGroupsResponse{
		Groups: protoGroups,
		PageInfo: &commonv1.PageInfo{
			HasNext: len(groups) == limit,
		},
	}
	if len(groups) > 0 && resp.PageInfo.HasNext {
		resp.PageInfo.NextCursor = groups[len(groups)-1].ID
	}
	return resp, nil
}

func (s *groupsServer) UpdateGroup(ctx context.Context, req *groupsv1.UpdateGroupRequest) (*groupsv1.Group, error) {
	g, err := s.svc.Update(ctx, req.Id, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "group not found")
		}
		return nil, status.Errorf(codes.Internal, "update group: %v", err)
	}
	return repoGroupToProto(g), nil
}

func (s *groupsServer) DeleteGroup(ctx context.Context, req *groupsv1.DeleteGroupRequest) (*commonv1.SuccessResponse, error) {
	if err := s.svc.Delete(ctx, req.Id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "group not found")
		}
		return nil, status.Errorf(codes.Internal, "delete group: %v", err)
	}
	return &commonv1.SuccessResponse{Success: true}, nil
}

func repoGroupToProto(g *repository.Group) *groupsv1.Group {
	return &groupsv1.Group{
		Id:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		CreatedAt:   g.CreatedAt.String(),
	}
}
