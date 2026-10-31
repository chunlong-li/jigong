package api

import (
	"net/http"
	"strings"
)

func (s *Server) listWorkers(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sqlStr := `SELECT id, name, phone, id_card, internal_rate, external_rate, overtime_rate, remark, created_at
	           FROM workers WHERE 1=1`
	args := []any{}
	if q != "" {
		sqlStr += ` AND (name LIKE ? OR phone LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	sqlStr += ` ORDER BY id DESC`

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]Worker, 0)
	for rows.Next() {
		var x Worker
		if err := rows.Scan(&x.ID, &x.Name, &x.Phone, &x.IDCard, &x.InternalRate, &x.ExternalRate, &x.OvertimeRate, &x.Remark, &x.CreatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

func (s *Server) createWorker(w http.ResponseWriter, r *http.Request) {
	var x Worker
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	x.Name = strings.TrimSpace(x.Name)
	if x.Name == "" {
		fail(w, 400, "姓名不能为空")
		return
	}
	t := now()
	err := s.db.QueryRow(`INSERT INTO workers (name, phone, id_card, internal_rate, external_rate, overtime_rate, remark, created_at)
		VALUES (?,?,?,?,?,?,?,?) RETURNING id`,
		x.Name, x.Phone, x.IDCard, round2(x.InternalRate), round2(x.ExternalRate), round2(x.OvertimeRate), x.Remark, t).Scan(&x.ID)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	x.CreatedAt = t
	writeJSON(w, 200, x)
}

func (s *Server) updateWorker(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var x Worker
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	x.Name = strings.TrimSpace(x.Name)
	if x.Name == "" {
		fail(w, 400, "姓名不能为空")
		return
	}
	res, err := s.db.Exec(`UPDATE workers SET name=?, phone=?, id_card=?, internal_rate=?, external_rate=?, overtime_rate=?, remark=? WHERE id=?`,
		x.Name, x.Phone, x.IDCard, round2(x.InternalRate), round2(x.ExternalRate), round2(x.OvertimeRate), x.Remark, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "工人不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) deleteWorker(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM worklogs WHERE worker_id = ?`, id).Scan(&n)
	if n > 0 {
		fail(w, 400, "该工人已有记工记录，不能删除")
		return
	}
	res, err := s.db.Exec(`DELETE FROM workers WHERE id = ?`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		fail(w, 404, "工人不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) workerNameOf(id int64) string {
	var name string
	_ = s.db.QueryRow(`SELECT name FROM workers WHERE id = ?`, id).Scan(&name)
	return name
}
