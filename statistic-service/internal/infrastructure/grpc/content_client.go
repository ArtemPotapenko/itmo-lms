package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	contentv1 "itmo-lms/content-service/gen"
	"itmo-lms/statistic-service/internal/domain"
)

type ContentClient struct {
	client contentv1.ContentServiceClient
}

func NewContentClient(target string) (*ContentClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &ContentClient{client: contentv1.NewContentServiceClient(conn)}, nil
}

func (c *ContentClient) ResolveTask(ctx context.Context, taskID string) ([]string, []domain.TagScore, int, error) {
	resp, err := c.client.GetTask(ctx, &contentv1.GetTaskRequest{TaskId: taskID})
	if err != nil {
		return nil, nil, 0, err
	}
	scores := make([]domain.TagScore, 0, len(resp.GetTags()))
	for _, item := range resp.GetTags() {
		scores = append(scores, domain.TagScore{
			TagID:  item.GetTagId(),
			Code:   item.GetCode(),
			Name:   item.GetName(),
			Kind:   item.GetKind(),
			Weight: item.GetWeight(),
		})
	}
	return resp.GetTopicIds(), scores, int(resp.GetDifficulty()), nil
}
