package api

import (
	"net/http"
	"strconv"
	"strings"
)

var workModes = map[string]bool{"day": true, "hour": true, "piece": true}

func (s *Server) listWorkLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"1=1"}
	args := []any{}

	if v := q.Get("project_id"); v != "" {
		where = append(where, "w.project_id = ?")
		args = append(args, v)
	}
	if v := q.Get("worker_id"); v != "" {
		where = append(where, "w.worker_id = ?")
		args = append(args, v)
	}
	if v := q.Get("from"); v != "" {
		where = append(where, "w.work_date >= ?")
		args = append(args, v)
	}
	if v := q.Get("to"); v != "" {
		where = append(where, "w.work_date <= ?")
		args = append(args, v)
	}
	if v := q.Get("mode"); v != "" {
		where = append(where, "w.mode = ?")
		args = append(args, v)
	}

	sqlStr := `SELECT w.id, w.project_id, p.name, w.worker_id, k.name, w.work_date, w.mode, w.day_type,
	                  w.quantity, w.unit_price, w.amount, w.content, w.remark, w.settled_at, w.created_at
	           FROM worklogs w
	           JOIN projects p ON p.id = w.project_id
	           JOIN workers k ON k.id = w.worker_id
	           WHERE ` + strings.Join(where, " AND ") + `
	           ORDER BY w.work_date DESC, w.id DESC LIMIT 2000`

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]WorkLog, 0)
	for rows.Next() {
		var x WorkLog
		if err := rows.Scan(&x.ID, &x.ProjectID, &x.ProjectName, &x.WorkerID, &x.WorkerName,
			&x.WorkDate, &x.Mode, &x.DayType, &x.Quantity, &x.UnitPrice, &x.Amount, &x.Content, &x.Remark, &x.SettledAt, &x.CreatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

func (s *Server) workerRate(workerID int64) float64 {
	var rate float64
	_ = s.db.QueryRow(`SELECT internal_rate FROM workers WHERE id = ?`, workerID).Scan(&rate)
	return rate
}

func (s *Server) normalizeWorkLog(x *WorkLog) string {
	if x.ProjectID == 0 {
		return "请选择项目"
	}
	if x.WorkerID == 0 {
		return "请选择工人"
	}
	if x.WorkDate == "" {
		return "请选择日期"
	}
	if x.Mode == "" {
		x.Mode = "day"
	}
	if !workModes[x.Mode] {
		return "计价方式无效"
	}
	if x.Quantity <= 0 {
		return "数量必须大于 0"
	}
	if x.UnitPrice <= 0 {
		x.UnitPrice = s.workerRate(x.WorkerID)
	}
	if x.UnitPrice < 0 {
		return "单价不能为负"
	}
	x.Amount = round2(x.Quantity * x.UnitPrice)
	return ""
}

func (s *Server) createWorkLog(w http.ResponseWriter, r *http.Request) {
	var x WorkLog
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if msg := s.normalizeWorkLog(&x); msg != "" {
		fail(w, 400, msg)
		return
	}
	t := now()
	err := s.db.QueryRow(`INSERT INTO worklogs
		(project_id, worker_id, work_date, mode, quantity, unit_price, amount, content, remark, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?) RETURNING id`,
		x.ProjectID, x.WorkerID, x.WorkDate, x.Mode, x.Quantity, x.UnitPrice, x.Amount,
		x.Content, x.Remark, t).Scan(&x.ID)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	x.CreatedAt = t
	x.ProjectName = s.projectNameOf(x.ProjectID)
	x.WorkerName = s.workerNameOf(x.WorkerID)
	writeJSON(w, 200, x)
}

func (s *Server) updateWorkLog(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var x WorkLog
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if msg := s.normalizeWorkLog(&x); msg != "" {
		fail(w, 400, msg)
		return
	}
	res, err := s.db.Exec(`UPDATE worklogs SET project_id=?, worker_id=?, work_date=?, mode=?,
		quantity=?, unit_price=?, amount=?, content=?, remark=? WHERE id=?`,
		x.ProjectID, x.WorkerID, x.WorkDate, x.Mode, x.Quantity, x.UnitPrice, x.Amount,
		x.Content, x.Remark, id)
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

func (s *Server) deleteWorkLog(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	res, err := s.db.Exec(`DELETE FROM worklogs WHERE id = ?`, id)
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

var _ = strconv.Itoa

// 按天类型：full=一天 half=半天 overtime=加班（按小时）
var dayTypes = map[string]bool{"full": true, "half": true, "overtime": true}

// batchWorkLogs 批量记工：多选工人 × 多选日期；按天可选一天/半天/加班。
func (s *Server) batchWorkLogs(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProjectID int64    `json:"project_id"`
		WorkerIDs []int64  `json:"worker_ids"`
		Dates     []string `json:"dates"`
		Mode      string   `json:"mode"`
		DayType   string   `json:"day_type"`
		Quantity  float64  `json:"quantity"`
		Hours     float64  `json:"hours"`
		UnitPrice float64  `json:"unit_price"`
		Content   string   `json:"content"`
		Remark    string   `json:"remark"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if in.ProjectID == 0 {
		fail(w, 400, "请选择项目")
		return
	}
	if len(in.WorkerIDs) == 0 {
		fail(w, 400, "请至少选择一名工人")
		return
	}
	if in.Mode == "" {
		in.Mode = "day"
	}
	if !workModes[in.Mode] {
		fail(w, 400, "计价方式无效")
		return
	}

	// 计算每种记录的数量（按天：一天 1 / 半天 0.5 / 加班 按小时）
	qty := in.Quantity
	if in.Mode == "day" {
		if in.DayType == "" {
			in.DayType = "full"
		}
		if !dayTypes[in.DayType] {
			fail(w, 400, "记工类型无效")
			return
		}
		switch in.DayType {
		case "full":
			qty = 1
		case "half":
			qty = 0.5
		case "overtime":
			qty = in.Hours
			if qty <= 0 {
				fail(w, 400, "加班请填写小时数")
				return
			}
		}
	} else if qty <= 0 {
		fail(w, 400, "数量必须大于 0")
		return
	}
	if in.UnitPrice < 0 {
		fail(w, 400, "单价不能为负")
		return
	}

	// 去重
	workers := make([]int64, 0, len(in.WorkerIDs))
	seenW := map[int64]bool{}
	for _, id := range in.WorkerIDs {
		if !seenW[id] {
			seenW[id] = true
			workers = append(workers, id)
		}
	}
	dates := make([]string, 0, len(in.Dates))
	seenD := map[string]bool{}
	for _, d := range in.Dates {
		if d == "" || seenD[d] {
			continue
		}
		seenD[d] = true
		dates = append(dates, d)
	}
	if len(dates) == 0 {
		fail(w, 400, "请至少选择一个有效日期")
		return
	}

	// 事务前把所有单价查好（连接池只有 1 条连接，事务内再查会自锁）
	prices := make([]float64, len(workers))
	for i, wid := range workers {
		p := in.UnitPrice
		if p <= 0 {
			if in.Mode == "day" && in.DayType == "overtime" {
				p = s.workerOvertimeRate(wid)
			} else {
				p = s.workerRate(wid)
			}
		}
		if p < 0 {
			p = 0
		}
		prices[i] = p
	}

	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	t := now()
	created := 0
	for i, wid := range workers {
		price := prices[i]
		amount := round2(qty * price)
		for _, d := range dates {
			// 一天/半天互斥：写入前清掉同一天的另一类
			if in.Mode == "day" && (in.DayType == "full" || in.DayType == "half") {
				if _, err := tx.Exec(`DELETE FROM worklogs
					WHERE project_id=? AND worker_id=? AND work_date=? AND mode='day'
					  AND (day_type IN ('','full','half'))`,
					in.ProjectID, wid, d); err != nil {
					fail(w, 500, err.Error())
					return
				}
			}
			// 加班同一天只保留一条
			if in.Mode == "day" && in.DayType == "overtime" {
				if _, err := tx.Exec(`DELETE FROM worklogs
					WHERE project_id=? AND worker_id=? AND work_date=? AND mode='day' AND day_type='overtime'`,
					in.ProjectID, wid, d); err != nil {
					fail(w, 500, err.Error())
					return
				}
			}
			if _, err := tx.Exec(`INSERT INTO worklogs
				(project_id, worker_id, work_date, mode, day_type, quantity, unit_price, amount, content, remark, created_at)
				VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
				in.ProjectID, wid, d, in.Mode, in.DayType, qty, price, amount,
				in.Content, in.Remark, t); err != nil {
				fail(w, 500, err.Error())
				return
			}
			created++
		}
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"created": created})
}

