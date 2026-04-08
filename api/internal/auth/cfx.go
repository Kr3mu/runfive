// Package auth provides Cfx.re authentication via Discourse User API Keys.
//
// forum.cfx.re does not offer standard OAuth2 — its IDMS is private.
// The only supported third-party auth flow uses Discourse's User API Key
// protocol: the app generates an RSA keypair, redirects the user to
// forum.cfx.re/user-api-key/new with the public key, and receives an
// RSA-encrypted payload containing a User API Key on callback.
//
// The API key is then used to fetch user data from /session/current.json.
//
// See https://meta.discourse.org/t/user-api-keys-specification/48536
package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// CfxForumURL is the base URL of the Cfx.re Discourse forum.
	CfxForumURL = "https://forum.cfx.re"
	// rsaKeyBits is the RSA key size for the Discourse handshake.
	rsaKeyBits = 2048
	// pendingAuthTTL is how long a pending auth attempt stays valid.
	pendingAuthTTL = 10 * time.Minute
	// cleanupTick is the interval for purging expired pending auths.
	cleanupTick = time.Minute
)

// pendingAuth holds the RSA private key and metadata for an in-flight auth attempt.
type pendingAuth struct {
	// RSA private key to decrypt the callback payload
	privateKey *rsa.PrivateKey
	// Random nonce sent to Discourse, must match on callback
	nonce string
	// If set, the callback links the Cfx account to this existing user instead of creating/finding one
	linkUserID *uint
	// Absolute expiry for this pending auth
	expiresAt time.Time
}

// CfxAuth manages the Discourse User API Key authentication flow.
type CfxAuth struct {
	// pending stores in-flight auth attempts keyed by state UUID
	pending sync.Map
	// baseURL is the application's public URL for constructing redirect URIs
	baseURL string
	// appName is displayed on the Discourse authorization page
	appName string
}

// discoursePayload is the RSA-decrypted JSON from the callback.
type discoursePayload struct {
	// Discourse User API Key (hex string)
	Key string `json:"key"`
	// Nonce echoed back for verification
	Nonce string `json:"nonce"`
}

// CfxUserData holds the user information fetched from forum.cfx.re.
type CfxUserData struct {
	// Discourse user ID
	ID int `json:"id"`
	// Discourse username
	Username string `json:"username"`
	// Avatar URL template with {size} placeholder
	AvatarTemplate string `json:"avatar_template"`
}

// discourseCurrentUser wraps the /session/current.json response.
type discourseCurrentUser struct {
	CurrentUser CfxUserData `json:"current_user"`
}

// NewCfxAuth creates the Cfx.re auth handler and starts background cleanup
// of expired pending auth attempts.
func NewCfxAuth(baseURL string) *CfxAuth {
	ca := &CfxAuth{
		baseURL: baseURL,
		appName: "RunFive",
	}
	go ca.cleanupLoop()
	return ca
}

// StartAuth generates an RSA keypair, stores it in memory, and returns
// the Discourse redirect URL the user should be sent to.
// If linkUserID is non-nil, the callback will link the Cfx account to this user.
func (ca *CfxAuth) StartAuth(linkUserID *uint) (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return "", fmt.Errorf("generate rsa key: %w", err)
	}

	nonce := uuid.New().String()
	state := uuid.New().String()

	ca.pending.Store(state, &pendingAuth{
		privateKey: privateKey,
		nonce:      nonce,
		linkUserID: linkUserID,
		expiresAt:  time.Now().Add(pendingAuthTTL),
	})

	pubKeyPEM, err := marshalPublicKeyPEM(&privateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("marshal public key: %w", err)
	}

	callbackURL := ca.baseURL + "/v1/auth/cfx/callback"

	params := url.Values{
		"auth_redirect":    {callbackURL + "?state=" + state},
		"application_name": {ca.appName},
		"client_id":        {state},
		"nonce":            {nonce},
		"scopes":           {"session_info"},
		"public_key":       {pubKeyPEM},
	}

	return CfxForumURL + "/user-api-key/new?" + params.Encode(), nil
}

// HandleCallback processes the callback from forum.cfx.re, decrypts the
// payload, verifies the nonce, and fetches user data.
func (ca *CfxAuth) HandleCallback(state string, encryptedPayload string) (*CfxUserData, string, *uint, error) {
	val, ok := ca.pending.LoadAndDelete(state)
	if !ok {
		return nil, "", nil, fmt.Errorf("unknown or expired auth state")
	}
	pa := val.(*pendingAuth)

	if time.Now().After(pa.expiresAt) {
		return nil, "", nil, fmt.Errorf("auth attempt expired")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedPayload)
	if err != nil {
		return nil, "", nil, fmt.Errorf("decode payload: %w", err)
	}

	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, pa.privateKey, ciphertext) //nolint:staticcheck // Discourse User API Keys protocol requires PKCS1v15
	if err != nil {
		return nil, "", nil, fmt.Errorf("decrypt payload: %w", err)
	}

	var payload discoursePayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, "", nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	if payload.Nonce != pa.nonce {
		return nil, "", nil, fmt.Errorf("nonce mismatch")
	}

	userData, err := fetchCfxUser(payload.Key)
	if err != nil {
		return nil, "", nil, fmt.Errorf("fetch user data: %w", err)
	}

	return userData, payload.Key, pa.linkUserID, nil
}

// fetchCfxUser calls forum.cfx.re/session/current.json with the User API Key
// to retrieve the authenticated user's profile.
func fetchCfxUser(apiKey string) (*CfxUserData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", CfxForumURL+"/session/current.json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Api-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("cfx api returned %d: %s", resp.StatusCode, string(body))
	}

	var result discourseCurrentUser
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.CurrentUser.ID == 0 {
		return nil, fmt.Errorf("empty user data from cfx api")
	}

	return &result.CurrentUser, nil
}

func marshalPublicKeyPEM(pub *rsa.PublicKey) (string, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}
	return string(pem.EncodeToMemory(block)), nil
}

func (ca *CfxAuth) cleanupLoop() {
	ticker := time.NewTicker(cleanupTick)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		ca.pending.Range(func(key, value interface{}) bool {
			pa := value.(*pendingAuth)
			if now.After(pa.expiresAt) {
				ca.pending.Delete(key)
			}
			return true
		})
	}
}
