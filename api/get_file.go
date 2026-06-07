package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/pkg/storage"
)

// signedURLTTL is the lifetime of GCS signed URLs. Short enough that a
// leaked URL stops working quickly; long enough that a user reading a long
// thread doesn't see expired image links.
const signedURLTTL = 15 * time.Minute

// GetFileHandler streams (or 302-redirects to) the bytes for
// /api/files/<id>.<ext>. The route param is the full storage key — UUID
// plus extension — but lookup is by UUID only; we then verify the URL's
// extension matches the stored extension to keep one canonical URL per
// file. Access is intentionally public: the UUID is the only access token.
//
//	@ID			get-file
//	@Summary	Fetch an uploaded file by id
//	@Description	Returns the file bytes. On Cloud Storage backends this 302-redirects to a short-lived signed URL. Path is `<id>.<extension>`.
//	@Tags		Files
//	@Produce	octet-stream
//	@Param		id	path	string	true	"file storage key (id.ext)"
//	@Success	200
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/files/{id} [get]
func GetFileHandler(do dataoperations.Store, store storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("id")
		id := name
		if i := strings.LastIndexByte(name, '.'); i > 0 {
			id = name[:i]
		}
		f, err := do.FindFileByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if f == nil || f.StorageKey() != name {
			response.NotFoundWithMessage(c, "File not found.")
			return
		}

		// Prefer redirect when the backend supports it (GCS).
		if url, err := store.SignedURL(c.Request.Context(), f.StorageKey(), signedURLTTL); err == nil && url != "" {
			c.Redirect(http.StatusFound, url)
			return
		}

		rc, err := store.Open(c.Request.Context(), f.StorageKey())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				response.NotFoundWithMessage(c, "File not found.")
				return
			}
			response.SystemError(c, err)
			return
		}
		defer rc.Close()

		c.Header("Content-Type", f.MimeType)
		c.Header("Content-Length", strconv.FormatInt(f.Size, 10))
		// Long cache: the URL is content-addressable by UUID; the bytes
		// behind a given id never change.
		c.Header("Cache-Control", "private, max-age=31536000, immutable")
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, rc)
	}
}
