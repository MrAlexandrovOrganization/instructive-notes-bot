package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	participantsv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/participants/v1"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/service"
)

type participantsServer struct {
	participantsv1.UnimplementedParticipantsServiceServer
	svc *service.ParticipantsService
}

func newParticipantsServer(svc *service.ParticipantsService) *participantsServer {
	return &participantsServer{svc: svc}
}

func (s *participantsServer) CreateParticipant(ctx context.Context, req *participantsv1.CreateParticipantRequest) (*participantsv1.Participant, error) {
	var telegramID *int64
	if req.TelegramId != 0 {
		telegramID = &req.TelegramId
	}
	var customID *string
	if req.CustomIdentifier != "" {
		customID = &req.CustomIdentifier
	}
	var groupID *string
	if req.GroupId != "" {
		groupID = &req.GroupId
	}

	p, err := s.svc.Create(ctx, req.Name, telegramID, customID, groupID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "create participant: %v", err)
	}
	return repoParticipantToProto(p), nil
}

func (s *participantsServer) GetParticipant(ctx context.Context, req *participantsv1.GetParticipantRequest) (*participantsv1.Participant, error) {
	p, err := s.svc.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "participant not found")
		}
		return nil, status.Errorf(codes.Internal, "get participant: %v", err)
	}
	return repoParticipantToProto(p), nil
}

func (s *participantsServer) ListParticipants(ctx context.Context, req *participantsv1.ListParticipantsRequest) (*participantsv1.ListParticipantsResponse, error) {
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = int(req.Pagination.Limit)
		}
		cursor = req.Pagination.Cursor
	}

	participants, err := s.svc.List(ctx, req.GroupId, req.Search, limit, cursor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list participants: %v", err)
	}

	protoParticipants := make([]*participantsv1.Participant, 0, len(participants))
	for _, p := range participants {
		protoParticipants = append(protoParticipants, repoParticipantToProto(p))
	}

	resp := &participantsv1.ListParticipantsResponse{
		Participants: protoParticipants,
		PageInfo: &commonv1.PageInfo{
			HasNext: len(participants) == limit,
		},
	}
	if len(participants) > 0 && resp.PageInfo.HasNext {
		resp.PageInfo.NextCursor = participants[len(participants)-1].ID
	}
	return resp, nil
}

func (s *participantsServer) UpdateParticipant(ctx context.Context, req *participantsv1.UpdateParticipantRequest) (*participantsv1.Participant, error) {
	var telegramID *int64
	if req.TelegramId != 0 {
		telegramID = &req.TelegramId
	}
	var customID *string
	if req.CustomIdentifier != "" {
		customID = &req.CustomIdentifier
	}
	var groupID *string
	if req.GroupId != "" {
		groupID = &req.GroupId
	}

	p, err := s.svc.Update(ctx, req.Id, req.Name, telegramID, customID, groupID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "participant not found")
		}
		return nil, status.Errorf(codes.Internal, "update participant: %v", err)
	}
	return repoParticipantToProto(p), nil
}

func (s *participantsServer) DeleteParticipant(ctx context.Context, req *participantsv1.DeleteParticipantRequest) (*commonv1.SuccessResponse, error) {
	if err := s.svc.Delete(ctx, req.Id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "participant not found")
		}
		return nil, status.Errorf(codes.Internal, "delete participant: %v", err)
	}
	return &commonv1.SuccessResponse{Success: true}, nil
}

func (s *participantsServer) SetParticipantPhoto(ctx context.Context, req *participantsv1.SetParticipantPhotoRequest) (*participantsv1.Participant, error) {
	p, err := s.svc.SetPhoto(ctx, req.ParticipantId, req.MediaId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "participant not found")
		}
		return nil, status.Errorf(codes.Internal, "set photo: %v", err)
	}
	return repoParticipantToProto(p), nil
}

func repoParticipantToProto(p *repository.Participant) *participantsv1.Participant {
	proto := &participantsv1.Participant{
		Id:         p.ID,
		Name:       p.Name,
		NotesCount: p.NotesCount,
		CreatedAt:  p.CreatedAt.String(),
		UpdatedAt:  p.UpdatedAt.String(),
	}
	if p.TelegramID != nil {
		proto.TelegramId = *p.TelegramID
	}
	if p.CustomIdentifier != nil {
		proto.CustomIdentifier = *p.CustomIdentifier
	}
	if p.GroupID != nil {
		proto.GroupId = *p.GroupID
	}
	if p.GroupName != nil {
		proto.GroupName = *p.GroupName
	}
	if p.PhotoMediaID != nil {
		proto.PhotoMediaId = *p.PhotoMediaID
	}
	return proto
}
