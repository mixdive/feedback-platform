package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/dataoperations"
)

// Mount wires the embedded SPAs and the first-run setup page onto the
// router. It replaces the handful of Gin routes the main package would
// otherwise have to manage:
//
//	/console        → 301 → /console/
//	/console/*      → console SPA (with SPA fallback to console/dist/index.html)
//	/  (no setup)   → inline setup form (web/setup/index.html)
//	/  (post-setup) → portal SPA
//	/* (anything)   → portal SPA fallback
//
// Already-registered routes (/api, /swagger, /health) are not overridden.
// Unknown /api or /swagger paths return a JSON 404 instead of the SPA HTML
// so API clients aren't confused.
func Mount(r *gin.Engine, do *dataoperations.DataOperations) error {
	consoleFS, err := fs.Sub(spaFS, "console/dist")
	if err != nil {
		return err
	}
	portalFS, err := fs.Sub(spaFS, "portal/dist")
	if err != nil {
		return err
	}

	consoleHandler, err := newSPAHandler(consoleFS)
	if err != nil {
		return err
	}
	portalHandler, err := newSPAHandler(portalFS)
	if err != nil {
		return err
	}

	r.GET("/console", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/console/")
	})
	r.GET("/console/*filepath", gin.WrapH(http.StripPrefix("/console", consoleHandler)))

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/swagger/") {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Route not found.",
				"code":    http.StatusNotFound,
			})
			return
		}

		// Root path before setup is complete: serve the inline setup form.
		// Any other path falls through to the portal SPA so static assets
		// (favicon, locales) keep working even pre-setup.
		if (p == "/" || p == "") && !setupCompleted(do) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Status(http.StatusOK)
			_, _ = c.Writer.Write(setupHTML)
			return
		}

		portalHandler.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}

// setupCompleted is a fail-open check: any error reading settings (e.g.,
// Mongo briefly unavailable) is treated as "not completed" so the user is
// directed to the setup form rather than a blank portal.
func setupCompleted(do *dataoperations.DataOperations) bool {
	s, err := do.GetSettings()
	if err != nil || s == nil {
		return false
	}
	return s.SetupCompleted
}

// newSPAHandler returns an http.Handler that serves the given embedded fs.
// Existing files are served via http.FileServer; missing paths fall back to
// /index.html so client-side routing works on hard refresh / direct links.
func newSPAHandler(efs fs.FS) (http.Handler, error) {
	indexBytes, err := fs.ReadFile(efs, "index.html")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(efs))

	serveIndex := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexBytes)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean == "" || clean == "." {
			serveIndex(w)
			return
		}
		f, err := efs.Open(clean)
		if err != nil {
			serveIndex(w)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	}), nil
}
