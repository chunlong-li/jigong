package api

import (
	"database/sql"
	"net/http"
	"strings"
)

// listCustomers 客户列表，附带结算汇总（应付/实收/未结款）。
func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.category, c.contact, c.phone, c.address, c.remark, c.created_at,
		       COALESCE(st.cnt, 0), COALESCE(st.payable, 0), COALESCE(st.received, 0)
		FROM customers c
		LEFT JOIN (
			SELECT customer_id, COUNT(*) AS cnt,
			       SUM(payable) AS payable, SUM(received) AS received
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
			&x.CreatedAt, &x.SettleCount, &x.Payable, &x.Received); err != nil {
			fail(w, 500, err.Error())
			return
		}
		x.Outstanding = round2(x.Payable - x.Received)
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
