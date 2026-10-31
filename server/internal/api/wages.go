package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// settleWorkLogs 结算某工人在某工地的未结算记工：先归档快照，再冻结原记录。
func (s *Server) settleWorkLogs(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProjectID int64  `json:"project_id"`
		WorkerID  int64  `json:"worker_id"`
		Through   string `json:"through"` // 只结算该日期及之前（按月查看时用）
		Remark    string `json:"remark"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if in.ProjectID == 0 || in.WorkerID == 0 {
		fail(w, 400, "缺少工地或工人")
		return
	}

	var (
		days, otHours, hourQty, pieceQty, amount float64
		otCount                                  int
		start                                    sql.NullString
		end                                      sql.NullString
	)
	cond := "settled_at=''"
	args := []any{in.ProjectID, in.WorkerID}
	if in.Through != "" {
		cond += " AND work_date <= ?"
		args = append(args, in.Through)
	}

	err := s.db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN mode='day' AND day_type<>'overtime' THEN quantity ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN mode='day' AND day_type='overtime' THEN quantity ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN mode='day' AND day_type='overtime' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN mode='hour' THEN quantity ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN mode='piece' THEN quantity ELSE 0 END),0),
		COALESCE(SUM(amount),0),
		MIN(work_date), MAX(work_date)
		FROM worklogs WHERE project_id=? AND worker_id=? AND `+cond,
		args...).
		Scan(&days, &otHours, &otCount, &hourQty, &pieceQty, &amount, &start, &end)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if days == 0 && otCount == 0 && hourQty == 0 && pieceQty == 0 {
		fail(w, 400, "没有可结算的记工")
		return
	}

	t := now()
	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	var id int64
	err = tx.QueryRow(`INSERT INTO wage_settlements
		(project_id, worker_id, period_start, period_end, days, overtime_hours, overtime_count,
		 hour_qty, piece_qty, amount, remark, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?) RETURNING id`,
		in.ProjectID, in.WorkerID, start.String, end.String, round2(days), round2(otHours), otCount,
		round2(hourQty), round2(pieceQty), round2(amount), in.Remark, t).Scan(&id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE worklogs SET settled_at=? WHERE project_id=? AND worker_id=? AND `+cond,
		append([]any{t}, args...)...); err != nil {
		fail(w, 500, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, err.Error())
		return
	}

	ws := WageSettlement{
		ID: id, ProjectID: in.ProjectID, WorkerID: in.WorkerID,
		ProjectName: s.projectNameOf(in.ProjectID), WorkerName: s.workerNameOf(in.WorkerID),
		PeriodStart: start.String, PeriodEnd: end.String,
		Days: round2(days), OvertimeHours: round2(otHours), OvertimeCount: otCount,
		HourQty: round2(hourQty), PieceQty: round2(pieceQty), Amount: round2(amount),
		Remark: in.Remark, CreatedAt: t,
	}
	writeJSON(w, 200, ws)
}

func (s *Server) listWageSettlements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"1=1"}
	args := []any{}
	if v := q.Get("project_id"); v != "" {
		where = append(where, "ws.project_id = ?")
		args = append(args, v)
	}
	if v := q.Get("worker_id"); v != "" {
		where = append(where, "ws.worker_id = ?")
		args = append(args, v)
	}
	sqlStr := `SELECT ws.id, ws.project_id, p.name, ws.worker_id, k.name, ws.period_start, ws.period_end,
	       ws.days, ws.overtime_hours, ws.overtime_count, ws.hour_qty, ws.piece_qty, ws.amount,
	       ws.remark, ws.created_at
	FROM wage_settlements ws
	JOIN projects p ON p.id = ws.project_id
	JOIN workers k ON k.id = ws.worker_id
	WHERE ` + strings.Join(where, " AND ") + `
	ORDER BY ws.id DESC LIMIT 500`
	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]WageSettlement, 0)
	for rows.Next() {
		var x WageSettlement
		if err := rows.Scan(&x.ID, &x.ProjectID, &x.ProjectName, &x.WorkerID, &x.WorkerName,
			&x.PeriodStart, &x.PeriodEnd, &x.Days, &x.OvertimeHours, &x.OvertimeCount,
			&x.HourQty, &x.PieceQty, &x.Amount, &x.Remark, &x.CreatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

func (s *Server) findWageSettlement(id int64) (WageSettlement, error) {
	var x WageSettlement
	err := s.db.QueryRow(`SELECT ws.id, ws.project_id, p.name, ws.worker_id, k.name,
		ws.period_start, ws.period_end, ws.days, ws.overtime_hours, ws.overtime_count,
		ws.hour_qty, ws.piece_qty, ws.amount, ws.remark, ws.created_at
		FROM wage_settlements ws
		JOIN projects p ON p.id = ws.project_id
		JOIN workers k ON k.id = ws.worker_id
		WHERE ws.id = ?`, id).
		Scan(&x.ID, &x.ProjectID, &x.ProjectName, &x.WorkerID, &x.WorkerName,
			&x.PeriodStart, &x.PeriodEnd, &x.Days, &x.OvertimeHours, &x.OvertimeCount,
			&x.HourQty, &x.PieceQty, &x.Amount, &x.Remark, &x.CreatedAt)
	return x, err
}

// exportWageSlip 生成工资单 Excel。
func (s *Server) exportWageSlip(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	x, err := s.findWageSettlement(id)
	if err == sql.ErrNoRows {
		fail(w, 404, "结算记录不存在")
		return
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	data, err := buildWageSlipXLSX(x)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	name := fmt.Sprintf("工资单-%s-%s", x.WorkerName, x.PeriodEnd)
	writeXLSX(w, name+".xlsx", data)
}

// buildWageSlipXLSX 工资单：一个工人一张，含出勤汇总与签字栏。
func buildWageSlipXLSX(x WageSettlement) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "工资单"
	f.SetSheetName("Sheet1", sheet)

	_ = f.SetColWidth(sheet, "A", "A", 16)
	_ = f.SetColWidth(sheet, "B", "B", 28)
	_ = f.SetColWidth(sheet, "C", "C", 16)
	_ = f.SetColWidth(sheet, "D", "D", 20)

	_ = f.MergeCell(sheet, "A1", "D1")
	_ = f.SetCellValue(sheet, "A1", "工 资 单")
	_ = f.SetRowHeight(sheet, 1, 36)
	applyStyle(f, sheet, "A1", "D1", styleTitle)

	rows := [][2]string{
		{"工地", x.ProjectName},
		{"工人", x.WorkerName},
		{"结算期间", x.PeriodStart + " 至 " + x.PeriodEnd},
		{"出勤天数", fmt.Sprintf("%g 天", x.Days)},
		{"加班", fmt.Sprintf("%d 次 / %g 小时", x.OvertimeCount, x.OvertimeHours)},
	}
	if x.HourQty > 0 || x.PieceQty > 0 {
		extra := []string{}
		if x.HourQty > 0 {
			extra = append(extra, fmt.Sprintf("按时 %g 小时", x.HourQty))
		}
		if x.PieceQty > 0 {
			extra = append(extra, fmt.Sprintf("计件 %g 件", x.PieceQty))
		}
		rows = append(rows, [2]string{"其他", strings.Join(extra, "，")})
	}

	r0 := 2
	for i, kv := range rows {
		_ = f.SetCellValue(sheet, cellName(1, r0+i), kv[0])
		_ = f.MergeCell(sheet, cellName(2, r0+i), cellName(4, r0+i))
		_ = f.SetCellValue(sheet, cellName(2, r0+i), kv[1])
		applyStyle(f, sheet, cellName(1, r0+i), cellName(1, r0+i), styleHead)
		applyStyle(f, sheet, cellName(2, r0+i), cellName(4, r0+i), styleText)
		_ = f.SetRowHeight(sheet, r0+i, 26)
	}
	row := r0 + len(rows)

	_ = f.SetCellValue(sheet, cellName(1, row), "应付金额")
	_ = f.MergeCell(sheet, cellName(2, row), cellName(3, row))
	_ = f.SetCellValue(sheet, cellName(2, row), x.Amount)
	_ = f.SetCellValue(sheet, cellName(4, row), "元")
	applyStyle(f, sheet, cellName(1, row), cellName(1, row), styleHead)
	applyStyle(f, sheet, cellName(2, row), cellName(3, row), styleTotalMoney)
	applyStyle(f, sheet, cellName(4, row), cellName(4, row), styleText)
	_ = f.SetRowHeight(sheet, row, 30)
	row++

	_ = f.MergeCell(sheet, cellName(1, row), cellName(4, row))
	_ = f.SetCellValue(sheet, cellName(1, row), "大写："+RMBUpper(x.Amount))
	applyStyle(f, sheet, cellName(1, row), cellName(4, row), styleNote)
	row++

	if x.Remark != "" {
		_ = f.MergeCell(sheet, cellName(1, row), cellName(4, row))
		_ = f.SetCellValue(sheet, cellName(1, row), "备注："+x.Remark)
		applyStyle(f, sheet, cellName(1, row), cellName(4, row), styleNote)
		row++
	}

	row++
	_ = f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	_ = f.SetCellValue(sheet, cellName(1, row), "制单日期："+time.Now().Format("2006-01-02"))
	_ = f.MergeCell(sheet, cellName(3, row), cellName(4, row))
	_ = f.SetCellValue(sheet, cellName(3, row), "工人签字：")
	applyStyle(f, sheet, cellName(1, row), cellName(4, row), styleMeta)

	return xlsxBytes(f)
}
