package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func ptr[T any](v T) *T { return &v }

var (
	styleTitle = &excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 20, Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	}
	styleMeta = &excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	}
	styleHead = &excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    thinBorder(),
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F2F2F2"}},
	}
	styleText = &excelize.Style{
		Font:      &excelize.Font{Size: 11, Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border:    thinBorder(),
	}
	styleCenter = &excelize.Style{
		Font:      &excelize.Font{Size: 11, Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    thinBorder(),
	}
	styleMoney = &excelize.Style{
		Font:         &excelize.Font{Size: 11, Family: "微软雅黑"},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder(),
		CustomNumFmt: ptr("#,##0.00"),
	}
	styleQty = &excelize.Style{
		Font:         &excelize.Font{Size: 11, Family: "微软雅黑"},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder(),
		CustomNumFmt: ptr("#,##0.###"),
	}
	styleTotalLabel = &excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:    thinBorder(),
	}
	styleTotalMoney = &excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 11, Family: "微软雅黑"},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder(),
		CustomNumFmt: ptr("#,##0.00"),
	}
	styleNote = &excelize.Style{
		Font:      &excelize.Font{Size: 9, Family: "微软雅黑", Color: "666666"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	}
)

func thinBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "BFBFBF", Style: 1},
		{Type: "right", Color: "BFBFBF", Style: 1},
		{Type: "top", Color: "BFBFBF", Style: 1},
		{Type: "bottom", Color: "BFBFBF", Style: 1},
	}
}

func applyStyle(f *excelize.File, sheet, from, to string, style *excelize.Style) int {
	id, err := f.NewStyle(style)
	if err != nil {
		return 0
	}
	_ = f.SetCellStyle(sheet, from, to, id)
	return id
}

func safeSheetName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Sheet"
	}
	r := []rune(name)
	if len(r) > 28 {
		r = r[:28]
	}
	return strings.Map(func(c rune) rune {
		switch c {
		case ':', '\\', '/', '?', '*', '[', ']':
			return '-'
		}
		return c
	}, string(r))
}

func moneyCell(f *excelize.File, sheet, cell string, v float64) {
	_ = f.SetCellValue(sheet, cell, v)
}

