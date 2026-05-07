package checkoutrunner

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

func (c *apiClient) acquireLease(ctx context.Context, input checkoutLeaseCreateRequest) (checkoutLease, bool, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return checkoutLease{}, false, fmt.Errorf("encode checkout lease request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/checkout-jobs/leases", bytes.NewReader(body))
	if err != nil {
		return checkoutLease{}, false, fmt.Errorf("build checkout lease request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	c.authorize(request)

	response, err := c.client.Do(request)
	if err != nil {
		return checkoutLease{}, false, fmt.Errorf("acquire checkout lease: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNoContent {
		return checkoutLease{}, false, nil
	}
	if response.StatusCode != http.StatusCreated {
		return checkoutLease{}, false, decodeAPIError(response)
	}
	var decoded checkoutLease
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return checkoutLease{}, false, fmt.Errorf("decode checkout lease: %w", err)
	}
	if strings.TrimSpace(decoded.JobID) == "" || strings.TrimSpace(decoded.TaskID) == "" || strings.TrimSpace(decoded.LeaseToken) == "" {
		return checkoutLease{}, false, errors.New("checkout lease response is missing required fields")
	}
	return decoded, true, nil
}

func (c *apiClient) submitReceipt(ctx context.Context, jobID string, input checkoutReceiptSubmitRequest) (checkoutReceipt, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return checkoutReceipt{}, fmt.Errorf("encode checkout receipt request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/checkout-jobs/"+url.PathEscape(jobID)+"/receipts", bytes.NewReader(body))
	if err != nil {
		return checkoutReceipt{}, fmt.Errorf("build checkout receipt request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	c.authorize(request)

	response, err := c.client.Do(request)
	if err != nil {
		return checkoutReceipt{}, fmt.Errorf("submit checkout receipt: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return checkoutReceipt{}, decodeAPIError(response)
	}
	var decoded checkoutReceipt
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return checkoutReceipt{}, fmt.Errorf("decode checkout receipt: %w", err)
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return checkoutReceipt{}, errors.New("checkout receipt response is missing id")
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
