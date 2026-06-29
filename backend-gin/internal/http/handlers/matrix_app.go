package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/http/swagger"
	"yuem-go/backend-gin/internal/http/swaggerui"
)

func (h NativeHandlers) SwaggerDocsPage(c *gin.Context) {
	specPath := "/api/" + h.Config.Debug.SwaggerDocsPath + ".json"
	config, err := json.Marshal(map[string]any{
		"url":                specPath,
		"hiddenClients":      []string{},
		"withDefaultFonts":   false,
		"showDeveloperTools": "never",
		"showSidebar":        true,
		"telemetry":          false,
		"defaultHttpClient": map[string]string{
			"targetKey": "shell",
			"clientKey": "curl",
		},
		"agent": map[string]any{
			"disabled": true,
		},
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="API Reference">
  <title>Yuem Go-Gin API</title>
  <link rel="stylesheet" href="`+swaggerui.ScalarCSSPath()+`">
  <style>
    html, body, #scalar-api-reference {
      min-height: 100%;
      margin: 0;
    }
  </style>
</head>
<body>
  <div id="scalar-api-reference"></div>
  <script src="`+swaggerui.ScalarJSPath()+`"></script>
  <script>
    (function () {
      const scalarConfig = `+string(config)+`;

      function removeEmptyAuthorizationHeader(headers) {
        if (!headers) return;
        if (typeof headers.has === "function" && typeof headers.set === "function") {
          const value = headers.get("authorization");
          if (value && /^Bearer\s*$/i.test(value)) {
            headers.delete("authorization");
          }
          return;
        }
        if (Array.isArray(headers)) {
          for (let index = headers.length - 1; index >= 0; index -= 1) {
            const entry = headers[index];
            if (Array.isArray(entry) && String(entry[0]).toLowerCase() === "authorization" && /^Bearer\s*$/i.test(String(entry[1] || ""))) {
              headers.splice(index, 1);
            }
          }
          return;
        }
        if (typeof headers === "object") {
          Object.keys(headers).forEach(function (key) {
            if (key.toLowerCase() === "authorization" && /^Bearer\s*$/i.test(String(headers[key] || ""))) {
              delete headers[key];
            }
          });
        }
      }

      scalarConfig.customFetch = function (input, init) {
        const requestHeaders = typeof Request !== "undefined" && input instanceof Request ? input.headers : undefined;
        const headers = new Headers(init && init.headers || requestHeaders);
        removeEmptyAuthorizationHeader(headers);
        return window.fetch(input, {
          ...(init || {}),
          headers,
          credentials: "same-origin",
        });
      };

      scalarConfig.onBeforeRequest = function (_ref) {
        const requestBuilder = _ref && _ref.requestBuilder;
        if (requestBuilder) {
          requestBuilder.headers = requestBuilder.headers || {};
          removeEmptyAuthorizationHeader(requestBuilder.headers);
        }
      };

      window.Scalar.createApiReference("#scalar-api-reference", scalarConfig);
    })();
  </script>
</body>
</html>`)
}

func (h NativeHandlers) SwaggerUICSS(c *gin.Context) {
	var (
		data []byte
		err  error
	)
	if scalarAssetRequested(c) {
		data, err = swaggerui.ScalarCSS()
	} else {
		data, err = swaggerui.CSS()
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	c.Header("Content-Type", "text/css; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "text/css; charset=utf-8", data)
}

func (h NativeHandlers) SwaggerUIJS(c *gin.Context) {
	var (
		data []byte
		err  error
	)
	if scalarAssetRequested(c) {
		data, err = swaggerui.ScalarJS()
	} else {
		data, err = swaggerui.JS()
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/javascript; charset=utf-8", data)
}

func scalarAssetRequested(c *gin.Context) bool {
	return c.Query("ui") == swaggerui.ScalarUIQuery()
}

func (h NativeHandlers) SwaggerUIStandalonePreset(c *gin.Context) {
	data, err := swaggerui.StandalonePreset()
	if err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/javascript; charset=utf-8", data)
}

func (h NativeHandlers) appDispatch(c *gin.Context) {
	path := c.Request.URL.Path
	method := matrixMethod(c)
	switch {
	case method == http.MethodGet && path == "/api/health":
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "timestamp": time.Now().UTC().Format(time.RFC3339Nano)})
	case method == http.MethodGet && path == "/api/ua-block/check":
		h.UABlockCheck(c)
	case method == http.MethodGet && path == "/api/app/download-config":
		h.AppDownloadConfig(c)
	case method == http.MethodGet && path == "/api/app/check-update":
		h.CheckAppUpdate(c)
	case method == http.MethodPost && path == "/api/app/report-event":
		h.ReportAppEvent(c)
	case method == http.MethodGet && strings.HasSuffix(path, ".json") && strings.Contains(path, h.Config.Debug.SwaggerDocsPath):
		c.Data(http.StatusOK, "application/json; charset=utf-8", swagger.JSON)
	case method == http.MethodPost && strings.Contains(path, h.Config.Debug.JWTTestTokenPath):
		h.debugJWTToken(c)
	case method == http.MethodGet && strings.Contains(path, h.Config.Debug.JWTTestTokenPath):
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, "<!doctype html><html><body><h1>JWT test token</h1></body></html>")
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "app route not found", nil)
	}
}

func (h NativeHandlers) debugJWTToken(c *gin.Context) {
	if h.Auth == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	body := readBodyMap(c)
	userID, ok := int64FromAny(body["userId"])
	if !ok || userID <= 0 {
		userID = 1
	}
	tokenType := toString(body["type"])
	displayID := toString(body["user_id"])
	payload := gin.H{}
	if tokenType == "admin" {
		if displayID == "" {
			displayID = "admin"
		}
		payload = gin.H{"adminId": userID, "username": displayID, "type": "admin"}
	} else {
		if displayID == "" {
			displayID = "test_user"
		}
		payload = gin.H{"userId": userID, "user_id": displayID}
	}
	access, err := h.Auth.GenerateAccessToken(payload)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	refresh, err := h.Auth.GenerateRefreshToken(payload)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	if tokenType != "admin" {
		h.createUserSession(c, userID, access, refresh)
	}
	writeSuccess(c, "test token generated", gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"payload":       payload,
		"usage":         "Bearer " + access,
		"userId":        strconv.FormatInt(userID, 10),
	})
}