// groupedWorkLogs 合并统计：默认按「工地 + 工人」，group_by=month 时按月再拆。
func (s *Server) groupedWorkLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"w.settled_at = ''"}
	args := []any{}

	if v := q.Get("project_id"); v != "" {
		where = append(where, "w.project_id = ?")
		args = append(args, v)
	}
	if v := q.Get("worker_id"); v != "" {
		where = append(where, "w.worker_id = ?")
		args = append(args, v)
	}
	if v := q.Get("from"); v != "" {
		where = append(where, "w.work_date >= ?")
		args = append(args, v)
	}
	if v := q.Get("to"); v != "" {
		where = append(where, "w.work_date <= ?")
		args = append(args, v)
	}

	byMonth := q.Get("group_by") == "month"
	monthExpr, groupBy, orderBy := "''", "w.project_id, w.worker_id", "MAX(w.work_date) DESC, k.name"
	if byMonth {
		monthExpr = "substr(w.work_date,1,7)"
		groupBy = "substr(w.work_date,1,7), w.project_id, w.worker_id"
		orderBy = "substr(w.work_date,1,7) DESC, w.project_id, k.name"
	}

	sqlStr := `SELECT ` + monthExpr + `, w.project_id, p.name, w.worker_id, k.name,
	       COALESCE(SUM(CASE WHEN w.mode='day' AND w.day_type<>'overtime' THEN w.quantity ELSE 0 END),0),
	       COALESCE(SUM(CASE WHEN w.mode='day' AND w.day_type='overtime' THEN w.quantity ELSE 0 END),0),
	       COALESCE(SUM(CASE WHEN w.mode='day' AND w.day_type='overtime' THEN 1 ELSE 0 END),0),
	       COALESCE(SUM(CASE WHEN w.mode='hour' THEN w.quantity ELSE 0 END),0),
	       COALESCE(SUM(CASE WHEN w.mode='piece' THEN w.quantity ELSE 0 END),0),
	       COALESCE(SUM(w.amount),0),
	       COUNT(*),
	       MAX(w.work_date)
	FROM worklogs w
	JOIN projects p ON p.id = w.project_id
	JOIN workers k ON k.id = w.worker_id
	WHERE ` + strings.Join(where, " AND ") + `
	GROUP BY ` + groupBy + `
	ORDER BY ` + orderBy

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]WorkLogGroup, 0)
	for rows.Next() {
		var g WorkLogGroup
		if err := rows.Scan(&g.Month, &g.ProjectID, &g.ProjectName, &g.WorkerID, &g.WorkerName,
			&g.Days, &g.OvertimeHours, &g.OvertimeCount, &g.HourQty, &g.PieceQty,
			&g.Amount, &g.RecordCount, &g.LastDate); err != nil {
			fail(w, 500, err.Error())
			return
		}
		g.Days = round2(g.Days)
		g.OvertimeHours = round2(g.OvertimeHours)
		g.HourQty = round2(g.HourQty)
		g.PieceQty = round2(g.PieceQty)
		g.Amount = round2(g.Amount)
		out = append(out, g)
	}
	writeJSON(w, 200, out)
}

