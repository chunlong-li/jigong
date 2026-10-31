package api

// 数据传输结构。金额字段单位为元，保留两位小数。

type Project struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	SiteAddress    string  `json:"site_address"`
	Client         string  `json:"client"`
	Contact        string  `json:"contact"`
	ContractAmount float64 `json:"contract_amount"`
	StartDate      string  `json:"start_date"`
	Status         string  `json:"status"`
	ContractType   string  `json:"contract_type"`
	Remark         string  `json:"remark"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type Worker struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone"`
	IDCard      string  `json:"id_card"`
	InternalRate float64 `json:"internal_rate"`
	ExternalRate float64 `json:"external_rate"`
	OvertimeRate float64 `json:"overtime_rate"`
	Remark      string  `json:"remark"`
	CreatedAt   string  `json:"created_at"`
}

type WorkLog struct {
	ID         int64   `json:"id"`
	ProjectID  int64   `json:"project_id"`
	ProjectName string `json:"project_name"`
	WorkerID   int64   `json:"worker_id"`
	WorkerName string  `json:"worker_name"`
	WorkDate   string  `json:"work_date"`
	Mode       string  `json:"mode"`
	DayType    string  `json:"day_type"`
	Quantity   float64 `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	Amount     float64 `json:"amount"`
	Content    string  `json:"content"`
	Remark     string  `json:"remark"`
	SettledAt  string  `json:"settled_at"`
	CreatedAt  string  `json:"created_at"`
}

type PriceCategory struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type PriceItem struct {
	ID        int64   `json:"id"`
	Kind      string  `json:"kind"`
	Category  string  `json:"category"`
	Room      string  `json:"room"`
	Name      string  `json:"name"`
	Spec      string  `json:"spec"`
	Unit      string  `json:"unit"`
	Price     float64 `json:"price"`
	ClientPrice float64 `json:"client_price"`
	Remark    string  `json:"remark"`
	CreatedAt string  `json:"created_at"`
}

type Material struct {
	ID          int64   `json:"id"`
	ProjectID   int64   `json:"project_id"`
	ProjectName string  `json:"project_name"`
	InDate      string  `json:"in_date"`
	Name        string  `json:"name"`
	Spec        string  `json:"spec"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
	Supplier    string  `json:"supplier"`
	Remark      string  `json:"remark"`
	CreatedAt   string  `json:"created_at"`
}

type QuotationItem struct {
	ID         int64   `json:"id"`
	Room       string  `json:"room"`
	Name       string  `json:"name"`
	Spec       string  `json:"spec"`
	Unit       string  `json:"unit"`
	Quantity   float64 `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	Amount     float64 `json:"amount"`
	SortOrder  int     `json:"sort_order"`
}

type Quotation struct {
	ID         int64           `json:"id"`
	ProjectID    int64          `json:"project_id"`
	ProjectName  string         `json:"project_name"`
	Room         string         `json:"room"`
	ContractType string         `json:"contract_type"`
	QuoteType    string         `json:"quote_type"`
	QuoteNo    string          `json:"quote_no"`
	QuoteDate  string          `json:"quote_date"`
	Client     string          `json:"client"`
	Title      string          `json:"title"`
	TaxRate    float64         `json:"tax_rate"`
	Subtotal   float64         `json:"subtotal"`
	TaxAmount  float64         `json:"tax_amount"`
	Total      float64         `json:"total"`
	Remark     string          `json:"remark"`
	CreatedAt  string          `json:"created_at"`
	Items      []QuotationItem `json:"items"`
}

type SettlementItem struct {
	ID        int64   `json:"id"`
	Category  string  `json:"category"`
	Room      string  `json:"room"`
	Name      string  `json:"name"`
	Spec      string  `json:"spec"`
	Unit      string  `json:"unit"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Amount    float64 `json:"amount"`
	SortOrder int     `json:"sort_order"`
}

type Settlement struct {
	ID             int64            `json:"id"`
	ProjectID      int64            `json:"project_id"`
	ProjectName    string           `json:"project_name"`
	Room           string           `json:"room"`
	SettleNo       string           `json:"settle_no"`
	SettleDate     string           `json:"settle_date"`
	PeriodStart    string           `json:"period_start"`
	PeriodEnd      string           `json:"period_end"`
	Payee          string           `json:"payee"`
	ContractType   string           `json:"contract_type"`
	Title          string           `json:"title"`
	WorkAmount     float64          `json:"work_amount"`
	MaterialAmount float64          `json:"material_amount"`
	Deduction      float64          `json:"deduction"`
	Payable        float64          `json:"payable"`
	Received       float64          `json:"received"`
	Remark         string           `json:"remark"`
	CreatedAt      string           `json:"created_at"`
	Items          []SettlementItem `json:"items"`
}

// WorkLogGroup 以「项目 + 工人」聚合后的记工统计。
type WorkLogGroup struct {
	Month       string  `json:"month"`
	ProjectID   int64   `json:"project_id"`
	ProjectName string  `json:"project_name"`
	WorkerID    int64   `json:"worker_id"`
	WorkerName  string  `json:"worker_name"`
	Days        float64 `json:"days"`
	OvertimeHours float64 `json:"overtime_hours"`
	OvertimeCount int     `json:"overtime_count"`
	HourQty     float64 `json:"hour_qty"`
	PieceQty    float64 `json:"piece_qty"`
	Amount      float64 `json:"amount"`
	RecordCount int     `json:"record_count"`
	LastDate    string  `json:"last_date"`
}

// WageSettlement 工资结算归档。
type WageSettlement struct {
	ID            int64   `json:"id"`
	ProjectID     int64   `json:"project_id"`
	ProjectName   string  `json:"project_name"`
	WorkerID      int64   `json:"worker_id"`
	WorkerName    string  `json:"worker_name"`
	PeriodStart   string  `json:"period_start"`
	PeriodEnd     string  `json:"period_end"`
	Days          float64 `json:"days"`
	OvertimeHours float64 `json:"overtime_hours"`
	OvertimeCount int     `json:"overtime_count"`
	HourQty       float64 `json:"hour_qty"`
	PieceQty      float64 `json:"piece_qty"`
	Amount        float64 `json:"amount"`
	Remark        string  `json:"remark"`
	CreatedAt     string  `json:"created_at"`
}

type Summary struct {
	ProjectCount  int     `json:"project_count"`
	WorkerCount   int     `json:"worker_count"`
	WorkAmount    float64 `json:"work_amount"`
	MaterialAmount float64 `json:"material_amount"`
	MonthWork     float64 `json:"month_work"`
	MonthMaterial float64 `json:"month_material"`
}
