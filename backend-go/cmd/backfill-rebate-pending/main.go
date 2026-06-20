package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/db"
)

type pendingRow struct {
	OrderID                      string  `json:"order_id"`
	FlightID                     string  `json:"flight_id"`
	ProductName                  string  `json:"product_name"`
	CustomerName                 string  `json:"customer_name"`
	RebateTarget                 string  `json:"rebate_target"`
	Principal                    float64 `json:"principal"`
	HasPrincipal                 bool    `json:"has_principal"`
	SubscribeReceivable          float64 `json:"subscribe_receivable"`
	HasSubscribeReceivable       bool    `json:"has_subscribe_receivable"`
	ManagementFeeReceived        float64 `json:"management_fee_received"`
	HasManagementFeeReceived     bool    `json:"has_management_fee_received"`
	PerformanceFeeReceivable     float64 `json:"performance_fee_receivable"`
	HasPerformanceFeeReceivable  bool    `json:"has_performance_fee_receivable"`
	SubscribeFeeRatio            float64 `json:"subscribe_fee_ratio"`
	HasSubscribeFeeRatio         bool    `json:"has_subscribe_fee_ratio"`
	ManagementFeeRatio           float64 `json:"management_fee_ratio"`
	HasManagementFeeRatio        bool    `json:"has_management_fee_ratio"`
	PerformanceFeeRatio          float64 `json:"performance_fee_ratio"`
	HasPerformanceFeeRatio       bool    `json:"has_performance_fee_ratio"`
	TaxSubscribeRatio            float64 `json:"tax_subscribe_ratio"`
	HasTaxSubscribeRatio         bool    `json:"has_tax_subscribe_ratio"`
	TaxManagementRatio           float64 `json:"tax_management_ratio"`
	HasTaxManagementRatio        bool    `json:"has_tax_management_ratio"`
	TaxPerformanceRatio          float64 `json:"tax_performance_ratio"`
	HasTaxPerformanceRatio       bool    `json:"has_tax_performance_ratio"`
	IsReturnable                 string  `json:"is_returnable"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: backfill-rebate-pending <rows.json>")
	}

	cfg := config.Load()
	store, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("read input json: %v", err)
	}

	for _, ddl := range []string{
		"ALTER TABLE transactions ADD COLUMN management_fee_received REAL",
		"ALTER TABLE transactions ADD COLUMN performance_fee_receivable REAL",
	} {
		if _, err := store.DB.Exec(ddl); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			log.Fatalf("migrate transactions: %v", err)
		}
	}

	var rows []pendingRow
	if err := json.Unmarshal(content, &rows); err != nil {
		log.Fatalf("parse input json: %v", err)
	}

	tx, err := store.DB.Begin()
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	updateTxStmt, err := tx.Prepare(`
		UPDATE transactions
		SET flight_id = COALESCE(?, flight_id),
		    product_name = COALESCE(?, product_name),
		    customer_name = COALESCE(?, customer_name),
		    rebate_target = COALESCE(?, rebate_target),
		    subscribe_amount = COALESCE(?, subscribe_amount),
		    amount = COALESCE(?, amount),
		    subscribe_fee_rate = COALESCE(?, subscribe_fee_rate),
		    subscribe_fee_ratio = COALESCE(?, subscribe_fee_ratio),
		    management_fee_ratio = COALESCE(?, management_fee_ratio),
		    performance_fee_ratio = COALESCE(?, performance_fee_ratio),
		    management_fee_received = COALESCE(?, management_fee_received),
		    performance_fee_receivable = COALESCE(?, performance_fee_receivable),
		    tax_subscribe_ratio = COALESCE(?, tax_subscribe_ratio),
		    tax_management_ratio = COALESCE(?, tax_management_ratio),
		    tax_performance_ratio = COALESCE(?, tax_performance_ratio)
		WHERE order_id = ?
	`)
	if err != nil {
		log.Fatalf("prepare transactions update: %v", err)
	}
	defer updateTxStmt.Close()

	insertStatusStmt, err := tx.Prepare(`
		INSERT INTO rebate_status (order_id, is_returnable, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(order_id) DO UPDATE SET
		  is_returnable = excluded.is_returnable,
		  updated_at = datetime('now')
	`)
	if err != nil {
		log.Fatalf("prepare status upsert: %v", err)
	}
	defer insertStatusStmt.Close()

	updatedTransactions := 0
	updatedStatuses := 0
	missingTransactions := 0

	for _, row := range rows {
		orderID := strings.TrimSpace(row.OrderID)
		if orderID == "" {
			continue
		}

		subscribeFeeRate := nullableFloat64(false, 0)
		if row.HasPrincipal && row.Principal > 0 && row.HasSubscribeReceivable {
			subscribeFeeRate = nullableFloat64(true, row.SubscribeReceivable/row.Principal)
		}

		result, err := updateTxStmt.Exec(
			nullableString(row.FlightID),
			nullableString(row.ProductName),
			nullableString(row.CustomerName),
			nullableString(row.RebateTarget),
			nullableFloat64(row.HasPrincipal, row.Principal),
			nullableFloat64(row.HasPrincipal, row.Principal),
			subscribeFeeRate,
			nullableFloat64(row.HasSubscribeFeeRatio, row.SubscribeFeeRatio),
			nullableFloat64(row.HasManagementFeeRatio, row.ManagementFeeRatio),
			nullableFloat64(row.HasPerformanceFeeRatio, row.PerformanceFeeRatio),
			nullableFloat64(row.HasManagementFeeReceived, row.ManagementFeeReceived),
			nullableFloat64(row.HasPerformanceFeeReceivable, row.PerformanceFeeReceivable),
			nullableFloat64(row.HasTaxSubscribeRatio, row.TaxSubscribeRatio),
			nullableFloat64(row.HasTaxManagementRatio, row.TaxManagementRatio),
			nullableFloat64(row.HasTaxPerformanceRatio, row.TaxPerformanceRatio),
			orderID,
		)
		if err != nil {
			log.Fatalf("update transaction %s: %v", orderID, err)
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			missingTransactions++
			continue
		}
		updatedTransactions += int(affected)

		if strings.TrimSpace(row.IsReturnable) != "" {
			if _, err := insertStatusStmt.Exec(orderID, row.IsReturnable); err != nil {
				log.Fatalf("upsert rebate_status %s: %v", orderID, err)
			}
			updatedStatuses++
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit tx: %v", err)
	}

	fmt.Printf("rows loaded: %d\n", len(rows))
	fmt.Printf("transactions updated: %d\n", updatedTransactions)
	fmt.Printf("rebate_status updated: %d\n", updatedStatuses)
	fmt.Printf("rows missing transactions: %d\n", missingTransactions)
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableFloat64(ok bool, value float64) any {
	if !ok {
		return nil
	}
	return value
}

var _ = sql.ErrNoRows
