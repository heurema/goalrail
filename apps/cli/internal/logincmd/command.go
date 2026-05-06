package logincmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const callbackPath = "/callback"

type Store interface {
	Save(authstore.Session) error
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Options struct {
	Store           Store
	HTTPClient      HTTPClient
	OpenBrowser     func(string) error
	skipWaitForTest bool
}

type callbackResult struct {
	code string
	err  error
}

type exchangeResponse struct {
	UserID               string `json:"user_id"`
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	AccessTokenExpiresAt string `json:"access_token_expires_at"`
	TokenType            string `json:"token_type"`
}

func Run(ctx context.Context, out *term.Output, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	parsed, err := parseLoginArgs(args)
	if err != nil {
		return err
	}
	if parsed.help {
		_, writeErr := fmt.Fprint(out.Stdout, Usage())
		return writeErr
	}
	if parsed.serverURL == "" {
		return exitcode.UsageError(errors.New("missing required server_url"))
	}

	serverURL, err := NormalizeServerURL(parsed.serverURL)
	if err != nil {
		return exitcode.UsageError(err)
	}
	store := options.Store
	if store == nil {
		path, err := authstore.DefaultPath()
		if err != nil {
			return exitcode.RuntimeError(err)
		}
		store = authstore.NewFileStore(path)
	}
	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	openBrowser := options.OpenBrowser
	if openBrowser == nil {
		openBrowser = OpenBrowser
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("start localhost callback listener: %w", err))
	}
	defer listener.Close()

	state, err := randomURLToken()
	if err != nil {
		return exitcode.RuntimeError(err)
	}
	codeVerifier, err := randomURLToken()
	if err != nil {
		return exitcode.RuntimeError(err)
	}
	codeChallenge := CodeChallengeS256(codeVerifier)
	callbackURL := "http://" + listener.Addr().String() + callbackPath
	browserURL := BuildBrowserURL(serverURL, callbackURL, state, codeChallenge)
	callbackCh := serveCallback(listener, state)

	if parsed.noBrowser {
		if _, err := fmt.Fprintf(out.Stdout, "Open this URL in your browser:\n%s\n", browserURL); err != nil {
			return err
		}
	} else {
		if err := openBrowser(browserURL); err != nil {
			if _, writeErr := fmt.Fprintf(out.Stdout, "Open this URL in your browser:\n%s\n", browserURL); writeErr != nil {
				return writeErr
			}
		}
	}
	if options.skipWaitForTest {
		return nil
	}

	var result callbackResult
	select {
	case result = <-callbackCh:
	case <-ctx.Done():
		return exitcode.RuntimeError(ctx.Err())
	}
	if result.err != nil {
		return exitcode.RuntimeError(result.err)
	}

	session, err := exchangeCode(ctx, client, serverURL, result.code, state, codeVerifier)
	if err != nil {
		return exitcode.RuntimeError(err)
	}
	session.ServerURL = serverURL
	if err := store.Save(session); err != nil {
		return exitcode.RuntimeError(err)
	}

	_, err = fmt.Fprintf(out.Stdout, "Logged in to %s\n", serverURL)
	return err
}

func Usage() string {
	return "Usage: goalrail login <server_url> [--no-browser]\n\nStarts a browser localhost-loopback login and stores tokens in the local Goalrail auth file. Organization, Project, and RepoBinding selection are not implemented yet.\n"
}

type loginArgs struct {
	serverURL string
	noBrowser bool
	help      bool
}

func parseLoginArgs(args []string) (loginArgs, error) {
	parsed := loginArgs{}
	for _, arg := range args {
		switch {
		case arg == "--help" || arg == "-h":
			parsed.help = true
		case arg == "--no-browser" || arg == "-no-browser":
			parsed.noBrowser = true
		case strings.HasPrefix(arg, "--no-browser="):
			noBrowser, err := parseBoolFlagValue(arg, strings.TrimPrefix(arg, "--no-browser="))
			if err != nil {
				return loginArgs{}, err
			}
			parsed.noBrowser = noBrowser
		case strings.HasPrefix(arg, "-no-browser="):
			noBrowser, err := parseBoolFlagValue(arg, strings.TrimPrefix(arg, "-no-browser="))
			if err != nil {
				return loginArgs{}, err
			}
			parsed.noBrowser = noBrowser
		case strings.HasPrefix(arg, "-"):
			return loginArgs{}, exitcode.UsageError(fmt.Errorf("unknown login flag %q", arg))
		case parsed.serverURL == "":
			parsed.serverURL = arg
		default:
			return loginArgs{}, exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", args))
		}
	}
	return parsed, nil
}

func parseBoolFlagValue(flag string, rawValue string) (bool, error) {
	value, err := strconv.ParseBool(rawValue)
	if err != nil {
		return false, exitcode.UsageError(fmt.Errorf("invalid login flag %q", flag))
	}
	return value, nil
}

func NormalizeServerURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return "", fmt.Errorf("invalid server_url %q", raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("server_url must use http or https")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = ""
	}
	return parsed.String(), nil
}

func BuildBrowserURL(serverURL string, callbackURL string, state string, codeChallenge string) string {
	parsed, _ := url.Parse(serverURL + "/cli/login")
	query := parsed.Query()
	query.Set("redirect_uri", callbackURL)
	query.Set("state", state)
	query.Set("code_challenge", codeChallenge)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func ValidateCallbackState(expectedState string, values url.Values) (string, error) {
	if values.Get("state") != expectedState {
		return "", fmt.Errorf("callback state mismatch")
	}
	code := strings.TrimSpace(values.Get("code"))
	if code == "" {
		return "", fmt.Errorf("callback missing code")
	}
	return code, nil
}

func OpenBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

func serveCallback(listener net.Listener, expectedState string) <-chan callbackResult {
	resultCh := make(chan callbackResult, 1)
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		code, err := ValidateCallbackState(expectedState, r.URL.Query())
		if err != nil {
			http.Error(w, "Goalrail login failed.", http.StatusBadRequest)
			resultCh <- callbackResult{err: err}
			go server.Close()
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, "<!doctype html><html><body><p>Goalrail login complete. You can close this window.</p></body></html>")
		resultCh <- callbackResult{code: code}
		go server.Close()
	})
	server.Handler = mux
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			resultCh <- callbackResult{err: err}
		}
	}()
	return resultCh
}

func exchangeCode(ctx context.Context, client HTTPClient, serverURL string, code string, state string, codeVerifier string) (authstore.Session, error) {
	payload, err := json.Marshal(map[string]string{
		"code":          code,
		"state":         state,
		"code_verifier": codeVerifier,
	})
	if err != nil {
		return authstore.Session{}, fmt.Errorf("encode exchange request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/v1/auth/cli/exchange", bytes.NewReader(payload))
	if err != nil {
		return authstore.Session{}, fmt.Errorf("build exchange request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return authstore.Session{}, fmt.Errorf("exchange CLI auth code: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return authstore.Session{}, fmt.Errorf("exchange CLI auth code failed: HTTP %d", response.StatusCode)
	}
	var decoded exchangeResponse
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return authstore.Session{}, fmt.Errorf("decode exchange response: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, decoded.AccessTokenExpiresAt)
	if err != nil {
		return authstore.Session{}, fmt.Errorf("decode access token expiry: %w", err)
	}
	return authstore.Session{
		AccessToken:          decoded.AccessToken,
		RefreshToken:         decoded.RefreshToken,
		AccessTokenExpiresAt: expiresAt,
		TokenType:            decoded.TokenType,
	}, nil
}

func randomURLToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate random state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func CodeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
