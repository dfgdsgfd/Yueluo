package handlers

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
)

type oauthStateEntry struct {
	CodeVerifier     string
	DPoPKey          *ecdsa.PrivateKey
	AppState         string
	AppCodeChallenge string
	AppCallbackURL   string
	AppReturnURL     string
}

func (h NativeHandlers) oauthLogin(c *gin.Context) {
	if !h.Config.OAuth2.Enabled {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "OAuth2登录未启用", nil)
		return
	}
	if h.Config.OAuth2.LoginURL == "" || h.Config.OAuth2.ClientID == "" {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "OAuth2配置不完整", nil)
		return
	}
	if h.Cache == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "OAuth2 state缓存不可用", nil)
		return
	}
	verifier := base64URLRandom(32)
	challenge := base64URLSHA256(verifier)
	state := base64URLRandom(24)
	entry := oauthStateEntry{CodeVerifier: verifier}
	if rawAppCallback := strings.TrimSpace(c.Query("app_callback")); rawAppCallback != "" {
		appCallback, ok := h.safeOAuthMobileCallbackURL(rawAppCallback)
		if !ok {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.oauth_app_invalid_request", nil)
			return
		}
		entry.AppReturnURL = h.safeOAuthMobileReturnURL(c, c.Query("app_return_url"))
		callbackURL, _ := url.Parse(appCallback)
		callbackQuery := callbackURL.Query()
		callbackQuery.Set("url", entry.AppReturnURL)
		callbackURL.RawQuery = callbackQuery.Encode()
		entry.AppCallbackURL = callbackURL.String()
	} else if strings.EqualFold(c.Query("client"), oauthAppClient) {
		appState := strings.TrimSpace(c.Query("app_state"))
		appChallenge := strings.TrimSpace(c.Query("code_challenge"))
		if !validOAuthAppState(appState) || c.Query("code_challenge_method") != "S256" || !validPKCEChallenge(appChallenge) {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.oauth_app_invalid_request", nil)
			return
		}
		entry.AppState = appState
		entry.AppCodeChallenge = appChallenge
	}
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {h.Config.OAuth2.ClientID},
		"redirect_uri":          {h.oauthCallbackURL(c)},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	if h.Config.OAuth2.EnableDPoP {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "DPoP密钥生成失败", nil)
			return
		}
		entry.DPoPKey = key
		params.Set("dpop_jkt", dpopThumbprint(key))
	}
	h.Cache.Set("oauth2:state:"+state, entry, 10*time.Minute)
	c.Redirect(http.StatusFound, h.Config.OAuth2.LoginURL+"/oauth2.1/authorize?"+params.Encode())
}

func (h NativeHandlers) oauthCallback(c *gin.Context) {
	if !h.Config.OAuth2.Enabled {
		h.recordSecurityAudit(c, "oauth_login", "callback", "failure", "oauth2_disabled", http.StatusFound, nil, "user", "", nil)
		c.Redirect(http.StatusFound, "/?error=oauth2_disabled")
		return
	}
	if oauthErr := c.Query("error"); oauthErr != "" {
		h.recordSecurityAudit(c, "oauth_login", "callback", "failure", "oauth2_auth_error", http.StatusFound, nil, "user", "", gin.H{"oauth_error": oauthErr})
		if entry, ok := h.consumeOAuthState(c.Query("state")); ok {
			if entry.AppCallbackURL != "" {
				c.Redirect(http.StatusFound, h.oauthMobileRedirectLocation(entry, "", "oauth2_auth_error"))
				return
			}
			if entry.AppState != "" {
				c.Redirect(http.StatusFound, h.oauthAppErrorRedirectLocation(entry.AppState, "oauth2_auth_error"))
				return
			}
		}
		values := url.Values{"error": {"oauth2_auth_error"}, "message": {firstNonEmpty(c.Query("error_description"), oauthErr)}}
		c.Redirect(http.StatusFound, "/?"+values.Encode())
		return
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		h.recordSecurityAudit(c, "oauth_login", "callback", "failure", "missing_code", http.StatusFound, nil, "user", "", nil)
		c.Redirect(http.StatusFound, "/?error=missing_code")
		return
	}
	entry, ok := h.consumeOAuthState(state)
	if !ok || entry.CodeVerifier == "" {
		h.recordSecurityAudit(c, "oauth_login", "callback", "failure", "invalid_state", http.StatusFound, nil, "user", "", nil)
		c.Redirect(http.StatusFound, "/?error=invalid_state")
		return
	}
	tokenData, err := h.exchangeOAuthToken(c.Request.Context(), code, entry, h.oauthCallbackURL(c))
	if err != nil {
		h.recordSecurityAudit(c, "oauth_login", "callback", "failure", "token_error", http.StatusFound, nil, "user", "", gin.H{"error": err.Error()})
		h.redirectOAuthCallbackFailure(c, entry, "token_error", err.Error())
		return
	}
	accessToken := toString(tokenData["access_token"])
	if accessToken == "" {
		h.recordSecurityAudit(c, "oauth_login", "callback", "failure", "missing_access_token", http.StatusFound, nil, "user", "", nil)
		h.redirectOAuthCallbackFailure(c, entry, "missing_access_token", "")
		return
	}
	userInfo, err := h.fetchOAuthUserInfo(c.Request.Context(), tokenData, entry)
	if err != nil {
		h.recordSecurityAudit(c, "oauth_login", "callback", "failure", "userinfo_error", http.StatusFound, nil, "user", "", gin.H{"error": err.Error()})
		h.redirectOAuthCallbackFailure(c, entry, "userinfo_error", err.Error())
		return
	}
	oauthID, ok := int64FromAny(firstPresent(userInfo, "user_id", "sub", "id"))
	if !ok || oauthID <= 0 {
		h.recordSecurityAudit(c, "oauth_login", "callback", "failure", "invalid_user_id", http.StatusFound, nil, "user", "", nil)
		h.redirectOAuthCallbackFailure(c, entry, "invalid_user_id", "")
		return
	}
	user, isNew, ok := h.findOrCreateOAuthUser(c, oauthID, toString(userInfo["username"]), toString(userInfo["email"]))
	if !ok {
		return
	}
	if entry.AppCallbackURL != "" {
		h.completeOAuthMobileCallback(c, entry, *user, isNew)
		return
	}
	if entry.AppState != "" {
		h.completeOAuthAppCallback(c, entry, *user, isNew)
		return
	}
	if _, _, ok := h.issueUserTokens(c, user.ID, user.UserID); !ok {
		return
	}
	h.recordSecurityAudit(c, "oauth_login", "callback", "success", "", http.StatusFound, &user.ID, "user", user.UserID, gin.H{"is_new_user": isNew})
	c.Redirect(http.StatusFound, oauthSuccessRedirectLocation(isNew))
}

