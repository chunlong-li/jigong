package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
)

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	sqlStr := `SELECT id, name, site_address, client, contact, contract_amount,
	                  start_date, status, contract_type, remark, created_at, updated_at
	           FROM projects WHERE 1=1`
	args := []any{}
	if q != "" {
		sqlStr += ` AND (name LIKE ? OR client LIKE ? OR site_address LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	if status != "" {
		sqlStr += ` AND status = ?`
		args = append(args, status)
	}
	sqlStr += ` ORDER BY id DESC`

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]Project, 0)
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.SiteAddress, &p.Client, &p.Contact,
			&p.ContractAmount, &p.StartDate, &p.Status, &p.ContractType, &p.Remark, &p.CreatedAt, &p.UpdatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		out = append(out, p)
	}
	writeJSON(w, 200, out)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var p Project
	if err := decode(r, &p); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		fail(w, 400, "项目名称不能为空")
		return
	}
	if p.Status == "" {
		p.Status = "active"
	}
	if p.ContractType == "" {
		p.ContractType = "包工包料"
	}
	t := now()
	err := s.db.QueryRow(`INSERT INTO projects
		(name, site_address, client, contact, contract_amount, start_date, status, contract_type, remark, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?) RETURNING id`,
		p.Name, p.SiteAddress, p.Client, p.Contact, round2(p.ContractAmount),
		p.StartDate, p.Status, p.ContractType, p.Remark, t, t).Scan(&p.ID)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	p.CreatedAt, p.UpdatedAt = t, t
	writeJSON(w, 200, p)
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var p Project
	err = s.db.QueryRow(`SELECT id, name, site_address, client, contact, contract_amount,
		start_date, status, contract_type, remark, created_at, updated_at FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.SiteAddress, &p.Client, &p.Contact, &p.ContractAmount,
			&p.StartDate, &p.Status, &p.ContractType, &p.Remark, &p.CreatedAt, &p.UpdatedAt)
	if isNoRows(err) {
		fail(w, 404, "项目不存在")
		return
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, p)
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var p Project
	if err := decode(r, &p); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		fail(w, 400, "项目名称不能为空")
		return
	}
	res, err := s.db.Exec(`UPDATE projects SET name=?, site_address=?, client=?, contact=?,
		contract_amount=?, start_date=?, status=?, contract_type=?, remark=?, updated_at=? WHERE id=?`,
		p.Name, p.SiteAddress, p.Client, p.Contact, round2(p.ContractAmount),
		p.StartDate, p.Status, p.ContractType, p.Remark, now(), id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "项目不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var workCount int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM worklogs WHERE project_id = ?`, id).Scan(&workCount)
	if workCount > 0 {
		fail(w, 400, "该项目下已有记工记录，请先删除记录")
		return
	}
	res, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "项目不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	var sum Summary
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&sum.ProjectCount)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM workers`).Scan(&sum.WorkerCount)

	month := now()[:7]
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM worklogs`).Scan(&sum.WorkAmount)
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM materials`).Scan(&sum.MaterialAmount)
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM worklogs WHERE substr(work_date,1,7) = ?`, month).Scan(&sum.MonthWork)
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM materials WHERE substr(in_date,1,7) = ?`, month).Scan(&sum.MonthMaterial)
	writeJSON(w, 200, sum)
}

var _ = sql.ErrNoRows

func (s *Server) projectNameOf(id int64) string {
	var name string
	_ = s.db.QueryRow(`SELECT name FROM projects WHERE id = ?`, id).Scan(&name)
	return name
}

// projectWorkerRates 取该项目参与记工的工人的平均单价（点工结算默认价用）。
func (s *Server) projectWorkerRates(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var count int
	var avgIn, avgEx sql.NullFloat64
	err = s.db.QueryRow(`SELECT COUNT(DISTINCT w.worker_id), AVG(k.internal_rate), AVG(k.external_rate)
		FROM worklogs w JOIN workers k ON k.id = w.worker_id
		WHERE w.project_id = ?`, id).Scan(&count, &avgIn, &avgEx)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{
		"count":        count,
		"avg_internal": round2(avgIn.Float64),
		"avg_external": round2(avgEx.Float64),
	})
}
