package aiassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/license"
	"github.com/divinecoid/one-backend/internal/modules/report/application"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const mcpDBKey = "aiassistant.mcpDB"

func licenseActive(c *fiber.Ctx, central *gorm.DB, companyID uuid.UUID) (bool, error) {
	return license.IsActive(c.UserContext(), central, companyID)
}

var supportedProtocols = map[string]bool{"2024-11-05": true, "2025-03-26": true, "2025-06-18": true}

const latestProtocol = "2025-06-18"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
	run         func(ctx context.Context, q *application.ReportQuery, args map[string]any) (any, error)
}

func dateProps() map[string]any {
	return map[string]any{
		"from": map[string]any{"type": "string", "description": "Start date YYYY-MM-DD (default: today, WIB)"},
		"to":   map[string]any{"type": "string", "description": "End date YYYY-MM-DD (default: today, WIB)"},
	}
}

func schema(props map[string]any) map[string]any {
	return map[string]any{"type": "object", "properties": props}
}

func str(args map[string]any, k string) string {
	s, _ := args[k].(string)
	return s
}

func today() string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func rangeArgs(args map[string]any) (string, string) {
	from, to := str(args, "from"), str(args, "to")
	if from == "" {
		from = today()
	}
	if to == "" {
		to = today()
	}
	return from, to
}

var readOnly = map[string]any{"readOnlyHint": true}

// All tools are read-only and go through the report application layer, never raw SQL.
var tools = []tool{
	{Name: "executive_summary", Description: "Ringkasan eksekutif perusahaan: indikator utama penjualan, keuangan, dan operasional.",
		InputSchema: schema(map[string]any{}), Annotations: readOnly,
		run: func(ctx context.Context, q *application.ReportQuery, _ map[string]any) (any, error) {
			return q.ExecutiveSummary(ctx)
		}},
	{Name: "inventory_valuation", Description: "Nilai persediaan per gudang dan daftar barang di bawah stok minimum.",
		InputSchema: schema(map[string]any{}), Annotations: readOnly,
		run: func(ctx context.Context, q *application.ReportQuery, _ map[string]any) (any, error) {
			return q.InventoryValuation(ctx)
		}},
	{Name: "sales_performance", Description: "Jumlah pesanan, omzet, dan omzet per kanal pada rentang tanggal.",
		InputSchema: schema(dateProps()), Annotations: readOnly,
		run: func(ctx context.Context, q *application.ReportQuery, a map[string]any) (any, error) {
			from, to := rangeArgs(a)
			return q.SalesPerformance(ctx, from, to)
		}},
	{Name: "sales_summary", Description: "Ringkasan penjualan pada rentang tanggal.",
		InputSchema: schema(dateProps()), Annotations: readOnly,
		run: func(ctx context.Context, q *application.ReportQuery, a map[string]any) (any, error) {
			from, to := rangeArgs(a)
			return q.SalesSummary(ctx, from, to)
		}},
	{Name: "sales_by_customer", Description: "Penjualan per pelanggan pada rentang tanggal.",
		InputSchema: schema(dateProps()), Annotations: readOnly,
		run: func(ctx context.Context, q *application.ReportQuery, a map[string]any) (any, error) {
			from, to := rangeArgs(a)
			return q.SalesByCustomer(ctx, from, to)
		}},
	{Name: "purchase_report", Description: "Laporan pembelian pada rentang tanggal.",
		InputSchema: schema(dateProps()), Annotations: readOnly,
		run: func(ctx context.Context, q *application.ReportQuery, a map[string]any) (any, error) {
			from, to := rangeArgs(a)
			return q.PurchaseReport(ctx, from, to)
		}},
	{Name: "receivable_balances", Description: "Saldo piutang pelanggan per tanggal tertentu.",
		InputSchema: schema(map[string]any{"asOf": map[string]any{"type": "string", "description": "Tanggal YYYY-MM-DD (default: hari ini)"}}),
		Annotations: readOnly,
		run: func(ctx context.Context, q *application.ReportQuery, a map[string]any) (any, error) {
			return q.PartyBalances(ctx, true, false, str(a, "asOf"))
		}},
	{Name: "payable_balances", Description: "Saldo hutang ke pemasok per tanggal tertentu.",
		InputSchema: schema(map[string]any{"asOf": map[string]any{"type": "string", "description": "Tanggal YYYY-MM-DD (default: hari ini)"}}),
		Annotations: readOnly,
		run: func(ctx context.Context, q *application.ReportQuery, a map[string]any) (any, error) {
			return q.PartyBalances(ctx, false, false, str(a, "asOf"))
		}},
}

func (h *Handler) MCP(c *fiber.Ctx) error {
	if _, err := h.authenticate(c); err != nil {
		return err
	}
	db := c.Locals(mcpDBKey).(*gorm.DB)

	var req rpcRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil || req.JSONRPC != "2.0" {
		return c.Status(fiber.StatusBadRequest).JSON(rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"),
			Error: &rpcError{Code: -32700, Message: "Parse error"}})
	}
	if len(req.ID) == 0 { // notification (e.g. notifications/initialized)
		return c.SendStatus(fiber.StatusAccepted)
	}
	reply := func(result any) error {
		return c.JSON(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
	}
	fail := func(code int, msg string) error {
		return c.JSON(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: code, Message: msg}})
	}

	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := latestProtocol
		if supportedProtocols[p.ProtocolVersion] {
			version = p.ProtocolVersion
		}
		return reply(fiber.Map{
			"protocolVersion": version,
			"capabilities":    fiber.Map{"tools": fiber.Map{"listChanged": false}},
			"serverInfo":      fiber.Map{"name": "manna-erp", "version": "1.0.0"},
			"instructions":    "Read-only access to Manna ERP reports. Amounts are in the company's base currency.",
		})
	case "ping":
		return reply(fiber.Map{})
	case "tools/list":
		return reply(fiber.Map{"tools": tools})
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fail(-32602, "Invalid params")
		}
		for _, t := range tools {
			if t.Name != p.Name {
				continue
			}
			ctx, cancel := context.WithTimeout(c.UserContext(), 30*time.Second)
			defer cancel()
			out, err := t.run(ctx, application.NewReportQuery(db), p.Arguments)
			if err != nil {
				return reply(fiber.Map{"isError": true, "content": []fiber.Map{{"type": "text", "text": fmt.Sprintf("Error: %v", err)}}})
			}
			body, _ := json.Marshal(out)
			return reply(fiber.Map{"content": []fiber.Map{{"type": "text", "text": string(body)}}})
		}
		return fail(-32602, "Unknown tool: "+p.Name)
	}
	return fail(-32601, "Method not found")
}

func (h *Handler) MCPMethodNotAllowed(c *fiber.Ctx) error {
	c.Set("Allow", "POST")
	return c.SendStatus(fiber.StatusMethodNotAllowed)
}
