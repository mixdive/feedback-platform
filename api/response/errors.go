package response

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"net/http"
	"strings"
	"unicode/utf8"
)

type ApiError struct {
	Message string `json:"message" validate:"required"` //Message of error
	Code    int    `json:"code" validate:"required"`    //Code of error which is always same with the returned HTTP status code
} //@name Error

func newApiError(statusCode int, err error) *ApiError {
	return newApiErrorFromMessage(statusCode, err.Error())
}

func newApiErrorFromMessage(statusCode int, errorMessage string) *ApiError {
	if len(errorMessage) != 0 {
		errorMessage = strings.ToUpper(errorMessage[:1]) + errorMessage[1:]
	}

	return &ApiError{
		Message: errorMessage,
		Code:    statusCode,
	}
}

func SystemError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, newApiError(http.StatusInternalServerError, err))
}

func BadRequest(c *gin.Context, err error) {
	BadRequestWithMessage(c, err.Error())
}

func BadRequestWithMessage(c *gin.Context, msg string) {
	msg = ensureUTF8(msg)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.AbortWithStatusJSON(http.StatusBadRequest, newApiErrorFromMessage(http.StatusBadRequest, msg))
}

func ValidationError(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, newApiError(http.StatusBadRequest, convertToValidationError(err)))
}

func UnauthorizedError(c *gin.Context, err error) {
	ErrorWithStatusCodeAndMessage(c, http.StatusUnauthorized, err.Error())
}

func UnauthorizedErrorWithMessage(c *gin.Context, msg string) {
	ErrorWithStatusCodeAndMessage(c, http.StatusUnauthorized, msg)
}

func ForbiddenErrorWithMessage(c *gin.Context, msg string) {
	ErrorWithStatusCodeAndMessage(c, http.StatusForbidden, msg)
}

func NotFoundWithMessage(c *gin.Context, msg string) {
	ErrorWithStatusCodeAndMessage(c, http.StatusNotFound, msg)
}

func ConflictWithMessage(c *gin.Context, msg string) {
	ErrorWithStatusCodeAndMessage(c, http.StatusConflict, msg)
}

// SetupRequired signals that the deployment hasn't been initialized yet.
// 503 with a setup-specific message — the frontend interprets this as
// "redirect to the setup form".
func SetupRequired(c *gin.Context) {
	ErrorWithStatusCodeAndMessage(c, http.StatusServiceUnavailable, "First-run setup has not been completed.")
}

func UnderMaintenance(c *gin.Context) {
	ErrorWithStatusCodeAndMessage(c, http.StatusServiceUnavailable, "System is under maintenance")
}

func ensureUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	r := transform.NewReader(bytes.NewReader([]byte(s)), charmap.ISO8859_9.NewDecoder())
	decoded, err := io.ReadAll(r)
	if err != nil {
		return s // en kötü ihtimal: bozma, ham haliyle geri ver
	}

	return string(decoded)
}

func ErrorWithStatusCodeAndMessage(c *gin.Context, statusCode int, msg string) {
	msg = ensureUTF8(msg)

	c.Header("Content-Type", "application/json; charset=utf-8")

	c.AbortWithStatusJSON(statusCode, newApiErrorFromMessage(statusCode, msg))
}

func Redirect(c *gin.Context, url string) {
	c.Redirect(http.StatusSeeOther, url)
}