func (h NativeHandlers) redirectOAuthCallbackFailure(c *gin.Context, entry oauthStateEntry, code string, message string) {
	if entry.AppCallbackURL != "" {
		c.Redirect(http.StatusFound, h.oauthMobileRedirectLocation(entry, "", code))
		return
	}
	if entry.AppState != "" {
		c.Redirect(http.StatusFound, h.oauthAppErrorRedirectLocation(entry.AppState, code))
		return
	}
	values := url.Values{"error": {code}}
	if message != "" {
		values.Set("message", message)
	}
	c.Redirect(http.StatusFound, "/?"+values.Encode())
}

func (h NativeHandlers) sendEmailVerificationCode(to string, code string) error {
	subject := "【汐社校园图文社区】邮箱验证"
	body := `<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
<h1 style="color: #333; text-align: center;">汐社校园图文社区</h1>
<p style="color: #666; font-size: 16px;">您的邮箱验证码是：</p>
<div style="text-align: center; margin: 30px 0;">
<span style="font-size: 32px; font-weight: bold; color: #000000; letter-spacing: 5px;">` + code + `</span>
</div>
<p style="color: #666; font-size: 16px;">请在10分钟内使用此验证码完成操作。</p>
</div>`
	return h.sendSMTPMail(to, subject, body)
}

func (h NativeHandlers) consumeOAuthState(state string) (oauthStateEntry, bool) {
	if h.Cache == nil || state == "" {
		return oauthStateEntry{}, false
	}
	value, ok := h.Cache.Get("oauth2:state:" + state)
	if !ok {
		return oauthStateEntry{}, false
	}
	h.Cache.Delete("oauth2:state:" + state)
	entry, ok := value.(oauthStateEntry)
	return entry, ok
}

func (h NativeHandlers) oauthCallbackURL(c *gin.Context) string {
	if h.Config.OAuth2.RedirectURI != "" {
		return h.Config.OAuth2.RedirectURI
	}
	path := oauthCallbackPath(h.Config.OAuth2.CallbackPath)
	if h.Config.OAuth2.RedirectBaseURL != "" {
		return strings.TrimRight(h.Config.OAuth2.RedirectBaseURL, "/") + path
	}
	proto := firstNonEmpty(c.GetHeader("X-Forwarded-Proto"), "http")
	host := firstNonEmpty(c.GetHeader("X-Forwarded-Host"), c.Request.Host)
	return proto + "://" + host + path
}

