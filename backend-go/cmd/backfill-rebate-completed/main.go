package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/db"
)

type workbookRow struct {
	OrderID                 string  `json:"order_id"`
	ExpenseCategory         string  `json:"expense_category"`
	ExpenseAmount           float64 `json:"expense_amount"`
	HasExpenseAmount        bool    `json:"has_expense_amount"`
	PaymentTime             string  `json:"payment_time"`
	PaymentYear             string  `json:"payment_year"`
	PaymentMonth            string  `json:"payment_month"`
	PaymentDay              string  `json:"payment_day"`
	ChannelSubscribeRatio   float64 `json:"channel_subscribe_ratio"`
	HasChannelSubscribe     bool    `json:"has_channel_subscribe_ratio"`
	ChannelManagementRatio  float64 `json:"channel_management_ratio"`
	HasChannelManagement    bool    `json:"has_channel_management_ratio"`
	ChannelPerformanceRatio float64 `json:"channel_performance_ratio"`
	HasChannelPerformance   bool    `json:"has_channel_performance_ratio"`
}

type rebateRecord struct {
	ID int64
}

func main() {
	cfg := config.Load()

	store, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	fmt.Printf("database: %s\n", store.Path)

	beforeRatioMissing := mustCount(store.DB, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND order_id <> ''
		  AND (
		    channel_subscribe_ratio IS NULL OR
		    channel_management_ratio IS NULL OR
		    channel_performance_ratio IS NULL
		  )
	`)
	beforeDateSplitMissing := mustCount(store.DB, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND payment_time IS NOT NULL AND payment_time <> ''
		  AND (
		    payment_year IS NULL OR payment_year = '' OR
		    payment_month IS NULL OR payment_month = '' OR
		    payment_day IS NULL OR payment_day = ''
		  )
	`)

	result, err := store.DB.Exec(`
		UPDATE rebate_completed
		SET
		  channel_subscribe_ratio = COALESCE(
		    channel_subscribe_ratio,
		    (SELECT t.subscribe_fee_ratio FROM transactions t WHERE t.order_id = rebate_completed.order_id LIMIT 1)
		  ),
		  channel_management_ratio = COALESCE(
		    channel_management_ratio,
		    (SELECT t.management_fee_ratio FROM transactions t WHERE t.order_id = rebate_completed.order_id LIMIT 1)
		  ),
		  channel_performance_ratio = COALESCE(
		    channel_performance_ratio,
		    (SELECT t.performance_fee_ratio FROM transactions t WHERE t.order_id = rebate_completed.order_id LIMIT 1)
		  ),
		  payment_year = CASE
		    WHEN (payment_year IS NULL OR payment_year = '') AND payment_time IS NOT NULL AND length(payment_time) >= 10
		      THEN substr(payment_time, 1, 4)
		    ELSE payment_year
		  END,
		  payment_month = CASE
		    WHEN (payment_month IS NULL OR payment_month = '') AND payment_time IS NOT NULL AND length(payment_time) >= 10
		      THEN substr(payment_time, 6, 2)
		    ELSE payment_month
		  END,
		  payment_day = CASE
		    WHEN (payment_day IS NULL OR payment_day = '') AND payment_time IS NOT NULL AND length(payment_time) >= 10
		      THEN substr(payment_time, 9, 2)
		    ELSE payment_day
		  END,
		  updated_at = datetime('now')
		WHERE source = 'upload'
		  AND order_id <> ''
		  AND (
		    channel_subscribe_ratio IS NULL OR
		    channel_management_ratio IS NULL OR
		    channel_performance_ratio IS NULL OR
		    (
		      payment_time IS NOT NULL AND payment_time <> '' AND (
		        payment_year IS NULL OR payment_year = '' OR
		        payment_month IS NULL OR payment_month = '' OR
		        payment_day IS NULL OR payment_day = ''
		      )
		    )
		  )
		  AND EXISTS (SELECT 1 FROM transactions t WHERE t.order_id = rebate_completed.order_id)
	`)
	if err != nil {
		log.Fatalf("update rebate_completed: %v", err)
	}

	affected, _ := result.RowsAffected()

	afterRatioMissing := mustCount(store.DB, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND order_id <> ''
		  AND (
		    channel_subscribe_ratio IS NULL OR
		    channel_management_ratio IS NULL OR
		    channel_performance_ratio IS NULL
		  )
	`)
	afterDateSplitMissing := mustCount(store.DB, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND payment_time IS NOT NULL AND payment_time <> ''
		  AND (
		    payment_year IS NULL OR payment_year = '' OR
		    payment_month IS NULL OR payment_month = '' OR
		    payment_day IS NULL OR payment_day = ''
		  )
	`)
	missingAmount := mustCount(store.DB, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND (expense_amount IS NULL OR expense_amount = 0)
	`)
	missingPaymentTime := mustCount(store.DB, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND (payment_time IS NULL OR payment_time = '')
	`)
	missingExpenseCategory := mustCount(store.DB, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND (expense_category IS NULL OR expense_category = '')
	`)

	fmt.Printf("rows touched: %d\n", affected)
	fmt.Printf("ratio-missing before: %d, after: %d\n", beforeRatioMissing, afterRatioMissing)
	fmt.Printf("payment y/m/d missing before: %d, after: %d\n", beforeDateSplitMissing, afterDateSplitMissing)
	fmt.Printf("still missing expense_amount: %d\n", missingAmount)
	fmt.Printf("still missing payment_time: %d\n", missingPaymentTime)
	fmt.Printf("still missing expense_category: %d\n", missingExpenseCategory)

	if len(os.Args) > 1 {
		if err := applyWorkbookBackfill(store.DB, os.Args[1]); err != nil {
			log.Fatalf("apply workbook backfill: %v", err)
		}
	}
}

func mustCount(dbConn *sql.DB, query string) int {
	var count int
	if err := dbConn.QueryRow(query).Scan(&count); err != nil {
		log.Fatalf("count query failed: %v", err)
	}
	return count
}

func applyWorkbookBackfill(dbConn *sql.DB, jsonPath string) error {
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read workbook json: %w", err)
	}

	var rows []workbookRow
	if err := json.Unmarshal(content, &rows); err != nil {
		return fmt.Errorf("parse workbook json: %w", err)
	}

	byOrder := map[string][]workbookRow{}
	for _, row := range rows {
		key := strings.TrimSpace(row.OrderID)
		if key == "" {
			continue
		}
		byOrder[key] = append(byOrder[key], row)
	}

	dbRows, err := loadUploadRecords(dbConn)
	if err != nil {
		return err
	}

	tx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	deleteResult, err := tx.Exec(`
		DELETE FROM rebate_completed
		WHERE source = 'upload'
		  AND COALESCE(order_id, '') = ''
		  AND COALESCE(flight_id, '') = ''
		  AND COALESCE(product_name, '') = ''
		  AND COALESCE(customer_name, '') = ''
	`)
	if err != nil {
		return fmt.Errorf("delete blank upload rows: %w", err)
	}
	deletedBlank, _ := deleteResult.RowsAffected()

	stmt, err := tx.Prepare(`
		UPDATE rebate_completed
		SET expense_category = ?,
		    expense_amount = ?,
		    payment_time = ?,
		    payment_year = ?,
		    payment_month = ?,
		    payment_day = ?,
		    channel_subscribe_ratio = ?,
		    channel_management_ratio = ?,
		    channel_performance_ratio = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("prepare update: %w", err)
	}
	defer stmt.Close()

	orderIDs := make([]string, 0, len(dbRows))
	for orderID := range dbRows {
		orderIDs = append(orderIDs, orderID)
	}
	sort.Strings(orderIDs)

	updated := 0
	missingInWorkbook := 0
	countMismatch := 0
	for _, orderID := range orderIDs {
		dbGroup := dbRows[orderID]
		wbGroup := byOrder[orderID]
		if len(wbGroup) == 0 {
			missingInWorkbook++
			continue
		}
		if len(dbGroup) != len(wbGroup) {
			countMismatch++
			continue
		}
		sort.Slice(dbGroup, func(i, j int) bool { return dbGroup[i].ID < dbGroup[j].ID })
		for i := range dbGroup {
			row := wbGroup[i]
			paymentTime := strings.TrimSpace(row.PaymentTime)
			if paymentTime == "" {
				paymentTime = synthesizePaymentTime(row.PaymentYear, row.PaymentMonth, row.PaymentDay)
			}
			if _, err := stmt.Exec(
				nullableStringArg(row.ExpenseCategory),
				nullableFloat64Arg(row.ExpenseAmount, row.HasExpenseAmount),
				nullableStringArg(paymentTime),
				nullableStringArg(row.PaymentYear),
				nullableStringArg(row.PaymentMonth),
				nullableStringArg(row.PaymentDay),
				nullableFloat64Arg(row.ChannelSubscribeRatio, row.HasChannelSubscribe),
				nullableFloat64Arg(row.ChannelManagementRatio, row.HasChannelManagement),
				nullableFloat64Arg(row.ChannelPerformanceRatio, row.HasChannelPerformance),
				dbGroup[i].ID,
			); err != nil {
				return fmt.Errorf("update order %s row %d: %w", orderID, i, err)
			}
			updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("workbook rows loaded: %d\n", len(rows))
	fmt.Printf("blank upload rows deleted: %d\n", deletedBlank)
	fmt.Printf("workbook-based rows updated: %d\n", updated)
	fmt.Printf("orders missing in workbook: %d\n", missingInWorkbook)
	fmt.Printf("orders skipped due to count mismatch: %d\n", countMismatch)
	fmt.Printf("remaining missing expense_amount: %d\n", mustCount(dbConn, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND (expense_amount IS NULL)
	`))
	fmt.Printf("remaining missing payment_year: %d\n", mustCount(dbConn, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND (payment_year IS NULL OR payment_year = '')
	`))
	fmt.Printf("remaining missing payment_month: %d\n", mustCount(dbConn, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND (payment_month IS NULL OR payment_month = '')
	`))
	fmt.Printf("remaining missing payment_day: %d\n", mustCount(dbConn, `
		SELECT COUNT(*)
		FROM rebate_completed
		WHERE source = 'upload'
		  AND (payment_day IS NULL OR payment_day = '')
	`))
	return nil
}

func loadUploadRecords(dbConn *sql.DB) (map[string][]rebateRecord, error) {
	rows, err := dbConn.Query(`
		SELECT id, order_id
		FROM rebate_completed
		WHERE source = 'upload'
		  AND COALESCE(order_id, '') <> ''
		ORDER BY order_id, id
	`)
	if err != nil {
		return nil, fmt.Errorf("query upload records: %w", err)
	}
	defer rows.Close()

	result := map[string][]rebateRecord{}
	for rows.Next() {
		var id int64
		var orderID sql.NullString
		if err := rows.Scan(&id, &orderID); err != nil {
			return nil, fmt.Errorf("scan upload record: %w", err)
		}
		key := strings.TrimSpace(orderID.String)
		result[key] = append(result[key], rebateRecord{ID: id})
	}
	return result, rows.Err()
}

func nullableStringArg(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableFloat64Arg(value float64, ok bool) any {
	if !ok {
		return nil
	}
	return value
}

func synthesizePaymentTime(year, month, day string) string {
	year = strings.TrimSpace(year)
	month = strings.TrimSpace(month)
	day = strings.TrimSpace(day)
	if year == "" {
		return ""
	}
	if month == "" {
		return year
	}
	if day == "" {
		return fmt.Sprintf("%s-%s", year, pad2(dayOrMonth(month)))
	}
	return fmt.Sprintf("%s-%s-%s", year, pad2(dayOrMonth(month)), pad2(dayOrMonth(day)))
}

func dayOrMonth(value string) string {
	return strings.TrimSpace(value)
}

func pad2(value string) string {
	if len(value) >= 2 {
		return value
	}
	return "0" + value
}
