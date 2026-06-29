package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess         = 200
	CodeError           = 500
	CodeValidationError = 400
	CodeUnauthorized    = 401
	CodeForbidden       = 403
	CodeNotFound        = 404
	CodeConflict        = 409
	CodeTooManyRequests = 429
)

func Success(c *gin.Context, data any, message string) {
	body := gin.H{
		"code":    CodeSuccess,
		"message": message,
	}
	if data != nil {
		body["data"] = data
	}
	c.JSON(http.StatusOK, body)
}

func JSON(c *gin.Context, status int, code int, message string, data any) {
	body := gin.H{
		"code":    code,
		"message": message,
	}
	if data != nil {
		body["data"] = data
	}
	c.JSON(status, body)
}

func Error(c *gin.Context, message string) {
	JSON(c, http.StatusInternalServerError, CodeError, message, nil)
}

func ValidationError(c *gin.Context, message string, data any) {
	JSON(c, http.StatusBadRequest, CodeValidationError, message, data)
}

func NotFound(c *gin.Context, message string) {
	JSON(c, http.StatusNotFound, CodeNotFound, message, nil)
}
