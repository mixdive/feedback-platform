package dataoperations

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// FindUserByKey returns the user whose Keys array contains the given value.
// Used by the email login flow (key = lowercased email) and by the portal
// custom-auth callback (key = userId from the inbound JWT). Returns nil if
// no user matches.
func (do *DataOperations) FindUserByKey(key string) (*models.User, error) {
	if key == "" {
		return nil, nil
	}
	return mongodb.QueryOne[models.User](do.DB, CollectionUsers, bson.M{"keys": key}, nil)
}

// FindUserByID returns the user with the given ID, or nil.
func (do *DataOperations) FindUserByID(id string) (*models.User, error) {
	return mongodb.GetOneById[models.User](do.DB, CollectionUsers, id)
}

// ListUsersByIDs returns every user whose _id is in ids. Order is not
// guaranteed; the caller is expected to index the result by ID. Empty
// input short-circuits with an empty slice.
func (do *DataOperations) ListUsersByIDs(ids []string) ([]models.User, error) {
	if len(ids) == 0 {
		return []models.User{}, nil
	}
	return mongodb.Query[models.User](do.DB, CollectionUsers, bson.M{"_id": bson.M{"$in": ids}}, nil)
}

// InsertUser persists a new user.
func (do *DataOperations) InsertUser(u *models.User) error {
	return mongodb.InsertOne(do.DB, CollectionUsers, *u)
}

// IncrementUserVotesSpent adjusts a user's running vote-quota counter
// atomically. delta is typically +1 (vote on open entry) or -1 (unvote
// from an open entry, or refund when an entry closes).
//
// Negative deltas can drive the stored value below zero — we floor it
// back to zero with a follow-up read+write so the quota check stays
// monotonic, even at the cost of an extra roundtrip. The race here is
// benign: the worst case is a single legitimate vote being rejected
// while the floor pass races with a concurrent decrement, which the
// user can retry.
func (do *DataOperations) IncrementUserVotesSpent(userID string, delta int) error {
	if userID == "" || delta == 0 {
		return nil
	}
	if err := mongodb.IncrementValue(do.DB, CollectionUsers, userID, "votesspent", delta); err != nil {
		return err
	}
	if delta >= 0 {
		return nil
	}
	u, err := do.FindUserByID(userID)
	if err != nil || u == nil {
		return err
	}
	if u.VotesSpent < 0 {
		return mongodb.SetValue(do.DB, CollectionUsers, userID, "votesspent", 0)
	}
	return nil
}

// IncrementUserFeatureRequestsOpen adjusts a user's running
// feature-request-quota counter atomically. delta is +1 (user submits a
// new feature-request) or -1 (one of their feature-requests closes, or
// a closed feature-request reopens). Mirrors IncrementUserVotesSpent
// including the floor-at-zero guard: an admin/editor never consumes
// quota at submission time, so a refund firing on their closed entry
// would otherwise drive the counter negative.
func (do *DataOperations) IncrementUserFeatureRequestsOpen(userID string, delta int) error {
	if userID == "" || delta == 0 {
		return nil
	}
	if err := mongodb.IncrementValue(do.DB, CollectionUsers, userID, "featurerequestsopen", delta); err != nil {
		return err
	}
	if delta >= 0 {
		return nil
	}
	u, err := do.FindUserByID(userID)
	if err != nil || u == nil {
		return err
	}
	if u.FeatureRequestsOpen < 0 {
		return mongodb.SetValue(do.DB, CollectionUsers, userID, "featurerequestsopen", 0)
	}
	return nil
}

// UpdateUser overwrites the persisted user document with the given value.
// Used after attaching/refreshing an account so Accounts + Keys + UpdatedAt
// land together.
func (do *DataOperations) UpdateUser(u *models.User) error {
	return mongodb.UpdateOne(do.DB, CollectionUsers, u.ID, u)
}

// ListUsersWithRoles returns every user that carries at least one role,
// sorted by creation time (oldest first so the bootstrap admin lands at
// the top). Backs the Console "Administrators" page.
func (do *DataOperations) ListUsersWithRoles() ([]models.User, error) {
	filter := bson.M{"roles.0": bson.M{"$exists": true}}
	opts := options.Find().SetSort(bson.D{{Key: "createdat", Value: 1}})
	return mongodb.Query[models.User](do.DB, CollectionUsers, filter, opts)
}