func (s *Server) workerOvertimeRate(workerID int64) float64 {
	var rate float64
	_ = s.db.QueryRow(`SELECT overtime_rate FROM workers WHERE id = ?`, workerID).Scan(&rate)
	return rate
}

// dailyWorkSummary 按日期汇总某工地的工数与人头（点工结算用）。
func (s *Server) dailyWorkSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	projectID := q.Get("project_id")
	if projectID == "" {
		fail(w, 400, "缺少 project_id")
		return
	}
	where := []string{"w.project_id = ?", "w.mode = 'day'", "w.day_type <> 'overtime'"}
	args := []any{projectID}
	if v := q.Get("from"); v != "" {
		where = append(where, "w.work_date >= ?")
		args = append(args, v)
	}
	if v := q.Get("to"); v != "" {
		where = append(where, "w.work_date <= ?")
		args = append(args, v)
	}
	rows, err := s.db.Query(`SELECT w.work_date, COUNT(DISTINCT w.worker_id), COALESCE(SUM(w.quantity),0)
		FROM worklogs w WHERE `+strings.Join(where, " AND ")+`
		GROUP BY w.work_date ORDER BY w.work_date`, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	type dayRow struct {
		Date    string  `json:"date"`
		Workers int     `json:"workers"`
		Days    float64 `json:"days"`
	}
	out := make([]dayRow, 0)
	for rows.Next() {
		var d dayRow
		if err := rows.Scan(&d.Date, &d.Workers, &d.Days); err != nil {
			fail(w, 500, err.Error())
			return
		}
		d.Days = round2(d.Days)
		out = append(out, d)
	}
	writeJSON(w, 200, out)
}
