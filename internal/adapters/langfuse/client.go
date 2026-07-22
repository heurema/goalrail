package langfuse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

const (
	defaultObservationPageSize = 100
	defaultObservationMaxPages = 10
	defaultResponseBytes       = int64(2 * 1024 * 1024)
	maxConfiguredResponseBytes = int64(8 * 1024 * 1024)
)

var (
	ErrInvalidClientConfig = errors.New("invalid Langfuse client configuration")
	ErrInvalidTraceQuery   = errors.New("invalid Langfuse trace query")
	ErrObservationRead     = errors.New("Langfuse observation read failed")
	ErrMalformedResponse   = errors.New("malformed Langfuse observation response")
	ErrPaginationLimit     = errors.New("Langfuse observation pagination limit reached")
)

type ClientConfig struct {
	BaseURL          string
	PublicKey        string
	SecretKey        string
	HTTPClient       *http.Client
	PageSize         int
	MaxPages         int
	MaxResponseBytes int64
}

// Client is a bounded read-only implementation of the observations v2 API.
// Credentials remain in adapter memory and are never returned in errors or
// domain values.
type Client struct {
	baseURL          *url.URL
	publicKey        string
	secretKey        string
	httpClient       *http.Client
	pageSize         int
	maxPages         int
	maxResponseBytes int64
}

