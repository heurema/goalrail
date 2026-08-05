package githubadmission

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

const maxGitHubResponseBytes = 4 << 20

type HTTPStatusError struct {
	Status int
	Route  string
}

func (err HTTPStatusError) Error() string {
	return fmt.Sprintf("GitHub read %s returned HTTP %d", err.Route, err.Status)
}

type RESTReader struct {
	baseURL string
	client  *http.Client
	token   string
}

func NewRESTReader(baseURL, token string, client *http.Client) (*RESTReader, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return nil, errors.New("GitHub API base URL must be absolute and credential-free")
	}
	loopback := parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && loopback) {
		return nil, errors.New("GitHub API base URL must use HTTPS outside loopback tests")
	}
	if strings.TrimSpace(token) == "" || domain.ContainsSecretShapedContent(baseURL) {
		return nil, errors.New("GitHub read-only credential is unavailable")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &RESTReader{baseURL: strings.TrimRight(baseURL, "/"), client: client, token: token}, nil
}

type pullWire struct {
	Number int64  `json:"number"`
	State  string `json:"state"`
	User   struct {
		Login string `json:"login"`
	} `json:"user"`
	Base struct {
		SHA string `json:"sha"`
	} `json:"base"`
	Head struct {
		SHA string `json:"sha"`
	} `json:"head"`
	UpdatedAt      time.Time  `json:"updated_at"`
	MergedAt       *time.Time `json:"merged_at"`
	MergeCommitSHA string     `json:"merge_commit_sha"`
}

