package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open 打开（必要时创建）SQLite 数据库。
func Open(dir string) (*sql.DB, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dir, "jigong.db")
	// 兼容旧文件名：gongji.db → jigong.db
	if _, err := os.Stat(path); os.IsNotExist(err) {
		old := filepath.Join(dir, "gongji.db")
		if _, e := os.Stat(old); e == nil {
			_ = os.Rename(old, path)
			_ = os.Rename(old+"-wal", path+"-wal")
			_ = os.Rename(old+"-shm", path+"-shm")
		}
	}
	dsn := "file:" + path +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=synchronous(NORMAL)"

	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// modernc.org/sqlite 单写者模型：限制连接数可避免 SQLITE_BUSY。
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)
	if err := d.Ping(); err != nil {
		return nil, err
	}
	if err := Migrate(d); err != nil {
		return nil, err
	}
	return d, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	username      TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	last_login    TEXT NOT NULL DEFAULT '',
	created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	name            TEXT NOT NULL,
	site_address    TEXT NOT NULL DEFAULT '',
	client          TEXT NOT NULL DEFAULT '',
	contact         TEXT NOT NULL DEFAULT '',
	contract_amount REAL NOT NULL DEFAULT 0,
	start_date      TEXT NOT NULL DEFAULT '',
	status          TEXT NOT NULL DEFAULT 'active',
	contract_type   TEXT NOT NULL DEFAULT '包工包料',
	remark          TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workers (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	name         TEXT NOT NULL,
	phone        TEXT NOT NULL DEFAULT '',
	id_card      TEXT NOT NULL DEFAULT '',
	internal_rate REAL NOT NULL DEFAULT 0,
	external_rate REAL NOT NULL DEFAULT 0,
	overtime_rate REAL NOT NULL DEFAULT 0,
	remark       TEXT NOT NULL DEFAULT '',
	created_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS worklogs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	worker_id  INTEGER NOT NULL REFERENCES workers(id) ON DELETE RESTRICT,
	work_date  TEXT NOT NULL,
	mode       TEXT NOT NULL DEFAULT 'day',
	day_type   TEXT NOT NULL DEFAULT '',
	quantity   REAL NOT NULL DEFAULT 0,
	unit_price REAL NOT NULL DEFAULT 0,
	amount     REAL NOT NULL DEFAULT 0,
	content    TEXT NOT NULL DEFAULT '',
	remark     TEXT NOT NULL DEFAULT '',
	settled_at TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_worklogs_project_date ON worklogs(project_id, work_date);
CREATE INDEX IF NOT EXISTS idx_worklogs_worker ON worklogs(worker_id);

CREATE TABLE IF NOT EXISTS wage_settlements (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id     INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	worker_id      INTEGER NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
	period_start   TEXT NOT NULL DEFAULT '',
	period_end     TEXT NOT NULL DEFAULT '',
	days           REAL NOT NULL DEFAULT 0,
	overtime_hours REAL NOT NULL DEFAULT 0,
	overtime_count INTEGER NOT NULL DEFAULT 0,
	hour_qty       REAL NOT NULL DEFAULT 0,
	piece_qty      REAL NOT NULL DEFAULT 0,
	amount         REAL NOT NULL DEFAULT 0,
	remark         TEXT NOT NULL DEFAULT '',
	created_at     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_wage_settle_worker ON wage_settlements(worker_id, project_id);

CREATE TABLE IF NOT EXISTS price_categories (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	kind       TEXT NOT NULL DEFAULT 'material',
	name       TEXT NOT NULL,
	created_at TEXT NOT NULL,
	UNIQUE(kind, name)
);

CREATE TABLE IF NOT EXISTS price_items (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	kind       TEXT NOT NULL DEFAULT 'material',
	category   TEXT NOT NULL DEFAULT '',
	name       TEXT NOT NULL,
	spec       TEXT NOT NULL DEFAULT '',
	unit       TEXT NOT NULL DEFAULT '',
	price      REAL NOT NULL DEFAULT 0,
	client_price REAL NOT NULL DEFAULT 0,
	remark     TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_price_items_kind ON price_items(kind);

CREATE TABLE IF NOT EXISTS materials (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	in_date    TEXT NOT NULL,
	name       TEXT NOT NULL,
	spec       TEXT NOT NULL DEFAULT '',
	unit       TEXT NOT NULL DEFAULT '',
	quantity   REAL NOT NULL DEFAULT 0,
	unit_price REAL NOT NULL DEFAULT 0,
	amount     REAL NOT NULL DEFAULT 0,
	supplier   TEXT NOT NULL DEFAULT '',
	remark     TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_materials_project_date ON materials(project_id, in_date);

CREATE TABLE IF NOT EXISTS quotations (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id    INTEGER REFERENCES projects(id) ON DELETE CASCADE,
	project_name  TEXT NOT NULL DEFAULT '',
	contract_type TEXT NOT NULL DEFAULT '包工包料',
	quote_no    TEXT NOT NULL DEFAULT '',
	quote_date  TEXT NOT NULL,
	client      TEXT NOT NULL DEFAULT '',
	title       TEXT NOT NULL DEFAULT '报价单',
	tax_rate    REAL NOT NULL DEFAULT 0,
	subtotal    REAL NOT NULL DEFAULT 0,
	tax_amount  REAL NOT NULL DEFAULT 0,
	total       REAL NOT NULL DEFAULT 0,
	remark      TEXT NOT NULL DEFAULT '',
	created_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS quotation_items (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	quotation_id INTEGER NOT NULL REFERENCES quotations(id) ON DELETE CASCADE,
	room         TEXT NOT NULL DEFAULT '',
	name         TEXT NOT NULL,
	spec         TEXT NOT NULL DEFAULT '',
	unit         TEXT NOT NULL DEFAULT '',
	quantity     REAL NOT NULL DEFAULT 0,
	unit_price   REAL NOT NULL DEFAULT 0,
	amount       REAL NOT NULL DEFAULT 0,
	sort_order   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_quotation_items_q ON quotation_items(quotation_id);

CREATE TABLE IF NOT EXISTS settlements (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id      INTEGER REFERENCES projects(id) ON DELETE CASCADE,
	project_name    TEXT NOT NULL DEFAULT '',
	contract_type   TEXT NOT NULL DEFAULT '包工包料',
	settle_no       TEXT NOT NULL DEFAULT '',
	settle_date     TEXT NOT NULL,
	period_start    TEXT NOT NULL DEFAULT '',
	period_end      TEXT NOT NULL DEFAULT '',
	payee           TEXT NOT NULL DEFAULT '',
	title           TEXT NOT NULL DEFAULT '结算单',
	work_amount     REAL NOT NULL DEFAULT 0,
	material_amount REAL NOT NULL DEFAULT 0,
	deduction       REAL NOT NULL DEFAULT 0,
	payable         REAL NOT NULL DEFAULT 0,
	remark          TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settlement_items (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	settlement_id INTEGER NOT NULL REFERENCES settlements(id) ON DELETE CASCADE,
	category      TEXT NOT NULL DEFAULT '记工',
	room          TEXT NOT NULL DEFAULT '',
	name          TEXT NOT NULL,
	spec          TEXT NOT NULL DEFAULT '',
	unit          TEXT NOT NULL DEFAULT '',
	quantity      REAL NOT NULL DEFAULT 0,
	unit_price    REAL NOT NULL DEFAULT 0,
	amount        REAL NOT NULL DEFAULT 0,
	sort_order    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_settlement_items_s ON settlement_items(settlement_id);
`

// Migrate 建表（幂等），并把老库的 default_rate 迁到 internal_rate。
func Migrate(d *sql.DB) error {
	if _, err := d.Exec(schema); err != nil {
		return err
	}
	if err := migrateWorkers(d); err != nil {
		return err
	}
	if err := migrateQuotations(d); err != nil {
		return err
	}
	if err := migrateSettlements(d); err != nil {
		return err
	}
	// 分类表种子数据
	if _, err := d.Exec(`INSERT OR IGNORE INTO price_categories (kind, name, created_at)
		SELECT kind, category, datetime('now','localtime') FROM price_items
		WHERE category <> '' GROUP BY kind, category`); err != nil {
		return err
	}
	// 施工价格拆成「包工包料 / 包工」两类，老数据归到包工包料
	if _, err := d.Exec(`UPDATE price_items SET kind='labor_all' WHERE kind='labor'`); err != nil {
		return err
	}
	pcols2, err := tableColumns(d, "price_items")
	if err != nil {
		return err
	}
	qcols, err := tableColumns(d, "quotations")
	if err != nil {
		return err
	}
	qicols, err := tableColumns(d, "quotation_items")
	if err != nil {
		return err
	}
	if !qicols["room"] {
		if _, err := d.Exec(`ALTER TABLE quotation_items ADD COLUMN room TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !qcols["room"] {
		if _, err := d.Exec(`ALTER TABLE quotations ADD COLUMN room TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !qcols["quote_type"] {
		if _, err := d.Exec(`ALTER TABLE quotations ADD COLUMN quote_type TEXT NOT NULL DEFAULT '客户报价'`); err != nil {
			return err
		}
	}
	ucols, err := tableColumns(d, "users")
	if err != nil {
		return err
	}
	if !ucols["last_login"] {
		if _, err := d.Exec(`ALTER TABLE users ADD COLUMN last_login TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
		// 老库已在用：视为登录过，避免升级后又弹一次默认账号提示
		if _, err := d.Exec(`UPDATE users SET last_login = created_at WHERE last_login = ''`); err != nil {
			return err
		}
	}
	if !pcols2["client_price"] {
		if _, err := d.Exec(`ALTER TABLE price_items ADD COLUMN client_price REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !pcols2["category"] {
		if _, err := d.Exec(`ALTER TABLE price_items ADD COLUMN category TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	return nil
}

func tableColumns(d *sql.DB, table string) (map[string]bool, error) {
	rows, err := d.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, nil
}

// migrateQuotations 让报价单不再强制关联已有项目（未中标也能先报价）。
func migrateQuotations(d *sql.DB) error {
	cols, err := tableColumns(d, "quotations")
	if err != nil {
		return err
	}
	if cols["project_name"] {
		return nil
	}
	ctx := context.Background()
	conn, err := d.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmts := []string{
		`CREATE TABLE quotations_new (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id    INTEGER REFERENCES projects(id) ON DELETE CASCADE,
			project_name  TEXT NOT NULL DEFAULT '',
			room          TEXT NOT NULL DEFAULT '',
			contract_type TEXT NOT NULL DEFAULT '包工包料',
			quote_type    TEXT NOT NULL DEFAULT '客户报价',
			quote_no      TEXT NOT NULL DEFAULT '',
			quote_date    TEXT NOT NULL,
			client        TEXT NOT NULL DEFAULT '',
			title         TEXT NOT NULL DEFAULT '报价单',
			tax_rate      REAL NOT NULL DEFAULT 0,
			subtotal      REAL NOT NULL DEFAULT 0,
			tax_amount    REAL NOT NULL DEFAULT 0,
			total         REAL NOT NULL DEFAULT 0,
			remark        TEXT NOT NULL DEFAULT '',
			created_at    TEXT NOT NULL
		)`,
		`INSERT INTO quotations_new
			(id, project_id, quote_no, quote_date, client, title, tax_rate, subtotal, tax_amount, total, remark, created_at)
		 SELECT id, project_id, quote_no, quote_date, client, title, tax_rate, subtotal, tax_amount, total, remark, created_at
		 FROM quotations`,
		`DROP TABLE quotations`,
		`ALTER TABLE quotations_new RENAME TO quotations`,
	}
	for _, q := range stmts {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, `PRAGMA foreign_keys=ON`)
	return err
}

// migrateSettlements 让结算单支持「自定义项目」（不必须关联已有项目）。
func migrateSettlements(d *sql.DB) error {
	cols, err := tableColumns(d, "settlements")
	if err != nil {
		return err
	}
	if cols["project_name"] {
		return nil
	}
	ctx := context.Background()
	conn, err := d.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmts := []string{
		`CREATE TABLE settlements_new (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id      INTEGER REFERENCES projects(id) ON DELETE CASCADE,
			project_name    TEXT NOT NULL DEFAULT '',
			room            TEXT NOT NULL DEFAULT '',
			contract_type   TEXT NOT NULL DEFAULT '包工包料',
			settle_no       TEXT NOT NULL DEFAULT '',
			settle_date     TEXT NOT NULL,
			period_start    TEXT NOT NULL DEFAULT '',
			period_end      TEXT NOT NULL DEFAULT '',
			payee           TEXT NOT NULL DEFAULT '',
			title           TEXT NOT NULL DEFAULT '结算单',
			work_amount     REAL NOT NULL DEFAULT 0,
			material_amount REAL NOT NULL DEFAULT 0,
			deduction       REAL NOT NULL DEFAULT 0,
			payable         REAL NOT NULL DEFAULT 0,
			received        REAL NOT NULL DEFAULT 0,
			remark          TEXT NOT NULL DEFAULT '',
			created_at      TEXT NOT NULL
		)`,
		`INSERT INTO settlements_new
			(id, project_id, settle_no, settle_date, period_start, period_end, payee, title,
			 work_amount, material_amount, deduction, payable, remark, created_at)
		 SELECT id, project_id, settle_no, settle_date, period_start, period_end, payee, title,
			 work_amount, material_amount, deduction, payable, remark, created_at
		 FROM settlements`,
		`DROP TABLE settlements`,
		`ALTER TABLE settlements_new RENAME TO settlements`,
	}
	for _, q := range stmts {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, `PRAGMA foreign_keys=ON`)
	return err
}

func migrateWorkers(d *sql.DB) error {
	cols, err := tableColumns(d, "workers")
	if err != nil {
		return err
	}
	if cols["default_rate"] && !cols["internal_rate"] {
		if _, err := d.Exec(`ALTER TABLE workers RENAME COLUMN default_rate TO internal_rate`); err != nil {
			return err
		}
		cols["internal_rate"] = true
	}
	if !cols["internal_rate"] {
		if _, err := d.Exec(`ALTER TABLE workers ADD COLUMN internal_rate REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !cols["external_rate"] {
		if _, err := d.Exec(`ALTER TABLE workers ADD COLUMN external_rate REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !cols["overtime_rate"] {
		if _, err := d.Exec(`ALTER TABLE workers ADD COLUMN overtime_rate REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	wcols, err := tableColumns(d, "worklogs")
	if err != nil {
		return err
	}
	if !wcols["day_type"] {
		if _, err := d.Exec(`ALTER TABLE worklogs ADD COLUMN day_type TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	pcols, err := tableColumns(d, "projects")
	if err != nil {
		return err
	}
	stcols, err := tableColumns(d, "settlements")
	if err != nil {
		return err
	}
	if !stcols["room"] {
		if _, err := d.Exec(`ALTER TABLE settlements ADD COLUMN room TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !stcols["received"] {
		if _, err := d.Exec(`ALTER TABLE settlements ADD COLUMN received REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := d.Exec(`UPDATE settlements SET received = payable WHERE received = 0`); err != nil {
			return err
		}
	}
	scols, err := tableColumns(d, "settlement_items")
	if err != nil {
		return err
	}
	if !scols["room"] {
		if _, err := d.Exec(`ALTER TABLE settlement_items ADD COLUMN room TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !scols["spec"] {
		if _, err := d.Exec(`ALTER TABLE settlement_items ADD COLUMN spec TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !scols["unit"] {
		if _, err := d.Exec(`ALTER TABLE settlement_items ADD COLUMN unit TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !pcols["contract_type"] {
		if _, err := d.Exec(`ALTER TABLE projects ADD COLUMN contract_type TEXT NOT NULL DEFAULT '包工包料'`); err != nil {
			return err
		}
	}
	if !wcols["settled_at"] {
		if _, err := d.Exec(`ALTER TABLE worklogs ADD COLUMN settled_at TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	return nil
}
