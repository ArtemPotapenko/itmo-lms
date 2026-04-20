package grpctransport

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	documentv1 "itmo-lms/document-service/gen"
	"itmo-lms/document-service/internal/application"
	"itmo-lms/document-service/internal/domain"
)

type Server struct {
	documentv1.UnimplementedDocumentServiceServer
	service *application.Service
}

func New(service *application.Service) *Server { return &Server{service: service} }

func (s *Server) CompileDocument(ctx context.Context, req *documentv1.CompileDocumentRequest) (*documentv1.DocumentJobReply, error) {
	tasks := make([]domain.DocumentTask, 0, len(req.GetTasks()))
	for _, item := range req.GetTasks() {
		tasks = append(tasks, domain.DocumentTask{ID: item.GetId(), Title: item.GetTitle(), LatexBody: item.GetLatexBody()})
	}
	job, err := s.service.Compile(ctx, domain.CompileRequest{Title: req.GetTitle(), Format: req.GetFormat(), Tasks: tasks})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return reply(job), nil
}

func (s *Server) GetDocumentJob(ctx context.Context, req *documentv1.GetDocumentJobRequest) (*documentv1.DocumentJobReply, error) {
	job, err := s.service.Get(ctx, req.GetJobId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return reply(job), nil
}

func reply(job domain.DocumentJob) *documentv1.DocumentJobReply {
	resp := &documentv1.DocumentJobReply{
		Id:          job.ID,
		Format:      job.Format,
		Status:      job.Status,
		Error:       job.Error,
		CreatedAt:   job.CreatedAt.Format(time.RFC3339),
		CompletedAt: job.CompletedAt.Format(time.RFC3339),
		Files:       make([]*documentv1.DocumentFileReply, 0, len(job.Files)),
	}
	for _, file := range job.Files {
		resp.Files = append(resp.Files, &documentv1.DocumentFileReply{
			Id:         file.ID,
			JobId:      file.JobID,
			Kind:       file.Kind,
			StorageKey: file.StorageKey,
			MimeType:   file.MimeType,
			Size:       file.Size,
			Checksum:   file.Checksum,
			CreatedAt:  file.CreatedAt.Format(time.RFC3339),
		})
	}
	return resp
}
