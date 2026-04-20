package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"itmo-lms/content-service/internal/domain"
	documentv1 "itmo-lms/document-service/gen"
)

type DocumentClient struct {
	client documentv1.DocumentServiceClient
}

func NewDocumentClient(target string) (*DocumentClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &DocumentClient{client: documentv1.NewDocumentServiceClient(conn)}, nil
}

func (c *DocumentClient) Compile(ctx context.Context, title string, tasks []domain.Task) (string, error) {
	items := make([]*documentv1.DocumentTask, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, &documentv1.DocumentTask{Id: task.ID, Title: task.Title, LatexBody: task.LatexBody})
	}
	resp, err := c.client.CompileDocument(ctx, &documentv1.CompileDocumentRequest{Title: title, Format: "pdf", Tasks: items})
	if err != nil {
		return "", err
	}
	return resp.GetId(), nil
}
