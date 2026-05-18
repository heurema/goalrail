package planningworker

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
	baseURL string
	client  *http.Client
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

func newAPIClient(serverURL string, client *http.Client) (*apiClient, error) {
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
		baseURL: strings.TrimRight(parsed.String(), "/"),
		client:  client,
	}, nil
}

func (c *apiClient) acquireLease(ctx context.Context, input leaseCreateRequest) (planLease, bool, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return planLease{}, false, fmt.Errorf("encode lease request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/plans/leases", bytes.NewReader(body))
	if err != nil {
		return planLease{}, false, fmt.Errorf("build lease request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return planLease{}, false, fmt.Errorf("acquire planning lease: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNoContent {
		return planLease{}, false, nil
	}
	if response.StatusCode != http.StatusCreated {
		return planLease{}, false, decodeAPIError(response)
	}

	var decoded planLease
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return planLease{}, false, fmt.Errorf("decode planning lease: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" || strings.TrimSpace(decoded.PlanID) == "" || strings.TrimSpace(decoded.LeaseToken) == "" {
		return planLease{}, false, errors.New("planning lease response is missing required fields")
	}
	return decoded, true, nil
}

func (c *apiClient) getPlan(ctx context.Context, id string, lease planLease) (workItemPlan, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/plans/"+url.PathEscape(id), nil)
	if err != nil {
		return workItemPlan{}, fmt.Errorf("build plan request: %w", err)
	}
	request.Header.Set("X-Goalrail-Lease-ID", lease.ID)
	request.Header.Set("X-Goalrail-Lease-Token", lease.LeaseToken)
	response, err := c.client.Do(request)
	if err != nil {
		return workItemPlan{}, fmt.Errorf("get planning plan: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return workItemPlan{}, decodeAPIError(response)
	}
	var decoded workItemPlan
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return workItemPlan{}, fmt.Errorf("decode planning plan: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return workItemPlan{}, errors.New("planning plan response is missing id")
	}
	return decoded, nil
}

func (c *apiClient) submitProposal(ctx context.Context, planID string, input proposalSubmitRequest) (planProposal, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return planProposal{}, fmt.Errorf("encode proposal request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/plans/"+url.PathEscape(planID)+"/proposals", bytes.NewReader(body))
	if err != nil {
		return planProposal{}, fmt.Errorf("build proposal request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return planProposal{}, fmt.Errorf("submit planning proposal: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return planProposal{}, decodeAPIError(response)
	}
	var decoded planProposal
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return planProposal{}, fmt.Errorf("decode planning proposal: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return planProposal{}, errors.New("planning proposal response is missing id")
	}
	return decoded, nil
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
