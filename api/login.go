package api

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/argon2"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// loginRequest carries the credentials for POST /api/login.
type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=1"`
} //@name loginRequest

// loginResponse confirms a successful login and echoes basic user info so
// the frontend can render the menu without a follow-up /me call.
type loginResponse struct {
	User userPayload `json:"user"`
} //@name LoginResult

// LoginHandler authenticates an admin via email + password and issues a
// session cookie. Errors are deliberately uniform (401 with the same
// message) regardless of which credential failed, so an attacker cannot
// enumerate accounts.
//
//	@ID			auth-login
//	@Summary	Log in
//	@Description	Verifies email + password against the stored argon2id hash and issues an HTTPOnly session cookie.
//	@Tags		Auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		loginRequest	true	"Credentials"
//	@Success	200		{object}	loginResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	401		{object}	response.ApiError
//	@Router		/api/login [post]
func LoginHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		email := strings.ToLower(strings.TrimSpace(req.Email))
		user, err := do.FindUserByKey(email)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if user == nil || user.IsBlocked || user.IsDeleted || !user.HasConsoleAccess() {
			response.UnauthorizedErrorWithMessage(c, "Invalid email or password.")
			return
		}
		emailAccount, ok := user.EmailAccount()
		if !ok || emailAccount.Email != email || emailAccount.Password == "" {
			response.UnauthorizedErrorWithMessage(c, "Invalid email or password.")
			return
		}
		match, err := VerifyPassword(req.Password, emailAccount.Password)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if !match {
			response.UnauthorizedErrorWithMessage(c, "Invalid email or password.")
			return
		}
		s, err := do.CreateSession(user.ID)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		middlewares.SetSessionCookie(c, s.Token)
		response.Success(c, loginResponse{User: NewUserPayloadWithQuota(do, user)})
	}
}

// VerifyPassword decodes the PHC-formatted argon2id hash produced by
// HashPassword and compares it with the candidate password in constant
// time.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("hash version: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("unsupported argon2 version %d", version)
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("hash params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("hash salt: %w", err)
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("hash key: %w", err)
	}

	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(got, expected) == 1, nil
}
