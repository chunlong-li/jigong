package api

import (
	"net/http"
	"strings"
)

func (s *Server) listMaterials(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"1=1"}
	args := []any{}

	if v := q.Get("project_id"); v != "" {
		where = append(where, "m.project_id = ?")
		args = append(args, v)
	}
	if v := q.Get("from"); v != "" {
		where = append(where, "m.in_date >= ?")
		args = append(args, v)
	}
	if v := q.Get("to"); v != "" {
		where = append(where, "m.in_date <= ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		where = append(where, "(m.name LIKE ? OR m.spec LIKE ? OR m.supplier LIKE ?)")
		like := "%" + v + "%"
		args = append(args, like, like, like)
	}

	sqlStr := `SELECT m.id, m.project_id, p.name, m.in_date, m.name, m.spec, m.unit,
	                  m.quantity, m.unit_price, m.amount, m.supplier, m.remark, m.created_at
	           FROM materials m
	           JOIN projects p ON p.id = m.project_id
	           WHERE ` + strings.Join(where, " AND ") + `
	           ORDER BY m.in_date DESC, m.id DESC LIMIT 2000`

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]Material, 0)
	for rows.Next() {
		var x Material
		if err := rows.Scan(&x.ID, &x.ProjectID, &x.ProjectName, &x.InDate, &x.Name, &x.Spec,
			&x.Unit, &x.Quantity, &x.UnitPrice, &x.Amount, &x.Supplier, &x.Remark, &x.CreatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

func (s *Server) normalizeMaterial(x *Material) string {
	if x.ProjectID == 0 {
		return "请选择项目"
	}
	x.Name = strings.TrimSpace(x.Name)
	if x.Name == "" {
		return "材料名称不能为空"
	}
	if x.InDate == "" {
		return "请选择进料日期"
	}
	if x.Quantity <= 0 {
		return "数量必须大于 0"
	}
	if x.UnitPrice < 0 {
		return "单价不能为负"
	}
	x.Amount = round2(x.Quantity * x.UnitPrice)
	return ""
}

func (s *Server) createMaterial(w http.ResponseWriter, r *http.Request) {
	var x Material
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if msg := s.normalizeMaterial(&x); msg != "" {
		fail(w, 400, msg)
		return
	}
	t := now()
	err := s.db.QueryRow(`INSERT INTO materials
		(project_id, in_date, name, spec, unit, quantity, unit_price, amount, supplier, remark, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?) RETURNING id`,
		x.ProjectID, x.InDate, x.Name, x.Spec, x.Unit, x.Quantity, x.UnitPrice, x.Amount,
		x.Supplier, x.Remark, t).Scan(&x.ID)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	x.CreatedAt = t
	x.ProjectName = s.projectNameOf(x.ProjectID)
	writeJSON(w, 200, x)
}

func (s *Server) updateMaterial(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var x Material
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if msg := s.normalizeMaterial(&x); msg != "" {
		fail(w, 400, msg)
		return
	}
	res, err := s.db.Exec(`UPDATE materials SET project_id=?, in_date=?, name=?, spec=?, unit=?,
		quantity=?, unit_price=?, amount=?, supplier=?, remark=? WHERE id=?`,
		x.ProjectID, x.InDate, x.Name, x.Spec, x.Unit, x.Quantity, x.UnitPrice, x.Amount,
		x.Supplier, x.Remark, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "记录不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) deleteMaterial(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	res, err := s.db.Exec(`DELETE FROM materials WHERE id = ?`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "记录不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