// CountActiveAdmins returns the number of users that carry the admin role,
// are not blocked, and are not soft-deleted. Used to enforce the "at least
// one administrator must remain" rule when changing roles or blocking
// someone — without this gate, an admin could lock the Console entirely.
func (do *DataOperations) CountActiveAdmins() (int64, error) {
	return mongodb.Count(do.DB, CollectionUsers, bson.M{
		"roles":     string(models.RoleAdmin),
		"isblocked": bson.M{"$ne": true},
		"isdeleted": bson.M{"$ne": true},
	}, nil)
}

// UserListFilter describes the Console "Users" list options. The page
// shows every (non-deleted) user — both administrative (any role) and
// normal (no roles, e.g. portal visitors signed in via the custom
// auth provider).
//
// Search runs a case-insensitive regex against the user's Keys array
// (every email and external ID across all attached accounts) plus the
// name/username fields on each known account. Role narrows by role
// bucket: "admin", "editor", or "none" (no role at all). Status filters
// by the IsBlocked flag. Sort accepts "new" (default), "old", or
// "email" (alphabetical by the email account address).
type UserListFilter struct {
	Search string
	Role   string // "" or "all" | "admin" | "editor" | "none"
	Status string // "" or "all" | "active" | "blocked"
	Sort   string // "" or "new" | "old" | "email"
	Page   int
	Limit  int
}

// ListUsers runs the paginated Console users query.
func (do *DataOperations) ListUsers(f UserListFilter) ([]models.User, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 25
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	sort := bson.D{}
	switch f.Sort {
	case "old":
		sort = append(sort, bson.E{Key: "createdat", Value: 1})
	case "email":
		sort = append(sort,
			bson.E{Key: "accounts.email.email", Value: 1},
			bson.E{Key: "createdat", Value: -1},
		)
	default:
		sort = append(sort, bson.E{Key: "createdat", Value: -1})
	}
	skip := int64((f.Page - 1) * f.Limit)
	opts := options.Find().SetSort(sort).SetSkip(skip).SetLimit(int64(f.Limit))
	return mongodb.Query[models.User](do.DB, CollectionUsers, userListFilter(f), opts)
}

// CountUsers returns the total number of users matching the same filter
// passed to ListUsers (ignoring Page/Limit/Sort).
func (do *DataOperations) CountUsers(f UserListFilter) (int64, error) {
	return mongodb.Count(do.DB, CollectionUsers, userListFilter(f), nil)
}

// ListAllUsers returns every user matching f's filter clauses, ignoring
// Page/Limit/Sort. Used by the Console Users page which merges in
// per-user entry counts and then sorts + paginates in Go (Mongo can't
// sort by computed counts without a heavier aggregation, and pilot
// volume is small enough that loading the full set is cheaper than the
// extra pipeline).
func (do *DataOperations) ListAllUsers(f UserListFilter) ([]models.User, error) {
	return mongodb.Query[models.User](do.DB, CollectionUsers, userListFilter(f), nil)
}

// userListFilter assembles the Mongo predicate for the Users page. Soft-
// deleted users are always excluded.
func userListFilter(f UserListFilter) bson.M {
	clauses := []bson.M{
		{"isdeleted": bson.M{"$ne": true}},
	}
	if f.Search != "" {
		clauses = append(clauses, bson.M{"$or": bson.A{
			bson.M{"keys": bson.M{"$regex": f.Search, "$options": "i"}},
			bson.M{"accounts.email.name": bson.M{"$regex": f.Search, "$options": "i"}},
			bson.M{"accounts.email.username": bson.M{"$regex": f.Search, "$options": "i"}},
			bson.M{"accounts.custom.name": bson.M{"$regex": f.Search, "$options": "i"}},
			bson.M{"accounts.custom.username": bson.M{"$regex": f.Search, "$options": "i"}},
		}})
	}
	switch f.Role {
	case "admin":
		clauses = append(clauses, bson.M{"roles": string(models.RoleAdmin)})
	case "editor":
		clauses = append(clauses, bson.M{"roles": string(models.RoleEditor)})
	case "none":
		clauses = append(clauses, bson.M{"roles.0": bson.M{"$exists": false}})
	}
	switch f.Status {
	case "active":
		clauses = append(clauses, bson.M{"isblocked": bson.M{"$ne": true}})
	case "blocked":
		clauses = append(clauses, bson.M{"isblocked": true})
	}
	if len(clauses) == 1 {
		return clauses[0]
	}
	return bson.M{"$and": clauses}
}
