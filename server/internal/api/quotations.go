package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *Server) listQuotations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"1=1"}
	args := []any{}
	if v := q.Get("project_id"); v != "" {
		where = append(where, "q.project_id = ?")
		args = append(args, v)
	}
	sqlStr := `SELECT q.id, COALESCE(q.project_id,0), COALESCE(NULLIF(p.name,''), q.project_name),
	                  q.room, q.contract_type, q.quote_type, q.quote_no, q.quote_date, q.client, q.title,
	                  q.tax_rate, q.subtotal, q.tax_amount, q.total, q.remark, q.created_at
	           FROM quotations q LEFT JOIN projects p ON p.id = q.project_id
	           WHERE ` + strings.Join(where, " AND ") + `
	           ORDER BY q.id DESC LIMIT 500`
	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]Quotation, 0)
	for rows.Next() {
		var x Quotation
		if err := rows.Scan(&x.ID, &x.ProjectID, &x.ProjectName, &x.Room, &x.ContractType, &x.QuoteType,
			&x.QuoteNo, &x.QuoteDate, &x.Client, &x.Title, &x.TaxRate, &x.Subtotal, &x.TaxAmount,
			&x.Total, &x.Remark, &x.CreatedAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		x.Items = []QuotationItem{}
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

func (s *Server) loadQuotationItems(id int64) ([]QuotationItem, error) {
	rows, err := s.db.Query(`SELECT id, room, name, spec, unit, quantity, unit_price, amount, sort_order
		FROM quotation_items WHERE quotation_id = ? ORDER BY sort_order, id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]QuotationItem, 0)
	for rows.Next() {
		var it QuotationItem
		if err := rows.Scan(&it.ID, &it.Room, &it.Name, &it.Spec, &it.Unit, &it.Quantity,
			&it.UnitPrice, &it.Amount, &it.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (s *Server) recalcQuotation(x *Quotation) string {
	x.ProjectName = strings.TrimSpace(x.ProjectName)
	if x.ProjectID == 0 && x.ProjectName == "" {
		return "请填写项目名称"
	}
	if x.ContractType == "" {
		x.ContractType = "包工包料"
	}
	if x.QuoteType == "" {
		x.QuoteType = "客户报价"
	}
	if x.QuoteDate == "" {
		x.QuoteDate = time.Now().Format("2006-01-02")
	}
	if x.Title == "" {
		x.Title = "报价单"
	}
	if len(x.Items) == 0 {
		return "至少添加一条报价明细"
	}
	subtotal := 0.0
	for i := range x.Items {
		it := &x.Items[i]
		it.Name = strings.TrimSpace(it.Name)
		if it.Name == "" {
			return fmt.Sprintf("第 %d 行缺少名称", i+1)
		}
		if it.Quantity < 0 || it.UnitPrice < 0 {
			return fmt.Sprintf("第 %d 行数量/单价不能为负", i+1)
		}
		it.Amount = round2(it.Quantity * it.UnitPrice)
		it.SortOrder = i
		subtotal += it.Amount
	}
	x.Subtotal = round2(subtotal)
	x.TaxAmount = round2(x.Subtotal * x.TaxRate / 100)
	x.Total = round2(x.Subtotal + x.TaxAmount)
	return ""
}

func (s *Server) createQuotation(w http.ResponseWriter, r *http.Request) {
	var x Quotation
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if msg := s.recalcQuotation(&x); msg != "" {
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
	err = tx.QueryRow(`INSERT INTO quotations
		(project_id, project_name, room, contract_type, quote_type, quote_no, quote_date, client, title,
		 tax_rate, subtotal, tax_amount, total, remark, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING id`,
		pid, x.ProjectName, x.Room, x.ContractType, x.QuoteType, x.QuoteNo, x.QuoteDate, x.Client, x.Title, x.TaxRate,
		x.Subtotal, x.TaxAmount, x.Total, x.Remark, t).Scan(&x.ID)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	for _, it := range x.Items {
		if _, err := tx.Exec(`INSERT INTO quotation_items
			(quotation_id, room, name, spec, unit, quantity, unit_price, amount, sort_order)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			x.ID, it.Room, it.Name, it.Spec, it.Unit, it.Quantity, it.UnitPrice, it.Amount, it.SortOrder); err != nil {
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

func (s *Server) getQuotation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	x, err := s.findQuotation(id)
	if isNoRows(err) {
		fail(w, 404, "报价单不存在")
		return
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, x)
}

func (s *Server) findQuotation(id int64) (Quotation, error) {
	var x Quotation
	err := s.db.QueryRow(`SELECT q.id, COALESCE(q.project_id,0),
		COALESCE(NULLIF(p.name,''), q.project_name), q.room, q.contract_type, q.quote_type, q.quote_no, q.quote_date, q.client,
		q.title, q.tax_rate, q.subtotal, q.tax_amount, q.total, q.remark, q.created_at
		FROM quotations q LEFT JOIN projects p ON p.id = q.project_id WHERE q.id = ?`, id).
		Scan(&x.ID, &x.ProjectID, &x.ProjectName, &x.Room, &x.ContractType, &x.QuoteType, &x.QuoteNo, &x.QuoteDate, &x.Client,
			&x.Title, &x.TaxRate, &x.Subtotal, &x.TaxAmount, &x.Total, &x.Remark, &x.CreatedAt)
	if err != nil {
		return x, err
	}
	x.Items, err = s.loadQuotationItems(id)
	return x, err
}

func (s *Server) updateQuotation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var x Quotation
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if msg := s.recalcQuotation(&x); msg != "" {
		fail(w, 400, msg)
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	var pidUpd any
	if x.ProjectID > 0 {
		pidUpd = x.ProjectID
	}
	res, err := tx.Exec(`UPDATE quotations SET project_id=?, project_name=?, room=?, contract_type=?, quote_type=?,
		quote_no=?, quote_date=?, client=?, title=?, tax_rate=?, subtotal=?, tax_amount=?, total=?, remark=?
		WHERE id=?`,
		pidUpd, x.ProjectName, x.Room, x.ContractType, x.QuoteType, x.QuoteNo, x.QuoteDate, x.Client, x.Title, x.TaxRate,
		x.Subtotal, x.TaxAmount, x.Total, x.Remark, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "报价单不存在")
		return
	}
	if _, err := tx.Exec(`DELETE FROM quotation_items WHERE quotation_id = ?`, id); err != nil {
		fail(w, 500, err.Error())
		return
	}
	for _, it := range x.Items {
		if _, err := tx.Exec(`INSERT INTO quotation_items
			(quotation_id, room, name, spec, unit, quantity, unit_price, amount, sort_order)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			id, it.Room, it.Name, it.Spec, it.Unit, it.Quantity, it.UnitPrice, it.Amount, it.SortOrder); err != nil {
			fail(w, 500, err.Error())
			return
		}
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) deleteQuotation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	res, err := s.db.Exec(`DELETE FROM quotations WHERE id = ?`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "报价单不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// exportQuotation 生成报价单 Excel。
func (s *Server) exportQuotation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	x, err := s.findQuotation(id)
	if isNoRows(err) {
		fail(w, 404, "报价单不存在")
		return
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	data, err := buildQuotationXLSX(x)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	name := strings.TrimSpace(x.ProjectName)
	if name == "" {
		name = strings.TrimSpace(x.QuoteNo)
	}
	if name == "" {
		name = fmt.Sprintf("报价单-%d", x.ID)
	}
	writeXLSX(w, sanitizeFilename(name)+".xlsx", data)
}

// 供 settlement 复用
func (s *Server) queryOneFloat(query string, args ...any) float64 {
	var v sql.NullFloat64
	_ = s.db.QueryRow(query, args...).Scan(&v)
	if v.Valid {
		return v.Float64
	}
	return 0
}

func sanitizeFilename(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	return strings.TrimSpace(r.Replace(name))
}

func writeXLSX(w http.ResponseWriter, filename string, data []byte) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	_, _ = w.Write(data)
}
