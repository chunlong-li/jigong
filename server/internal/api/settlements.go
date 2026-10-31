package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *Server) listSettlements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"1=1"}
	args := []any{}
	if v := q.Get("project_id"); v != "" {
		where = append(where, "s.project_id = ?")
		args = append(args, v)
	}
	sqlStr := `SELECT s.id, COALESCE(s.project_id,0), COALESCE(NULLIF(p.name,''), s.project_name),
	                  s.room, s.contract_type, s.settle_no, s.settle_date, s.period_start,
	                  s.period_end, s.payee, s.title, s.work_amount, s.material_amount,
	                  s.deduction, s.payable, s.received, s.remark, s.created_at
	           FROM settlements s LEFT JOIN projects p ON p.id = s.project_id
	           WHERE ` + strings.Join(where, " AND ") + `
	           ORDER BY s.id DESC LIMIT 500`
	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]Settlement, 0)
	for rows.Next() {
		var x Settlement
		if err := rows.Scan(&x.ID, &x.ProjectID, &x.ProjectName, &x.Room, &x.ContractType, &x.SettleNo, &x.SettleDate,
			&x.PeriodStart, &x.PeriodEnd, &x.Payee, &x.Title, &x.WorkAmount, &x.MaterialAmount,
			&x.Deduction, &x.Payable, &x.Received, &x.Remark, &x.CreatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		x.Items = []SettlementItem{}
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

// previewSettlement 按项目 + 期间汇总记工和进料，生成结算明细草稿。
func (s *Server) previewSettlement(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProjectID int64   `json:"project_id"`
		From      string  `json:"period_start"`
		To        string  `json:"period_end"`
		Deduction float64 `json:"deduction"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if in.ProjectID == 0 {
		fail(w, 400, "请选择项目")
		return
	}

	items := make([]SettlementItem, 0)
	order := 0

	// 记工：按工人 + 计价方式 + 单价 汇总
	rows, err := s.db.Query(`SELECT k.name, w.mode, w.unit_price,
	       SUM(w.quantity), SUM(w.amount), COUNT(*)
		FROM worklogs w JOIN workers k ON k.id = w.worker_id
		WHERE w.project_id = ? AND w.work_date >= ? AND w.work_date <= ?
		GROUP BY w.worker_id, w.mode, w.unit_price
		ORDER BY k.name`, in.ProjectID, in.From, in.To)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name, mode string
		var price, qty, amount float64
		var cnt int
		if err := rows.Scan(&name, &mode, &price, &qty, &amount, &cnt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		items = append(items, SettlementItem{
			Category:  "记工",
			Name:      fmt.Sprintf("%s（%s，%d 条）", name, modeLabel(mode), cnt),
			Quantity:  round2(qty),
			UnitPrice: price,
			Amount:    round2(amount),
			SortOrder: order,
		})
		order++
	}

	// 进料：按材料 + 规格 + 单位 + 单价 汇总
	mrows, err := s.db.Query(`SELECT name, spec, unit, unit_price,
	       SUM(quantity), SUM(amount), COUNT(*)
		FROM materials
		WHERE project_id = ? AND in_date >= ? AND in_date <= ?
		GROUP BY name, spec, unit, unit_price
		ORDER BY name`, in.ProjectID, in.From, in.To)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer mrows.Close()
	for mrows.Next() {
		var name, spec, unit string
		var price, qty, amount float64
		var cnt int
		if err := mrows.Scan(&name, &spec, &unit, &price, &qty, &amount, &cnt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		display := name
		if spec != "" {
			display += " " + spec
		}
		items = append(items, SettlementItem{
			Category:  "进料",
			Name:      fmt.Sprintf("%s（%d 条）", display, cnt),
			Quantity:  round2(qty),
			UnitPrice: price,
			Amount:    round2(amount),
			SortOrder: order,
		})
		order++
	}

	work, material := 0.0, 0.0
	for _, it := range items {
		if it.Category == "记工" {
			work += it.Amount
		} else {
			material += it.Amount
		}
	}
	work, material = round2(work), round2(material)

	writeJSON(w, 200, map[string]any{
		"items":           items,
		"work_amount":     work,
		"material_amount": material,
		"deduction":       round2(in.Deduction),
		"payable":         round2(work + material - in.Deduction),
	})
}

func modeLabel(m string) string {
	switch m {
	case "day":
		return "按天"
	case "hour":
		return "按时"
	case "piece":
		return "计件"
	}
	return m
}

func (s *Server) recalcSettlement(x *Settlement) string {
	x.ProjectName = strings.TrimSpace(x.ProjectName)
	if x.ProjectID == 0 && x.ProjectName == "" {
		return "请选择项目或填写项目名称"
	}
	if x.ContractType == "" {
		x.ContractType = "包工包料"
	}
	if x.SettleDate == "" {
		x.SettleDate = time.Now().Format("2006-01-02")
	}
	if x.Title == "" {
		x.Title = "结算单"
	}
	work, material := 0.0, 0.0
	for i := range x.Items {
		it := &x.Items[i]
		it.Name = strings.TrimSpace(it.Name)
		if it.Name == "" {
			return fmt.Sprintf("第 %d 行缺少名称", i+1)
		}
		if it.Category == "" {
			it.Category = "记工"
		}
		it.Amount = round2(it.Quantity * it.UnitPrice)
		it.SortOrder = i
		if it.Category == "记工" {
			work += it.Amount
		} else {
			material += it.Amount
		}
	}
	x.WorkAmount = round2(work)
	x.MaterialAmount = round2(material)
	x.Payable = round2(x.WorkAmount + x.MaterialAmount - x.Deduction)
	if x.Received == 0 {
		x.Received = x.Payable
	}
	x.Received = round2(x.Received)
	return ""
}

func (s *Server) createSettlement(w http.ResponseWriter, r *http.Request) {
	var x Settlement
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if msg := s.recalcSettlement(&x); msg != "" {
		fail(w, 400, msg)
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	t := now()
	var pid any
	if x.ProjectID > 0 {
		pid = x.ProjectID
	}
	err = tx.QueryRow(`INSERT INTO settlements
		(project_id, project_name, room, contract_type, settle_no, settle_date, period_start, period_end,
		 payee, title, work_amount, material_amount, deduction, payable, received, remark, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING id`,
		pid, x.ProjectName, x.Room, x.ContractType, x.SettleNo, x.SettleDate, x.PeriodStart, x.PeriodEnd,
		x.Payee, x.Title, x.WorkAmount, x.MaterialAmount, x.Deduction, x.Payable, x.Received, x.Remark, t).Scan(&x.ID)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	for _, it := range x.Items {
		if _, err := tx.Exec(`INSERT INTO settlement_items
			(settlement_id, category, room, name, spec, unit, quantity, unit_price, amount, sort_order)
			VALUES (?,?,?,?,?,?,?,?,?,?)`,
			x.ID, it.Category, it.Room, it.Name, it.Spec, it.Unit, it.Quantity, it.UnitPrice, it.Amount, it.SortOrder); err != nil {
			fail(w, 500, err.Error())
			return
		}
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, err.Error())
		return
	}
	x.CreatedAt = t
	if x.ProjectID > 0 {
		x.ProjectName = s.projectNameOf(x.ProjectID)
	}
	writeJSON(w, 200, x)
}

func (s *Server) getSettlement(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	x, err := s.findSettlement(id)
	if err == sql.ErrNoRows {
		fail(w, 404, "结算单不存在")
		return
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, x)
}

func (s *Server) findSettlement(id int64) (Settlement, error) {
	var x Settlement
	err := s.db.QueryRow(`SELECT s.id, COALESCE(s.project_id,0),
		COALESCE(NULLIF(p.name,''), s.project_name), s.room, s.contract_type, s.settle_no, s.settle_date,
		s.period_start, s.period_end, s.payee, s.title, s.work_amount, s.material_amount,
		s.deduction, s.payable, s.received, s.remark, s.created_at
		FROM settlements s LEFT JOIN projects p ON p.id = s.project_id WHERE s.id = ?`, id).
		Scan(&x.ID, &x.ProjectID, &x.ProjectName, &x.Room, &x.ContractType, &x.SettleNo, &x.SettleDate, &x.PeriodStart,
			&x.PeriodEnd, &x.Payee, &x.Title, &x.WorkAmount, &x.MaterialAmount,
			&x.Deduction, &x.Payable, &x.Received, &x.Remark, &x.CreatedAt)
	if err != nil {
		return x, err
	}
	rows, err := s.db.Query(`SELECT id, category, room, name, spec, unit, quantity, unit_price, amount, sort_order
		FROM settlement_items WHERE settlement_id = ? ORDER BY sort_order, id`, id)
	if err != nil {
		return x, err
	}
	defer rows.Close()
	x.Items = make([]SettlementItem, 0)
	for rows.Next() {
		var it SettlementItem
		if err := rows.Scan(&it.ID, &it.Category, &it.Room, &it.Name, &it.Spec, &it.Unit, &it.Quantity, &it.UnitPrice,
			&it.Amount, &it.SortOrder); err != nil {
			return x, err
		}
		x.Items = append(x.Items, it)
	}
	return x, nil
}

func (s *Server) deleteSettlement(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	res, err := s.db.Exec(`DELETE FROM settlements WHERE id = ?`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "结算单不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) exportSettlement(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	x, err := s.findSettlement(id)
	if err == sql.ErrNoRows {
		fail(w, 404, "结算单不存在")
		return
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	data, err := buildSettlementXLSX(x)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	name := strings.TrimSpace(x.ProjectName)
	if name == "" {
		name = strings.TrimSpace(x.SettleNo)
	}
	if name == "" {
		name = fmt.Sprintf("结算单-%d", x.ID)
	}
	writeXLSX(w, sanitizeFilename(name)+".xlsx", data)
}

// updateSettlementReceived 登记实收金额。
func (s *Server) updateSettlementReceived(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var in struct {
		Received float64 `json:"received"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if in.Received < 0 {
		fail(w, 400, "实收金额不能为负")
		return
	}
	res, err := s.db.Exec(`UPDATE settlements SET received = ? WHERE id = ?`, round2(in.Received), id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "结算单不存在")
		return
	}
	writeJSON(w, 200, map[string]any{"status": "ok", "received": round2(in.Received)})
}
