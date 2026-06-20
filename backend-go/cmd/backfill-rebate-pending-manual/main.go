package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/db"
)

type pendingManualRow struct {
	OrderID                    string  `json:"order_id"`
	Principal                  float64 `json:"principal"`
	HasPrincipal               bool    `json:"has_principal"`
	SubscribeReceivable        float64 `json:"subscribe_receivable"`
	HasSubscribeReceivable     bool    `json:"has_subscribe_receivable"`
	ManagementFeeReceived      float64 `json:"management_fee_received"`
	HasManagementFeeReceived   bool    `json:"has_management_fee_received"`
	PerformanceFeeReceivable   float64 `json:"performance_fee_receivable"`
	HasPerformanceFeeReceivable bool   `json:"has_performance_fee_receivable"`
	SubscribeFeeRatio          float64 `json:"subscribe_fee_ratio"`
	HasSubscribeFeeRatio       bool    `json:"has_subscribe_fee_ratio"`
	ManagementFeeRatio         float64 `json:"management_fee_ratio"`
	HasManagementFeeRatio      bool    `json:"has_management_fee_ratio"`
	PerformanceFeeRatio        float64 `json:"performance_fee_ratio"`
	HasPerformanceFeeRatio     bool    `json:"has_performance_fee_ratio"`
	TaxSubscribeRatio          float64 `json:"tax_subscribe_ratio"`
	HasTaxSubscribeRatio       bool    `json:"has_tax_subscribe_ratio"`
	TaxManagementRatio         float64 `json:"tax_management_ratio"`
	HasTaxManagementRatio      bool    `json:"has_tax_management_ratio"`
	TaxPerformanceRatio        float64 `json:"tax_performance_ratio"`
	HasTaxPerformanceRatio     bool    `json:"has_tax_performance_ratio"`
	ExpectedSubscribe          float64 `json:"expected_subscribe"`
	HasExpectedSubscribe       bool    `json:"has_expected_subscribe"`
	ExpectedManagement         float64 `json:"expected_management"`
	HasExpectedManagement      bool    `json:"has_expected_management"`
	ExpectedPerformance        float64 `json:"expected_performance"`
	HasExpectedPerformance     bool    `json:"has_expected_performance"`
	ReturnedSubscribe          float64 `json:"returned_subscribe"`
	HasReturnedSubscribe       bool    `json:"has_returned_subscribe"`
	ReturnedManagement         float64 `json:"returned_management"`
	HasReturnedManagement      bool    `json:"has_returned_management"`
	ReturnedPerformance        float64 `json:"returned_performance"`
	HasReturnedPerformance     bool    `json:"has_returned_performance"`
	OutstandingSubscribe       float64 `json:"outstanding_subscribe"`
	HasOutstandingSubscribe    bool    `json:"has_outstanding_subscribe"`
	OutstandingManagement      float64 `json:"outstanding_management"`
	HasOutstandingManagement   bool    `json:"has_outstanding_management"`
	OutstandingPerformance     float64 `json:"outstanding_performance"`
	HasOutstandingPerformance  bool    `json:"has_outstanding_performance"`
	IsReturnable               string  `json:"is_returnable"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: backfill-rebate-pending-manual <rows.json>")
	}
	cfg := config.Load()
	store, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if _, err := store.DB.Exec(`
		CREATE TABLE IF NOT EXISTS rebate_pending_manual (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id TEXT NOT NULL UNIQUE,
			principal REAL,
			subscribe_receivable REAL,
			management_fee_received REAL,
			performance_fee_receivable REAL,
			subscribe_fee_ratio REAL,
			management_fee_ratio REAL,
			performance_fee_ratio REAL,
			tax_subscribe_ratio REAL,
			tax_management_ratio REAL,
			tax_performance_ratio REAL,
			expected_subscribe REAL,
			expected_management REAL,
			expected_performance REAL,
			returned_subscribe REAL,
			returned_management REAL,
			returned_performance REAL,
			outstanding_subscribe REAL,
			outstanding_management REAL,
			outstanding_performance REAL,
			is_returnable TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)
	`); err != nil {
		log.Fatalf("create rebate_pending_manual: %v", err)
	}

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("read input: %v", err)
	}
	var rows []pendingManualRow
	if err := json.Unmarshal(content, &rows); err != nil {
		log.Fatalf("parse input: %v", err)
	}

	tx, err := store.DB.Begin()
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO rebate_pending_manual (
			order_id, principal, subscribe_receivable, management_fee_received, performance_fee_receivable,
			subscribe_fee_ratio, management_fee_ratio, performance_fee_ratio,
			tax_subscribe_ratio, tax_management_ratio, tax_performance_ratio,
			expected_subscribe, expected_management, expected_performance,
			returned_subscribe, returned_management, returned_performance,
			outstanding_subscribe, outstanding_management, outstanding_performance,
			is_returnable, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(order_id) DO UPDATE SET
			principal = excluded.principal,
			subscribe_receivable = excluded.subscribe_receivable,
			management_fee_received = excluded.management_fee_received,
			performance_fee_receivable = excluded.performance_fee_receivable,
			subscribe_fee_ratio = excluded.subscribe_fee_ratio,
			management_fee_ratio = excluded.management_fee_ratio,
			performance_fee_ratio = excluded.performance_fee_ratio,
			tax_subscribe_ratio = excluded.tax_subscribe_ratio,
			tax_management_ratio = excluded.tax_management_ratio,
			tax_performance_ratio = excluded.tax_performance_ratio,
			expected_subscribe = excluded.expected_subscribe,
			expected_management = excluded.expected_management,
			expected_performance = excluded.expected_performance,
			returned_subscribe = excluded.returned_subscribe,
			returned_management = excluded.returned_management,
			returned_performance = excluded.returned_performance,
			outstanding_subscribe = excluded.outstanding_subscribe,
			outstanding_management = excluded.outstanding_management,
			outstanding_performance = excluded.outstanding_performance,
			is_returnable = excluded.is_returnable,
			updated_at = datetime('now')
	`)
	if err != nil {
		log.Fatalf("prepare upsert: %v", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		orderID := strings.TrimSpace(row.OrderID)
		if orderID == "" {
			continue
		}
		if _, err := stmt.Exec(
			orderID,
			nullableFloat(row.HasPrincipal, row.Principal),
			nullableFloat(row.HasSubscribeReceivable, row.SubscribeReceivable),
			nullableFloat(row.HasManagementFeeReceived, row.ManagementFeeReceived),
			nullableFloat(row.HasPerformanceFeeReceivable, row.PerformanceFeeReceivable),
			nullableFloat(row.HasSubscribeFeeRatio, row.SubscribeFeeRatio),
			nullableFloat(row.HasManagementFeeRatio, row.ManagementFeeRatio),
			nullableFloat(row.HasPerformanceFeeRatio, row.PerformanceFeeRatio),
			nullableFloat(row.HasTaxSubscribeRatio, row.TaxSubscribeRatio),
			nullableFloat(row.HasTaxManagementRatio, row.TaxManagementRatio),
			nullableFloat(row.HasTaxPerformanceRatio, row.TaxPerformanceRatio),
			nullableFloat(row.HasExpectedSubscribe, row.ExpectedSubscribe),
			nullableFloat(row.HasExpectedManagement, row.ExpectedManagement),
			nullableFloat(row.HasExpectedPerformance, row.ExpectedPerformance),
			nullableFloat(row.HasReturnedSubscribe, row.ReturnedSubscribe),
			nullableFloat(row.HasReturnedManagement, row.ReturnedManagement),
			nullableFloat(row.HasReturnedPerformance, row.ReturnedPerformance),
			nullableFloat(row.HasOutstandingSubscribe, row.OutstandingSubscribe),
			nullableFloat(row.HasOutstandingManagement, row.OutstandingManagement),
			nullableFloat(row.HasOutstandingPerformance, row.OutstandingPerformance),
			nullableString(row.IsReturnable),
		); err != nil {
			log.Fatalf("upsert %s: %v", orderID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}
	log.Printf("manual overrides upserted: %d", len(rows))
}

func nullableFloat(ok bool, value float64) any {
	if !ok {
		return nil
	}
	return value
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
