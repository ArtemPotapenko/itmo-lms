package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"itmo-lms/content-service/internal/domain"
)

type AIEvaluatorClient struct {
	baseURL string
	client  *http.Client
}

func NewAIEvaluatorClient(baseURL string) *AIEvaluatorClient {
	return &AIEvaluatorClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

func (c *AIEvaluatorClient) EvaluateTask(ctx context.Context, task domain.Task, tags []domain.Tag, topicTitles []string) (int, map[string]float64, error) {
	requestBody := map[string]any{
		"title":          task.Title,
		"latex_body":     task.LatexBody,
		"topic_titles":   topicTitles,
		"correct_answer": task.CorrectAnswer,
		"tags":           tagsPayload(tags),
	}
	raw, err := json.Marshal(requestBody)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/evaluate/task", bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("ai evaluator returned status %d", resp.StatusCode)
	}
	var decoded struct {
		Difficulty int `json:"difficulty"`
		TagWeights []struct {
			TagID  string  `json:"tag_id"`
			Weight float64 `json:"weight"`
		} `json:"tag_weights"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, nil, err
	}
	weights := make(map[string]float64, len(decoded.TagWeights))
	for _, item := range decoded.TagWeights {
		weights[item.TagID] = item.Weight
	}
	return decoded.Difficulty, weights, nil
}

func tagsPayload(items []domain.Tag) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"tag_id": item.ID,
			"code":   item.Code,
			"name":   item.Name,
			"kind":   item.Kind,
		})
	}
	return out
}