// buildQuotationXLSX 生成报价单。
func buildQuotationXLSX(q Quotation) ([]byte, error) {
	f := excelize.NewFile()
	sheet := safeSheetName(q.Title)
	f.SetSheetName("Sheet1", sheet)

	widths := map[string]float64{"A": 6, "B": 12, "C": 26, "D": 14, "E": 8, "F": 10, "G": 12, "H": 14}
	for col, w := range widths {
		_ = f.SetColWidth(sheet, col, col, w)
	}

	title := q.Title
	if title == "" {
		title = "报价单"
	}
	_ = f.MergeCell(sheet, "A1", "H1")
	_ = f.SetCellValue(sheet, "A1", strings.Join(strings.Split(title, ""), " "))
	_ = f.SetRowHeight(sheet, 1, 36)
	applyStyle(f, sheet, "A1", "H1", styleTitle)

	client := q.Client
	if client == "" {
		client = "-"
	}
	_ = f.SetCellValue(sheet, "A2", "报价编号："+dashIfEmpty(q.QuoteNo))
	_ = f.MergeCell(sheet, "A2", "C2")
	_ = f.SetCellValue(sheet, "D2", "报价日期："+dashIfEmpty(q.QuoteDate))
	_ = f.MergeCell(sheet, "D2", "E2")
	_ = f.SetCellValue(sheet, "F2", "客户："+client)
	_ = f.MergeCell(sheet, "F2", "H2")
	applyStyle(f, sheet, "A2", "H2", styleMeta)
	_ = f.SetRowHeight(sheet, 2, 22)

	projLine := "工程/项目：" + dashIfEmpty(q.ProjectName)
	if q.ContractType != "" {
		projLine += "　承包：" + q.ContractType
	}
	_ = f.SetCellValue(sheet, "A3", projLine)
	_ = f.MergeCell(sheet, "A3", "H3")
	applyStyle(f, sheet, "A3", "H3", styleMeta)
	_ = f.SetRowHeight(sheet, 3, 22)

	headers := []string{"序号", "房间", "项目名称", "尺寸", "单位", "数量", "单价(元)", "金额(元)"}
	headRow := 4
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headRow)
		_ = f.SetCellValue(sheet, cell, h)
	}
	applyStyle(f, sheet, "A4", "H4", styleHead)
	_ = f.SetRowHeight(sheet, headRow, 26)

	row := headRow + 1
	for i, it := range q.Items {
		_ = f.SetCellValue(sheet, cellName(1, row), i+1)
		_ = f.SetCellValue(sheet, cellName(2, row), it.Room)
		_ = f.SetCellValue(sheet, cellName(3, row), it.Name)
		_ = f.SetCellValue(sheet, cellName(4, row), it.Spec)
		_ = f.SetCellValue(sheet, cellName(5, row), it.Unit)
		moneyCell(f, sheet, cellName(6, row), it.Quantity)
		moneyCell(f, sheet, cellName(7, row), it.UnitPrice)
		moneyCell(f, sheet, cellName(8, row), it.Amount)
		applyStyle(f, sheet, cellName(1, row), cellName(2, row), styleCenter)
		applyStyle(f, sheet, cellName(3, row), cellName(5, row), styleText)
		applyStyle(f, sheet, cellName(6, row), cellName(6, row), styleQty)
		applyStyle(f, sheet, cellName(7, row), cellName(8, row), styleMoney)
		_ = f.SetRowHeight(sheet, row, 22)
		row++
	}

	totalStyle := &excelize.Style{
		Font:      &excelize.Font{Size: 11, Family: "微软雅黑"},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:    thinBorder(),
	}
	_ = f.MergeCell(sheet, cellName(1, row), cellName(7, row))
	_ = f.SetCellValue(sheet, cellName(1, row), "小计")
	applyStyle(f, sheet, cellName(1, row), cellName(7, row), totalStyle)
	moneyCell(f, sheet, cellName(8, row), q.Subtotal)
	applyStyle(f, sheet, cellName(8, row), cellName(8, row), styleMoney)
	row++

	if q.TaxRate > 0 {
		_ = f.MergeCell(sheet, cellName(1, row), cellName(7, row))
		_ = f.SetCellValue(sheet, cellName(1, row), fmt.Sprintf("税金（%g%%）", q.TaxRate))
		applyStyle(f, sheet, cellName(1, row), cellName(7, row), totalStyle)
		moneyCell(f, sheet, cellName(8, row), q.TaxAmount)
		applyStyle(f, sheet, cellName(8, row), cellName(8, row), styleMoney)
		row++
	}

	_ = f.MergeCell(sheet, cellName(1, row), cellName(7, row))
	_ = f.SetCellValue(sheet, cellName(1, row), "合计金额（大写）："+RMBUpper(q.Total))
	applyStyle(f, sheet, cellName(1, row), cellName(7, row), styleTotalLabel)
	moneyCell(f, sheet, cellName(8, row), q.Total)
	applyStyle(f, sheet, cellName(8, row), cellName(8, row), styleTotalMoney)
	_ = f.SetRowHeight(sheet, row, 26)
	row++

	_ = f.MergeCell(sheet, cellName(1, row), cellName(8, row))
	note := "备注：" + q.Remark
	if q.Remark == "" {
		note = "备注："
	}
	_ = f.SetCellValue(sheet, cellName(1, row), note)
	applyStyle(f, sheet, cellName(1, row), cellName(8, row), styleNote)
	_ = f.SetRowHeight(sheet, row, 22)
	row += 2

	_ = f.MergeCell(sheet, cellName(1, row), cellName(4, row))
	_ = f.SetCellValue(sheet, cellName(1, row), "报价单位（盖章）：")
	_ = f.MergeCell(sheet, cellName(5, row), cellName(8, row))
	_ = f.SetCellValue(sheet, cellName(5, row), "客户确认（签字）：")
	applyStyle(f, sheet, cellName(1, row), cellName(8, row), styleMeta)

	return xlsxBytes(f)
}

