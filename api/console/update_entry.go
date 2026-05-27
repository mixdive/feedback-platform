package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// updateEntryRequest is the body for PATCH /api/console/entry/{id}.
//
// Sparse update: only fields present on the wire are applied. Entry content
// (title, description) is owned by the Portal author and not editable from
// the Console — admins triage metadata only.
type updateEntryRequest struct {
	IsInternal *bool     `json:"isInternal,omitempty"`
	EntryType  *string   `json:"entryType,omitempty"`
	Status     *string   `json:"status,omitempty"`
	TopicIDs   *[]string `json:"topicIds,omitempty"`
	ReleaseID  *string   `json:"releaseId,omitempty"`
} //@name consoleUpdateEntryRequest

// UpdateEntryHandler patches a single entry from the Console.
//
//	@ID			console-update-entry
//	@Summary	Update entry (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"entry id"
//	@Param		request	body		updateEntryRequest	true	"Patch"
//	@Success	200		{object}	entryResponse
//	@Failure	404		{object}	response.ApiError
//	@Router		/api/console/entry/{id} [patch]
func UpdateEntryHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req updateEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		existing, err := do.FindEntryByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if existing == nil {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		actorID := ""
		if u := middlewares.CurrentUser(c); u != nil {
			actorID = u.ID
		}
		if req.IsInternal != nil && *req.IsInternal != existing.IsInternal {
			if err := do.SetEntryIsInternal(id, *req.IsInternal); err != nil {
				response.SystemError(c, err)
				return
			}
			t := models.ActivityTypeInternalEnabled
			if !*req.IsInternal {
				t = models.ActivityTypeInternalDisabled
			}
			if err := recordAdminActivity(do, id, actorID, t, "", "", ""); err != nil {
				response.SystemError(c, err)
				return
			}
		}
		if req.EntryType != nil {
			ft, err := resolveEntryType(*req.EntryType)
			if err != nil {
				response.BadRequestWithMessage(c, err.Error())
				return
			}
			if ft != existing.EntryType {
				if err := do.SetEntryType(id, ft); err != nil {
					response.SystemError(c, err)
					return
				}
				if err := recordAdminActivity(do, id, actorID, models.ActivityTypeEntryTypeChanged, string(existing.EntryType), string(ft), ""); err != nil {
					response.SystemError(c, err)
					return
				}
			}
		}
		if req.Status != nil {
			status, err := resolveEntryStatus(*req.Status)
			if err != nil {
				response.BadRequestWithMessage(c, err.Error())
				return
			}
			if status != existing.Status {
				if err := do.SetEntryStatus(id, status); err != nil {
					response.SystemError(c, err)
					return
				}
				if err := recordAdminActivity(do, id, actorID, models.ActivityTypeStatusChanged, string(existing.Status), string(status), ""); err != nil {
					response.SystemError(c, err)
					return
				}
			}
		}
		if req.TopicIDs != nil {
			topicIDs, err := resolveTopicIDs(do, *req.TopicIDs)
			if err != nil {
				response.BadRequestWithMessage(c, err.Error())
				return
			}
			added, removed := diffTopicIDs(existing.TopicIDs, topicIDs)
			if len(added) > 0 || len(removed) > 0 {
				if err := do.SetEntryTopics(id, topicIDs); err != nil {
					response.SystemError(c, err)
					return
				}
				// One activity per added/removed topic so the timeline
				// can render individual chips and tooltip topic names.
				for _, t := range added {
					if err := recordAdminActivity(do, id, actorID, models.ActivityTypeTopicAdded, "", "", t); err != nil {
						response.SystemError(c, err)
						return
					}
				}
				for _, t := range removed {
					if err := recordAdminActivity(do, id, actorID, models.ActivityTypeTopicRemoved, "", "", t); err != nil {
						response.SystemError(c, err)
						return
					}
				}
			}
		}
		if req.ReleaseID != nil {
			releaseID, err := resolveReleaseID(do, *req.ReleaseID)
			if err != nil {
				response.BadRequestWithMessage(c, err.Error())
				return
			}
			if releaseID != existing.ReleaseID {
				if err := do.SetEntryRelease(id, releaseID); err != nil {
					response.SystemError(c, err)
					return
				}
				t := models.ActivityTypeReleaseSet
				if releaseID == "" {
					t = models.ActivityTypeReleaseCleared
				}
				if err := recordAdminActivity(do, id, actorID, t, existing.ReleaseID, releaseID, releaseID); err != nil {
					response.SystemError(c, err)
					return
				}
			}
		}
		updated, err := do.FindEntryByID(id)
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		creators, err := api.LoadEntryCreators(do, []string{updated.UserID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		topics, err := LoadEntryTopics(do, updated.TopicIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		releases, err := api.LoadReleases(do, []string{updated.ReleaseID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		isVoted := false
		if u := middlewares.CurrentUser(c); u != nil {
			v, err := do.FindVote(u.ID, id)
			if err != nil {
				response.SystemError(c, err)
				return
			}
			isVoted = v != nil
		}
		relations, err := resolveRelationsForEntry(do, *updated)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, entryToResponse(*updated, creators, topics, releases, isVoted, relations))
	}
}

// recordAdminActivity is the per-handler shortcut for admin/editor
// metadata mutations. Caller passes Type + payload columns; we stamp
// Source=admin and the calling user's ID.
func recordAdminActivity(do *dataoperations.DataOperations, entryID, actorID string, t models.ActivityType, from, to, target string) error {
	a := models.NewActivity()
	a.EntryID = entryID
	a.Type = t
	a.Source = models.ActivitySourceAdmin
	a.ActorID = actorID
	a.FromValue = from
	a.ToValue = to
	a.TargetID = target
	return do.InsertActivity(a)
}

// diffTopicIDs returns the added and removed sets between two topic
// ID lists. Order is preserved from the input slices so the activity
// log reads in the order the admin chose them.
func diffTopicIDs(old, next []string) (added, removed []string) {
	oldSet := make(map[string]struct{}, len(old))
	for _, id := range old {
		oldSet[id] = struct{}{}
	}
	nextSet := make(map[string]struct{}, len(next))
	for _, id := range next {
		nextSet[id] = struct{}{}
	}
	for _, id := range next {
		if _, ok := oldSet[id]; !ok {
			added = append(added, id)
		}
	}
	for _, id := range old {
		if _, ok := nextSet[id]; !ok {
			removed = append(removed, id)
		}
	}
	return added, removed
}
