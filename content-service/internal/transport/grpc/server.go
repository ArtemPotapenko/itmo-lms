package grpctransport

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	contentv1 "itmo-lms/content-service/gen"
	"itmo-lms/content-service/internal/application"
	"itmo-lms/content-service/internal/domain"
)

type Server struct {
	contentv1.UnimplementedContentServiceServer
	service *application.Service
}

func New(service *application.Service) *Server { return &Server{service: service} }

func (s *Server) GetTask(ctx context.Context, req *contentv1.GetTaskRequest) (*contentv1.TaskReply, error) {
	task, err := s.service.GetTask(ctx, req.GetTaskId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &contentv1.TaskReply{
		Id:            task.ID,
		Title:         task.Title,
		LatexBody:     task.LatexBody,
		TopicIds:      task.TopicIDs,
		Difficulty:    int32(task.Difficulty),
		CorrectAnswer: task.CorrectAnswer,
		Status:        task.Status,
		AuthorId:      task.AuthorID,
		CreatedAt:     task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     task.UpdatedAt.Format(time.RFC3339),
		Tags:          tagsReply(task.Tags),
	}, nil
}

func (s *Server) GetWorkTemplate(ctx context.Context, req *contentv1.GetWorkTemplateRequest) (*contentv1.WorkTemplateReply, error) {
	work, err := s.service.GetWork(ctx, req.GetWorkId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	taskIDs := make([]string, 0)
	for _, item := range work.Items {
		if item.Kind == "task" {
			taskIDs = append(taskIDs, item.ContentID)
		}
	}
	return &contentv1.WorkTemplateReply{
		Id:        work.ID,
		Title:     work.Title,
		TaskIds:   taskIDs,
		Status:    work.Status,
		CreatedBy: work.CreatedBy,
		CreatedAt: work.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) BuildWorkLatex(ctx context.Context, req *contentv1.BuildWorkLatexRequest) (*contentv1.BuildWorkLatexReply, error) {
	latex, err := s.service.BuildWorkLatex(ctx, req.GetWorkId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &contentv1.BuildWorkLatexReply{WorkId: req.GetWorkId(), Latex: latex}, nil
}

func tagsReply(items []domain.TaskTag) []*contentv1.TaskTagReply {
	result := make([]*contentv1.TaskTagReply, 0, len(items))
	for _, item := range items {
		result = append(result, &contentv1.TaskTagReply{
			TagId:  item.TagID,
			Code:   item.Code,
			Name:   item.Name,
			Kind:   item.Kind,
			Weight: item.Weight,
		})
	}
	return result
}
