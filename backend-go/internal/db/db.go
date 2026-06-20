package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB   *sql.DB
	Path string
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{DB: db, Path: path}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) InitSchema() error {
	_, err := s.DB.Exec(schemaSQL)
	if err != nil {
		return err
	}
	if err := s.MigrateHoldingColumns(); err != nil {
		return err
	}
	return s.MigrateRebateColumns()
}

func (s *Store) MigrateHoldingColumns() error {
	productCols := map[string]string{
		"knock_in":      "TEXT",
		"duration_days": "INTEGER",
		"knocked_in":    "TEXT",
		"margin_ratio":  "REAL",
		"custodian":     "TEXT",
		"counterparty":  "TEXT",
	}
	for col, colType := range productCols {
		_, err := s.DB.Exec(fmt.Sprintf("ALTER TABLE products ADD COLUMN %s %s", col, colType))
		if err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("migrate products.%s: %w", col, err)
		}
	}

	txCols := map[string]string{
		"product_name":          "TEXT",
		"customer_name":         "TEXT",
		"actual_buyer":          "TEXT",
		"channel_or_direct":     "TEXT",
		"amount":                "REAL",
		"subscribe_fee_rate":    "REAL",
		"subscribe_fee_ratio":   "REAL",
		"management_fee_ratio":  "REAL",
		"performance_fee_ratio": "REAL",
		"management_fee_received": "REAL",
		"performance_fee_receivable": "REAL",
		"rebate_target":         "TEXT",
		"flight_date":           "TEXT",
		"holding_status":        "TEXT",
		"complete_date":         "TEXT",
		"underlying":            "TEXT",
		"structure_type":        "TEXT",
		"lock_period":           "TEXT",
		"dividend_barrier":      "REAL",
		"monthly_coupon":        "REAL",
		"coupon_1st":            "REAL",
		"tax_subscribe_ratio":   "REAL",
		"tax_management_ratio":  "REAL",
		"tax_performance_ratio": "REAL",
	}
	for col, colType := range txCols {
		_, err := s.DB.Exec(fmt.Sprintf("ALTER TABLE transactions ADD COLUMN %s %s", col, colType))
		if err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("migrate transactions.%s: %w", col, err)
		}
	}
	return nil
}

func (s *Store) MigrateRebateColumns() error {
	txCols := map[string]string{
		"order_id":          "TEXT",
		"channel_or_direct": "TEXT",
		"subscribe_fee_rate": "REAL",
	}
	for col, colType := range txCols {
		_, err := s.DB.Exec(fmt.Sprintf("ALTER TABLE transactions ADD COLUMN %s %s", col, colType))
		if err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("migrate transactions.%s: %w", col, err)
		}
	}
	rebateCompletedCols := map[string]string{
		"ignored_conflicts": "TEXT",
	}
	for col, colType := range rebateCompletedCols {
		_, err := s.DB.Exec(fmt.Sprintf("ALTER TABLE rebate_completed ADD COLUMN %s %s", col, colType))
		if err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("migrate rebate_completed.%s: %w", col, err)
		}
	}
	if _, err := s.DB.Exec(`
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
		return fmt.Errorf("migrate rebate_pending_manual: %w", err)
	}
	return nil
}
