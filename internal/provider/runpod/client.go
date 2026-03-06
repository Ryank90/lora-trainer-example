package runpod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
}

func NewClient(endpoint, apiKey string) *Client {
	if endpoint == "" {
		endpoint = "https://api.runpod.io"
	}
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoint:   endpoint,
		apiKey:     apiKey,
	}
}

type createPodRequest struct {
	Name            string            `json:"name"`
	ImageName       string            `json:"imageName"`
	GPUTypeID       string            `json:"gpuTypeId"`
	GPUCount        int               `json:"gpuCount"`
	VolumeMountPath string            `json:"volumeMountPath,omitempty"`
	VolumeInGB      int               `json:"volumeInGb"`
	ContainerDiskGB int               `json:"containerDiskInGb"`
	Env             map[string]string `json:"env,omitempty"`
	DockerArgs      string            `json:"dockerArgs,omitempty"`
}

type podResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"desiredStatus"`
	GPUType     string  `json:"gpuTypeId"`
	GPUCount    int     `json:"gpuCount"`
	CostPerHr   float64 `json:"costPerHr"`
	Runtime     *podRuntime `json:"runtime"`
}

type podRuntime struct {
	Ports []struct {
		IP         string `json:"ip"`
		PublicPort int    `json:"publicPort"`
		PrivatePort int   `json:"privatePort"`
		Type       string `json:"type"`
	} `json:"ports"`
}

type gpuTypeResponse struct {
	ID            string  `json:"id"`
	DisplayName   string  `json:"displayName"`
	MemoryInGB    int     `json:"memoryInGb"`
	CommunityCount int    `json:"communityCount"`
	SecureCount    int    `json:"secureCount"`
	CommunityPrice float64 `json:"communityPrice"`
	SecurePrice    float64 `json:"securePrice"`
}

func (c *Client) CreatePod(ctx context.Context, req createPodRequest) (*podResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v2/pods", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("creating pod: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result podResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

func (c *Client) GetPod(ctx context.Context, id string) (*podResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v2/pods/"+id, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("getting pod: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result podResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

func (c *Client) DeletePod(ctx context.Context, id string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint+"/v2/pods/"+id, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("deleting pod: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) GetGPUTypes(ctx context.Context) ([]gpuTypeResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v2/gpu-types", nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("getting GPU types: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result []gpuTypeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}
