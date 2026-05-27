package middlewares

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SessionCookieName is the single, hardcoded cookie name used everywhere.
const SessionCookieName = "mixdive_session"

// SessionCookieMaxAge mirrors the session TTL in dataoperations. Kept in
// sync manually — both are 14 days.
const SessionCookieMaxAge = int(14 * 24 * time.Hour / time.Second)

// SetSessionCookie writes the HTTPOnly session cookie. Secure is derived
// from whether the request itself was TLS, so local http development works
// without setup.
func SetSessionCookie(c *gin.Context, token string) {
	secure := c.Request.TLS != nil
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   SessionCookieMaxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie expires the session cookie on the client.
func ClearSessionCookie(c *gin.Context) {
	secure := c.Request.TLS != nil
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadSessionToken returns the raw session token from the request cookie,
// or "" if absent.
func ReadSessionToken(c *gin.Context) string {
	cookie, err := c.Request.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
