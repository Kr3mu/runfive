// Package auth provides Discord OAuth2 authentication.
//
// Discord uses standard OAuth2 with authorization code flow.
// The app redirects the user to Discord's authorization endpoint,
// Discord redirects back with a code, which is exchanged for an
// access token. The token is then used to fetch user data from
// the Discord API.
//
// See https://discord.com/developers/docs/topics/oauth2
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	DiscordAPIURL      = "https://discord.com/api/v10"
	discordAuthURL     = "https://discord.com/oauth2/authorize"
	discordTokenURL    = "https://discord.com/api/oauth2/token" //nolint:gosec // OAuth endpoint URL, not a credential
	discordPendingTTL  = 10 * time.Minute
	discordCleanupTick = time.Minute
)

type discordPendingAuth struct {
	nonce      string
	linkUserID *uint
	expiresAt  time.Time
}

// DiscordAuth manages the Discord OAuth2 authentication flow.
// clientID and clientSecret may be empty at startup and set later via Reconfigure.
type DiscordAuth struct {
	pending      sync.Map
	mu           sync.RWMutex
	baseURL      string
	clientID     string
	clientSecret string
}

// DiscordUserData holds the user information fetched from the Discord API.
type DiscordUserData struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"`
}

// AvatarURL returns the full CDN URL for the user's avatar.
func (u *DiscordUserData) AvatarURL() string {
	if u.Avatar == "" {
		return "https://cdn.discordapp.com/embed/avatars/0.png"
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", u.ID, u.Avatar)
}

type discordTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// NewDiscordAuth creates the Discord OAuth2 handler and starts background cleanup.
// clientID and clientSecret may be empty if not yet configured.
func NewDiscordAuth(baseURL, clientID, clientSecret string) *DiscordAuth {
	da := &DiscordAuth{
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
	go da.cleanupLoop()
	return da
}

// Reconfigure updates the Discord OAuth2 credentials at runtime without restart.
// Safe to call concurrently with ongoing auth flows.
func (da *DiscordAuth) Reconfigure(clientID, clientSecret string) {
	da.mu.Lock()
	defer da.mu.Unlock()
	da.clientID = clientID
	da.clientSecret = clientSecret
}

// IsConfigured reports whether Discord credentials have been set.
func (da *DiscordAuth) IsConfigured() bool {
	da.mu.RLock()
	defer da.mu.RUnlock()
	return da.clientID != "" && da.clientSecret != ""
}

// StartAuth generates a state token and returns the Discord authorization URL.
// Returns an error if credentials are not configured.
func (da *DiscordAuth) StartAuth(linkUserID *uint) (string, error) {
	if !da.IsConfigured() {
		return "", fmt.Errorf("discord oauth is not configured")
	}

	state := uuid.New().String()
	da.pending.Store(state, &discordPendingAuth{
		nonce:      uuid.New().String(),
		linkUserID: linkUserID,
		expiresAt:  time.Now().Add(discordPendingTTL),
	})

	da.mu.RLock()
	clientID := da.clientID
	da.mu.RUnlock()

	params := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {da.baseURL + "/v1/auth/discord/callback"},
		"response_type": {"code"},
		"scope":         {"identify email"},
		"state":         {state},
	}

	return discordAuthURL + "?" + params.Encode(), nil
}

// HandleCallback processes the Discord callback, exchanges the code, and fetches user data.
func (da *DiscordAuth) HandleCallback(state, code string) (userData *DiscordUserData, accessToken string, linkUserID *uint, err error) {
	val, ok := da.pending.LoadAndDelete(state)
	if !ok {
		return nil, "", nil, fmt.Errorf("unknown or expired auth state")
	}
	pa := val.(*discordPendingAuth)

	if time.Now().After(pa.expiresAt) {
		return nil, "", nil, fmt.Errorf("auth attempt expired")
	}

	token, err := da.exchangeCode(code)
	if err != nil {
		return nil, "", nil, fmt.Errorf("exchange code: %w", err)
	}

	userData, err = fetchDiscordUser(token.AccessToken)
	if err != nil {
		return nil, "", nil, fmt.Errorf("fetch user data: %w", err)
	}

	return userData, token.AccessToken, pa.linkUserID, nil
}

func (da *DiscordAuth) exchangeCode(code string) (*discordTokenResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	da.mu.RLock()
	clientID := da.clientID
	clientSecret := da.clientSecret
	da.mu.RUnlock()

	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {da.baseURL + "/v1/auth/discord/callback"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", discordTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("discord returned %d: %s", resp.StatusCode, string(body))
	}

	var token discordTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &token, nil
}

func fetchDiscordUser(accessToken string) (*DiscordUserData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", DiscordAPIURL+"/users/@me", http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("discord api returned %d: %s", resp.StatusCode, string(body))
	}

	var user DiscordUserData
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if user.ID == "" {
		return nil, fmt.Errorf("empty user data from discord api")
	}

	return &user, nil
}

func (da *DiscordAuth) cleanupLoop() {
	ticker := time.NewTicker(discordCleanupTick)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		da.pending.Range(func(key, value interface{}) bool {
			pa := value.(*discordPendingAuth)
			if now.After(pa.expiresAt) {
				da.pending.Delete(key)
			}
			return true
		})
	}
}
