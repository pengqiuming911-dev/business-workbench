package main

import (
	"fmt"
	"log"

	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/db"
)

func main() {
	cfg := config.Load()
	store, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	var before int
	if err := store.DB.QueryRow(`
		SELECT COUNT(*)
		FROM transactions
		WHERE (subscribe_amount IS NULL OR subscribe_amount = 0)
		  AND amount > 0 AND amount < 10000
		  AND (
		    subscribe_fee_rate > 0 OR
		    subscribe_fee_ratio > 0 OR
		    management_fee_ratio > 0 OR
		    performance_fee_ratio > 0
		  )
	`).Scan(&before); err != nil {
		log.Fatalf("count suspicious rows: %v", err)
	}

	result, err := store.DB.Exec(`
		UPDATE transactions
		SET subscribe_amount = amount * 10000
		WHERE (subscribe_amount IS NULL OR subscribe_amount = 0)
		  AND amount > 0 AND amount < 10000
		  AND (
		    subscribe_fee_rate > 0 OR
		    subscribe_fee_ratio > 0 OR
		    management_fee_ratio > 0 OR
		    performance_fee_ratio > 0
		  )
	`)
	if err != nil {
		log.Fatalf("update subscribe_amount: %v", err)
	}
	affected, _ := result.RowsAffected()

	var after int
	if err := store.DB.QueryRow(`
		SELECT COUNT(*)
		FROM transactions
		WHERE (subscribe_amount IS NULL OR subscribe_amount = 0)
		  AND amount > 0 AND amount < 10000
		  AND (
		    subscribe_fee_rate > 0 OR
		    subscribe_fee_ratio > 0 OR
		    management_fee_ratio > 0 OR
		    performance_fee_ratio > 0
		  )
	`).Scan(&after); err != nil {
		log.Fatalf("count remaining suspicious rows: %v", err)
	}

	fmt.Printf("principal rows before: %d\n", before)
	fmt.Printf("rows updated: %d\n", affected)
	fmt.Printf("principal rows after: %d\n", after)
}
