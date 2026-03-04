package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	notesv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/notes/v1"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/service"
)

type notesServer struct {
	notesv1.UnimplementedNotesServiceServer
	svc *service.NotesService
}

func newNotesServer(svc *service.NotesService) *notesServer {
	return &notesServer{svc: svc}
}

func (s *notesServer) CreateNote(ctx context.Context, req *notesv1.CreateNoteRequest) (*notesv1.Note, error) {
	var participantID *string
	if req.ParticipantId != "" {
		participantID = &req.ParticipantId
	}
	n, err := s.svc.Create(ctx, req.AuthorId, participantID, req.Text)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "create note: %v", err)
	}
	return repoNoteToProto(n), nil
}

func (s *notesServer) GetNote(ctx context.Context, req *notesv1.GetNoteRequest) (*notesv1.Note, error) {
	n, err := s.svc.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "note not found")
		}
		return nil, status.Errorf(codes.Internal, "get note: %v", err)
	}
	return repoNoteToProto(n), nil
}

func (s *notesServer) ListNotes(ctx context.Context, req *notesv1.ListNotesRequest) (*notesv1.ListNotesResponse, error) {
	f := repository.ListFilter{
		AuthorID:       req.AuthorId,
		ParticipantID:  req.ParticipantId,
		UnassignedOnly: req.UnassignedOnly,
		AllNotes:       req.AllNotes,
		Limit:          20,
	}
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			f.Limit = int(req.Pagination.Limit)
		}
		f.Cursor = req.Pagination.Cursor
	}

	notes, err := s.svc.List(ctx, f)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list notes: %v", err)
	}

	protoNotes := make([]*notesv1.Note, 0, len(notes))
	for _, n := range notes {
		protoNotes = append(protoNotes, repoNoteToProto(n))
	}

	resp := &notesv1.ListNotesResponse{
		Notes: protoNotes,
		PageInfo: &commonv1.PageInfo{
			HasNext: len(notes) == f.Limit,
		},
	}
	if len(notes) > 0 && resp.PageInfo.HasNext {
		resp.PageInfo.NextCursor = notes[len(notes)-1].ID
	}
	return resp, nil
}

func (s *notesServer) UpdateNote(ctx context.Context, req *notesv1.UpdateNoteRequest) (*notesv1.Note, error) {
	n, err := s.svc.Update(ctx, req.Id, req.Text)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "note not found")
		}
		return nil, status.Errorf(codes.Internal, "update note: %v", err)
	}
	return repoNoteToProto(n), nil
}

func (s *notesServer) DeleteNote(ctx context.Context, req *notesv1.DeleteNoteRequest) (*commonv1.SuccessResponse, error) {
	if err := s.svc.Delete(ctx, req.Id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "note not found")
		}
		return nil, status.Errorf(codes.Internal, "delete note: %v", err)
	}
	return &commonv1.SuccessResponse{Success: true}, nil
}

func (s *notesServer) AssignNoteToParticipant(ctx context.Context, req *notesv1.AssignNoteToParticipantRequest) (*notesv1.Note, error) {
	n, err := s.svc.AssignToParticipant(ctx, req.NoteId, req.ParticipantId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "note not found")
		}
		return nil, status.Errorf(codes.Internal, "assign note: %v", err)
	}
	return repoNoteToProto(n), nil
}

func repoNoteToProto(n *repository.Note) *notesv1.Note {
	proto := &notesv1.Note{
		Id:         n.ID,
		AuthorId:   n.AuthorID,
		AuthorName: n.AuthorName,
		Text:       n.Text,
		CreatedAt:  n.CreatedAt.String(),
		UpdatedAt:  n.UpdatedAt.String(),
	}
	if n.ParticipantID != nil {
		proto.ParticipantId = *n.ParticipantID
	}
	if n.ParticipantName != nil {
		proto.ParticipantName = *n.ParticipantName
	}
	return proto
}
