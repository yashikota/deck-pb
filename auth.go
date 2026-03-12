package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

const (
	envServiceAccountKey = "DECK_SERVICE_ACCOUNT_KEY"
	envEnableADC         = "DECK_ENABLE_ADC"
	envAccessToken       = "DECK_ACCESS_TOKEN"
)

var scopes = []string{
	"https://www.googleapis.com/auth/presentations",
	"https://www.googleapis.com/auth/drive",
}

// NewSlidesService creates a Google Slides API service using deck-compatible authentication.
func NewSlidesService(ctx context.Context, profile string) (*slides.Service, error) {
	client, err := getHTTPClient(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}
	srv, err := slides.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Slides service: %w", err)
	}
	return srv, nil
}

func getHTTPClient(ctx context.Context, profile string) (*http.Client, error) {
	if credsJSON := os.Getenv(envServiceAccountKey); credsJSON != "" {
		config, err := google.JWTConfigFromJSON([]byte(credsJSON), scopes...)
		if err != nil {
			return nil, err
		}
		return config.Client(ctx), nil
	}
	if os.Getenv(envEnableADC) != "" {
		return google.DefaultClient(ctx, scopes...)
	}
	if token := os.Getenv(envAccessToken); token != "" {
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		})), nil
	}
	return getDefaultHTTPClient(ctx, profile)
}

func getDefaultHTTPClient(ctx context.Context, profile string) (*http.Client, error) {
	creds := credentialsPath(profile)
	b, err := os.ReadFile(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file %s: %w", creds, err)
	}
	cfg, err := google.ConfigFromJSON(b, scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	tokenPath := tokenFilePath(profile)
	token, err := tokenFromFile(tokenPath)
	if err != nil {
		token, err = getTokenFromWeb(ctx, cfg)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenPath, token); err != nil {
			return nil, err
		}
	} else if token.Expiry.Before(time.Now()) {
		if token.RefreshToken != "" {
			tokenSource := cfg.TokenSource(ctx, token)
			newToken, err := tokenSource.Token()
			if err != nil {
				newToken, err = getTokenFromWeb(ctx, cfg)
				if err != nil {
					return nil, err
				}
			}
			if err := saveToken(tokenPath, newToken); err != nil {
				return nil, err
			}
			token = newToken
		} else {
			token, err = getTokenFromWeb(ctx, cfg)
			if err != nil {
				return nil, err
			}
			if err := saveToken(tokenPath, token); err != nil {
				return nil, err
			}
		}
	}
	return cfg.Client(ctx, token), nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	var authCode string
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	listenCtx, listening := context.WithCancel(ctx)
	defer listening()
	doneCtx, done := context.WithCancel(ctx)
	defer done()

	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("code") == "" {
			return
		}
		authCode = r.URL.Query().Get("code")
		_, _ = w.Write([]byte("Received code. You may now close this tab."))
		done()
	})
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	var listenErr error
	go func() {
		ln, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			listenErr = fmt.Errorf("listen: %w", err)
			listening()
			done()
			return
		}
		srv.Addr = ln.Addr().String()
		listening()
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			listenErr = fmt.Errorf("serve: %w", err)
			done()
		}
	}()
	<-listenCtx.Done()
	if listenErr != nil {
		return nil, listenErr
	}
	config.RedirectURL = "http://" + srv.Addr + "/"

	authURL := config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))

	fmt.Printf("Opening browser for authentication...\n")
	if err := browser.OpenURL(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	<-doneCtx.Done()
	if err := srv.Shutdown(ctx); err != nil {
		return nil, err
	}

	token, err := config.Exchange(ctx, authCode,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return nil, err
	}
	return token, nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}
	return token, nil
}

func saveToken(path string, token *oauth2.Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func dataHomePath() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "deck")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get home directory: %v", err))
	}
	return filepath.Join(home, ".local", "share", "deck")
}

func stateHomePath() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "deck")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get home directory: %v", err))
	}
	return filepath.Join(home, ".local", "state", "deck")
}

func credentialsPath(profile string) string {
	creds := filepath.Join(dataHomePath(), "credentials.json")
	if profile != "" {
		p := filepath.Join(dataHomePath(), fmt.Sprintf("credentials-%s.json", profile))
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			creds = p
		}
	}
	return creds
}

func tokenFilePath(profile string) string {
	if profile != "" {
		return filepath.Join(stateHomePath(), fmt.Sprintf("token-%s.json", profile))
	}
	return filepath.Join(stateHomePath(), "token.json")
}
