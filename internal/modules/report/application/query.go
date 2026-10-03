package application

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"gorm.io/gorm"
)

type ReportQuery struct {
	db *gorm.DB
}

func NewReportQuery(db *gorm.DB) *ReportQuery {
	return &ReportQuery{db: db}
}

func (q *ReportQuery) SalesPerformance(ctx context.Context, from, to string) (*SalesPerformanceDTO, error) {
	dto := &SalesPerformanceDTO{From: from, To: to}

	base := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("sales_orders")).Where("deleted_at IS NULL")
	if from != "" {
		base = base.Where("order_date >= ?", from)
	}
	if to != "" {
		base = base.Where("order_date <= ?", to)
	}

	var totals struct {
		Count   int64
		Revenue float64
	}
	if err := base.Session(&gorm.Session{}).
		Select("COUNT(*) as count, COALESCE(SUM(total_amount),0) as revenue").
		Scan(&totals).Error; err != nil {
		return nil, err
	}
	dto.TotalOrders = totals.Count
	dto.TotalRevenue = totals.Revenue

	var byChannel []ChannelRevenue
	if err := base.Session(&gorm.Session{}).
		Select("channel, COALESCE(SUM(total_amount),0) as revenue").
		Group("channel").
		Scan(&byChannel).Error; err != nil {
		return nil, err
	}
	dto.RevenueByChannel = byChannel

	var byStatus []StatusRevenue
	if err := base.Session(&gorm.Session{}).
		Select("status, COALESCE(SUM(total_amount),0) as revenue").
		Group("status").
		Scan(&byStatus).Error; err != nil {
		return nil, err
	}
	dto.RevenueByStatus = byStatus

	var topCustomers []TopCustomer
	if err := base.Session(&gorm.Session{}).
		Select("customer_name, COALESCE(SUM(total_amount),0) as total_value").
		Group("customer_name").
		Order("total_value DESC").
		Limit(5).
		Scan(&topCustomers).Error; err != nil {
		return nil, err
	}
	dto.TopCustomers = topCustomers

	return dto, nil
}

func (q *ReportQuery) InventoryValuation(ctx context.Context) (*InventoryValuationDTO, error) {
	dto := &InventoryValuationDTO{}

	var total struct{ Value float64 }
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_levels sl")).
		Joins("JOIN products p ON p.id = sl.product_id AND p.deleted_at IS NULL").
		Where("sl.deleted_at IS NULL").
		Select("COALESCE(SUM(sl.quantity * p.cost_price),0) as value").
		Scan(&total).Error; err != nil {
		return nil, err
	}
	dto.TotalInventoryValue = total.Value

	var byWarehouse []WarehouseValue
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_levels sl")).
		Joins("JOIN products p ON p.id = sl.product_id AND p.deleted_at IS NULL").
		Joins("JOIN inventory_warehouses w ON w.id = sl.warehouse_id AND w.deleted_at IS NULL").
		Where("sl.deleted_at IS NULL").
		Select("w.id as warehouse_id, w.name as warehouse_name, COALESCE(SUM(sl.quantity * p.cost_price),0) as total_value").
		Group("w.id, w.name").
		Scan(&byWarehouse).Error; err != nil {
		return nil, err
	}
	dto.ByWarehouse = byWarehouse

	var byCategory []CategoryValue
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_levels sl")).
		Joins("JOIN products p ON p.id = sl.product_id AND p.deleted_at IS NULL").
		Where("sl.deleted_at IS NULL").
		Select("p.category as category, COALESCE(SUM(sl.quantity * p.cost_price),0) as total_value").
		Group("p.category").
		Scan(&byCategory).Error; err != nil {
		return nil, err
	}
	dto.ByCategory = byCategory

	var lowStock []LowStockItem
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_levels sl")).
		Joins("JOIN products p ON p.id = sl.product_id AND p.deleted_at IS NULL").
		Joins("JOIN inventory_warehouses w ON w.id = sl.warehouse_id AND w.deleted_at IS NULL").
		Where("sl.deleted_at IS NULL AND sl.quantity <= sl.min_stock").
		Select("p.sku as product_sku, p.name as product_name, w.name as warehouse_name, sl.quantity as quantity, sl.min_stock as min_stock").
		Order("sl.quantity ASC").
		Scan(&lowStock).Error; err != nil {
		return nil, err
	}
	dto.LowStockItems = lowStock

	return dto, nil
}

func (q *ReportQuery) HRMSummary(ctx context.Context) (*HRMSummaryDTO, error) {
	dto := &HRMSummaryDTO{}

	var total int64
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("employees")).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
		return nil, err
	}
	dto.TotalHeadcount = total

	var byDept []DepartmentHeadcount
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("employees")).
		Where("deleted_at IS NULL").
		Select("department, COUNT(*) as count").
		Group("department").
		Scan(&byDept).Error; err != nil {
		return nil, err
	}
	dto.ByDepartment = byDept

	var byStatus []StatusHeadcount
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("employees")).
		Where("deleted_at IS NULL").
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&byStatus).Error; err != nil {
		return nil, err
	}
	dto.ByStatus = byStatus

	monthPrefix := time.Now().Format("2006-01")
	var attCount int64
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("attendances")).
		Where("deleted_at IS NULL AND date LIKE ?", monthPrefix+"%").
		Count(&attCount).Error; err != nil {
		return nil, err
	}

	if attCount == 0 {
		dto.AttendanceRateNote = "No attendance records for the current month"
	} else {
		var presentCount int64
		if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("attendances")).
			Where("deleted_at IS NULL AND date LIKE ? AND clock_in IS NOT NULL AND clock_in <> ''", monthPrefix+"%").
			Count(&presentCount).Error; err != nil {
			return nil, err
		}
		dto.AttendanceRatePct = (float64(presentCount) / float64(attCount)) * 100
		dto.AttendanceRateNote = "Present (clocked in) records as a share of attendance records this month"
	}

	return dto, nil
}

func (q *ReportQuery) ExecutiveSummary(ctx context.Context) (*ExecutiveSummaryDTO, error) {
	dto := &ExecutiveSummaryDTO{
		FinanceNote: "Finance headline omitted: computing a revenue/expense figure requires resolving chart-of-accounts account types, which lives in finance's application layer and is not appropriate to duplicate here read-only. See finance/reports/profit-loss for that number.",
	}

	monthPrefix := time.Now().Format("2006-01")
	var sales struct{ Revenue float64 }
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("sales_orders")).
		Where("deleted_at IS NULL AND order_date LIKE ?", monthPrefix+"%").
		Select("COALESCE(SUM(total_amount),0) as revenue").
		Scan(&sales).Error; err != nil {
		return nil, err
	}
	dto.SalesRevenueThisMonth = sales.Revenue

	var inv struct{ Value float64 }
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_levels sl")).
		Joins("JOIN products p ON p.id = sl.product_id AND p.deleted_at IS NULL").
		Where("sl.deleted_at IS NULL").
		Select("COALESCE(SUM(sl.quantity * p.cost_price),0) as value").
		Scan(&inv).Error; err != nil {
		return nil, err
	}
	dto.TotalInventoryValue = inv.Value

	var headcount int64
	if err := tenantctx.Scope(ctx, q.db.WithContext(ctx).Table("employees")).Where("deleted_at IS NULL").Count(&headcount).Error; err != nil {
		return nil, err
	}
	dto.TotalHeadcount = headcount

	return dto, nil
}
