package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/storage"
)

// MaxUploadBytes caps any single upload at 25 MiB. The cap covers
// screenshots, short screen recordings, and PDFs comfortably and stays small
// enough to keep upload memory usage bounded — the handler buffers the full
// payload to compute a checksum and sniff the mime type before writing
// through to storage. Bumpable later via settings if it bites.
const MaxUploadBytes = 25 * 1024 * 1024

// mimeExtensions is both the upload allowlist and the canonical
// mime → extension table. We sniff the bytes server-side and only accept
// types listed here; the matching extension is what we store on the File
// record (and therefore in the storage key + URL).
//
// SVG is excluded on purpose — SVGs can carry script and we serve files at
// /api/files/:id without sandboxing.
var mimeExtensions = map[string]string{
	// Images.
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/gif":  "gif",
	"image/webp": "webp",
	// Video.
	"video/mp4":       "mp4",
	"video/webm":      "webm",
	"video/quicktime": "mov",
	// Documents.
	"application/pdf": "pdf",
	"application/msword": "doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
	"application/vnd.ms-excel": "xls",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": "xlsx",
	"application/vnd.ms-powerpoint": "ppt",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
	"application/zip": "zip",
	// Plain text family.
	"text/plain":    "txt",
	"text/csv":      "csv",
	"text/markdown": "md",
}

// FileResponse is the wire shape returned by an upload and the metadata
// fetch. URL is the relative path the client should use as the markdown
// link target — `![alt](url)` for images, `[name](url)` for everything
// else.
type FileResponse struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	Size         int64  `json:"size"`
	IsImage      bool   `json:"isImage"`
} //@name File

// BuildFileResponse renders a File for the wire. The URL embeds the
// extension so browsers / link unfurlers see a recognizable suffix.
func BuildFileResponse(f models.File) FileResponse {
	return FileResponse{
		ID:           f.ID,
		URL:          fmt.Sprintf("/api/files/%s", f.StorageKey()),
		OriginalName: f.OriginalName,
		MimeType:     f.MimeType,
		Size:         f.Size,
		IsImage:      f.IsImage(),
	}
}

// UploadFileHandler is a cross-surface multipart upload endpoint. The same
// handler is registered under both /api/portal/files and /api/console/files
// — the route-level middleware stack decides who gets to call it. The
// uploader is recorded as the authenticated user when a session is attached
// and as the empty string for anonymous portal uploads (matching anonymous
// entry submission).
//
//	@ID			upload-file
//	@Summary	Upload a file (image, video, or document)
//	@Description	Multipart form upload. Field name "file". Returns metadata + a relative URL to embed in markdown — `![alt](url)` for images, `[name](url)` for other attachments.
//	@Tags		Files
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		file	formData	file	true	"File to upload"
//	@Success	201		{object}	api.FileResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	413		{object}	response.ApiError
//	@Failure	415		{object}	response.ApiError
//	@Router		/api/files [post]
func UploadFileHandler(do dataoperations.Store, store storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			response.BadRequestWithMessage(c, "No file provided. Use multipart/form-data with field name \"file\".")
			return
		}
		if fh.Size <= 0 {
			response.BadRequestWithMessage(c, "Empty file.")
			return
		}
		if fh.Size > MaxUploadBytes {
			response.ErrorWithStatusCodeAndMessage(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("File too large. Max upload size is %d bytes.", MaxUploadBytes))
			return
		}

		src, err := fh.Open()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		defer src.Close()

		// Buffer the upload so we can sniff + hash before writing through.
		// LimitReader guards against a lying Content-Length / Size header.
		buf, err := io.ReadAll(io.LimitReader(src, MaxUploadBytes+1))
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if int64(len(buf)) > MaxUploadBytes {
			response.ErrorWithStatusCodeAndMessage(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("File too large. Max upload size is %d bytes.", MaxUploadBytes))
			return
		}

		mimeType := http.DetectContentType(buf)
		// DetectContentType returns text/plain; charset=utf-8 for plaintext —
		// strip the charset so the allowlist key matches.
		if i := strings.IndexByte(mimeType, ';'); i >= 0 {
			mimeType = strings.TrimSpace(mimeType[:i])
		}
		ext, ok := mimeExtensions[mimeType]
		if !ok {
			response.ErrorWithStatusCodeAndMessage(c, http.StatusUnsupportedMediaType, fmt.Sprintf("Unsupported file type: %s.", mimeType))
			return
		}

		sum := sha256.Sum256(buf)
		f := models.NewFile()
		f.OriginalName = fh.Filename
		f.Extension = ext
		f.MimeType = mimeType
		f.Size = int64(len(buf))
		f.Checksum = hex.EncodeToString(sum[:])
		if u := middlewares.CurrentUser(c); u != nil {
			f.UserID = u.ID
		}

		if err := store.Put(c.Request.Context(), f.StorageKey(), f.MimeType, f.Size, bytes.NewReader(buf)); err != nil {
			response.SystemError(c, err)
			return
		}
		if err := do.InsertFile(f); err != nil {
			// Try to roll back the blob so the metadata and storage stay in
			// sync. Best-effort — if delete also fails, we leave an orphan
			// for a future cleanup pass.
			_ = store.Delete(c.Request.Context(), f.StorageKey())
			response.SystemError(c, err)
			return
		}

		response.Created(c, BuildFileResponse(*f))
	}
}
