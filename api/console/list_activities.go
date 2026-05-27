package console

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// activityResponse is the wire shape for a single audit-log row on the
// Console timeline. Source identifies the originator (user/admin/ai)
// so the Console can render an "AI" badge without a synthetic user.
// Actor is the lightweight User projection — present only when the
// row has a known acting user (admin/editor row, or portal author for
// entry-created); nil for AI rows and anonymous portal submissions.
//
// FromValue / ToValue carry the before/after snapshot for transitions
// (status changes, entry-type changes). TargetID identifies the
// related entity for scoped events:
//   - topic-added / topic-removed → topic ID
//   - release-set                 → release ID  ("" on release-cleared)
//   - relation-added / removed    → peer entry ID
//   - merged-into                 → target entry ID
type activityResponse struct {
	ID        string              `json:"id"`
	EntryID   string              `json:"entryId"`
	Type      string              `json:"type"`
	Source    string              `json:"source"`
	Actor     *api.EntryCreator   `json:"actor,omitempty"`
	FromValue string              `json:"fromValue,omitempty"`
	ToValue   string              `json:"toValue,omitempty"`
	TargetID  string              `json:"targetId,omitempty"`
	CreatedAt string              `json:"createdAt"`
} //@name Activity

// activityListResponse is the envelope returned by the list endpoint.
// Like comments, pagination is intentionally absent — we ship every
// activity on the entry sorted oldest-first.
type activityListResponse struct {
	Data []activityResponse `json:"data"`
} //@name ActivityList

// ListActivitiesHandler returns every activity row on an entry,
// oldest first. The Console detail page merges this stream with the
// entry's comments client-side by CreatedAt to produce the unified
// timeline shown under the description.
//
//	@ID			console-list-activities
//	@Summary	List activities on an entry (admin/editor)
//	@Tags		Console
//	@Produce	json
//	@Param		id	path		string	true	"entry id"
//	@Success	200	{object}	activityListResponse
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/console/entry/{id}/activity [get]
func ListActivitiesHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entry, err := do.FindEntryByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if entry == nil {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		activities, err := do.ListActivitiesByEntryID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		actorIDs := make([]string, 0, len(activities))
		for _, a := range activities {
			actorIDs = append(actorIDs, a.ActorID)
		}
		actors, err := api.LoadEntryCreators(do, actorIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]activityResponse, 0, len(activities))
		for _, a := range activities {
			out = append(out, buildActivityResponse(a, actors))
		}
		response.Success(c, activityListResponse{Data: out})
	}
}

// buildActivityResponse projects a stored Activity into the wire
// shape, looking up the acting user from actors when ActorID is set.
// AI rows never have an actor populated, by design.
func buildActivityResponse(a models.Activity, actors map[string]api.EntryCreator) activityResponse {
	out := activityResponse{
		ID:        a.ID,
		EntryID:   a.EntryID,
		Type:      string(a.Type),
		Source:    string(a.Source),
		FromValue: a.FromValue,
		ToValue:   a.ToValue,
		TargetID:  a.TargetID,
		CreatedAt: activityISO(a.CreatedAt),
	}
	if a.ActorID != "" {
		if u, ok := actors[a.ActorID]; ok {
			out.Actor = &u
		}
	}
	return out
}

// activityISO mirrors commentISO — UTC ISO-8601 string, empty when zero.
func activityISO(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
