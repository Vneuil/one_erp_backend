package application

// Sales performance

type ChannelRevenue struct {
	Channel string  `json:"channel"`
	Revenue float64 `json:"revenue"`
}

type StatusRevenue struct {
	Status  string  `json:"status"`
	Revenue float64 `json:"revenue"`
}

type TopCustomer struct {
	CustomerName string  `json:"customerName"`
	TotalValue   float64 `json:"totalValue"`
}

type SalesPerformanceDTO struct {
	From             string           `json:"from,omitempty"`
	To               string           `json:"to,omitempty"`
	TotalOrders      int64            `json:"totalOrders"`
	TotalRevenue     float64          `json:"totalRevenue"`
	RevenueByChannel []ChannelRevenue `json:"revenueByChannel"`
	RevenueByStatus  []StatusRevenue  `json:"revenueByStatus"`
	TopCustomers     []TopCustomer    `json:"topCustomers"`
}

// Inventory valuation

type WarehouseValue struct {
	WarehouseID   string  `json:"warehouseId"`
	WarehouseName string  `json:"warehouseName"`
	TotalValue    float64 `json:"totalValue"`
}

type CategoryValue struct {
	Category   string  `json:"category"`
	TotalValue float64 `json:"totalValue"`
}

type LowStockItem struct {
	ProductSKU    string `json:"productSku"`
	ProductName   string `json:"productName"`
	WarehouseName string `json:"warehouseName"`
	Quantity      int    `json:"quantity"`
	MinStock      int    `json:"minStock"`
}

type InventoryValuationDTO struct {
	TotalInventoryValue float64          `json:"totalInventoryValue"`
	ByWarehouse         []WarehouseValue `json:"byWarehouse"`
	ByCategory          []CategoryValue  `json:"byCategory"`
	LowStockItems       []LowStockItem   `json:"lowStockItems"`
}

// HRM summary

type DepartmentHeadcount struct {
	Department string `json:"department"`
	Count      int64  `json:"count"`
}

type StatusHeadcount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type HRMSummaryDTO struct {
	TotalHeadcount     int64                 `json:"totalHeadcount"`
	ByDepartment       []DepartmentHeadcount `json:"byDepartment"`
	ByStatus           []StatusHeadcount     `json:"byStatus"`
	AttendanceRatePct  float64               `json:"attendanceRatePct"`
	AttendanceRateNote string                `json:"attendanceRateNote,omitempty"`
}

// Executive summary

type ExecutiveSummaryDTO struct {
	SalesRevenueThisMonth float64 `json:"salesRevenueThisMonth"`
	TotalInventoryValue   float64 `json:"totalInventoryValue"`
	TotalHeadcount        int64   `json:"totalHeadcount"`
	FinanceNote           string  `json:"financeNote"`
}
