package api

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "jigong_session"

// Server 持有数据库连接和会话表。
type Server struct {
	db       *sql.DB
	webDir   string
	mu       sync.Mutex
	sessions map[string]time.Time
}

func NewServer(db *sql.DB, webDir string) *Server {
	return &Server{
		db:       db,
		webDir:   webDir,
		sessions: make(map[string]time.Time),
	}
}

// EnsureAdmin 首次启动时创建管理员账号。
func (s *Server) EnsureAdmin(user, pass string) error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO users (username, password_hash, created_at) VALUES (?,?,?)`,
		user, string(hash), now())
	if err == nil {
		log.Printf("[init] 已创建管理员账号: %s，请登录后尽快修改密码", user)
	}
	return err
}

func now() string { return time.Now().Format("2006-01-02 15:04:05") }

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("POST /api/logout", s.handleLogout)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/init-status", s.handleInitStatus)

	mux.HandleFunc("GET /api/me", s.handleMe)
	mux.HandleFunc("POST /api/password", s.handleChangePassword)

	mux.HandleFunc("GET /api/projects", s.listProjects)
	mux.HandleFunc("POST /api/projects", s.createProject)
	mux.HandleFunc("GET /api/projects/{id}", s.getProject)
	mux.HandleFunc("PUT /api/projects/{id}", s.updateProject)
	mux.HandleFunc("DELETE /api/projects/{id}", s.deleteProject)
	mux.HandleFunc("GET /api/projects/{id}/worker-rates", s.projectWorkerRates)

	mux.HandleFunc("GET /api/workers", s.listWorkers)
	mux.HandleFunc("POST /api/workers", s.createWorker)
	mux.HandleFunc("PUT /api/workers/{id}", s.updateWorker)
	mux.HandleFunc("DELETE /api/workers/{id}", s.deleteWorker)

	mux.HandleFunc("GET /api/worklogs", s.listWorkLogs)
	mux.HandleFunc("GET /api/worklogs/grouped", s.groupedWorkLogs)
	mux.HandleFunc("POST /api/worklogs/settle", s.settleWorkLogs)
	mux.HandleFunc("GET /api/worklogs/daily", s.dailyWorkSummary)
	mux.HandleFunc("POST /api/worklogs", s.createWorkLog)
	mux.HandleFunc("POST /api/worklogs/batch", s.batchWorkLogs)
	mux.HandleFunc("PUT /api/worklogs/{id}", s.updateWorkLog)
	mux.HandleFunc("DELETE /api/worklogs/{id}", s.deleteWorkLog)

	mux.HandleFunc("GET /api/price-categories", s.listPriceCategories)
	mux.HandleFunc("POST /api/price-categories", s.createPriceCategory)
	mux.HandleFunc("PUT /api/price-categories/{id}", s.renamePriceCategory)

	mux.HandleFunc("GET /api/price-items", s.listPriceItems)
	mux.HandleFunc("POST /api/price-items", s.createPriceItem)
	mux.HandleFunc("PUT /api/price-items/{id}", s.updatePriceItem)
	mux.HandleFunc("DELETE /api/price-items/{id}", s.deletePriceItem)

	mux.HandleFunc("GET /api/materials", s.listMaterials)
	mux.HandleFunc("POST /api/materials", s.createMaterial)
	mux.HandleFunc("PUT /api/materials/{id}", s.updateMaterial)
	mux.HandleFunc("DELETE /api/materials/{id}", s.deleteMaterial)

	mux.HandleFunc("GET /api/quotations", s.listQuotations)
	mux.HandleFunc("POST /api/quotations", s.createQuotation)
	mux.HandleFunc("GET /api/quotations/{id}", s.getQuotation)
	mux.HandleFunc("PUT /api/quotations/{id}", s.updateQuotation)
	mux.HandleFunc("DELETE /api/quotations/{id}", s.deleteQuotation)
	mux.HandleFunc("GET /api/quotations/{id}/excel", s.exportQuotation)

	mux.HandleFunc("GET /api/settlements", s.listSettlements)
	mux.HandleFunc("POST /api/settlements/preview", s.previewSettlement)
	mux.HandleFunc("POST /api/settlements", s.createSettlement)
	mux.HandleFunc("GET /api/settlements/{id}", s.getSettlement)
	mux.HandleFunc("DELETE /api/settlements/{id}", s.deleteSettlement)
	mux.HandleFunc("PUT /api/settlements/{id}/received", s.updateSettlementReceived)
	mux.HandleFunc("GET /api/settlements/{id}/excel", s.exportSettlement)

	mux.HandleFunc("GET /api/wage-settlements", s.listWageSettlements)
	mux.HandleFunc("GET /api/wage-settlements/{id}/excel", s.exportWageSlip)

	mux.HandleFunc("GET /api/summary", s.summary)

	return s.withSession(s.withStatic(mux))
}

func (s *Server) withStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		s.serveStatic(w, r)
	})
}

// withSession 对 /api/*（登录、健康检查除外）校验会话。
func (s *Server) withSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") ||
			r.URL.Path == "/api/login" || r.URL.Path == "/api/health" ||
			r.URL.Path == "/api/init-status" {
			next.ServeHTTP(w, r)
			return
		}
		c, err := r.Cookie(sessionCookie)
		if err != nil || !s.validSession(c.Value) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "未登录或登录已过期"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) validSession(token string) bool {
	if token == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.sessions, token)
		return false
	}
	return true
}

func (s *Server) newSession() (string, time.Time) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	token := hex.EncodeToString(b)
	exp := time.Now().Add(7 * 24 * time.Hour)
	s.mu.Lock()
	s.sessions[token] = exp
	s.mu.Unlock()
	return token, exp
}

func (s *Server) dropSession(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}
	var hash string
	err := s.db.QueryRow(`SELECT password_hash FROM users WHERE username = ?`, in.Username).Scan(&hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "账号或密码错误"})
		return
	}
	_, _ = s.db.Exec(`UPDATE users SET last_login = ? WHERE username = ?`, now(), in.Username)
	token, exp := s.newSession()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
	})
	writeJSON(w, http.StatusOK, map[string]any{"username": in.Username})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.dropSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleInitStatus 登录页用：是否还是「默认账号未登录过」状态。
func (s *Server) handleInitStatus(w http.ResponseWriter, r *http.Request) {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE last_login = ''`).Scan(&n)
	writeJSON(w, http.StatusOK, map[string]bool{"first_time": n > 0})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"username": "admin"})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Old string `json:"old_password"`
		New string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}
	if len(in.New) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "新密码至少 6 位"})
		return
	}
	var hash string
	if err := s.db.QueryRow(`SELECT password_hash FROM users WHERE username = 'admin'`).Scan(&hash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "账号异常"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Old)) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "原密码错误"})
		return
	}
	nh, err := bcrypt.GenerateFromPassword([]byte(in.New), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "加密失败"})
		return
	}
	if _, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE username = 'admin'`, string(nh)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- 工具函数 ----

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

var errNotFound = errors.New("记录不存在")

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
