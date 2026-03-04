package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/common/v1"
	mediav1 "github.com/mrralexandrov/instructive-notes-bot/gen/go/media/v1"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/service"
)

type mediaServer struct {
	mediav1.UnimplementedMediaServiceServer
	svc *service.MediaService
}

func newMediaServer(svc *service.MediaService) *mediaServer {
	return &mediaServer{svc: svc}
}

func (s *mediaServer) UploadMedia(ctx context.Context, req *mediav1.UploadMediaRequest) (*mediav1.Media, error) {
	m, err := s.svc.Upload(ctx, req.Data, req.MimeType, req.OriginalName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upload media: %v", err)
	}
	return repoMediaToProto(m), nil
}

func (s *mediaServer) GetMedia(ctx context.Context, req *mediav1.GetMediaRequest) (*mediav1.GetMediaResponse, error) {
	m, data, err := s.svc.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get media: %v", err)
	}
	return &mediav1.GetMediaResponse{
		Media: repoMediaToProto(m),
		Data:  data,
	}, nil
}

func (s *mediaServer) DeleteMedia(ctx context.Context, req *mediav1.DeleteMediaRequest) (*commonv1.SuccessResponse, error) {
	if err := s.svc.Delete(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "delete media: %v", err)
	}
	return &commonv1.SuccessResponse{Success: true}, nil
}

func repoMediaToProto(m *service.MediaRecord) *mediav1.Media {
	return &mediav1.Media{
		Id:           m.ID,
		FilePath:     m.FilePath,
		MimeType:     m.MimeType,
		OriginalName: m.OriginalName,
		SizeBytes:    m.SizeBytes,
		CreatedAt:    m.CreatedAt.String(),
	}
}
