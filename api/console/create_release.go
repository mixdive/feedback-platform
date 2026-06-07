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

// createReleaseRequest is the body for POST /api/console/release.
//
// State is optional — an unspecified state defaults to "planned" via
// NewRelease(). Date is YYYY-MM-DD; the server parses it into UTC
// midnight. An empty date is allowed (planned release whose schedule
// hasn't been set yet).
type createReleaseRequest struct {
	VersionName string `json:"versionName" binding:"required,min=1,max=60"`
	Title       string `json:"title,omitempty"        binding:"max=200"`
	Description string `json:"description,omitempty"  binding:"max=10000"`
	ReleaseDate string `json:"releaseDate,omitempty"`
	State       string `json:"state,omitempty"`
	// Optional PDF attachment. URL is the relative path returned by
	// the upload endpoint (/api/files/...); Name is the original
	// uploader filename rendered next to the download link; Size is
	// in bytes. All three travel together — pass either the full
	// trio or none.
	PdfFileURL  string `json:"pdfFileUrl,omitempty"  binding:"max=500"`
	PdfFileName string `json:"pdfFileName,omitempty" binding:"max=255"`
	PdfFileSize int64  `json:"pdfFileSize,omitempty"`
} //@name consoleCreateReleaseRequest

// CreateReleaseHandler creates a new release. Admin-only.
//
//	@ID			console-create-release
//	@Summary	Create release (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		request	body		createReleaseRequest	true	"Release"
//	@Success	201		{object}	api.ReleaseResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	409		{object}	response.ApiError
//	@Router		/api/console/release [post]
func CreateReleaseHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createReleaseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		req.VersionName = strings.TrimSpace(req.VersionName)
		if req.VersionName == "" {
			response.BadRequestWithMessage(c, "Version name is required.")
			return
		}
		existing, err := do.FindReleaseByVersionName(req.VersionName)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if existing != nil {
			response.ConflictWithMessage(c, "A release with this version name already exists.")
			return
		}
		var releaseDate time.Time
		if strings.TrimSpace(req.ReleaseDate) != "" {
			d, err := time.Parse("2006-01-02", strings.TrimSpace(req.ReleaseDate))
			if err != nil {
				response.BadRequestWithMessage(c, "Release date must be YYYY-MM-DD.")
				return
			}
			releaseDate = d.UTC()
		}
		state := models.ReleaseStatePlanned
		if req.State != "" {
			s := models.ReleaseState(req.State)
			if !models.IsValidReleaseState(s) {
				response.BadRequestWithMessage(c, "State must be 'planned' or 'completed'.")
				return
			}
			state = s
		}
		r := models.NewRelease()
		r.VersionName = req.VersionName
		r.Title = strings.TrimSpace(req.Title)
		r.Description = req.Description
		r.ReleaseDate = releaseDate
		r.State = state
		r.PdfFileURL = strings.TrimSpace(req.PdfFileURL)
		r.PdfFileName = strings.TrimSpace(req.PdfFileName)
		if r.PdfFileURL != "" && req.PdfFileSize > 0 {
			r.PdfFileSize = req.PdfFileSize
		}
		if err := do.InsertRelease(r); err != nil {
			response.SystemError(c, err)
			return
		}
		response.Created(c, api.BuildRelease(*r))
	}
}
