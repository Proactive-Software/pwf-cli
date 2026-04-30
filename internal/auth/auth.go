package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	IdentityBase = "https://identity-dev.proworkflow.com"
	CallbackPort = "9876"
	CallbackPath = "/callback"
)

type Tokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// Login runs the browser-based authorization code flow and returns tokens.
func Login(clientID, clientSecret string) (*Tokens, error) {
	state, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	redirectURI := fmt.Sprintf("http://localhost:%s%s", CallbackPort, CallbackPath)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "invalid state", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch — possible CSRF")
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- fmt.Errorf("no code in callback")
			return
		}
		fmt.Fprintln(w, "<html><body><p>Login successful. You can close this tab.</p></body></html>")
		codeCh <- code
	})

	ln, err := net.Listen("tcp", ":"+CallbackPort)
	if err != nil {
		return nil, fmt.Errorf("listen on port %s: %w", CallbackPort, err)
	}

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer srv.Shutdown(context.Background())

	authURL := fmt.Sprintf(
		"%s/auth?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=openid%%20email",
		IdentityBase,
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		state,
	)

	fmt.Printf("Opening browser for login...\nIf it doesn't open, visit:\n%s\n", authURL)
	openBrowser(authURL)

	select {
	case code := <-codeCh:
		return exchangeCode(clientID, clientSecret, code, redirectURI)
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("login timed out after 5 minutes")
	}
}

// Refresh exchanges a refresh token for a new access token.
func Refresh(clientID, clientSecret, refreshToken string) (*Tokens, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
	return postToken(form, refreshToken)
}

func exchangeCode(clientID, clientSecret, code, redirectURI string) (*Tokens, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
	}
	return postToken(form, "")
}

func postToken(form url.Values, existingRefresh string) (*Tokens, error) {
	resp, err := http.PostForm(IdentityBase+"/token", form)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var tr tokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("token error %s: %s", tr.Error, tr.ErrorDesc)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in response")
	}

	expiresIn := tr.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 300
	}

	refresh := tr.RefreshToken
	if refresh == "" {
		refresh = existingRefresh
	}

	return &Tokens{
		AccessToken:  tr.AccessToken,
		RefreshToken: refresh,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(u string) {
	// pkg/browser is already a dependency; import via indirect to avoid cycle.
	// We shell out to avoid importing pkg/browser here — commands package does it.
	_ = u
}

// BuildAuthURL returns the authorization URL (for callers that open the browser).
func BuildAuthURL(clientID, redirectURI, state string) string {
	return fmt.Sprintf(
		"%s/auth?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=openid%%20email",
		IdentityBase,
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		state,
	)
}

// LoginWithBrowserOpener runs the flow, calling openFn to open the browser.
func LoginWithBrowserOpener(clientID, clientSecret string, openFn func(string) error) (*Tokens, error) {
	state, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	redirectURI := fmt.Sprintf("http://localhost:%s%s", CallbackPort, CallbackPath)
	authURL := BuildAuthURL(clientID, redirectURI, state)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "invalid state", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch — possible CSRF")
			return
		}
		code := q.Get("code")
		if code == "" {
			errStr := q.Get("error")
			if errStr != "" {
				http.Error(w, errStr, http.StatusBadRequest)
				errCh <- fmt.Errorf("auth error: %s — %s", errStr, q.Get("error_description"))
				return
			}
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- fmt.Errorf("no code in callback")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, "<html><body style='font-family:sans-serif;padding:2rem'><h2>Login successful</h2><p>You can close this tab.</p></body></html>")
		codeCh <- code
	})

	ln, err := net.Listen("tcp", ":"+CallbackPort)
	if err != nil {
		return nil, fmt.Errorf("listen on port %s: %w", CallbackPort, err)
	}

	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			select {
			case errCh <- serveErr:
			default:
			}
		}
	}()
	defer srv.Shutdown(context.Background())

	if err := openFn(authURL); err != nil {
		fmt.Printf("Could not open browser automatically.\nVisit: %s\n", authURL)
	}

	select {
	case code := <-codeCh:
		return exchangeCode(clientID, clientSecret, code, redirectURI)
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("login timed out after 5 minutes")
	}
}

// IsExpired reports whether the access token needs refreshing (with 30s buffer).
func (t *Tokens) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-30 * time.Second))
}

// MarshalJSON encodes Tokens as JSON for storage.
func (t *Tokens) Marshal() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalTokens decodes a JSON string back to Tokens.
func UnmarshalTokens(s string) (*Tokens, error) {
	var t Tokens
	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return nil, err
	}
	// Legacy: plain JWT string stored directly (no JSON structure)
	if t.AccessToken == "" && !strings.HasPrefix(s, "{") {
		return &Tokens{
			AccessToken:  s,
			ExpiresAt:    time.Time{}, // zero = treat as expired
		}, nil
	}
	return &t, nil
}