func NewClient(config ClientConfig) (*Client, error) {
	baseURL, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || baseURL == nil || (baseURL.Scheme != "https" && baseURL.Scheme != "http") ||
		baseURL.Host == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" ||
		(baseURL.Path != "" && baseURL.Path != "/") ||
		(baseURL.Scheme == "http" && !isLoopbackHost(baseURL.Hostname())) {
		return nil, ErrInvalidClientConfig
	}
	if strings.TrimSpace(config.PublicKey) == "" || strings.TrimSpace(config.SecretKey) == "" ||
		config.HTTPClient == nil || config.HTTPClient.Timeout <= 0 {
		return nil, ErrInvalidClientConfig
	}
	pageSize := config.PageSize
	if pageSize == 0 {
		pageSize = defaultObservationPageSize
	}
	maxPages := config.MaxPages
	if maxPages == 0 {
		maxPages = defaultObservationMaxPages
	}
	maxBytes := config.MaxResponseBytes
	if maxBytes == 0 {
		maxBytes = defaultResponseBytes
	}
	if pageSize < 1 || pageSize > defaultObservationPageSize || maxPages < 1 || maxPages > defaultObservationMaxPages ||
		maxBytes < 1 || maxBytes > maxConfiguredResponseBytes {
		return nil, ErrInvalidClientConfig
	}
	httpClient := *config.HTTPClient
	httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/")
	return &Client{
		baseURL: baseURL, publicKey: config.PublicKey, secretKey: config.SecretKey,
		httpClient: &httpClient, pageSize: pageSize, maxPages: maxPages,
		maxResponseBytes: maxBytes,
	}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

type observationResponse struct {
	Data []observationRow `json:"data"`
	Meta *observationMeta `json:"meta"`
}

type observationMeta struct {
	Cursor *string `json:"cursor"`
}

type observationRow struct {
	ID                  string     `json:"id"`
	TraceID             string     `json:"traceId"`
	StartTime           time.Time  `json:"startTime"`
	EndTime             *time.Time `json:"endTime"`
	ProjectID           string     `json:"projectId"`
	ParentObservationID *string    `json:"parentObservationId"`
	Type                string     `json:"type"`
	Name                string     `json:"name"`
	Level               string     `json:"level"`
	StatusMessage       *string    `json:"statusMessage"`
	Version             *string    `json:"version"`
	Environment         string     `json:"environment"`
	Bookmarked          bool       `json:"bookmarked"`
	Public              bool       `json:"public"`
	UserID              *string    `json:"userId"`
	SessionID           *string    `json:"sessionId"`
}

type sessionFilter struct {
	Type     string `json:"type"`
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

func (c *Client) ListSessionObservations(
	ctx context.Context,
	query domain.TraceObservationQuery,
) ([]domain.TraceObservation, error) {
	if !domain.IsCanonicalID(string(query.SessionID)) ||
		query.FromStartTime.IsZero() || query.FromStartTime.Location() != time.UTC ||
		query.ToStartTime.IsZero() || query.ToStartTime.Location() != time.UTC ||
		!query.FromStartTime.Before(query.ToStartTime) {
		return nil, ErrInvalidTraceQuery
	}
	filterJSON, err := json.Marshal([]sessionFilter{{
		Type: "string", Column: "sessionId", Operator: "=", Value: string(query.SessionID),
	}})
	if err != nil {
		return nil, ErrInvalidTraceQuery
	}

	var observations []domain.TraceObservation
	seenCursors := make(map[string]struct{})
	cursor := ""
	for page := 0; page < c.maxPages; page++ {
		response, readErr := c.readObservationPage(ctx, query, string(filterJSON), cursor)
		if readErr != nil {
			return nil, readErr
		}
		for _, row := range response.Data {
			if !domain.IsCanonicalID(row.ID) || !traceIDPattern.MatchString(row.TraceID) || row.SessionID == nil ||
				domain.SessionID(*row.SessionID) != query.SessionID || row.StartTime.IsZero() ||
				row.EndTime == nil || row.EndTime.Before(row.StartTime) {
				return nil, ErrMalformedResponse
			}
			observations = append(observations, domain.TraceObservation{
				TraceReference: domain.EvidenceReference(traceReferenceScheme + strings.ToLower(row.TraceID)),
				SessionID:      query.SessionID,
				StartedAt:      row.StartTime.UTC(),
				EndedAt:        row.EndTime.UTC(),
			})
		}
		if response.Meta == nil || response.Meta.Cursor == nil || *response.Meta.Cursor == "" {
			return observations, nil
		}
		next := *response.Meta.Cursor
		if len(next) > 4096 {
			return nil, ErrMalformedResponse
		}
		if _, repeated := seenCursors[next]; repeated {
			return nil, ErrMalformedResponse
		}
		seenCursors[next] = struct{}{}
		cursor = next
	}
	return nil, ErrPaginationLimit
}

func (c *Client) readObservationPage(
	ctx context.Context,
	query domain.TraceObservationQuery,
	filter string,
	cursor string,
) (observationResponse, error) {
	requestURL := *c.baseURL
	requestURL.Path = strings.TrimSuffix(requestURL.Path, "/") + "/api/public/v2/observations"
	values := requestURL.Query()
	values.Set("fields", "core,basic")
	values.Set("filter", filter)
	values.Set("fromStartTime", query.FromStartTime.Format(time.RFC3339Nano))
	values.Set("toStartTime", query.ToStartTime.Format(time.RFC3339Nano))
	values.Set("limit", fmt.Sprintf("%d", c.pageSize))
	if cursor != "" {
		values.Set("cursor", cursor)
	}
	requestURL.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return observationResponse{}, ErrObservationRead
	}
	request.SetBasicAuth(c.publicKey, c.secretKey)
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return observationResponse{}, fmt.Errorf("%w: transport %T", ErrObservationRead, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, c.maxResponseBytes+1))
		return observationResponse{}, fmt.Errorf("%w: HTTP %d", ErrObservationRead, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil {
		return observationResponse{}, ErrObservationRead
	}
	if int64(len(body)) > c.maxResponseBytes {
		return observationResponse{}, ErrMalformedResponse
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var decoded observationResponse
	if err := decoder.Decode(&decoded); err != nil {
		return observationResponse{}, ErrMalformedResponse
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return observationResponse{}, ErrMalformedResponse
	}
	if decoded.Meta == nil {
		return observationResponse{}, ErrMalformedResponse
	}
	return decoded, nil
}
