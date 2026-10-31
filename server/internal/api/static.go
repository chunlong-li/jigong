package api

import (
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// serveStatic 提供前端静态资源；非文件路径回退到 index.html（SPA 路由）。
func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clean := path.Clean("/" + r.URL.Path)
	file := filepath.Join(s.webDir, filepath.FromSlash(clean))
	if !strings.HasPrefix(file, filepath.Clean(s.webDir)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	info, err := os.Stat(file)
	if err != nil || info.IsDir() {
		// 前端路由回退
		index := filepath.Join(s.webDir, "index.html")
		if _, err := os.Stat(index); err != nil {
			http.Error(w, "前端资源未找到，请先构建 web 或检查 -web-dir", http.StatusNotFound)
			return
		}
		s.serveFile(w, r, index, false)
		return
	}
	s.serveFile(w, r, file, strings.HasPrefix(clean, "/assets/"))
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, file string, immutable bool) {
	if ct := mime.TypeByExtension(filepath.Ext(file)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if immutable {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeFile(w, r, file)
}
