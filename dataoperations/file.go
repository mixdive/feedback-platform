package dataoperations

import (
	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// InsertFile persists a new File metadata record.
func (do *DataOperations) InsertFile(f *models.File) error {
	return mongodb.InsertOne(do.DB, CollectionFiles, *f)
}

// FindFileByID returns the file's metadata or (nil, nil) if it doesn't exist.
// Mirrors the rest of the data layer's "missing → nil, nil" convention.
func (do *DataOperations) FindFileByID(id string) (*models.File, error) {
	if id == "" {
		return nil, nil
	}
	return mongodb.GetOneById[models.File](do.DB, CollectionFiles, id)
}
