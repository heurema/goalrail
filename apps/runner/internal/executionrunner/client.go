package executionrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultHTTPClientTimeout = 30 * time.Second

type apiClient struct {
	baseURL     string
	bearerToken string
	client      *http.Client
}

type apiError struct {
	statusCode int
	code       string
	message    string
}

func (e *apiError) Error() string {
	if e.code == "" {
		return fmt.Sprintf("goalrail api returned status %d", e.statusCode)
	}
	return fmt.Sprintf("goalrail api returned %s: %s", e.code, e.message)
}

func newAPIClient(serverURL string, bearerToken string, client *http.Client) (*apiClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return nil, fmt.Errorf("parse server url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("server url must use http or https")
	}
	if parsed.Host == "" {
		return nil, errors.New("server url must include host")
	}
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPClientTimeout}
	}
	return &apiClient{
		baseURL:     strings.TrimRight(parsed.String(), "/"),
		bearerToken: strings.TrimSpace(bearerToken),
		client:      client,
	}, nil
}

func (c *apiClient) acquireLease(ctx context.Context, input executionLeaseCreateRequest) (executionLease, bool, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return executionLease{}, false, fmt.Errorf("encode execution lease request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/execution-jobs/leases", bytes.NewReader(body))
	if err != nil {
		return executionLease{}, false, fmt.Errorf("build execution lease request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	c.authorize(request)

	response, err := c.client.Do(request)
	if err != nil {
		return executionLease{}, false, fmt.Errorf("acquire execution lease: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNoContent {
		return executionLease{}, false, nil
	}
	if response.StatusCode != http.StatusCreated {
		return executionLease{}, false, decodeAPIError(response)
	}
	var decoded executionLease
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return executionLease{}, false, fmt.Errorf("decode execution lease: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" || strings.TrimSpace(decoded.ExecutionJobID) == "" || strings.TrimSpace(decoded.LeaseToken) == "" {
		return executionLease{}, false, errors.New("execution lease response is missing required fields")
	}
	return decoded, true, nil
}

func (c *apiClient) startRun(ctx context.Context, executionJobID string, input runStartRequest) (runStarted, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return runStarted{}, fmt.Errorf("encode run start request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/execution-jobs/"+url.PathEscape(executionJobID)+"/runs", bytes.NewReader(body))
	if err != nil {
		return runStarted{}, fmt.Errorf("build run start request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	c.authorize(request)

	response, err := c.client.Do(request)
	if err != nil {
		return runStarted{}, fmt.Errorf("start run: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return runStarted{}, decodeAPIError(response)
	}
	var decoded runStarted
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return runStarted{}, fmt.Errorf("decode run start: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return runStarted{}, errors.New("run start response is missing id")
	}
	return decoded, nil
}

func (c *apiClient) createCommandPlan(ctx context.Context, runID string, input executionCommandPlanRequest) (executionCommandPlan, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return executionCommandPlan{}, fmt.Errorf("encode execution command plan request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/runs/"+url.PathEscape(runID)+"/command-plans", bytes.NewReader(body))
	if err != nil {
		return executionCommandPlan{}, fmt.Errorf("build execution command plan request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	c.authorize(request)

	response, err := c.client.Do(request)
	if err != nil {
		return executionCommandPlan{}, fmt.Errorf("create execution command plan: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return executionCommandPlan{}, decodeAPIError(response)
	}
	var decoded executionCommandPlan
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return executionCommandPlan{}, fmt.Errorf("decode execution command plan: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return executionCommandPlan{}, errors.New("execution command plan response is missing id")
	}
	return decoded, nil
}

func (c *apiClient) getCommandPlan(ctx context.Context, runID string, commandKind string, action string) (executionCommandPlan, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/runs/"+url.PathEscape(runID)+"/command-plans/"+url.PathEscape(commandKind)+"/"+url.PathEscape(action), nil)
	if err != nil {
		return executionCommandPlan{}, fmt.Errorf("build execution command plan get request: %w", err)
	}
	c.authorize(request)

	response, err := c.client.Do(request)
	if err != nil {
		return executionCommandPlan{}, fmt.Errorf("get execution command plan: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return executionCommandPlan{}, decodeAPIError(response)
	}
	var decoded executionCommandPlan
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return executionCommandPlan{}, fmt.Errorf("decode execution command plan: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return executionCommandPlan{}, errors.New("execution command plan response is missing id")
	}
	return decoded, nil
}

func (c *apiClient) submitReceipt(ctx context.Context, runID string, input executionReceiptRequest) (executionReceipt, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return executionReceipt{}, fmt.Errorf("encode execution receipt request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/runs/"+url.PathEscape(runID)+"/receipts", bytes.NewReader(body))
	if err != nil {
		return executionReceipt{}, fmt.Errorf("build execution receipt request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	c.authorize(request)

	response, err := c.client.Do(request)
	if err != nil {
		return executionReceipt{}, fmt.Errorf("submit execution receipt: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return executionReceipt{}, decodeAPIError(response)
	}
	var decoded executionReceipt
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return executionReceipt{}, fmt.Errorf("decode execution receipt: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return executionReceipt{}, errors.New("execution receipt response is missing id")
	}
	return decoded, nil
}

func (c *apiClient) submitRunnerCapabilityReport(ctx context.Context, input runnerCapabilityReportRequest) (runnerCapabilityReport, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return runnerCapabilityReport{}, fmt.Errorf("encode runner capability report request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/runner-capability-reports", bytes.NewReader(body))
	if err != nil {
		return runnerCapabilityReport{}, fmt.Errorf("build runner capability report request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	c.authorize(request)

	response, err := c.client.Do(request)
	if err != nil {
		return runnerCapabilityReport{}, fmt.Errorf("submit runner capability report: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return runnerCapabilityReport{}, decodeAPIError(response)
	}
	var decoded runnerCapabilityReport
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return runnerCapabilityReport{}, fmt.Errorf("decode runner capability report: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return runnerCapabilityReport{}, errors.New("runner capability report response is missing id")
	}
	return decoded, nil
}

func (c *apiClient) authorize(request *http.Request) {
	if c.bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
}

func decodeAPIError(response *http.Response) error {
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	var decoded struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(payload, &decoded)
	return &apiError{
		statusCode: response.StatusCode,
		code:       decoded.Error.Code,
		message:    decoded.Error.Message,
	}
}

func apiErrorCode(err error) string {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr.code
	}
	return ""
}
