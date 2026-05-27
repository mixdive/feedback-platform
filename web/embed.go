// Package web embeds the two pre-built React applications (Console and
// Portal) plus the inline first-run setup page into the Go binary via
// go:embed, and exposes Mount() to wire them onto the Gin router.
//
// The dist/ folders for each SPA must exist at compile time. They are
// produced by `bun run build` in web/console/ and web/portal/ — `make build`
// runs both before invoking `go build`.
package web

import "embed"

//go:embed all:console/dist all:portal/dist
var spaFS embed.FS

//go:embed setup/index.html
var setupHTML []byte
