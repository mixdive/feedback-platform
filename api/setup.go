package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/argon2"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// setupStatusResponse tells the frontend whether the first-run wizard is
// still required.
type setupStatusResponse struct {
	Completed bool `json:"completed"`
} //@name SetupStatus

// SetupStatusHandler reports whether the deployment has been initialized.
// Public endpoint with no side effects.
//
//	@ID			setup-status
//	@Summary	Setup status
//	@Description	Returns whether the first-run setup wizard has been completed.
//	@Tags		Setup
//	@Produce	json
//	@Success	200	{object}	setupStatusResponse
//	@Router		/api/setup/status [get]
func SetupStatusHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := do.GetSettings()
		completed := false
		if err == nil && s != nil {
			completed = s.SetupCompleted
		}
		response.Success(c, setupStatusResponse{Completed: completed})
	}
}

// setupRequest is the body for POST /api/setup. Carries the values the
// admin fills in the first-run form: project name and the bootstrap
// admin's email + password.
type setupRequest struct {
	ProjectName   string `json:"projectName"      binding:"required,min=1,max=120"`
	AdminEmail    string `json:"adminEmail"       binding:"required,email"`
	AdminPassword string `json:"adminPassword"    binding:"required,min=10"`
} //@name setupRequest

// setupResponse is the success payload for POST /api/setup. v0.1 has no
// session/login flow yet, so we just confirm completion and let the
// frontend redirect to /console/.
type setupResponse struct {
	Completed bool `json:"completed"`
} //@name SetupResult

// SetupHandler runs the first-run wizard. Behavior:
//
//   - Refuses with 409 if setup has already been completed.
//   - Persists the singleton Settings document with the provided values.
//   - Creates the bootstrap admin user with an argon2id-hashed password.
//
// Login + session issuance are deferred — the form just gets a 201 and
// redirects on its own. Without auth, every endpoint behind RequireSetup
// is reachable to anyone, which is acceptable for the v0.1 scope.
//
//	@ID			setup-complete
//	@Summary	Complete first-run setup
//	@Description	One-shot endpoint that initializes the deployment. After it succeeds, GET /api/setup/status returns completed=true and this endpoint refuses.
//	@Tags		Setup
//	@Accept		json
//	@Produce	json
//	@Param		request	body		setupRequest	true	"Setup data"
//	@Success	201		{object}	setupResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	409		{object}	response.ApiError
//	@Router		/api/setup [post]
func SetupHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		existing, err := do.GetSettings()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if existing != nil && existing.SetupCompleted {
			response.ConflictWithMessage(c, "Setup has already been completed.")
			return
		}
		var req setupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		passwordHash, err := HashPassword(req.AdminPassword)
		if err != nil {
			response.SystemError(c, err)
			return
		}

		portal, err := models.NewPortalSettings()
		if err != nil {
			response.SystemError(c, err)
			return
		}

		s := models.NewSettings()
		s.ProjectName = req.ProjectName
		s.Portal = portal
		s.Feedback = models.FeedbackSettings{
			MaxVotesPerUser:           models.DefaultMaxVotesPerUser,
			MaxFeatureRequestsPerUser: models.DefaultMaxFeatureRequestsPerUser,
			EntryTypeTemplatesByLang:  models.DefaultEntryTypeTemplates(),
		}
		s.SetupCompleted = true
		if existing == nil {
			if err := do.InsertSettings(s); err != nil {
				response.SystemError(c, err)
				return
			}
		} else {
			if err := do.UpdateSettings(map[string]any{
				"projectname":    s.ProjectName,
				"portal":         s.Portal,
				"feedback":       s.Feedback,
				"setupcompleted": true,
			}); err != nil {
				response.SystemError(c, err)
				return
			}
		}

		now := time.Now().UTC()
		email := strings.ToLower(strings.TrimSpace(req.AdminEmail))
		u := models.NewUser()
		u.Roles = []models.UserRole{models.RoleAdmin}
		u.SetAccount(models.UserAccountTypeEmail, models.UserAccount{
			ID:              email,
			Email:           email,
			Password:        passwordHash,
			IsEmailVerified: true,
			CreatedAt:       now,
		})
		if err := do.InsertUser(u); err != nil {
			response.SystemError(c, err)
			return
		}

		response.Created(c, setupResponse{Completed: true})
	}
}

// argon2id parameters tuned for v0.1. Per OWASP 2024 guidance: m=64MiB, t=1, p=4.
const (
	argonMemory      uint32 = 64 * 1024
	argonIterations  uint32 = 1
	argonParallelism uint8  = 4
	argonSaltLength  uint32 = 16
	argonKeyLength   uint32 = 32
)

// HashPassword returns a PHC-formatted argon2id hash. Used by the setup
// handler to seed the bootstrap admin's password and by the profile
// password-change handler when an admin rotates their own credential.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}
