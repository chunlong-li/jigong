package api

import (
	"net/http"
	"strings"
)

var priceKinds = map[string]bool{"material": true, "labor_all": true, "labor_only": true}

func (s *Server) listPriceItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"1=1"}
	args := []any{}
	if v := q.Get("kind"); v != "" {
		where = append(where, "kind = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(q.Get("category")); v != "" {
		where = append(where, "category = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		where = append(where, "(name LIKE ? OR spec LIKE ?)")
		like := "%" + v + "%"
		args = append(args, like, like)
	}
	rows, err := s.db.Query(`SELECT id, kind, category, name, spec, unit, price, client_price, remark, created_at
		FROM price_items WHERE `+strings.Join(where, " AND ")+`
		ORDER BY CASE WHEN category = '' THEN 1 ELSE 0 END, category, name, id`, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]PriceItem, 0)
	for rows.Next() {
		var x PriceItem
		if err := rows.Scan(&x.ID, &x.Kind, &x.Category, &x.Name, &x.Spec, &x.Unit, &x.Price, &x.ClientPrice, &x.Remark, &x.CreatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

func (s *Server) createPriceItem(w http.ResponseWriter, r *http.Request) {
	var x PriceItem
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	x.Name = strings.TrimSpace(x.Name)
	if x.Name == "" {
		fail(w, 400, "名称不能为空")
		return
	}
	if x.Kind == "" {
		x.Kind = "material"
	}
	if !priceKinds[x.Kind] {
		fail(w, 400, "类型无效")
		return
	}
	t := now()
	err := s.db.QueryRow(`INSERT INTO price_items (kind, category, name, spec, unit, price, client_price, remark, created_at)
		VALUES (?,?,?,?,?,?,?,?,?) RETURNING id`,
		x.Kind, x.Category, x.Name, x.Spec, x.Unit, round2(x.Price), round2(x.ClientPrice), x.Remark, t).Scan(&x.ID)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	// 手填的新分类也自动进分类表
	if x.Category != "" {
		_, _ = s.db.Exec(`INSERT OR IGNORE INTO price_categories (kind, name, created_at) VALUES (?,?,?)`,
			x.Kind, x.Category, t)
	}
	x.CreatedAt = t
	writeJSON(w, 200, x)
}

func (s *Server) updatePriceItem(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var x PriceItem
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	x.Name = strings.TrimSpace(x.Name)
	if x.Name == "" {
		fail(w, 400, "名称不能为空")
		return
	}
	if x.Kind == "" {
		x.Kind = "material"
	}
	if !priceKinds[x.Kind] {
		fail(w, 400, "类型无效")
		return
	}
	res, err := s.db.Exec(`UPDATE price_items SET kind=?, category=?, name=?, spec=?, unit=?, price=?, client_price=?, remark=? WHERE id=?`,
		x.Kind, x.Category, x.Name, x.Spec, x.Unit, round2(x.Price), round2(x.ClientPrice), x.Remark, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "条目不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) deletePriceItem(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	res, err := s.db.Exec(`DELETE FROM price_items WHERE id = ?`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "条目不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
