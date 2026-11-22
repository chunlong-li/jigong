package api

import (
	_ "embed"
	"net/http"
)

//go:embed changelog.md
var changelogMD string

// getChangelog 返回随二进制一起发布的更新日志（Markdown 原文）。
func (s *Server) getChangelog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{
		"version": Version,
		"content": changelogMD,
	})
}