type reviewWire struct {
	ID       int64  `json:"id"`
	State    string `json:"state"`
	CommitID string `json:"commit_id"`
	User     struct {
		Login string `json:"login"`
	} `json:"user"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type commentWire struct {
	ID int64 `json:"id"`
}

type checkRunsWire struct {
	TotalCount int `json:"total_count"`
	CheckRuns  []struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		App        *struct {
			Slug string `json:"slug"`
		} `json:"app"`
		CompletedAt *time.Time `json:"completed_at"`
	} `json:"check_runs"`
}

type permissionWire struct {
	Permission string `json:"permission"`
	User       struct {
		Login string `json:"login"`
	} `json:"user"`
}

func (reader *RESTReader) Read(ctx context.Context, repository Repository, number int64) (Snapshot, error) {
	if number <= 0 {
		return Snapshot{}, errors.New("GitHub pull-request number must be positive")
	}
	prefix := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name)
	var pull pullWire
	observedAt, err := reader.get(ctx, prefix+"/pulls/"+strconv.FormatInt(number, 10), &pull)
	if err != nil {
		return Snapshot{}, err
	}
	var reviews []reviewWire
	when, err := reader.get(ctx, prefix+"/pulls/"+strconv.FormatInt(number, 10)+"/reviews?per_page=100", &reviews)
	if err != nil {
		return Snapshot{}, err
	}
	observedAt = later(observedAt, when)
	var issueComments []commentWire
	when, err = reader.get(ctx, prefix+"/issues/"+strconv.FormatInt(number, 10)+"/comments?per_page=100", &issueComments)
	if err != nil {
		return Snapshot{}, err
	}
	observedAt = later(observedAt, when)
	var reviewComments []commentWire
	when, err = reader.get(ctx, prefix+"/pulls/"+strconv.FormatInt(number, 10)+"/comments?per_page=100", &reviewComments)
	if err != nil {
		return Snapshot{}, err
	}
	observedAt = later(observedAt, when)
	var checks checkRunsWire
	when, err = reader.get(ctx, prefix+"/commits/"+url.PathEscape(pull.Head.SHA)+"/check-runs?per_page=100", &checks)
	if err != nil {
		return Snapshot{}, err
	}
	observedAt = later(observedAt, when)
	if checks.TotalCount != len(checks.CheckRuns) {
		return Snapshot{}, errors.New("GitHub check-run response exceeds the bounded single-page collector")
	}

	snapshot := Snapshot{
		PullRequest: PullRequest{
			Number: pull.Number, BaseSHA: pull.Base.SHA, HeadSHA: pull.Head.SHA,
			State: pull.State, Author: pull.User.Login, UpdatedAt: pull.UpdatedAt,
			MergedAt: pull.MergedAt, MergeCommitSHA: pull.MergeCommitSHA,
		},
		AuthorizedActors: make(map[string]string), AuthorityAvailable: true,
		ObservedAt: observedAt, Authenticated: true,
	}
	actors := make(map[string]struct{})
	for _, review := range reviews {
		snapshot.Reviews = append(snapshot.Reviews, Review{
			ID: review.ID, Actor: review.User.Login, State: review.State,
			CommitSHA: review.CommitID, SubmittedAt: review.SubmittedAt,
		})
		if review.User.Login != "" {
			actors[review.User.Login] = struct{}{}
		}
	}
	for _, comment := range issueComments {
		if comment.ID <= 0 {
			return Snapshot{}, errors.New("GitHub issue comment index contains an invalid ID")
		}
		snapshot.IssueCommentIDs = append(snapshot.IssueCommentIDs, comment.ID)
	}
	for _, comment := range reviewComments {
		if comment.ID <= 0 {
			return Snapshot{}, errors.New("GitHub review comment index contains an invalid ID")
		}
		snapshot.ReviewCommentIDs = append(snapshot.ReviewCommentIDs, comment.ID)
	}
	for _, check := range checks.CheckRuns {
		app := ""
		if check.App != nil {
			app = check.App.Slug
		}
		snapshot.Checks = append(snapshot.Checks, Check{
			ID: check.ID, Name: check.Name, Status: check.Status, Conclusion: check.Conclusion,
			AppSlug: app, CompletedAt: check.CompletedAt,
		})
	}

	actorList := make([]string, 0, len(actors))
	for actor := range actors {
		actorList = append(actorList, actor)
	}
	sort.Strings(actorList)
	if len(actorList) > domain.MaxPolicyActors {
		return Snapshot{}, errors.New("GitHub review actor set exceeds the policy bound")
	}
	for _, actor := range actorList {
		var permission permissionWire
		when, err := reader.get(ctx, prefix+"/collaborators/"+url.PathEscape(actor)+"/permission", &permission)
		if err != nil {
			var status HTTPStatusError
			if errors.As(err, &status) && (status.Status == http.StatusForbidden || status.Status == http.StatusNotFound) {
				return Snapshot{}, ErrAuthorityUnavailable
			}
			return Snapshot{}, err
		}
		observedAt = later(observedAt, when)
		if permission.User.Login != "" && permission.User.Login != actor {
			return Snapshot{}, errors.New("GitHub permission response names a different actor")
		}
		switch strings.ToLower(permission.Permission) {
		case "admin", "maintain":
			snapshot.AuthorizedActors[actor] = "github-permission:" + strings.ToLower(permission.Permission)
		case "write", "triage", "read", "none":
		default:
			return Snapshot{}, ErrAuthorityUnavailable
		}
	}
	snapshot.ObservedAt = observedAt
	return snapshot, nil
}

func (reader *RESTReader) get(ctx context.Context, route string, output any) (time.Time, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, reader.baseURL+route, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("build GitHub read request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+reader.token)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := reader.client.Do(request)
	if err != nil {
		return time.Time{}, errors.New("GitHub read request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return time.Time{}, HTTPStatusError{Status: response.StatusCode, Route: routeWithoutQuery(route)}
	}
	if strings.Contains(response.Header.Get("Link"), `rel="next"`) {
		return time.Time{}, errors.New("GitHub response exceeds the bounded single-page collector")
	}
	observedAt, err := http.ParseTime(response.Header.Get("Date"))
	if err != nil || observedAt.IsZero() {
		return time.Time{}, ErrUntrustedObservation
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxGitHubResponseBytes+1))
	if err != nil {
		return time.Time{}, errors.New("read GitHub response failed")
	}
	if len(raw) > maxGitHubResponseBytes {
		return time.Time{}, fmt.Errorf("GitHub response exceeds %d bytes", maxGitHubResponseBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(output); err != nil {
		return time.Time{}, errors.New("decode GitHub response failed")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return time.Time{}, errors.New("GitHub response contains trailing JSON")
	}
	return observedAt.UTC(), nil
}

func routeWithoutQuery(route string) string {
	if index := strings.IndexByte(route, '?'); index >= 0 {
		return route[:index]
	}
	return route
}

func later(left, right time.Time) time.Time {
	if right.After(left) {
		return right
	}
	return left
}
