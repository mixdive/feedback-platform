package portal

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// submitEntryRequest is the body for POST /api/portal/entry.
//
// Portal-created entries are always public; internal visibility can
// only be set later from the Console.
type submitEntryRequest struct {
	Title       string `json:"title"        binding:"required,min=3,max=200" example:"Add dark mode"`
	Description string `json:"description,omitempty"                         example:"Would love a system-aware dark mode toggle."`
	EntryType   string `json:"entryType,omitempty"                           example:"feature-request"`
} //@name portalSubmitEntryRequest

// SubmitEntryHandler creates a new entry. When the visitor is signed
// in (custom auth) we stamp UserID so the Portal can credit the author;
// anonymous submissions leave it empty and render without a creator.
//
//	@ID			portal-submit-entry
//	@Summary	Submit entry
//	@Tags		Portal
//	@Accept		json
//	@Produce	json
//	@Param		request	body		submitEntryRequest	true	"Entry"
//	@Success	201		{object}	entryResponse
//	@Failure	400		{object}	response.ApiError
//	@Router		/api/portal/entry [post]
func SubmitEntryHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req submitEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		ft, err := resolveEntryType(req.EntryType)
		if err != nil {
			response.BadRequestWithMessage(c, err.Error())
			return
		}
		// Support entries are no longer created from the Portal —
		// the "New Support Request" button is a link out to an
		// admin-configured URL instead. The type stays valid on the
		// model (Console still lists existing support records) but
		// the submission path refuses it for every Portal caller,
		// admin and end-user alike.
		if ft == models.EntryTypeSupport {
			response.BadRequestWithMessage(c, "Support requests cannot be submitted from the portal.")
			return
		}

		e := models.NewEntry()
		e.Title = req.Title
		e.Description = req.Description
		e.EntryType = ft
		e.Source = models.EntrySourcePortal
		e.IsInternal = false
		// AI entry-type analysis runs only on entries that were
		// created without a type. If the submitter picked one, stamp
		// the analysis as disabled so the worker never claims it.
		if ft != "" {
			e.EntryTypeAnalysis.Status = models.AnalysisStatusDisabled
		}
		creators := map[string]api.EntryCreator{}
		if u := middlewares.CurrentUser(c); u != nil {
			e.UserID = u.ID
			creators[u.ID] = api.BuildEntryCreator(*u)
		}

		if err := do.InsertEntry(e); err != nil {
			response.SystemError(c, err)
			return
		}
		// Audit-log: the entry-created activity row anchors the
		// Console timeline. Source=user is the portal-author origin
		// even when the visitor is anonymous (ActorID then stays "").
		act := models.NewActivity()
		act.EntryID = e.ID
		act.Type = models.ActivityTypeEntryCreated
		act.Source = models.ActivitySourceUser
		act.ActorID = e.UserID
		act.CreatedAt = e.CreatedAt
		if err := do.InsertActivity(act); err != nil {
			response.SystemError(c, err)
			return
		}
		// Best-effort Slack notification for the new public entry.
		// Internal entries (admin-authored notes) never leave the
		// Console, so they never reach Slack either.
		if !e.IsInternal {
			msg := fmt.Sprintf(":sparkles: *New feedback* — %s\n_%s · by %s_",
				slackEntryRef(c, e.ID, e.Title),
				slackEntryTypeLabel(e.EntryType),
				slackAuthorName(middlewares.CurrentUser(c)),
			)
			notifySlack(do, func(s models.SlackIntegration) bool { return s.NotifyOnEntry }, msg)
		}
		// AI analysis is poller-driven (pkg/aianalyzer.Worker) — an
		// empty EntryTypeAnalysis.Status is picked up on the next
		// tick. New records can't have a vote yet — passing false
		// avoids the lookup.
		releases := map[string]api.ReleaseResponse{}
		response.Created(c, entryToResponse(*e, creators, releases, false))
	}
}
