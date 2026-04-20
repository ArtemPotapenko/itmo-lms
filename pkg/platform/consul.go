package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func RegisterConsul(consulURL, serviceName, serviceID, address string, port int) {
	if consulURL == "" {
		return
	}
	payload := map[string]any{
		"Name":    serviceName,
		"ID":      serviceID,
		"Address": address,
		"Port":    port,
		"Check": map[string]any{
			"HTTP":     fmt.Sprintf("http://%s:%d/healthz", address, port),
			"Interval": "10s",
			"Timeout":  "2s",
		},
	}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(consulURL, "/") + "/v1/agent/service/register"
	client := http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		slog.Warn("consul registration request failed", "service", serviceName, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("consul registration failed", "service", serviceName, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Warn("consul registration rejected", "service", serviceName, "status", resp.Status)
	}
}
