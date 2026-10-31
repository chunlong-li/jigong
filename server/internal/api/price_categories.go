package api

import (
	"net/http"
	"strings"
)

func (s *Server) listPriceCategories(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	where, args := "1=1", []any{}
	if kind != "" {
		where = "kind = ?"
		args = append(args, kind)
	}
	rows, err := s.db.Query(`SELECT id, kind, name, created_at FROM price_categories
		WHERE `+where+` ORDER BY name`, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]PriceCategory, 0)
	for rows.Next() {
		var x PriceCategory
		if err := rows.Scan(&x.ID, &x.Kind, &x.Name, &x.CreatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

func (s *Server) createPriceCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		fail(w, 400, "分类名称不能为空")
		return
	}
	if in.Kind == "" {
		in.Kind = "material"
	}
	if !priceKinds[in.Kind] {
		fail(w, 400, "类型无效")
		return
	}
	var id int64
	err := s.db.QueryRow(`INSERT INTO price_categories (kind, name, created_at) VALUES (?,?,?)
		RETURNING id`, in.Kind, in.Name, now()).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			fail(w, 400, "该分类已存在")
			return
		}
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "kind": in.Kind, "name": in.Name})
}

// renamePriceCategory 改分类名，并同步该分类下所有价目条目。
func (s *Server) renamePriceCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		fail(w, 400, "分类名称不能为空")
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	var kind, oldName string
	if err := tx.QueryRow(`SELECT kind, name FROM price_categories WHERE id = ?`, id).Scan(&kind, &oldName); err != nil {
		fail(w, 404, "分类不存在")
		return
	}
	if _, err := tx.Exec(`UPDATE price_categories SET name = ? WHERE id = ?`, in.Name, id); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			fail(w, 400, "该分类已存在")
			return
		}
		fail(w, 500, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE price_items SET category = ? WHERE kind = ? AND category = ?`,
		in.Name, kind, oldName); err != nil {
		fail(w, 500, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"status": "ok", "kind": kind, "name": in.Name})
}
