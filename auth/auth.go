package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/searchconsole/v1"
	"google.golang.org/api/webmasters/v3"
)

const (
	// TokenCacheDir is the directory in home folder to store auth tokens
	TokenCacheDir = ".gsc-cli"
	// TokenCacheFile is the file name for cached token
	TokenCacheFile = "token.json"
	// ClientSecretCacheFile is the file name for cached client secret
	ClientSecretCacheFile = "client_secret.json"
)

// Services represents the Google Search Console clients
type Services struct {
	Webmasters    *webmasters.Service
	SearchConsole *searchconsole.Service
}

// GetServices initializes and returns the Search Console services based on available credentials.
func GetServices(ctx context.Context, credsFile string, clientSecretFile string) (*Services, error) {
	var opts []option.ClientOption

	// 1. Try Service Account credentials from CLI flag or Env
	if credsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credsFile))
	} else if envCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); envCreds != "" {
		opts = append(opts, option.WithCredentialsFile(envCreds))
	} else {
		// 2. Try OAuth2 flow (explicitly passed or from cache)
		secretFileToUse := clientSecretFile
		if secretFileToUse == "" {
			if cachedPath, err := cachedClientSecretPath(); err == nil {
				if _, err := os.Stat(cachedPath); err == nil {
					secretFileToUse = cachedPath
				}
			}
		}

		if secretFileToUse != "" {
			// Cache the client secret if it was explicitly passed in and differs from the cached location
			if clientSecretFile != "" {
				if cachedPath, err := cachedClientSecretPath(); err == nil {
					_ = cacheClientSecret(clientSecretFile, cachedPath)
				}
			}

			tokenSource, err := getOAuth2TokenSource(ctx, secretFileToUse)
			if err != nil {
				return nil, fmt.Errorf("OAuth2 authentication failed: %w", err)
			}
			opts = append(opts, option.WithTokenSource(tokenSource))
		} else {
			// 3. Try Application Default Credentials (ADC) without explicit file
			creds, err := google.FindDefaultCredentials(ctx, webmasters.WebmastersScope)
			if err == nil {
				opts = append(opts, option.WithCredentials(creds))
			} else {
				return nil, fmt.Errorf("no credentials found. Please provide either:\n" +
					"  - A Service Account key file via -c/--credentials flag or GOOGLE_APPLICATION_CREDENTIALS\n" +
					"  - An OAuth2 Client Secret JSON file via -s/--client-secret flag")
			}
		}
	}

	// Add Scopes to ensure they are set
	opts = append(opts, option.WithScopes(webmasters.WebmastersScope))

	// Initialize the Services
	webmastersSvc, err := webmasters.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Webmasters service: %w", err)
	}

	searchConsoleSvc, err := searchconsole.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Search Console (URL Inspection) service: %w", err)
	}

	return &Services{
		Webmasters:    webmastersSvc,
		SearchConsole: searchConsoleSvc,
	}, nil
}

// getOAuth2TokenSource loads cached token or runs the OAuth2 authorization flow.
func getOAuth2TokenSource(ctx context.Context, clientSecretFile string) (oauth2.TokenSource, error) {
	b, err := os.ReadFile(clientSecretFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %w", err)
	}

	// Create config for OAuth2 using both webmaster and searchconsole scopes
	config, err := google.ConfigFromJSON(b, webmasters.WebmastersScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret: %w", err)
	}



	// Try reading cached token
	tokenFile, err := tokenCachePath()
	if err != nil {
		return nil, err
	}

	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		// If reading from file fails (e.g. doesn't exist), run interactive flow
		fmt.Printf("\033[33mNo cached token found. Starting interactive OAuth2 login...\033[0m\n")
		tok, err = tokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}
		err = saveToken(tokenFile, tok)
		if err != nil {
			fmt.Printf("\033[31mWarning: Failed to cache token: %v\033[0m\n", err)
		} else {
			fmt.Printf("\033[32mSuccessfully authenticated and cached token to %s\033[0m\n", tokenFile)
		}
	}

	return config.TokenSource(ctx, tok), nil
}

// tokenFromWeb starts a local web server to receive the authorization code.
func tokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	codeChan := make(chan string)
	errChan := make(chan error)

	// 1. Find a free local TCP port on loopback interface
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to allocate local port for OAuth callback: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	// 2. Dynamically set redirect URL to the allocated loopback port
	config.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/oauth2callback", port)

	// Create standard authentication URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code found in callback")
			fmt.Fprint(w, "Authentication failed! No code was received.")
			return
		}
		codeChan <- code
		fmt.Fprint(w, "Authentication successful! You can now close this tab and return to the CLI terminal.")
	})

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start local callback server: %w", err)
		}
	}()

	fmt.Printf("\033[1;36m1. Please open the following URL in your browser to authorize this CLI:\033[0m\n\n%s\n\n", authURL)
	fmt.Printf("\033[1;36m2. Waiting for login callback on %s...\033[0m\n", config.RedirectURL)

	// Set a timeout of 3 minutes for authorization
	select {
	case code := <-codeChan:
		// Shut down local server
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)

		// Exchange code for token
		tok, err := config.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
		}
		return tok, nil

	case err := <-errChan:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return nil, err

	case <-time.After(3 * time.Minute):
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return nil, fmt.Errorf("authentication timed out (3 minutes)")
	}
}

// tokenCachePath returns the path to the cached token file.
func tokenCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home directory: %w", err)
	}
	dir := filepath.Join(home, TokenCacheDir)
	return filepath.Join(dir, TokenCacheFile), nil
}

// tokenFromFile retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("unable to create directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache OAuth token: %w", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

// cachedClientSecretPath returns the path to the cached client secret JSON file.
func cachedClientSecretPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home directory: %w", err)
	}
	dir := filepath.Join(home, TokenCacheDir)
	return filepath.Join(dir, ClientSecretCacheFile), nil
}

// cacheClientSecret saves a copy of the client secret JSON file to the cache directory.
func cacheClientSecret(src, dst string) error {
	if src == dst {
		return nil
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0600)
}

