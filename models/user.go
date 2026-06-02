// Package models holds the canonical Go structs for every persisted entity.
// The struct is the contract.
//
// Conventions:
//   - All IDs are `string` (UUID values produced by uuid.New().String()).
//   - The only bson tag allowed is `bson:"_id"` on the ID field.
//   - Nullable fields use plain types and rely on Go zero values; we accept
//     defaults being persisted to Mongo rather than introducing pointer types.
//   - Each model has a NewX() constructor immediately after the struct.
package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserRole names a single permission bucket attached to a User. Defined as
// a named string (mirroring UserAccountType below) so handlers can't pass
// arbitrary literals where a role is expected, and so the set of valid
// values is greppable from one spot.
type UserRole string

// RoleAdmin marks a user with full Console access, including the ability to
// manage other administrators. RoleEditor grants every other Console
// capability (settings + entries) but cannot list or change the
// administrator roster — that gate stays admin-only.
const (
	RoleAdmin  UserRole = "admin"
	RoleEditor UserRole = "editor"
)

// UserAccountType identifies which external (or built-in) identity backs a
// UserAccount entry. v0.1 ships two: Email (the bootstrap admin's local
// password account) and Custom (a portal visitor authenticated via the
// admin-configured external auth URL). Google is reserved for v0.2.
type UserAccountType string

const (
	UserAccountTypeGoogle UserAccountType = "google"
	UserAccountTypeEmail  UserAccountType = "email"
	UserAccountTypeCustom UserAccountType = "custom"
)

// UserAccount is one identity attached to a User. A single User may carry
// many — e.g. an admin who later signs into the portal via the custom
// auth provider gets a Custom account added next to their existing Email
// one. The fields are a superset across providers; each provider populates
// only the columns it knows.
//
// Password is the argon2id PHC hash for Email accounts; empty for every
// other type. Mongo persists empty strings — we accept that to keep the
// no-pointer convention used across the models package.
type UserAccount struct {
	ID              string
	Name            string
	Username        string
	ImageURL        string
	Email           string
	Password        string
	IsEmailVerified bool
	CreatedAt       time.Time
}

// User carries a single identity that may be backed by multiple accounts
// (one per UserAccountType). The Email account holds the bootstrap admin's
// password; the Custom account holds the portal visitor identity sent over
// from the admin-configured external auth URL.
//
// Keys is a flat list of every searchable identifier across the user's
// accounts (email addresses, external user IDs, …). It exists so a single
// {"keys": value} Mongo query can locate a user without knowing which
// account type the value belongs to. We keep the per-account fields
// authoritative — Keys is rebuilt from them via RebuildKeys.
type User struct {
	ID        string `bson:"_id"`
	Roles     []UserRole
	Accounts  map[UserAccountType]UserAccount
	Keys      []string
	CreatedAt time.Time
	UpdatedAt time.Time
	IsDeleted bool
	IsBlocked bool
	// VotesSpent is the running count of this user's votes that
	// currently sit on OPEN entries (status new/evaluation/in-progress).
	// Compared against Settings.Feedback.MaxVotesPerUser at vote time.
	// Stored as a plain int so the Go zero value reads as "no quota
	// consumed" — existing users on a pre-feature deployment start
	// with their full quota available without any backfill.
	VotesSpent int
	// FeatureRequestsOpen is the running count of this user's
	// authored feature-request entries currently in an OPEN status
	// (new/evaluation/in-progress). Compared against
	// Settings.Feedback.MaxFeatureRequestsPerUser at submission time.
	// Same zero-value-means-empty discipline as VotesSpent so the
	// rollout needs no backfill — existing users start with their
	// full quota available.
	FeatureRequestsOpen int
}

// NewUser constructs a User with a fresh ID, empty roles/accounts/keys, and
// current timestamps. The caller fills in accounts via SetAccount.
func NewUser() *User {
	now := time.Now().UTC()
	return &User{
		ID:        uuid.New().String(),
		Roles:     []UserRole{},
		Accounts:  map[UserAccountType]UserAccount{},
		Keys:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsAdmin reports whether the user has the admin role. Stays strict — the
// administrator-management endpoints check this and only this.
func (u *User) IsAdmin() bool {
	for _, r := range u.Roles {
		if r == RoleAdmin {
			return true
		}
	}
	return false
}

// HasConsoleAccess reports whether the user is allowed into the Console at
// all. Both admins and editors qualify; everything except the
// administrator-management endpoints uses this gate.
func (u *User) HasConsoleAccess() bool {
	for _, r := range u.Roles {
		if r == RoleAdmin || r == RoleEditor {
			return true
		}
	}
	return false
}

// SetAccount attaches (or replaces) an account on the user and rebuilds the
// Keys index off of it. UpdatedAt is bumped.
func (u *User) SetAccount(t UserAccountType, a UserAccount) {
	if u.Accounts == nil {
		u.Accounts = map[UserAccountType]UserAccount{}
	}
	u.Accounts[t] = a
	u.RebuildKeys()
	u.UpdatedAt = time.Now().UTC()
}

// EmailAccount returns the Email account, or zero-value + false if absent.
func (u *User) EmailAccount() (UserAccount, bool) {
	a, ok := u.Accounts[UserAccountTypeEmail]
	return a, ok
}

// CustomAccount returns the Custom account, or zero-value + false if absent.
func (u *User) CustomAccount() (UserAccount, bool) {
	a, ok := u.Accounts[UserAccountTypeCustom]
	return a, ok
}

// GoogleAccount returns the Google account, or zero-value + false if absent.
func (u *User) GoogleAccount() (UserAccount, bool) {
	a, ok := u.Accounts[UserAccountTypeGoogle]
	return a, ok
}

// RebuildKeys regenerates the Keys slice from every account currently on
// the user. Stable order so equal user states produce equal Keys.
func (u *User) RebuildKeys() {
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(u.Accounts)*2)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, dup := seen[v]; dup {
			return
		}
		seen[v] = struct{}{}
		keys = append(keys, v)
	}
	// Iterate in a fixed order so Keys is deterministic.
	for _, t := range []UserAccountType{UserAccountTypeEmail, UserAccountTypeCustom, UserAccountTypeGoogle} {
		a, ok := u.Accounts[t]
		if !ok {
			continue
		}
		add(a.ID)
		add(a.Email)
	}
	u.Keys = keys
}
