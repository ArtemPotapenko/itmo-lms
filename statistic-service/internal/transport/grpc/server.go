package grpctransport

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	statisticv1 "itmo-lms/statistic-service/gen"
	"itmo-lms/statistic-service/internal/application"
	"itmo-lms/statistic-service/internal/domain"
)

type Server struct {
	statisticv1.UnimplementedStatisticServiceServer
	service *application.Service
}

func New(service *application.Service) *Server { return &Server{service: service} }

func (s *Server) RecordAttempt(ctx context.Context, req *statisticv1.RecordAttemptRequest) (*statisticv1.AttemptReply, error) {
	item, err := s.service.CreateAttempt(ctx, domain.Attempt{
		UserID: req.GetUserId(), ContentID: req.GetContentId(), Answer: req.GetAnswer(), IsCorrect: req.GetIsCorrect(), Source: req.GetSource(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &statisticv1.AttemptReply{
		Id:        item.ID,
		UserId:    item.UserID,
		ContentId: item.ContentID,
		TopicIds:  item.TopicIDs,
		Answer:    item.Answer,
		IsCorrect: item.IsCorrect,
		Source:    item.Source,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
		TagScores: tagScoreReplies(item.TagScores, item.IsCorrect),
	}, nil
}

func (s *Server) GetKnowledgeProfile(ctx context.Context, req *statisticv1.GetKnowledgeProfileRequest) (*statisticv1.KnowledgeProfileReply, error) {
	profile, err := s.service.Profile(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	resp := &statisticv1.KnowledgeProfileReply{
		UserId:    profile.UserID,
		UpdatedAt: profile.UpdatedAt.Format(time.RFC3339),
		Topics:    make([]*statisticv1.TopicStatReply, 0, len(profile.Topics)),
		Tags:      make([]*statisticv1.TagStatReply, 0, len(profile.Tags)),
	}
	for _, stat := range profile.Topics {
		resp.Topics = append(resp.Topics, &statisticv1.TopicStatReply{
			UserId:    stat.UserID,
			TopicId:   stat.TopicID,
			Attempts:  int32(stat.Attempts),
			Correct:   int32(stat.Correct),
			Accuracy:  stat.Accuracy,
			UpdatedAt: stat.UpdatedAt.Format(time.RFC3339),
		})
	}
	for _, stat := range profile.Tags {
		resp.Tags = append(resp.Tags, &statisticv1.TagStatReply{
			UserId:           stat.UserID,
			TagId:            stat.TagID,
			Code:             stat.Code,
			Name:             stat.Name,
			Kind:             stat.Kind,
			WeightedAttempts: stat.WeightedAttempts,
			WeightedCorrect:  stat.WeightedCorrect,
			Mastery:          stat.Mastery,
			UpdatedAt:        stat.UpdatedAt.Format(time.RFC3339),
		})
	}
	return resp, nil
}

func tagScoreReplies(items []domain.TagScore, isCorrect bool) []*statisticv1.TagStatReply {
	result := make([]*statisticv1.TagStatReply, 0, len(items))
	for _, item := range items {
		weightedCorrect := 0.0
		mastery := 0.0
		if isCorrect {
			weightedCorrect = item.Weight
			mastery = 1
		}
		result = append(result, &statisticv1.TagStatReply{
			TagId:            item.TagID,
			Code:             item.Code,
			Name:             item.Name,
			Kind:             item.Kind,
			WeightedAttempts: item.Weight,
			WeightedCorrect:  weightedCorrect,
			Mastery:          mastery,
		})
	}
	return result
}
