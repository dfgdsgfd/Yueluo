package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const configFileName = "config.json"

const (
	stateTTL      = 10 * time.Minute
	userCookieTTL = time.Hour
)

var defaultConfig = Config{
	Port:         "8083",
	LoginURL:     "",
	CallbackURL:  "http://localhost:8083/callback",
	ClientID:     "",
	ClientSecret: "",
	EnableDPoP:   true,
}

// Config holds the OAuth2.1 demo configuration.
type Config struct {
	Port         string `json:"port"`
	LoginURL     string `json:"login_url"`
	CallbackURL  string `json:"callback_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	EnableDPoP   bool   `json:"enable_dpop"`
}

// TokenResponse represents the OAuth2.1 token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// UserInfo represents the user information from the OAuth2.1 userinfo endpoint.
type UserInfo struct {
	Sub      string  `json:"sub"`
	UserID   uint    `json:"user_id"`
	Username string  `json:"username"`
	VIPLevel int     `json:"vip_level"`
	Balance  float64 `json:"balance"`
	Email    string  `json:"email"`
	UAHash   string  `json:"ua_hash"`
}

type stateEntry struct {
	Verifier  string
	DPoPKey   *ecdsa.PrivateKey
	DPoPJKT   string
	ExpiresAt time.Time
}

type userCookiePayload struct {
	User      UserInfo `json:"user"`
	ExpiresAt int64    `json:"exp"`
}

var (
	config         Config
	stateStore     = make(map[string]stateEntry)
	stateMu        sync.Mutex
	dpopNonceStore = make(map[string]string)
	dpopNonceMu    sync.Mutex
	cookieSecret   []byte
	httpClient     = &http.Client{Timeout: 15 * time.Second}
)

func main() {
	config = loadConfig()
	if config.LoginURL == "" {
		log.Printf("warning: login_url is empty, edit %s before using the demo", configFileName)
	}
	if config.CallbackURL == "" {
		config.CallbackURL = defaultCallbackURL(config.Port)
	}
	if config.ClientID == "" || config.ClientSecret == "" {
		log.Printf("warning: client_id/client_secret is empty, create an OAuth2.1 client in the admin panel first")
	}
	var err error
	cookieSecret, err = randomBytes(32)
	if err != nil {
		log.Fatalf("failed to generate cookie signing secret: %v", err)
	}
	if config.EnableDPoP {
		log.Printf("DPoP demo mode enabled; a fresh proof key is generated for each authorization request")
	}

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/logout", handleLogout)

	log.Printf("OAuth2.1 demo started on http://localhost:%s", config.Port)
	log.Printf("OAuth2.1 server: %s", config.LoginURL)
	log.Printf("callback URL: %s", getCallbackURL())
	server := &http.Server{
		Addr:              ":" + config.Port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func loadConfig() Config {
	cfg := defaultConfig
	if data, err := os.ReadFile(configFileName); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Printf("warning: failed to parse %s: %v", configFileName, err)
		}
	} else if os.IsNotExist(err) {
		if err := generateDefaultConfig(); err != nil {
			log.Printf("warning: failed to create default %s: %v", configFileName, err)
		} else {
			log.Printf("created %s, please fill login_url, client_id, and client_secret", configFileName)
		}
	}

	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}
	if loginURL := os.Getenv("LOGIN_URL"); loginURL != "" {
		cfg.LoginURL = loginURL
	}
	if callbackURL := os.Getenv("CALLBACK_URL"); callbackURL != "" {
		cfg.CallbackURL = callbackURL
	}
	if clientID := os.Getenv("CLIENT_ID"); clientID != "" {
		cfg.ClientID = clientID
	}
	if clientSecret := os.Getenv("CLIENT_SECRET"); clientSecret != "" {
		cfg.ClientSecret = clientSecret
	}
	if enableDPoP := os.Getenv("ENABLE_DPOP"); enableDPoP != "" {
		cfg.EnableDPoP = strings.EqualFold(enableDPoP, "true") || enableDPoP == "1"
	}
	return cfg
}

func generateDefaultConfig() error {
	// #nosec G117 -- Demo config intentionally includes an empty client_secret placeholder and is written 0600.
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFileName, data, 0600)
}

func defaultCallbackURL(port string) string {
	if port == "" {
		port = defaultConfig.Port
	}
	return fmt.Sprintf("http://localhost:%s/callback", port)
}

func getCallbackURL() string {
	if config.CallbackURL != "" {
		return config.CallbackURL
	}
	return defaultCallbackURL(config.Port)
}

func randomBytes(byteLen int) ([]byte, error) {
	raw := make([]byte, byteLen)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func generateRandomURLString(byteLen int) (string, error) {
	raw, err := randomBytes(byteLen)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func generatePKCEPair() (verifier, challenge string, err error) {
	verifier, err = generateRandomURLString(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func createState(verifier string, dpopKey *ecdsa.PrivateKey, dpopJKT string) (string, error) {
	state, err := generateRandomURLString(32)
	if err != nil {
		return "", err
	}
	now := time.Now()
	stateMu.Lock()
	pruneExpiredStatesLocked(now)
	stateStore[state] = stateEntry{
		Verifier:  verifier,
		DPoPKey:   dpopKey,
		DPoPJKT:   dpopJKT,
		ExpiresAt: now.Add(stateTTL),
	}
	stateMu.Unlock()
	return state, nil
}

func consumeState(state string) (stateEntry, bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	now := time.Now()
	pruneExpiredStatesLocked(now)
	entry, ok := stateStore[state]
	if ok {
		delete(stateStore, state)
	}
	if !ok || entry.ExpiresAt.Before(now) {
		return stateEntry{}, false
	}
	return entry, true
}

func pruneExpiredStatesLocked(now time.Time) {
	for state, entry := range stateStore {
		if entry.ExpiresAt.Before(now) {
			delete(stateStore, state)
		}
	}
}

func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	var user *UserInfo
	if cookie, err := r.Cookie("oauth21_user"); err == nil && cookie.Value != "" {
		user = parseUserCookie(cookie.Value)
	}
	tmpl := template.Must(template.New("home").Parse(homeTemplate))
	_ = tmpl.Execute(w, map[string]any{
		"LoginURL":    config.LoginURL,
		"CallbackURL": config.CallbackURL,
		"ClientID":    config.ClientID,
		"EnableDPoP":  config.EnableDPoP,
		"User":        user,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	verifier, challenge, err := generatePKCEPair()
	if err != nil {
		renderError(w, "failed to generate PKCE verifier")
		return
	}
	var dpopKey *ecdsa.PrivateKey
	var dpopJKT string
	if config.EnableDPoP {
		dpopKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			renderError(w, "failed to generate DPoP key")
			return
		}
		dpopJKT, err = publicJWKThumbprint(dpopKey)
		if err != nil {
			renderError(w, "failed to generate DPoP key thumbprint")
			return
		}
	}
	state, err := createState(verifier, dpopKey, dpopJKT)
	if err != nil {
		renderError(w, "failed to generate state")
		return
	}

	authEndpoint := strings.TrimRight(config.LoginURL, "/") + "/oauth2.1/authorize"
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", getCallbackURL())
	params.Set("state", state)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	if dpopJKT != "" {
		params.Set("dpop_jkt", dpopJKT)
	}
	http.Redirect(w, r, authEndpoint+"?"+params.Encode(), http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		renderError(w, fmt.Sprintf("authorization failed: %s - %s", errCode, r.URL.Query().Get("error_description")))
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		renderError(w, "missing authorization code")
		return
	}
	stateEntry, ok := consumeState(state)
	if !ok {
		renderError(w, "invalid state")
		return
	}

	tokenResp, err := exchangeCodeForToken(code, stateEntry.Verifier, getCallbackURL(), stateEntry.DPoPKey)
	if err != nil {
		renderError(w, fmt.Sprintf("token exchange failed: %v", err))
		return
	}
	if tokenResp.Error != "" {
		renderError(w, fmt.Sprintf("token error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc))
		return
	}

	userInfo, err := getUserInfo(tokenResp, stateEntry.DPoPKey)
	if err != nil {
		renderError(w, fmt.Sprintf("userinfo failed: %v", err))
		return
	}

	cookieValue, err := encodeUserCookie(userInfo)
	if err != nil {
		renderError(w, fmt.Sprintf("failed to sign user cookie: %v", err))
		return
	}

	// #nosec G124 -- Demo supports localhost HTTP; Secure follows the request scheme while HttpOnly/SameSite are set.
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth21_user",
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func exchangeCodeForToken(code, verifier, redirectURI string, proofKey *ecdsa.PrivateKey) (*TokenResponse, error) {
	tokenURL := strings.TrimRight(config.LoginURL, "/") + "/oauth2.1/token"
	for attempt := 0; attempt < 2; attempt++ {
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("redirect_uri", redirectURI)
		form.Set("client_id", config.ClientID)
		form.Set("client_secret", config.ClientSecret)
		form.Set("code_verifier", verifier)

		req, err := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if proofKey != nil {
			proof, err := createDPoPProof(proofKey, "POST", tokenURL, "", getDPoPNonce(tokenURL))
			if err != nil {
				return nil, err
			}
			req.Header.Set("DPoP", proof)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := readResponseBody(resp)
		if err != nil {
			return nil, err
		}
		storeDPoPNonce(tokenURL, resp.Header.Get("DPoP-Nonce"))
		if shouldRetryDPoPNonce(resp.StatusCode, body) && attempt == 0 && getDPoPNonce(tokenURL) != "" {
			continue
		}

		var tokenResp TokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			if resp.StatusCode >= http.StatusBadRequest {
				return nil, fmt.Errorf("%s: %s", resp.Status, string(body))
			}
			return nil, fmt.Errorf("parse token response: %v, body: %s", err, string(body))
		}
		if resp.StatusCode >= http.StatusBadRequest && tokenResp.Error == "" {
			return nil, fmt.Errorf("%s: %s", resp.Status, string(body))
		}
		if resp.StatusCode < http.StatusBadRequest && tokenResp.Error == "" && tokenResp.AccessToken == "" {
			return nil, fmt.Errorf("token response did not include an access token")
		}
		return &tokenResp, nil
	}
	return nil, fmt.Errorf("token endpoint requested a DPoP nonce but retry did not succeed")
}

func getUserInfo(tokenResp *TokenResponse, proofKey *ecdsa.PrivateKey) (*UserInfo, error) {
	userInfoURL := strings.TrimRight(config.LoginURL, "/") + "/oauth2.1/userinfo"
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest("GET", userInfoURL, nil)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(tokenResp.TokenType, "DPoP") {
			if proofKey == nil {
				return nil, fmt.Errorf("DPoP token returned but no DPoP key is available")
			}
			req.Header.Set("Authorization", "DPoP "+tokenResp.AccessToken)
			proof, err := createDPoPProof(proofKey, "GET", userInfoURL, tokenResp.AccessToken, getDPoPNonce(userInfoURL))
			if err != nil {
				return nil, err
			}
			req.Header.Set("DPoP", proof)
		} else {
			req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := readResponseBody(resp)
		if err != nil {
			return nil, err
		}
		storeDPoPNonce(userInfoURL, resp.Header.Get("DPoP-Nonce"))
		if shouldRetryDPoPNonce(resp.StatusCode, body) && attempt == 0 && getDPoPNonce(userInfoURL) != "" {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s: %s", resp.Status, string(body))
		}

		var userInfo UserInfo
		if err := json.Unmarshal(body, &userInfo); err != nil {
			return nil, err
		}
		return &userInfo, nil
	}
	return nil, fmt.Errorf("userinfo endpoint requested a DPoP nonce but retry did not succeed")
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return body, nil
}

func shouldRetryDPoPNonce(statusCode int, body []byte) bool {
	if statusCode < http.StatusBadRequest {
		return false
	}
	var oauthErr struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &oauthErr); err == nil && oauthErr.Error == "use_dpop_nonce" {
		return true
	}
	return strings.Contains(string(body), "use_dpop_nonce")
}

func getDPoPNonce(endpoint string) string {
	dpopNonceMu.Lock()
	defer dpopNonceMu.Unlock()
	return dpopNonceStore[endpoint]
}

func storeDPoPNonce(endpoint, nonce string) {
	if nonce == "" {
		return
	}
	dpopNonceMu.Lock()
	dpopNonceStore[endpoint] = nonce
	dpopNonceMu.Unlock()
}

func createDPoPProof(proofKey *ecdsa.PrivateKey, method, htu, accessToken, nonce string) (string, error) {
	if proofKey == nil {
		return "", fmt.Errorf("DPoP key is not initialized")
	}
	jwk, err := publicJWK(proofKey)
	if err != nil {
		return "", err
	}
	header := map[string]any{
		"typ": "dpop+jwt",
		"alg": "ES256",
		"jwk": jwk,
	}

	jti, err := generateRandomURLString(16)
	if err != nil {
		return "", err
	}
	claims := map[string]any{
		"jti": jti,
		"htm": method,
		"htu": htuWithoutQuery(htu),
		"iat": time.Now().Unix(),
	}
	if accessToken != "" {
		sum := sha256.Sum256([]byte(accessToken))
		claims["ath"] = base64.RawURLEncoding.EncodeToString(sum[:])
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	sum := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, proofKey, sum[:])
	if err != nil {
		return "", err
	}
	signature := append(padP256(r.Bytes()), padP256(s.Bytes())...)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func publicJWK(key *ecdsa.PrivateKey) (map[string]string, error) {
	publicKeyBytes, err := key.PublicKey.Bytes()
	if err != nil {
		return nil, fmt.Errorf("encode DPoP public key: %w", err)
	}
	if len(publicKeyBytes) != 65 || publicKeyBytes[0] != 4 {
		return nil, fmt.Errorf("unexpected DPoP public key format")
	}

	return map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(publicKeyBytes[1:33]),
		"y":   base64.RawURLEncoding.EncodeToString(publicKeyBytes[33:65]),
	}, nil
}

func publicJWKThumbprint(key *ecdsa.PrivateKey) (string, error) {
	jwk, err := publicJWK(key)
	if err != nil {
		return "", err
	}
	thumbprintJSON := fmt.Sprintf(`{"crv":"%s","kty":"%s","x":"%s","y":"%s"}`, jwk["crv"], jwk["kty"], jwk["x"], jwk["y"])
	sum := sha256.Sum256([]byte(thumbprintJSON))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func padP256(value []byte) []byte {
	padded := make([]byte, 32)
	copy(padded[32-len(value):], value)
	return padded
}

func htuWithoutQuery(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	// #nosec G124 -- Demo supports localhost HTTP; Secure follows the request scheme while HttpOnly/SameSite are set.
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth21_user",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func parseUserCookie(value string) *UserInfo {
	parts := strings.Split(value, ".")
	if len(parts) != 2 || len(cookieSecret) == 0 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	if !hmac.Equal(signature, signCookiePayload(payload)) {
		return nil
	}
	var payloadData userCookiePayload
	if err := json.Unmarshal(payload, &payloadData); err != nil {
		return nil
	}
	if time.Now().Unix() > payloadData.ExpiresAt {
		return nil
	}
	return &payloadData.User
}

func encodeUserCookie(user *UserInfo) (string, error) {
	payload := userCookiePayload{
		User:      *user,
		ExpiresAt: time.Now().Add(userCookieTTL).Unix(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	signature := signCookiePayload(data)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func signCookiePayload(payload []byte) []byte {
	mac := hmac.New(sha256.New, cookieSecret)
	mac.Write(payload)
	return mac.Sum(nil)
}

func renderError(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	tmpl := template.Must(template.New("error").Parse(errorTemplate))
	_ = tmpl.Execute(w, message)
}

const homeTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth2.1 Demo</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; background: #f6f7fb; color: #1f2937; }
        main { width: min(920px, calc(100% - 32px)); margin: 48px auto; }
        .panel { background: #fff; border: 1px solid #e5e7eb; border-radius: 8px; padding: 28px; box-shadow: 0 10px 30px rgba(15,23,42,.08); }
        .muted { color: #6b7280; }
        .btn { display: inline-flex; align-items: center; gap: 8px; padding: 12px 18px; border-radius: 8px; color: #fff; background: #2563eb; text-decoration: none; font-weight: 600; }
        .btn.secondary { background: #4b5563; }
        table { width: 100%; border-collapse: collapse; margin-top: 18px; }
        td { padding: 10px 8px; border-bottom: 1px solid #e5e7eb; }
        td:first-child { width: 160px; color: #6b7280; }
        code { background: #eef2ff; padding: 2px 6px; border-radius: 4px; word-break: break-all; }
    </style>
</head>
<body>
<main>
    <div class="panel">
        <h1>OAuth2.1 + PKCE Demo</h1>
        <p class="muted">使用独立 client_id/client_secret，并在授权码流程中强制启用 PKCE S256。{{if .EnableDPoP}}当前 demo 会发送 DPoP proof。{{end}}</p>
        <p>登录服务：<code>{{.LoginURL}}</code></p>
        <p>Callback URL：<code>{{.CallbackURL}}</code></p>
        <p>Client ID：<code>{{.ClientID}}</code></p>
        <p>DPoP：<code>{{if .EnableDPoP}}enabled{{else}}disabled{{end}}</code></p>
        {{if .User}}
            <h2>登录成功</h2>
            <table>
                <tr><td>User ID</td><td>{{.User.UserID}}</td></tr>
                <tr><td>Username</td><td>{{.User.Username}}</td></tr>
                <tr><td>Email</td><td>{{.User.Email}}</td></tr>
                <tr><td>VIP Level</td><td>{{.User.VIPLevel}}</td></tr>
                <tr><td>Balance</td><td>{{printf "%.2f" .User.Balance}}</td></tr>
                <tr><td>UA Hash</td><td><code>{{.User.UAHash}}</code></td></tr>
            </table>
            <p><a class="btn secondary" href="/logout">退出</a></p>
        {{else}}
            <p><a class="btn" href="/login">使用 OAuth2.1 登录</a></p>
        {{end}}
    </div>
</main>
</body>
</html>`

const errorTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth2.1 Demo Error</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; background: #fef2f2; color: #7f1d1d; }
        main { width: min(720px, calc(100% - 32px)); margin: 48px auto; background: #fff; border: 1px solid #fecaca; border-radius: 8px; padding: 28px; }
        a { color: #2563eb; }
    </style>
</head>
<body>
<main>
    <h1>OAuth2.1 授权失败</h1>
    <p>{{.}}</p>
    <p><a href="/">返回首页</a></p>
</main>
</body>
</html>`