func oauthCallbackPath(path string) string {
	path = firstNonEmpty(path, "/api/auth/oauth2/callback")
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func (h NativeHandlers) exchangeOAuthToken(ctx context.Context, code string, entry oauthStateEntry, redirectURI string) (map[string]any, error) {
	tokenURL := h.Config.OAuth2.LoginURL + "/oauth2.1/token"
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {h.Config.OAuth2.ClientID},
		"redirect_uri":  {redirectURI},
		"code_verifier": {entry.CodeVerifier},
	}
	if h.Config.OAuth2.ClientSecret != "" {
		form.Set("client_secret", h.Config.OAuth2.ClientSecret)
	}
	var nonce string
	for attempt := range 2 {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if entry.DPoPKey != nil {
			req.Header.Set("DPoP", createDPoPProof(entry.DPoPKey, http.MethodPost, tokenURL, "", nonce))
		}
		resp, body, err := doOAuthRequest(req)
		if err != nil {
			return nil, err
		}
		if value := resp.Header.Get("DPoP-Nonce"); value != "" {
			nonce = value
		}
		if entry.DPoPKey != nil && attempt == 0 && nonce != "" && shouldRetryDPoPNonce(resp.StatusCode, body) {
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, errors.New(firstNonEmpty(toString(parsed["error_description"]), toString(parsed["error"]), resp.Status))
		}
		if errText := toString(parsed["error"]); errText != "" {
			return nil, errors.New(firstNonEmpty(toString(parsed["error_description"]), errText))
		}
		return parsed, nil
	}
	return nil, errors.New("token端点要求DPoP nonce，但重试未成功")
}

func (h NativeHandlers) fetchOAuthUserInfo(ctx context.Context, tokenData map[string]any, entry oauthStateEntry) (map[string]any, error) {
	userInfoURL := h.Config.OAuth2.LoginURL + "/oauth2.1/userinfo"
	accessToken := toString(tokenData["access_token"])
	tokenType := strings.ToLower(toString(tokenData["token_type"]))
	useDPoP := tokenType == "dpop"
	if useDPoP && entry.DPoPKey == nil {
		return nil, errors.New("授权服务器返回DPoP令牌，但本次流程没有可用DPoP密钥")
	}
	var nonce string
	for attempt := range 2 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
		if err != nil {
			return nil, err
		}
		if useDPoP {
			req.Header.Set("Authorization", "DPoP "+accessToken)
			req.Header.Set("DPoP", createDPoPProof(entry.DPoPKey, http.MethodGet, userInfoURL, accessToken, nonce))
		} else {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
		resp, body, err := doOAuthRequest(req)
		if err != nil {
			return nil, err
		}
		if value := resp.Header.Get("DPoP-Nonce"); value != "" {
			nonce = value
		}
		if useDPoP && attempt == 0 && nonce != "" && shouldRetryDPoPNonce(resp.StatusCode, body) {
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("获取用户信息失败: %s", resp.Status)
		}
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	}
	return nil, errors.New("userinfo端点要求DPoP nonce，但重试未成功")
}

func doOAuthRequest(req *http.Request) (*http.Response, []byte, error) {
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return resp, body, err
}

func createDPoPProof(key *ecdsa.PrivateKey, method string, rawURL string, accessToken string, nonce string) string {
	header := gin.H{"typ": "dpop+jwt", "alg": "ES256", "jwk": publicJWK(key)}
	claims := gin.H{
		"jti": base64URLRandom(16),
		"htm": method,
		"htu": urlWithoutQuery(rawURL),
		"iat": time.Now().Unix(),
	}
	if accessToken != "" {
		claims["ath"] = base64URLSHA256(accessToken)
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	signingInput := base64URLJSON(header) + "." + base64URLJSON(claims)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		return ""
	}
	signature := append(padBigInt(r, 32), padBigInt(s, 32)...)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func shouldRetryDPoPNonce(status int, body []byte) bool {
	if status < 400 || len(body) == 0 {
		return false
	}
	var parsed map[string]any
	if json.Unmarshal(body, &parsed) == nil && toString(parsed["error"]) == "use_dpop_nonce" {
		return true
	}
	return bytes.Contains(body, []byte("use_dpop_nonce"))
}

func publicJWK(key *ecdsa.PrivateKey) gin.H {
	return gin.H{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(padBigInt(key.X, 32)),
		"y":   base64.RawURLEncoding.EncodeToString(padBigInt(key.Y, 32)),
	}
}

func dpopThumbprint(key *ecdsa.PrivateKey) string {
	jwk := publicJWK(key)
	canonical := `{"crv":"` + toString(jwk["crv"]) + `","kty":"` + toString(jwk["kty"]) + `","x":"` + toString(jwk["x"]) + `","y":"` + toString(jwk["y"]) + `"}`
	return base64URLSHA256(canonical)
}

func base64URLRandom(n int) string {
	data := make([]byte, n)
	if _, err := rand.Read(data); err != nil {
		return randomHex(n)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64URLSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func base64URLJSON(value any) string {
	data, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(data)
}

func padBigInt(value *big.Int, size int) []byte {
	raw := value.Bytes()
	if len(raw) >= size {
		return raw[len(raw)-size:]
	}
	out := make([]byte, size)
	copy(out[size-len(raw):], raw)
	return out
}

func urlWithoutQuery(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func strconvBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func oauthSuccessRedirectLocation(isNew bool) string {
	values := url.Values{
		"oauth2_login": {"success"},
		"is_new_user":  {strconvBool(isNew)},
	}
	return "/explore?" + values.Encode()
}
