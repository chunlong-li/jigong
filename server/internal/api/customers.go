package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
)

// listCustomers 客户列表，附带结算汇总（应付/实收/未结款）。
func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.category, c.contact, c.phone, c.address, c.remark, c.created_at,
		       COALESCE(st.cnt, 0), COALESCE(st.payable, 0), COALESCE(st.received, 0),
		       COALESCE(st.open_cnt, 0), COALESCE(st.open_left, 0)
		FROM customers c
		LEFT JOIN (
			SELECT customer_id, COUNT(*) AS cnt,
			       SUM(payable) AS payable, SUM(received) AS received,
			       SUM(CASE WHEN COALESCE(settled,0) = 0 THEN 1 ELSE 0 END) AS open_cnt,
			       SUM(CASE WHEN COALESCE(settled,0) = 0 THEN payable - received ELSE 0 END) AS open_left
			FROM settlements
			WHERE COALESCE(customer_id, 0) > 0
			GROUP BY customer_id
		) st ON st.customer_id = c.id
		ORDER BY c.id DESC`)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := make([]Customer, 0)
	for rows.Next() {
		var x Customer
		if err := rows.Scan(&x.ID, &x.Name, &x.Category, &x.Contact, &x.Phone, &x.Address, &x.Remark,
			&x.CreatedAt, &x.SettleCount, &x.Payable, &x.Received, &x.OpenCount, &x.Outstanding); err != nil {
			fail(w, 500, err.Error())
			return
		}
		x.Outstanding = round2(x.Outstanding)
		out = append(out, x)
	}
	writeJSON(w, 200, out)
}

func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	var x Customer
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	x.Name = strings.TrimSpace(x.Name)
	if x.Name == "" {
		fail(w, 400, "请填写客户名称")
		return
	}
	x.CreatedAt = now()
	if err := s.db.QueryRow(`INSERT INTO customers (name, category, contact, phone, address, remark, created_at)
		VALUES (?,?,?,?,?,?,?) RETURNING id`,
		x.Name, x.Category, x.Contact, x.Phone, x.Address, x.Remark, x.CreatedAt).Scan(&x.ID); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, x)
}

func (s *Server) updateCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var x Customer
	if err := decode(r, &x); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	x.Name = strings.TrimSpace(x.Name)
	if x.Name == "" {
		fail(w, 400, "请填写客户名称")
		return
	}
	res, err := s.db.Exec(`UPDATE customers SET name=?, category=?, contact=?, phone=?, address=?, remark=? WHERE id=?`,
		x.Name, x.Category, x.Contact, x.Phone, x.Address, x.Remark, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "客户不存在")
		return
	}
	x.ID = id
	writeJSON(w, 200, x)
}

func (s *Server) deleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM settlements WHERE customer_id = ?`, id).Scan(&n)
	if n > 0 {
		fail(w, 409, "该客户还有结算单，不能删除")
		return
	}
	_, _ = s.db.Exec(`UPDATE quotations SET customer_id = NULL WHERE customer_id = ?`, id)
	res, err := s.db.Exec(`DELETE FROM customers WHERE id = ?`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if k, _ := res.RowsAffected(); k == 0 {
		fail(w, 404, "客户不存在")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func isNoRowsCustomer(err error) bool { return err == sql.ErrNoRows }

// nullableID 0 值转 NULL，便于外键列存"未选择"。
func nullableID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

// receivePayment 客户打款：按最早未结的结算单依次冲抵实收。
func (s *Server) receivePayment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	var in struct {
		Amount float64 `json:"amount"`
		Note   string  `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	in.Amount = round2(in.Amount)
	if in.Amount <= 0 {
		fail(w, 400, "收款金额必须大于 0")
		return
	}

	rows, err := s.db.Query(`SELECT id, settle_no, payable, received FROM settlements
		WHERE customer_id = ? AND COALESCE(settled,0) = 0 AND payable - received > 0.005
		ORDER BY settle_date, id`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	type openItem struct {
		ID       int64
		SettleNo string
		Left     float64
	}
	var open []openItem
	var totalLeft float64
	for rows.Next() {
		var it openItem
		var payable, received float64
		if err := rows.Scan(&it.ID, &it.SettleNo, &payable, &received); err != nil {
			rows.Close()
			fail(w, 500, err.Error())
			return
		}
		it.Left = round2(payable - received)
		totalLeft = round2(totalLeft + it.Left)
		open = append(open, it)
	}
	rows.Close()

	if len(open) == 0 || totalLeft <= 0 {
		fail(w, 400, "该客户没有未结款")
		return
	}
	if in.Amount > totalLeft+0.005 {
		fail(w, 400, "收款金额超过未结款（未结 ¥"+money2(totalLeft)+"）")
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	type alloc struct {
		SettleNo string  `json:"settle_no"`
		Amount   float64 `json:"amount"`
	}
	allocs := make([]alloc, 0)
	left := in.Amount
	for _, it := range open {
		if left <= 0.004 {
			break
		}
		take := it.Left
		if take > left {
			take = left
		}
		take = round2(take)
		if _, err := tx.Exec(`UPDATE settlements SET received = ROUND(received + ?, 2) WHERE id = ?`, take, it.ID); err != nil {
			fail(w, 500, err.Error())
			return
		}
		allocs = append(allocs, alloc{SettleNo: it.SettleNo, Amount: take})
		left = round2(left - take)
	}
	if _, err := tx.Exec(`INSERT INTO customer_payments (customer_id, amount, note, paid_at) VALUES (?,?,?,?)`,
		id, in.Amount, in.Note, now()); err != nil {
		fail(w, 500, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, err.Error())
		return
	}

	var payable, received float64
	var cnt int
	_ = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(payable),0), COALESCE(SUM(received),0)
		FROM settlements WHERE customer_id = ?`, id).Scan(&cnt, &payable, &received)

	writeJSON(w, 200, map[string]any{
		"status":      "ok",
		"amount":      in.Amount,
		"allocations": allocs,
		"payable":     round2(payable),
		"received":    round2(received),
		"outstanding": round2(payable - received),
	})
}

// listCustomerPayments 收款记录
func (s *Server) listCustomerPayments(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, 400, "无效 ID")
		return
	}
	rows, err := s.db.Query(`SELECT id, amount, note, paid_at FROM customer_payments
		WHERE customer_id = ? ORDER BY id DESC LIMIT 200`, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var pid int64
		var amount float64
		var note, paidAt string
		if err := rows.Scan(&pid, &amount, &note, &paidAt); err != nil {
			fail(w, 500, err.Error())
			return
		}
		out = append(out, map[string]any{"id": pid, "amount": amount, "note": note, "paid_at": paidAt})
	}
	writeJSON(w, 200, out)
}

// money2 金额格式化（两位小数，不带符号）
func money2(v float64) string {
	return strconv.FormatFloat(round2(v), 'f', 2, 64)
}
