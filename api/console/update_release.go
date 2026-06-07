package console

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// updateReleaseRequest is the body for PATCH
// /api/console/release/{id}.
//
// Sparse: only fields present on the wire are applied. ReleaseDate is
// a string at the boundary so the same shape as create is preserved;
// the empty string clears the date back to the zero value.
type updateReleaseRequest struct {
	VersionName *string `json:"versionName,omitempty" binding:"omitempty,min=1,max=60"`
	Title       *string `json:"title,omitempty"       binding:"omitempty,max=200"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=10000"`
	ReleaseDate *string `json:"releaseDate,omitempty"`
	State       *string `json:"state,omitempty"`
	// PdfFile carries the attachment slot for the release. The
	// triple is applied together: an empty URL clears the slot
	// (and is the only way to remove an existing attachment). nil
	// pointer leaves the existing PDF untouched.
	PdfFile *updateReleasePdf `json:"pdfFile,omitempty"`
} //@name consoleUpdateReleaseRequest

type updateReleasePdf struct {
	URL  string `json:"url"  binding:"max=500"`
	Name string `json:"name" binding:"max=255"`
	Size int64  `json:"size"`
} //@name consoleUpdateReleasePdf

// UpdateReleaseHandler patches a single release. Admin-only.
//
//	@ID			console-update-release
//	@Summary	Update release (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"release id"
//	@Param		request	body		updateReleaseRequest	true	"Patch"
//	@Success	200		{object}	api.ReleaseResponse
//	@Failure	404		{object}	response.ApiError
//	@Failure	409		{object}	response.ApiError
//	@Router		/api/console/release/{id} [patch]
func UpdateReleaseHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		existing, err := findReleaseOrNotFound(do, id)
		if err != nil {
			response.BadRequestWithMessage(c, err.Error())
			return
		}
		if existing == nil {
			response.NotFoundWithMessage(c, "Release not found.")
			return
		}
		var req updateReleaseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		var trimmedVersionName *string
		if req.VersionName != nil {
			v := strings.TrimSpace(*req.VersionName)
			if v == "" {
				response.BadRequestWithMessage(c, "Version name cannot be empty.")
				return
			}
			if v != existing.VersionName {
				dup, err := do.FindReleaseByVersionName(v)
				if err != nil {
					response.SystemError(c, err)
					return
				}
				if dup != nil && dup.ID != id {
					response.ConflictWithMessage(c, "A release with this version name already exists.")
					return
				}
			}
			trimmedVersionName = &v
		}
		var trimmedTitle *string
		if req.Title != nil {
			v := strings.TrimSpace(*req.Title)
			trimmedTitle = &v
		}
		var releaseDate *time.Time
		if req.ReleaseDate != nil {
			s := strings.TrimSpace(*req.ReleaseDate)
			if s == "" {
				z := time.Time{}
				releaseDate = &z
			} else {
				d, err := time.Parse("2006-01-02", s)
				if err != nil {
					response.BadRequestWithMessage(c, "Release date must be YYYY-MM-DD.")
					return
				}
				ud := d.UTC()
				releaseDate = &ud
			}
		}
		var state *models.ReleaseState
		if req.State != nil {
			s := models.ReleaseState(*req.State)
			if !models.IsValidReleaseState(s) {
				response.BadRequestWithMessage(c, "State must be 'planned' or 'completed'.")
				return
			}
			state = &s
		}
		var pdf *dataoperations.ReleasePdfPatch
		if req.PdfFile != nil {
			p := dataoperations.ReleasePdfPatch{
				URL:  strings.TrimSpace(req.PdfFile.URL),
				Name: strings.TrimSpace(req.PdfFile.Name),
			}
			if p.URL != "" && req.PdfFile.Size > 0 {
				p.Size = req.PdfFile.Size
			}
			pdf = &p
		}
		if err := do.UpdateReleaseFields(id, trimmedVersionName, trimmedTitle, req.Description, releaseDate, state, pdf); err != nil {
			response.SystemError(c, err)
			return
		}
		updated, err := do.FindReleaseByID(id)
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, api.BuildRelease(*updated))
	}
}
