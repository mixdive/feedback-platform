package models

import (
	"time"

	"github.com/google/uuid"
)

// EntryTopic is an admin-managed topic that groups entries by the
// area of the customer product they belong to (sign up, feed,
// settings, billing…). Topics are console-only metadata — they
// never appear on the Portal. The DB allows multiple topics per
// entry so the model can grow without a migration; the v0.1 UI
// only assigns one.
//
// Topics are admin-managed (add/edit/delete from Console settings)
// so each record is a real document with a UUID. There is no seed
// list — a fresh deployment starts with zero topics. Admins (or the
// AI analyzer) create them on demand.
//
// Color is a CSS color string (hex preferred). SortOrder controls
// display order in pickers and chip rows; lower comes first. Source
// records who created the topic so the Console can surface an
// "AI-created" hint in the settings list.
type EntryTopic struct {
	ID          string `bson:"_id"`
	Title       string
	Description string
	Color       string
	SortOrder   int
	Source      EntryTopicSource
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EntryTopicSource records who created a topic. Defined as a named
// string so handlers can't accidentally pass an arbitrary literal.
type EntryTopicSource string

const (
	EntryTopicSourceAdmin EntryTopicSource = "admin"
	EntryTopicSourceAI    EntryTopicSource = "ai"
)

// NewEntryTopic constructs a topic with a fresh UUID, current
// timestamps, and Source defaulted to admin (the most common
// caller). The TopicAnalyzer overrides Source to "ai" before
// inserting.
func NewEntryTopic() *EntryTopic {
	now := time.Now().UTC()
	return &EntryTopic{
		ID:        uuid.New().String(),
		Source:    EntryTopicSourceAdmin,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