// buildSettlementXLSX 生成结算单（明细不再显示「类别」列）。
func buildSettlementXLSX(s Settlement) ([]byte, error) {
	f := excelize.NewFile()
	sheet := safeSheetName(s.Title)
	f.SetSheetName("Sheet1", sheet)

	widths := map[string]float64{"A": 6, "B": 12, "C": 28, "D": 14, "E": 8, "F": 10, "G": 12, "H": 14}
	for col, w := range widths {
		_ = f.SetColWidth(sheet, col, col, w)
	}

	title := s.Title
	if title == "" {
		title = "结算单"
	}
	_ = f.MergeCell(sheet, "A1", "H1")
	_ = f.SetCellValue(sheet, "A1", strings.Join(strings.Split(title, ""), " "))
	_ = f.SetRowHeight(sheet, 1, 36)
	applyStyle(f, sheet, "A1", "H1", styleTitle)

	_ = f.SetCellValue(sheet, "A2", "结算编号："+dashIfEmpty(s.SettleNo))
	_ = f.MergeCell(sheet, "A2", "C2")
	_ = f.SetCellValue(sheet, "D2", "结算日期："+dashIfEmpty(s.SettleDate))
	_ = f.MergeCell(sheet, "D2", "H2")
	applyStyle(f, sheet, "A2", "H2", styleMeta)
	_ = f.SetRowHeight(sheet, 2, 22)

	projLine := "项目：" + dashIfEmpty(s.ProjectName)
	if s.ContractType != "" {
		projLine += "　承包：" + s.ContractType
	}
	_ = f.SetCellValue(sheet, "A3", projLine)
	_ = f.MergeCell(sheet, "A3", "H3")
	applyStyle(f, sheet, "A3", "H3", styleMeta)
	_ = f.SetRowHeight(sheet, 3, 22)

	_ = f.SetCellValue(sheet, "A4", "收款方："+dashIfEmpty(s.Payee))
	_ = f.MergeCell(sheet, "A4", "C4")
	period := s.PeriodStart + " 至 " + s.PeriodEnd
	if s.PeriodStart == "" && s.PeriodEnd == "" {
		period = "-"
	}
	_ = f.SetCellValue(sheet, "D4", "结算期间："+period+"　制单日期："+time.Now().Format("2006-01-02"))
	_ = f.MergeCell(sheet, "D4", "H4")
	applyStyle(f, sheet, "A4", "H4", styleMeta)
	_ = f.SetRowHeight(sheet, 4, 22)

	headers := []string{"序号", "房间", "名称", "尺寸", "单位", "数量", "单价(元)", "金额(元)"}
	headRow := 5
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headRow)
		_ = f.SetCellValue(sheet, cell, h)
	}
	applyStyle(f, sheet, "A5", "H5", styleHead)
	_ = f.SetRowHeight(sheet, headRow, 26)

	row := headRow + 1
	for i, it := range s.Items {
		_ = f.SetCellValue(sheet, cellName(1, row), i+1)
		_ = f.SetCellValue(sheet, cellName(2, row), it.Room)
		_ = f.SetCellValue(sheet, cellName(3, row), it.Name)
		_ = f.SetCellValue(sheet, cellName(4, row), it.Spec)
		_ = f.SetCellValue(sheet, cellName(5, row), it.Unit)
		moneyCell(f, sheet, cellName(6, row), it.Quantity)
		moneyCell(f, sheet, cellName(7, row), it.UnitPrice)
		moneyCell(f, sheet, cellName(8, row), it.Amount)
		applyStyle(f, sheet, cellName(1, row), cellName(2, row), styleCenter)
		applyStyle(f, sheet, cellName(3, row), cellName(5, row), styleText)
		applyStyle(f, sheet, cellName(6, row), cellName(6, row), styleQty)
		applyStyle(f, sheet, cellName(7, row), cellName(8, row), styleMoney)
		_ = f.SetRowHeight(sheet, row, 22)
		row++
	}

	type total struct {
		label string
		value float64
		bold  bool
	}
	rows := []total{
		{"明细合计", round2(s.WorkAmount + s.MaterialAmount), false},
		{"扣款/借支", s.Deduction, false},
		{"应付金额", s.Payable, true},
	}
	for _, t := range rows {
		label := t.label
		if t.bold {
			label = "应付金额（大写）：" + RMBUpper(t.value)
		}
		_ = f.MergeCell(sheet, cellName(1, row), cellName(7, row))
		_ = f.SetCellValue(sheet, cellName(1, row), label)
		st := &excelize.Style{
			Font:      &excelize.Font{Size: 11, Family: "微软雅黑", Bold: t.bold},
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			Border:    thinBorder(),
		}
		applyStyle(f, sheet, cellName(1, row), cellName(7, row), st)
		moneyCell(f, sheet, cellName(8, row), t.value)
		ms := styleMoney
		if t.bold {
			ms = styleTotalMoney
		}
		applyStyle(f, sheet, cellName(8, row), cellName(8, row), ms)
		_ = f.SetRowHeight(sheet, row, 24)
		row++
	}

	row++
	_ = f.MergeCell(sheet, cellName(1, row), cellName(8, row))
	note := "备注：" + s.Remark
	if s.Remark == "" {
		note = "备注："
	}
	_ = f.SetCellValue(sheet, cellName(1, row), note)
	applyStyle(f, sheet, cellName(1, row), cellName(8, row), styleNote)
	row += 2

	_ = f.MergeCell(sheet, cellName(1, row), cellName(4, row))
	_ = f.SetCellValue(sheet, cellName(1, row), "结算单位（盖章）：")
	_ = f.MergeCell(sheet, cellName(5, row), cellName(8, row))
	_ = f.SetCellValue(sheet, cellName(5, row), "收款人（签字）：")
	applyStyle(f, sheet, cellName(1, row), cellName(8, row), styleMeta)

	return xlsxBytes(f)
}

func xlsxBytes(f *excelize.File) ([]byte, error) {
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func dashIfEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
