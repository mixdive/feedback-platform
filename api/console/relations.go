package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// EntryRelationPeer is the slim projection of a related entry surfaced
// inside an EntryRelationResponse. We deliberately keep this minimal —
// expanding the full Console entry shape here would create a circular
// definition (an entry's relations would carry entries that carry
// relations). The Console UI follows the link to fetch full details on
// demand.
type EntryRelationPeer struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	VoteCount  int    `json:"voteCount"`
	IsInternal bool   `json:"isInternal"`
} //@name EntryRelationPeer

// EntryRelationResponse is one entry-to-entry link as rendered on the
// Console entry detail Relations tab. Type is the kebab-case enum
// (`duplicate` / `related`); IsAI is true when the link was applied by
// the relation analyzer rather than by an admin.
//
// Relations are Console-only — Portal responses never carry this
// field. The type lives in the console package for that reason.
type EntryRelationResponse struct {
	Type string            `json:"type"`
	IsAI bool              `json:"isAI"`
	Peer EntryRelationPeer `json:"peer"`
} //@name EntryRelation

// LoadEntryRelationPeers batch-resolves a list of peer entry IDs into
// the slim peer projection, keyed by ID. Missing IDs (peer deleted,
// never existed) are absent from the map; the caller silently drops
// orphan relations from the wire payload.
func LoadEntryRelationPeers(do dataoperations.Store, ids []string) (map[string]EntryRelationPeer, error) {
	uniq := map[string]struct{}{}
	for _, id := range ids {
		if id == "" {
			continue
		}
		uniq[id] = struct{}{}
	}
	out := map[string]EntryRelationPeer{}
	if len(uniq) == 0 {
		return out, nil
	}
	for id := range uniq {
		e, err := do.FindEntryByID(id)
		if err != nil {
			return nil, err
		}
		if e == nil {
			continue
		}
		out[e.ID] = EntryRelationPeer{
			ID:         e.ID,
			Title:      e.Title,
			VoteCount:  e.VoteCount,
			IsInternal: e.IsInternal,
		}
	}
	return out, nil
}

// projectEntryRelations resolves entry.Relations against the loaded
// peer map and returns the wire payload, marking the AI-provenance
// subset via the entry's AIRelations array. Orphan peers (resolution
// returned nothing) are silently dropped — the Console must never
// render a half-link that points at a missing entry.
func projectEntryRelations(rels, aiRels []models.EntryRelation, peers map[string]EntryRelationPeer) []EntryRelationResponse {
	aiByPeer := make(map[string]struct{}, len(aiRels))
	for _, r := range aiRels {
		aiByPeer[r.EntryID] = struct{}{}
	}
	out := make([]EntryRelationResponse, 0, len(rels))
	for _, r := range rels {
		peer, ok := peers[r.EntryID]
		if !ok {
			continue
		}
		_, isAI := aiByPeer[r.EntryID]
		out = append(out, EntryRelationResponse{
			Type: string(r.Type),
			IsAI: isAI,
			Peer: peer,
		})
	}
	return out
}

// validateRelationType rejects unknown relation types up front so a
// handler never persists an invalid value. Empty input is rejected —
// callers must pick one of the kebab-case enum values.
func validateRelationType(t string) (models.EntryRelationType, bool) {
	rel := models.EntryRelationType(t)
	if !models.IsValidEntryRelationType(rel) {
		return "", false
	}
	return rel, true
}

// writeEntryDetailResponse loads the full Console entry payload for
// the given ID and writes it to c. Centralizes the load-and-respond
// pattern shared by the relation add/remove handlers — both mutate
// the entry then return its fresh detail view.
func writeEntryDetailResponse(c *gin.Context, do dataoperations.Store, id string) {
	updated, err := do.FindEntryByID(id)
	if err != nil {
		response.SystemError(c, err)
		return
	}
	if updated == nil {
		response.NotFoundWithMessage(c, "Entry not found.")
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

// resolveRelationsForEntry batch-resolves the entry's Relations into
// the wire shape used on the Console entry detail page. Returns an
// empty slice when the entry has no relations or when every peer has
// disappeared.
//
// One Mongo round-trip per peer in v0.1 — pilot dataset is small. A
// future optimization is a single ListByIDs call once we add it to
// dataoperations.
func resolveRelationsForEntry(do dataoperations.Store, e models.Entry) ([]EntryRelationResponse, error) {
	if len(e.Relations) == 0 {
		return []EntryRelationResponse{}, nil
	}
	ids := make([]string, 0, len(e.Relations))
	for _, r := range e.Relations {
		ids = append(ids, r.EntryID)
	}
	peers, err := LoadEntryRelationPeers(do, ids)
	if err != nil {
		return nil, err
	}
	return projectEntryRelations(e.Relations, e.AIRelations, peers), nil
}
