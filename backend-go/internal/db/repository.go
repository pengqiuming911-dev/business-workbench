package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"business-workbench/backend-go/internal/model"
)

func (s *Store) QueryProducts(startDate, endDate string) ([]model.Product, error) {
	query := "SELECT * FROM products WHERE 1=1"
	args := []any{}
	if startDate != "" {
		query += " AND issue_date >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND issue_date <= ?"
		args = append(args, endDate)
	}
	query += " ORDER BY issue_date DESC"
	return s.scanProducts(query, args...)
}

func (s *Store) ImportProducts(products []model.Product) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	incoming := map[string]bool{}
	for _, product := range products {
		incoming[product.ID] = true
	}
	rows, err := tx.Query("SELECT id FROM products")
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	existingIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			_ = tx.Rollback()
			return err
		}
		existingIDs = append(existingIDs, id)
	}
	if err := rows.Close(); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, id := range existingIDs {
		if !incoming[id] {
			if _, err := tx.Exec("DELETE FROM products WHERE id = ?", id); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO products
			(id, name, is_main, issue_date, complete_date, subscribe_amount, outstanding_amount,
			 manager, holding_status, structure_type, code, lock_days, lock_months,
			 first_knockout_ratio, entry_price, monthly_decrease, term, parachute,
			 dividend_barrier, monthly_coupon, coupon_1st, coupon_2nd, coupon_3rd,
			 duration_months, absolute_return, holiday_adjust, raw,
			 knock_in, duration_days, knocked_in, margin_ratio, custodian, counterparty)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, product := range products {
		if _, err := stmt.Exec(
			product.ID, nullableDBString(product.Name), nullableInt(product.IsMain), nullableDBString(product.IssueDate), nullableDBString(product.CompleteDate),
			nullableFloat(product.SubscribeAmount), nullableFloat(product.OutstandingAmount),
			nullableDBString(product.Manager), nullableDBString(product.HoldingStatus), nullableDBString(product.StructureType), nullableDBString(product.Code),
			nullableInt(product.LockDays), nullableInt(product.LockMonths), nullableFloat(product.FirstKnockoutRatio),
			nullableFloat(product.EntryPrice), nullableFloat(product.MonthlyDecrease), nullableDBString(product.Term), nullableDBString(product.Parachute),
			nullableFloat(product.DividendBarrier), nullableFloat(product.MonthlyCoupon),
			nullableFloat(product.Coupon1st), nullableFloat(product.Coupon2nd), nullableFloat(product.Coupon3rd),
			nullableFloat(product.DurationMonths), nullableFloat(product.AbsoluteReturn), nullableDBString(product.HolidayAdjust), nullableDBString(product.Raw),
			nullableDBString(product.KnockIn), nullableInt(product.DurationDays), nullableDBString(product.KnockedIn),
			nullableFloat(product.MarginRatio), nullableDBString(product.Custodian), nullableDBString(product.Counterparty),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ImportTransactions(rows []map[string]any) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM transactions"); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO transactions
			(transaction_date, flight_id, counterparty, channel_or_direct, subscribe_amount, subscribe_fee_rate,
			 product_name, customer_name, actual_buyer, amount,
			 subscribe_fee_ratio, management_fee_ratio, performance_fee_ratio,
			 rebate_target, flight_date, holding_status, complete_date,
			 underlying, structure_type, lock_period,
			 dividend_barrier, monthly_coupon, coupon_1st, raw, order_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, row := range rows {
		if _, err := stmt.Exec(
			row["transaction_date"], row["flight_id"], row["counterparty"], row["channel_or_direct"], row["subscribe_amount"], row["subscribe_fee_rate"],
			row["product_name"], row["customer_name"], row["actual_buyer"], row["amount"],
			row["subscribe_fee_ratio"], row["management_fee_ratio"], row["performance_fee_ratio"],
			row["rebate_target"], row["flight_date"], row["holding_status"], row["complete_date"],
			row["underlying"], row["structure_type"], row["lock_period"],
			row["dividend_barrier"], row["monthly_coupon"], row["coupon_1st"], row["raw"], row["order_id"],
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ImportCoInvestUsers(rows []map[string]any) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM co_invest_users"); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO co_invest_users
			(user_name, actual_buyer, phone, wechat, total_assets, risk_tolerance, industry,
			 is_actual_deal, lead_source, asset_match, is_dedicated_account, is_competitor, raw)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, row := range rows {
		if _, err := stmt.Exec(
			row["user_name"], row["actual_buyer"], row["phone"], row["wechat"], row["total_assets"],
			row["risk_tolerance"], row["industry"], row["is_actual_deal"], row["lead_source"],
			row["asset_match"], row["is_dedicated_account"], row["is_competitor"], row["raw"],
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ImportProductDocs(rows []map[string]any) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM product_docs"); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO product_docs
			(doc_token, doc_name, parent_path, folder_token, raw_content, structure_json, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, row := range rows {
		if _, err := stmt.Exec(
			row["doc_token"], row["doc_name"], row["parent_path"], row["folder_token"],
			row["raw_content"], row["structure_json"], row["synced_at"],
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) LogSync(rowCount int) error {
	_, err := s.DB.Exec("INSERT INTO sync_log (synced_at, row_count) VALUES (?, ?)", isoNow(), rowCount)
	return err
}

func (s *Store) LogCoInvestSync(rowCount int) error {
	_, err := s.DB.Exec("INSERT INTO co_invest_sync_log (synced_at, row_count) VALUES (?, ?)", isoNow(), rowCount)
	return err
}

func (s *Store) LogProductDocsSync(docCount int, folderCount int) error {
	_, err := s.DB.Exec("INSERT INTO product_docs_sync_log (synced_at, doc_count, folder_count) VALUES (?, ?, ?)", isoNow(), docCount, folderCount)
	return err
}

func (s *Store) LogActivity(logType string, action string, detail string) error {
	_, err := s.DB.Exec("INSERT INTO activity_logs (type, action, detail) VALUES (?, ?, ?)", logType, action, nullableDBString(detail))
	return err
}

func (s *Store) QueryOngoingProducts() ([]model.Product, error) {
	return s.scanProducts("SELECT * FROM products WHERE holding_status LIKE ? OR holding_status LIKE ?", "%存续%", "%持有%")
}

func (s *Store) QueryCompletedProducts() ([]model.Product, error) {
	return s.scanProducts("SELECT * FROM products WHERE holding_status LIKE ?", "%完结%")
}

func (s *Store) LastSync() (map[string]any, error) {
	return s.queryOneMap("SELECT * FROM sync_log ORDER BY synced_at DESC LIMIT 1")
}

func (s *Store) LastCoInvestSync() (map[string]any, error) {
	return s.queryOneMap("SELECT * FROM co_invest_sync_log ORDER BY synced_at DESC LIMIT 1")
}

func (s *Store) LastProductDocsSync() (map[string]any, error) {
	return s.queryOneMap("SELECT * FROM product_docs_sync_log ORDER BY synced_at DESC LIMIT 1")
}

func (s *Store) DashboardStats() (map[string]any, error) {
	stats := map[string]any{}
	queries := map[string]string{
		"totalProducts":  "SELECT COUNT(*) FROM products",
		"activeProducts": "SELECT COUNT(*) FROM products WHERE holding_status LIKE '%存续%' OR holding_status LIKE '%持有%'",
		"totalCustomers": "SELECT COUNT(DISTINCT customer_name) FROM customers",
		"totalChannels":  "SELECT COUNT(DISTINCT channel_name) FROM channels",
	}
	for key, query := range queries {
		var count int64
		if err := s.DB.QueryRow(query).Scan(&count); err != nil {
			return nil, err
		}
		stats[key] = count
	}
	return stats, nil
}

func (s *Store) LastObservationUpdate() (string, error) {
	var updated sql.NullString
	err := s.DB.QueryRow("SELECT updated_at FROM observations ORDER BY updated_at DESC LIMIT 1").Scan(&updated)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return nullString(updated), nil
}

func (s *Store) MonthlyTrend() ([]map[string]any, error) {
	rows, err := s.DB.Query(`
		SELECT strftime('%Y-%m', transaction_date) as month,
		       SUM(subscribe_amount) as amount,
		       COUNT(*) as count
		FROM transactions
		GROUP BY month ORDER BY month
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var month sql.NullString
		var amount sql.NullFloat64
		var count int64
		if err := rows.Scan(&month, &amount, &count); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{
			"month":  nullString(month),
			"amount": nullFloat(amount),
			"count":  count,
		})
	}
	return result, rows.Err()
}

func (s *Store) ChannelDistribution() ([]map[string]any, error) {
	rows, err := s.DB.Query(`
		SELECT c.channel_name, SUM(t.subscribe_amount) as amount
		FROM channels c
		LEFT JOIN transactions t ON t.counterparty = c.channel_name
		GROUP BY c.channel_name ORDER BY amount DESC LIMIT 8
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var channel sql.NullString
		var amount sql.NullFloat64
		if err := rows.Scan(&channel, &amount); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{
			"channel": nullString(channel),
			"amount":  nullFloat(amount),
		})
	}
	return result, rows.Err()
}

func (s *Store) QueryObservationsByProduct(productID string) ([]model.Observation, error) {
	rows, err := s.DB.Query(`
		SELECT id, product_id, observation_date, knockout_price, dividend_line,
		       underlying_price, is_knocked_out, is_dividend, months_since_entry, updated_at
		FROM observations WHERE product_id = ? ORDER BY observation_date
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.Observation{}
	for rows.Next() {
		var row model.Observation
		var knockout, dividend, underlying sql.NullFloat64
		var months sql.NullInt64
		if err := rows.Scan(
			&row.ID, &row.ProductID, &row.ObservationDate, &knockout, &dividend,
			&underlying, &row.IsKnockedOut, &row.IsDividend, &months, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		row.KnockoutPrice = floatPtr(knockout)
		row.DividendLine = floatPtr(dividend)
		row.UnderlyingPrice = floatPtr(underlying)
		row.MonthsSinceEntry = intPtr(months)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *Store) QueryPostersByDate(date string) ([]model.Poster, error) {
	return s.queryPosters("SELECT * FROM posters WHERE observation_date = ? ORDER BY created_at DESC", date)
}

func (s *Store) QueryPostersByProduct(productID string) ([]model.Poster, error) {
	return s.queryPosters("SELECT * FROM posters WHERE product_id = ? ORDER BY observation_date DESC", productID)
}

func (s *Store) QueryAllPosters() ([]model.Poster, error) {
	return s.queryPosters("SELECT * FROM posters ORDER BY observation_date DESC, created_at DESC")
}

func (s *Store) QueryActivityLogs(logType string, limit int) ([]model.ActivityLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := "SELECT id, type, action, detail, created_at FROM activity_logs"
	args := []any{}
	if logType != "" {
		query += " WHERE type = ?"
		args = append(args, logType)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.ActivityLog{}
	for rows.Next() {
		var row model.ActivityLog
		var detail sql.NullString
		if err := rows.Scan(&row.ID, &row.Type, &row.Action, &detail, &row.CreatedAt); err != nil {
			return nil, err
		}
		row.Detail = nullString(detail)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *Store) GetPushConfig(fallbackWebhook string) (model.PushConfig, error) {
	row := model.PushConfig{
		WebhookURL: fallbackWebhook,
		CronHour:   9,
		CronMinute: 0,
		Enabled:    false,
	}
	var enabled int
	var lastPushTime, lastPushResult sql.NullString
	err := s.DB.QueryRow(`
		SELECT webhook_url, cron_hour, cron_minute, enabled, last_push_time, last_push_result
		FROM push_config WHERE id = 1
	`).Scan(&row.WebhookURL, &row.CronHour, &row.CronMinute, &enabled, &lastPushTime, &lastPushResult)
	if err == sql.ErrNoRows {
		return row, nil
	}
	if err != nil {
		return row, err
	}
	row.Enabled = enabled != 0
	row.LastPushTime = nullString(lastPushTime)
	row.LastPushResult = nullString(lastPushResult)
	return row, nil
}

func (s *Store) UpsertPushConfig(config model.PushConfig) error {
	enabled := 0
	if config.Enabled {
		enabled = 1
	}
	_, err := s.DB.Exec(`
		INSERT INTO push_config (id, webhook_url, cron_hour, cron_minute, enabled, updated_at)
		VALUES (1, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			webhook_url = excluded.webhook_url,
			cron_hour = excluded.cron_hour,
			cron_minute = excluded.cron_minute,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`, config.WebhookURL, config.CronHour, config.CronMinute, enabled)
	return err
}

func (s *Store) UpdatePushResult(lastPushTime string, lastPushResult string) error {
	_, err := s.DB.Exec(`
		INSERT INTO push_config (id, webhook_url, cron_hour, cron_minute, enabled, last_push_time, last_push_result, updated_at)
		VALUES (1, '', 9, 0, 0, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			last_push_time = excluded.last_push_time,
			last_push_result = excluded.last_push_result,
			updated_at = excluded.updated_at
	`, lastPushTime, lastPushResult, isoNow())
	return err
}

func (s *Store) SearchProducts(q string) ([]map[string]any, error) {
	rows, err := s.DB.Query("SELECT id, name FROM products WHERE name LIKE ? LIMIT 5", LikeContains(q))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{
			"type": "product",
			"id":   nullString(id),
			"name": nullString(name),
			"path": "/holding-analysis",
		})
	}
	return result, rows.Err()
}

func (s *Store) SearchProductsForAgent(keyword string) ([]map[string]any, error) {
	like := LikeContains(keyword)
	rows, err := s.DB.Query(
		"SELECT id, name, holding_status, code FROM products WHERE name LIKE ? OR code LIKE ? LIMIT 20",
		like, like,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var id, name, holdingStatus, code sql.NullString
		if err := rows.Scan(&id, &name, &holdingStatus, &code); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{
			"id":             nullString(id),
			"name":           nullString(name),
			"holding_status": nullString(holdingStatus),
			"code":           nullString(code),
		})
	}
	return result, rows.Err()
}

func (s *Store) ProductByID(id string) (*model.Product, error) {
	products, err := s.scanProducts("SELECT * FROM products WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, nil
	}
	return &products[0], nil
}

func (s *Store) LatestPrice(code string) (map[string]any, error) {
	return s.queryOneMap(
		"SELECT code, price_date, price, updated_at FROM price_cache WHERE code = ? ORDER BY price_date DESC LIMIT 1",
		code,
	)
}

func (s *Store) PriceByDate(code string, priceDate string) (map[string]any, error) {
	return s.queryOneMap(
		"SELECT code, price_date, price, updated_at FROM price_cache WHERE code = ? AND price_date = ?",
		code, priceDate,
	)
}

func (s *Store) UpsertPrice(code string, priceDate string, price float64) error {
	_, err := s.DB.Exec(`
		INSERT INTO price_cache (code, price_date, price, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(code, price_date) DO UPDATE SET
			price = excluded.price,
			updated_at = excluded.updated_at
	`, code, priceDate, price, isoNow())
	return err
}

func (s *Store) UpsertObservation(productID string, eval map[string]any) error {
	_, err := s.DB.Exec(`
		INSERT INTO observations
			(product_id, observation_date, knockout_price, dividend_line, underlying_price,
			 is_knocked_out, is_dividend, months_since_entry, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_id, observation_date) DO UPDATE SET
			knockout_price = excluded.knockout_price,
			dividend_line = excluded.dividend_line,
			underlying_price = excluded.underlying_price,
			is_knocked_out = excluded.is_knocked_out,
			is_dividend = excluded.is_dividend,
			months_since_entry = excluded.months_since_entry,
			updated_at = excluded.updated_at
	`, productID, eval["observation_date"], eval["knockout_price"], eval["dividend_line"], eval["underlying_price"],
		eval["is_knocked_out"], eval["is_dividend"], eval["months_since_entry"], isoNow())
	return err
}

func (s *Store) UpsertPoster(row model.Poster) error {
	_, err := s.DB.Exec(`
		INSERT INTO posters
			(product_id, poster_type, observation_date, product_name, date_display, months_since_entry,
			 underlying_name, absolute_return, annualized_return, duration_months,
			 parachute_value, knockout_value, dividend_barrier_value,
			 dividend_count, cumulative_rate, monthly_coupon, entry_date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_id, poster_type, observation_date) DO UPDATE SET
			product_name = excluded.product_name,
			date_display = excluded.date_display,
			months_since_entry = excluded.months_since_entry,
			underlying_name = excluded.underlying_name,
			absolute_return = excluded.absolute_return,
			annualized_return = excluded.annualized_return,
			duration_months = excluded.duration_months,
			parachute_value = excluded.parachute_value,
			knockout_value = excluded.knockout_value,
			dividend_barrier_value = excluded.dividend_barrier_value,
			dividend_count = excluded.dividend_count,
			cumulative_rate = excluded.cumulative_rate,
			monthly_coupon = excluded.monthly_coupon,
			entry_date = excluded.entry_date,
			created_at = excluded.created_at
	`, row.ProductID, row.PosterType, row.ObservationDate, row.ProductName, row.DateDisplay, nullableInt(row.MonthsSinceEntry),
		row.UnderlyingName, nullableFloat(row.AbsoluteReturn), nullableFloat(row.AnnualizedReturn), nullableInt(row.DurationMonths),
		nullableDBString(row.ParachuteValue), nullableDBString(row.KnockoutValue), nullableDBString(row.DividendBarrierValue),
		nullableInt(row.DividendCount), nullableFloat(row.CumulativeRate), nullableFloat(row.MonthlyCoupon), row.EntryDate, isoNow())
	return err
}

func (s *Store) SearchCustomersForAgent(keyword, industry, isDedicated, isCompetitor string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT user_name, actual_buyer, phone, wechat, total_assets, risk_tolerance, industry,
		       is_actual_deal, lead_source, is_dedicated_account, is_competitor
		FROM co_invest_users WHERE 1=1
	`
	args := []any{}
	if keyword != "" {
		query += " AND (user_name LIKE ? OR actual_buyer LIKE ? OR wechat LIKE ?)"
		like := LikeContains(keyword)
		args = append(args, like, like, like)
	}
	if industry != "" {
		query += " AND industry LIKE ?"
		args = append(args, LikeContains(industry))
	}
	if isDedicated != "" {
		query += " AND is_dedicated_account = ?"
		args = append(args, isDedicated)
	}
	if isCompetitor != "" {
		query += " AND is_competitor = ?"
		args = append(args, isCompetitor)
	}
	query += " ORDER BY id LIMIT ?"
	args = append(args, limit)
	return s.queryMaps(query, args...)
}

func (s *Store) DistinctIndustries() ([]string, error) {
	rows, err := s.DB.Query("SELECT DISTINCT industry FROM co_invest_users WHERE industry IS NOT NULL AND industry != '' ORDER BY industry")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []string{}
	for rows.Next() {
		var industry sql.NullString
		if err := rows.Scan(&industry); err != nil {
			return nil, err
		}
		if industry.Valid && strings.TrimSpace(industry.String) != "" {
			result = append(result, industry.String)
		}
	}
	return result, rows.Err()
}

func (s *Store) UserProfiles(actualBuyer, nominalBuyer, isDedicated, isCompetitor, industry string) ([]map[string]any, error) {
	query := `
		SELECT user_name AS nominal_buyer, actual_buyer, phone, wechat, total_assets,
		       risk_tolerance, industry, is_actual_deal, lead_source, asset_match,
		       is_dedicated_account, is_competitor
		FROM co_invest_users WHERE 1=1
	`
	args := []any{}
	if actualBuyer != "" {
		query += " AND actual_buyer LIKE ?"
		args = append(args, LikeContains(actualBuyer))
	}
	if nominalBuyer != "" {
		query += " AND user_name LIKE ?"
		args = append(args, LikeContains(nominalBuyer))
	}
	if isDedicated != "" {
		query += " AND is_dedicated_account = ?"
		args = append(args, isDedicated)
	}
	if isCompetitor != "" {
		query += " AND is_competitor = ?"
		args = append(args, isCompetitor)
	}
	if industry != "" {
		query += " AND industry = ?"
		args = append(args, industry)
	}
	query += " ORDER BY id"

	rows, err := s.queryMaps(query, args...)
	if err != nil {
		return nil, err
	}
	peaks, err := s.CustomerPeakAnalysis("")
	if err != nil {
		return nil, err
	}
	peakByCustomer := map[string]map[string]any{}
	for _, peak := range peaks {
		peakByCustomer[fmt.Sprint(peak["customer_name"])] = peak
	}
	for _, row := range rows {
		name := strings.TrimSpace(fmt.Sprint(row["actual_buyer"]))
		if name == "" {
			name = strings.TrimSpace(fmt.Sprint(row["nominal_buyer"]))
		}
		peak := peakByCustomer[name]
		if peak != nil {
			row["peak_balance"] = peak["peak_balance"]
			row["peak_diff"] = numberValue(peak["peak_balance"]) - numberValue(row["total_assets"])
		}
		row["bought_before_yanxuan"] = row["is_actual_deal"]
		row["asset_range"] = row["total_assets"]
	}
	return rows, nil
}

func (s *Store) CustomerProductsForAgent(customerName string) ([]map[string]any, error) {
	return s.queryMaps(`
		SELECT DISTINCT p.id, p.name, p.holding_status, p.manager, p.code, p.outstanding_amount,
		       l.user_name, l.actual_buyer
		FROM customer_product_link l
		LEFT JOIN products p ON p.id = l.product_id
		WHERE l.actual_buyer LIKE ? OR l.user_name LIKE ?
		ORDER BY p.issue_date DESC
	`, LikeContains(customerName), LikeContains(customerName))
}

func (s *Store) CustomerPeakAnalysis(customerName string) ([]map[string]any, error) {
	links, err := s.queryMaps(`
		SELECT l.product_id, l.user_name, l.actual_buyer, p.name, p.outstanding_amount
		FROM customer_product_link l
		LEFT JOIN products p ON p.id = l.product_id
		WHERE l.actual_buyer LIKE ? OR l.user_name LIKE ?
	`, LikeContains(customerName), LikeContains(customerName))
	if err != nil {
		return nil, err
	}

	byCustomer := map[string]map[string]any{}
	for _, link := range links {
		name := strings.TrimSpace(fmt.Sprint(link["actual_buyer"]))
		if name == "" {
			name = strings.TrimSpace(fmt.Sprint(link["user_name"]))
		}
		if name == "" {
			name = "unknown"
		}
		item, ok := byCustomer[name]
		if !ok {
			item = map[string]any{
				"customer_name":        name,
				"product_count":        int64(0),
				"peak_balance":         float64(0),
				"current_balance":      float64(0),
				"representative_items": []map[string]any{},
			}
			byCustomer[name] = item
		}

		amount := numberValue(link["outstanding_amount"])
		item["product_count"] = item["product_count"].(int64) + 1
		item["current_balance"] = item["current_balance"].(float64) + amount
		if amount > item["peak_balance"].(float64) {
			item["peak_balance"] = amount
		}
		items := item["representative_items"].([]map[string]any)
		if len(items) < 5 {
			item["representative_items"] = append(items, map[string]any{
				"product_id":         link["product_id"],
				"product_name":       link["name"],
				"outstanding_amount": link["outstanding_amount"],
			})
		}
	}

	result := make([]map[string]any, 0, len(byCustomer))
	for _, item := range byCustomer {
		result = append(result, item)
	}
	return result, nil
}

func (s *Store) QueryTransactionsForAgent(productID, counterparty, startDate, endDate string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 100
	}
	query := "SELECT transaction_date, flight_id, counterparty, subscribe_amount FROM transactions WHERE 1=1"
	args := []any{}
	if productID != "" {
		query += " AND flight_id = ?"
		args = append(args, productID)
	}
	if counterparty != "" {
		query += " AND counterparty LIKE ?"
		args = append(args, LikeContains(counterparty))
	}
	if startDate != "" {
		query += " AND transaction_date >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND transaction_date <= ?"
		args = append(args, endDate)
	}
	query += " ORDER BY transaction_date DESC LIMIT ?"
	args = append(args, limit)
	return s.queryMaps(query, args...)
}

func (s *Store) ProductAnalytics(groupBy string) ([]map[string]any, error) {
	column := map[string]string{
		"manager":        "manager",
		"holding_status": "holding_status",
		"structure_type": "structure_type",
		"issue_month":    "strftime('%Y-%m', issue_date)",
	}[groupBy]
	if column == "" {
		column = "manager"
	}
	return s.queryMaps(fmt.Sprintf(`
		SELECT %s AS group_key,
		       COUNT(*) AS product_count,
		       SUM(COALESCE(subscribe_amount, 0)) AS subscribe_amount,
		       SUM(COALESCE(outstanding_amount, 0)) AS outstanding_amount
		FROM products
		GROUP BY group_key
		ORDER BY outstanding_amount DESC, product_count DESC
		LIMIT 50
	`, column))
}

func (s *Store) AllProductDocs() ([]map[string]any, error) {
	query := `SELECT doc_token, doc_name, parent_path, folder_token, raw_content, structure_json, synced_at
		FROM product_docs ORDER BY parent_path, doc_name`
	return s.queryMaps(query)
}

func (s *Store) SearchProductDocs(keyword, month string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT doc_token, doc_name, parent_path, folder_token, substr(raw_content, 1, 500) AS content_preview, synced_at
		FROM product_docs WHERE 1=1
	`
	args := []any{}
	if keyword != "" {
		like := LikeContains(keyword)
		query += " AND (doc_name LIKE ? OR raw_content LIKE ? OR parent_path LIKE ?)"
		args = append(args, like, like, like)
	}
	if month != "" {
		query += " AND parent_path LIKE ?"
		args = append(args, LikeContains(month))
	}
	query += " ORDER BY parent_path, doc_name LIMIT ?"
	args = append(args, limit)
	return s.queryMaps(query, args...)
}

func (s *Store) ProductDocs(month string) ([]map[string]any, error) {
	query := `
		SELECT doc_token, doc_name, parent_path, folder_token, raw_content, structure_json, synced_at
		FROM product_docs
	`
	args := []any{}
	if month != "" {
		query += " WHERE parent_path LIKE ? OR doc_name LIKE ?"
		like := LikeContains(month)
		args = append(args, like, like)
	}
	query += " ORDER BY parent_path, doc_name"
	rows, err := s.queryMaps(query, args...)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if raw, ok := row["structure_json"].(string); ok && strings.TrimSpace(raw) != "" {
			var structured any
			if err := json.Unmarshal([]byte(raw), &structured); err == nil {
				row["structured"] = structured
			}
		}
	}
	return rows, nil
}

func (s *Store) ChannelsForAgent() ([]map[string]any, []map[string]any, error) {
	channels, err := s.queryMaps("SELECT id, channel_name FROM channels ORDER BY id")
	if err != nil {
		return nil, nil, err
	}
	sources, err := s.queryMaps("SELECT id, source_name FROM direct_customer_sources ORDER BY id")
	if err != nil {
		return nil, nil, err
	}
	return channels, sources, nil
}

func (s *Store) SyncStatusForAgent() (map[string]any, error) {
	regular, err := s.LastSync()
	if err != nil {
		return nil, err
	}
	coinvest, err := s.LastCoInvestSync()
	if err != nil {
		return nil, err
	}
	docs, err := s.LastProductDocsSync()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"products":      regular,
		"co_invest":     coinvest,
		"product_docs":  docs,
		"generated_at":  isoNow(),
		"database_path": s.Path,
	}, nil
}

func (s *Store) AgentConversations() ([]model.AgentConversation, error) {
	rows, err := s.DB.Query("SELECT id, title, created_at, updated_at FROM agent_conversations ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.AgentConversation{}
	for rows.Next() {
		var row model.AgentConversation
		if err := rows.Scan(&row.ID, &row.Title, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *Store) CreateAgentConversation(title string) (int64, error) {
	if strings.TrimSpace(title) == "" {
		title = "新对话"
	}
	now := isoNow()
	result, err := s.DB.Exec(
		"INSERT INTO agent_conversations (title, created_at, updated_at) VALUES (?, ?, ?)",
		title, now, now,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) UpdateAgentConversationTitle(id int64, title string) error {
	_, err := s.DB.Exec(
		"UPDATE agent_conversations SET title = ?, updated_at = ? WHERE id = ?",
		title, isoNow(), id,
	)
	return err
}

func (s *Store) DeleteAgentConversation(id int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM agent_messages WHERE conversation_id = ?", id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM agent_conversations WHERE id = ?", id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) AgentMessages(conversationID int64) ([]model.AgentMessage, error) {
	rows, err := s.DB.Query(`
		SELECT id, conversation_id, role, content, tool_calls, tool_call_id, created_at
		FROM agent_messages WHERE conversation_id = ? ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.AgentMessage{}
	for rows.Next() {
		var row model.AgentMessage
		var toolCalls, toolCallID sql.NullString
		if err := rows.Scan(&row.ID, &row.ConversationID, &row.Role, &row.Content, &toolCalls, &toolCallID, &row.CreatedAt); err != nil {
			return nil, err
		}
		row.ToolCalls = nullString(toolCalls)
		row.ToolCallID = nullString(toolCallID)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *Store) AddAgentMessage(conversationID int64, role string, content string, toolCalls string, toolCallID string) error {
	now := isoNow()
	_, err := s.DB.Exec(`
		INSERT INTO agent_messages (conversation_id, role, content, tool_calls, tool_call_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, conversationID, role, content, nullableDBString(toolCalls), nullableDBString(toolCallID), now)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec("UPDATE agent_conversations SET updated_at = ? WHERE id = ?", now, conversationID)
	return err
}

func (s *Store) AgentMessageCount(conversationID int64) (int64, error) {
	var count int64
	err := s.DB.QueryRow("SELECT COUNT(*) FROM agent_messages WHERE conversation_id = ?", conversationID).Scan(&count)
	return count, err
}

func (s *Store) queryPosters(query string, args ...any) ([]model.Poster, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.Poster{}
	for rows.Next() {
		var row model.Poster
		var months, duration, dividendCount sql.NullInt64
		var absoluteReturn, annualizedReturn, cumulativeRate, monthlyCoupon sql.NullFloat64
		if err := rows.Scan(
			&row.ID, &row.ProductID, &row.PosterType, &row.ObservationDate, &row.ProductName,
			&row.DateDisplay, &months, &row.UnderlyingName, &absoluteReturn, &annualizedReturn,
			&duration, &row.ParachuteValue, &row.KnockoutValue, &row.DividendBarrierValue,
			&dividendCount, &cumulativeRate, &monthlyCoupon, &row.EntryDate, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		row.MonthsSinceEntry = intPtr(months)
		row.DurationMonths = intPtr(duration)
		row.DividendCount = intPtr(dividendCount)
		row.AbsoluteReturn = floatPtr(absoluteReturn)
		row.AnnualizedReturn = floatPtr(annualizedReturn)
		row.CumulativeRate = floatPtr(cumulativeRate)
		row.MonthlyCoupon = floatPtr(monthlyCoupon)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *Store) scanProducts(query string, args ...any) ([]model.Product, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	result := []model.Product{}
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		v := func(name string) any {
			if i := colIndex(cols, name); i >= 0 {
				return values[i]
			}
			return nil
		}
		p := model.Product{
			ID:         toString(v("id")),
			Name:          toString(v("name")),
			IssueDate:     toString(v("issue_date")),
			CompleteDate:  toString(v("complete_date")),
			Manager:       toString(v("manager")),
			HoldingStatus: toString(v("holding_status")),
			StructureType: toString(v("structure_type")),
			Code:          toString(v("code")),
			Term:          toString(v("term")),
			Parachute:     toString(v("parachute")),
			HolidayAdjust: toString(v("holiday_adjust")),
			Raw:           toString(v("raw")),
			KnockIn:       toString(v("knock_in")),
			KnockedIn:     toString(v("knocked_in")),
			Custodian:     toString(v("custodian")),
			Counterparty:  toString(v("counterparty")),
			IsMain:          toIntPtr(v("is_main")),
			LockDays:        toIntPtr(v("lock_days")),
			LockMonths:      toIntPtr(v("lock_months")),
			SubscribeAmount:   toFloatPtr(v("subscribe_amount")),
			OutstandingAmount: toFloatPtr(v("outstanding_amount")),
			FirstKnockoutRatio: toFloatPtr(v("first_knockout_ratio")),
			EntryPrice:         toFloatPtr(v("entry_price")),
			MonthlyDecrease:    toFloatPtr(v("monthly_decrease")),
			DividendBarrier:    toFloatPtr(v("dividend_barrier")),
			MonthlyCoupon:      toFloatPtr(v("monthly_coupon")),
			Coupon1st:          toFloatPtr(v("coupon_1st")),
			Coupon2nd:          toFloatPtr(v("coupon_2nd")),
			Coupon3rd:          toFloatPtr(v("coupon_3rd")),
			DurationMonths:     toFloatPtr(v("duration_months")),
			AbsoluteReturn:     toFloatPtr(v("absolute_return")),
			DurationDays:       toIntPtr(v("duration_days")),
			MarginRatio:        toFloatPtr(v("margin_ratio")),
		}
		p.FirstKnockoutRatio = normalizeFirstKnockoutRatioPtr(p.FirstKnockoutRatio, p.EntryPrice)
		result = append(result, p)
	}
	return result, rows.Err()
}

func normalizeFirstKnockoutRatioPtr(value, entryPrice *float64) *float64 {
	if value == nil || entryPrice == nil {
		return value
	}
	if *value <= 0 || *entryPrice <= 0 {
		return value
	}
	if *value > 2 || *value < 1 {
		normalized := *value / *entryPrice
		return &normalized
	}
	return value
}

func (s *Store) queryOneMap(query string, args ...any) (map[string]any, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]any, len(columns))
	scanTargets := make([]any, len(columns))
	for i := range values {
		scanTargets[i] = &values[i]
	}
	if err := rows.Scan(scanTargets...); err != nil {
		return nil, err
	}
	result := map[string]any{}
	for i, column := range columns {
		result[column] = normalizeDBValue(values[i])
	}
	return result, rows.Err()
}

func (s *Store) queryMaps(query string, args ...any) ([]map[string]any, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		scanTargets := make([]any, len(columns))
		for i := range values {
			scanTargets[i] = &values[i]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, err
		}
		item := map[string]any{}
		for i, column := range columns {
			item[column] = normalizeDBValue(values[i])
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func normalizeDBValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func mustJSONString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	data, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseJSONStringSlice(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return nil
	}
	return out
}

func intPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}

func floatPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullFloat(value sql.NullFloat64) float64 {
	if !value.Valid {
		return 0
	}
	return value.Float64
}

func numberValue(value any) float64 {
	switch v := value.(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	case []byte:
		var out float64
		_, _ = fmt.Sscanf(string(v), "%f", &out)
		return out
	case string:
		var out float64
		_, _ = fmt.Sscanf(v, "%f", &out)
		return out
	default:
		return 0
	}
}

func LikeContains(value string) string {
	return fmt.Sprintf("%%%s%%", strings.TrimSpace(value))
}

func nullableDBString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func isoNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

type ProductFilter struct {
	IssueDateStart    string
	IssueDateEnd      string
	HoldingStatus     string
	Manager           string
	CompleteDateStart string
	CompleteDateEnd   string
	Code              string
	StructureType     string
	LockMonths        string
	MarginRatio       string
}

func (s *Store) QueryHoldingProducts(f ProductFilter) ([]model.Product, error) {
	query := "SELECT * FROM products WHERE 1=1"
	args := []any{}
	if f.IssueDateStart != "" {
		query += " AND issue_date >= ?"
		args = append(args, f.IssueDateStart)
	}
	if f.IssueDateEnd != "" {
		query += " AND issue_date <= ?"
		args = append(args, f.IssueDateEnd)
	}
	if f.HoldingStatus != "" {
		query, args = appendHoldingStatusClause(query, args, f.HoldingStatus)
	}
	if f.Manager != "" {
		query += " AND manager LIKE ?"
		args = append(args, "%"+f.Manager+"%")
	}
	if f.CompleteDateStart != "" {
		query += " AND complete_date >= ?"
		args = append(args, f.CompleteDateStart)
	}
	if f.CompleteDateEnd != "" {
		query += " AND complete_date <= ?"
		args = append(args, f.CompleteDateEnd)
	}
	if f.Code != "" {
		query += " AND code LIKE ?"
		args = append(args, "%"+f.Code+"%")
	}
	if f.StructureType != "" {
		query += " AND structure_type = ?"
		args = append(args, f.StructureType)
	}
	if f.LockMonths != "" {
		query += " AND lock_months = ?"
		args = append(args, f.LockMonths)
	}
	if f.MarginRatio != "" {
		query += " AND margin_ratio = ?"
		args = append(args, f.MarginRatio)
	}
	query += " ORDER BY issue_date DESC"
	return s.scanProducts(query, args...)
}

type TransactionFilter struct {
	CustomerName      string
	ActualBuyer       string
	MatchType         string
	RebateTarget      string
	HoldingStatus     string
	FlightDateStart   string
	FlightDateEnd     string
	CompleteDateStart string
	CompleteDateEnd   string
	ProductName       string
}

func (s *Store) QueryHoldingTransactions(f TransactionFilter) ([]model.TransactionRow, error) {
	query := "SELECT * FROM transactions WHERE 1=1"
	args := []any{}

	if f.CustomerName != "" {
		pattern := "%" + f.CustomerName + "%"
		if f.MatchType == "name_only" {
			query += " AND customer_name LIKE ?"
			args = append(args, pattern)
		} else if f.MatchType == "buyer_only" {
			query += " AND actual_buyer LIKE ?"
			args = append(args, pattern)
		} else {
			query += " AND (customer_name LIKE ? OR actual_buyer LIKE ?)"
			args = append(args, pattern, pattern)
		}
	}
	if f.RebateTarget != "" {
		query += " AND rebate_target = ?"
		args = append(args, f.RebateTarget)
	}
	if f.HoldingStatus != "" {
		query, args = appendHoldingStatusClause(query, args, f.HoldingStatus)
	}
	if f.FlightDateStart != "" {
		query += " AND flight_date >= ?"
		args = append(args, f.FlightDateStart)
	}
	if f.FlightDateEnd != "" {
		query += " AND flight_date <= ?"
		args = append(args, f.FlightDateEnd)
	}
	if f.CompleteDateStart != "" {
		query += " AND complete_date >= ?"
		args = append(args, f.CompleteDateStart)
	}
	if f.CompleteDateEnd != "" {
		query += " AND complete_date <= ?"
		args = append(args, f.CompleteDateEnd)
	}
	if f.ProductName != "" {
		query += " AND product_name LIKE ?"
		args = append(args, "%"+f.ProductName+"%")
	}
	query += " ORDER BY flight_date DESC"
	return s.scanTransactions(query, args...)
}

func appendHoldingStatusClause(query string, args []any, value string) (string, []any) {
	value = strings.TrimSpace(value)
	if value == "" {
		return query, args
	}
	if strings.Contains(value, "已完结") || strings.Contains(value, "完结") {
		query += " AND (holding_status LIKE ? OR holding_status LIKE ? OR holding_status LIKE ? OR holding_status = ?)"
		args = append(args, "%完结%", "%已完结%", "%瀹岀粨%", value)
		return query, args
	}
	query += " AND holding_status = ?"
	args = append(args, value)
	return query, args
}

func (s *Store) scanTransactions(query string, args ...any) ([]model.TransactionRow, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []model.TransactionRow
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := model.TransactionRow{
			ID:                  toInt64(values[colIndex(cols, "id")]),
			TransactionDate:     toString(values[colIndex(cols, "transaction_date")]),
			FlightID:            toString(values[colIndex(cols, "flight_id")]),
			Counterparty:        toString(values[colIndex(cols, "counterparty")]),
			SubscribeAmount:     toFloatPtr(values[colIndex(cols, "subscribe_amount")]),
			ProductName:         toString(values[colIndex(cols, "product_name")]),
			CustomerName:        toString(values[colIndex(cols, "customer_name")]),
			ActualBuyer:         toString(values[colIndex(cols, "actual_buyer")]),
			Amount:              toFloatPtr(values[colIndex(cols, "amount")]),
			SubscribeFeeRatio:   toFloatPtr(values[colIndex(cols, "subscribe_fee_ratio")]),
			ManagementFeeRatio:  toFloatPtr(values[colIndex(cols, "management_fee_ratio")]),
			PerformanceFeeRatio: toFloatPtr(values[colIndex(cols, "performance_fee_ratio")]),
			RebateTarget:        toString(values[colIndex(cols, "rebate_target")]),
			FlightDate:          toString(values[colIndex(cols, "flight_date")]),
			HoldingStatus:       toString(values[colIndex(cols, "holding_status")]),
			CompleteDate:        toString(values[colIndex(cols, "complete_date")]),
			Underlying:          toString(values[colIndex(cols, "underlying")]),
			StructureType:       toString(values[colIndex(cols, "structure_type")]),
			LockPeriod:          toString(values[colIndex(cols, "lock_period")]),
			DividendBarrier:     toFloatPtr(values[colIndex(cols, "dividend_barrier")]),
			MonthlyCoupon:       toFloatPtr(values[colIndex(cols, "monthly_coupon")]),
			Coupon1st:           toFloatPtr(values[colIndex(cols, "coupon_1st")]),
			Raw:                 toString(values[colIndex(cols, "raw")]),
			OrderID:             toString(values[colIndex(cols, "order_id")]),
		}
		results = append(results, row)
	}
	return results, nil
}

func colIndex(cols []string, name string) int {
	for i, c := range cols {
		if c == name {
			return i
		}
	}
	return -1
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case []byte:
		var out int64
		fmt.Sscanf(string(val), "%d", &out)
		return out
	case string:
		var out int64
		fmt.Sscanf(val, "%d", &out)
		return out
	default:
		return 0
	}
}

func toString(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprint(val)
	}
}

func toFloatPtr(v any) *float64 {
	switch val := v.(type) {
	case nil:
		return nil
	case float64:
		return &val
	case int64:
		f := float64(val)
		return &f
	case []byte:
		var out float64
		if _, err := fmt.Sscanf(string(val), "%f", &out); err == nil {
			return &out
		}
		return nil
	case string:
		var out float64
		if _, err := fmt.Sscanf(val, "%f", &out); err == nil {
			return &out
		}
		return nil
	default:
		return nil
	}
}

func toIntPtr(v any) *int {
	switch val := v.(type) {
	case nil:
		return nil
	case int64:
		i := int(val)
		return &i
	case float64:
		i := int(val)
		return &i
	case []byte:
		var out int
		if _, err := fmt.Sscanf(string(val), "%d", &out); err == nil {
			return &out
		}
		return nil
	case string:
		var out int
		if _, err := fmt.Sscanf(val, "%d", &out); err == nil {
			return &out
		}
		return nil
	default:
		return nil
	}
}

func (s *Store) QueryPendingRebates(filters map[string]string) ([]map[string]any, error) {
	query := `
		SELECT t.id, t.order_id, t.flight_id, t.product_name, t.customer_name,
		       t.counterparty, t.channel_or_direct, t.subscribe_amount, t.amount, t.rebate_target,
		       t.subscribe_fee_rate, t.subscribe_fee_ratio, t.management_fee_ratio, t.performance_fee_ratio,
		       t.management_fee_received, t.performance_fee_receivable,
		       t.tax_subscribe_ratio, t.tax_management_ratio, t.tax_performance_ratio,
		       t.flight_date, t.holding_status, t.complete_date,
		       rs.is_returnable, rs.plan_subscribe, rs.plan_management, rs.plan_performance,
		       rpm.order_id AS manual_order_id,
		       rpm.principal AS manual_principal,
		       rpm.subscribe_receivable AS manual_subscribe_receivable,
		       rpm.management_fee_received AS manual_management_fee_received,
		       rpm.performance_fee_receivable AS manual_performance_fee_receivable,
		       rpm.subscribe_fee_ratio AS manual_subscribe_fee_ratio,
		       rpm.management_fee_ratio AS manual_management_fee_ratio,
		       rpm.performance_fee_ratio AS manual_performance_fee_ratio,
		       rpm.tax_subscribe_ratio AS manual_tax_subscribe_ratio,
		       rpm.tax_management_ratio AS manual_tax_management_ratio,
		       rpm.tax_performance_ratio AS manual_tax_performance_ratio,
		       rpm.expected_subscribe AS manual_expected_subscribe,
		       rpm.expected_management AS manual_expected_management,
		       rpm.expected_performance AS manual_expected_performance,
		       rpm.returned_subscribe AS manual_returned_subscribe,
		       rpm.returned_management AS manual_returned_management,
		       rpm.returned_performance AS manual_returned_performance,
		       rpm.outstanding_subscribe AS manual_outstanding_subscribe,
		       rpm.outstanding_management AS manual_outstanding_management,
		       rpm.outstanding_performance AS manual_outstanding_performance,
		       rpm.is_returnable AS manual_is_returnable,
		       EXISTS(SELECT 1 FROM rebate_completed rc WHERE rc.order_id = t.order_id AND rc.expense_category = '申购费') AS has_completed_subscribe,
		       EXISTS(SELECT 1 FROM rebate_completed rc WHERE rc.order_id = t.order_id AND rc.expense_category = '管理费') AS has_completed_management,
		       EXISTS(SELECT 1 FROM rebate_completed rc WHERE rc.order_id = t.order_id AND rc.expense_category = '业绩报酬') AS has_completed_performance,
		       COALESCE((SELECT SUM(rc.expense_amount) FROM rebate_completed rc WHERE rc.order_id = t.order_id AND rc.expense_category = '申购费'), 0) AS returned_subscribe,
		       COALESCE((SELECT SUM(rc.expense_amount) FROM rebate_completed rc WHERE rc.order_id = t.order_id AND rc.expense_category = '管理费'), 0) AS returned_management,
		       COALESCE((SELECT SUM(rc.expense_amount) FROM rebate_completed rc WHERE rc.order_id = t.order_id AND rc.expense_category = '业绩报酬'), 0) AS returned_performance
		FROM transactions t
		LEFT JOIN rebate_status rs ON t.order_id = rs.order_id
		LEFT JOIN rebate_pending_manual rpm ON t.order_id = rpm.order_id
		WHERE t.order_id IS NOT NULL AND t.order_id != ''
		  AND t.rebate_target IS NOT NULL AND TRIM(t.rebate_target) != ''
		  AND TRIM(t.rebate_target) NOT IN ('-', '--', '0')
	`
	args := []any{}
	if v, ok := filters["customer_name"]; ok && v != "" {
		query += " AND t.customer_name LIKE ?"
		args = append(args, LikeContains(v))
	}
	if v, ok := filters["rebate_target"]; ok && v != "" {
		query += " AND t.rebate_target LIKE ?"
		args = append(args, LikeContains(v))
	}
	if v, ok := filters["order_id"]; ok && v != "" {
		query += " AND t.order_id LIKE ?"
		args = append(args, LikeContains(v))
	}
	if v, ok := filters["flight_id"]; ok && v != "" {
		query += " AND t.flight_id LIKE ?"
		args = append(args, LikeContains(v))
	}
	if v, ok := filters["product_name"]; ok && v != "" {
		query += " AND t.product_name LIKE ?"
		args = append(args, LikeContains(v))
	}
	query += " ORDER BY t.flight_date DESC"
	return s.queryMaps(query, args...)
}

func (s *Store) GetTransactionByOrderID(orderID string) (*model.TransactionRow, error) {
	rows, err := s.DB.Query(`
		SELECT id, transaction_date, flight_id, counterparty, channel_or_direct, subscribe_amount, subscribe_fee_rate,
		       product_name, customer_name, actual_buyer, amount,
		       subscribe_fee_ratio, management_fee_ratio, performance_fee_ratio,
		       rebate_target, flight_date, holding_status, complete_date,
		       underlying, structure_type, lock_period, dividend_barrier, monthly_coupon, coupon_1st, raw, order_id
		FROM transactions
		WHERE order_id = ?
		LIMIT 1
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}

	var row model.TransactionRow
	var subscribeAmount, subscribeFeeRate, amount, subscribeFeeRatio sql.NullFloat64
	var managementFeeRatio, performanceFeeRatio, dividendBarrier, monthlyCoupon, coupon1st sql.NullFloat64
	var transactionDate, flightID, counterparty, channelOrDirect, productName, customerName sql.NullString
	var actualBuyer, rebateTarget, flightDate, holdingStatus, completeDate sql.NullString
	var underlying, structureType, lockPeriod, raw, storedOrderID sql.NullString
	if err := rows.Scan(
		&row.ID, &transactionDate, &flightID, &counterparty, &channelOrDirect, &subscribeAmount, &subscribeFeeRate,
		&productName, &customerName, &actualBuyer, &amount,
		&subscribeFeeRatio, &managementFeeRatio, &performanceFeeRatio,
		&rebateTarget, &flightDate, &holdingStatus, &completeDate,
		&underlying, &structureType, &lockPeriod, &dividendBarrier, &monthlyCoupon, &coupon1st, &raw, &storedOrderID,
	); err != nil {
		return nil, err
	}

	row.TransactionDate = nullString(transactionDate)
	row.FlightID = nullString(flightID)
	row.Counterparty = nullString(counterparty)
	row.ChannelOrDirect = nullString(channelOrDirect)
	row.SubscribeAmount = floatPtr(subscribeAmount)
	row.SubscribeFeeRate = floatPtr(subscribeFeeRate)
	row.ProductName = nullString(productName)
	row.CustomerName = nullString(customerName)
	row.ActualBuyer = nullString(actualBuyer)
	row.Amount = floatPtr(amount)
	row.SubscribeFeeRatio = floatPtr(subscribeFeeRatio)
	row.ManagementFeeRatio = floatPtr(managementFeeRatio)
	row.PerformanceFeeRatio = floatPtr(performanceFeeRatio)
	row.RebateTarget = nullString(rebateTarget)
	row.FlightDate = nullString(flightDate)
	row.HoldingStatus = nullString(holdingStatus)
	row.CompleteDate = nullString(completeDate)
	row.Underlying = nullString(underlying)
	row.StructureType = nullString(structureType)
	row.LockPeriod = nullString(lockPeriod)
	row.DividendBarrier = floatPtr(dividendBarrier)
	row.MonthlyCoupon = floatPtr(monthlyCoupon)
	row.Coupon1st = floatPtr(coupon1st)
	row.Raw = nullString(raw)
	row.OrderID = nullString(storedOrderID)
	return &row, rows.Err()
}

func (s *Store) FindMatchingTransactions(record model.RebateCompleted) ([]model.TransactionRow, error) {
	query := `
		SELECT id, transaction_date, flight_id, counterparty, channel_or_direct, subscribe_amount, subscribe_fee_rate,
		       product_name, customer_name, actual_buyer, amount,
		       subscribe_fee_ratio, management_fee_ratio, performance_fee_ratio,
		       rebate_target, flight_date, holding_status, complete_date,
		       underlying, structure_type, lock_period, dividend_barrier, monthly_coupon, coupon_1st, raw, order_id
		FROM transactions
		WHERE 1=1
	`
	args := []any{}

	if strings.TrimSpace(record.OrderID) != "" {
		query += " AND order_id = ?"
		args = append(args, strings.TrimSpace(record.OrderID))
	}
	if strings.TrimSpace(record.FlightID) != "" {
		query += " AND flight_id = ?"
		args = append(args, strings.TrimSpace(record.FlightID))
	}
	if strings.TrimSpace(record.ProductName) != "" {
		query += " AND product_name = ?"
		args = append(args, strings.TrimSpace(record.ProductName))
	}
	if strings.TrimSpace(record.CustomerName) != "" {
		query += " AND customer_name = ?"
		args = append(args, strings.TrimSpace(record.CustomerName))
	}
	if strings.TrimSpace(record.RebateTarget) != "" {
		query += " AND rebate_target = ?"
		args = append(args, strings.TrimSpace(record.RebateTarget))
	}
	if record.Principal != nil {
		query += " AND subscribe_amount = ?"
		args = append(args, *record.Principal)
	}

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.TransactionRow
	for rows.Next() {
		var row model.TransactionRow
		var subscribeAmount, subscribeFeeRate, amount, subscribeFeeRatio sql.NullFloat64
		var managementFeeRatio, performanceFeeRatio, dividendBarrier, monthlyCoupon, coupon1st sql.NullFloat64
		var transactionDate, flightID, counterparty, channelOrDirect, productName, customerName sql.NullString
		var actualBuyer, rebateTarget, flightDate, holdingStatus, completeDate sql.NullString
		var underlying, structureType, lockPeriod, raw, storedOrderID sql.NullString
		if err := rows.Scan(
			&row.ID, &transactionDate, &flightID, &counterparty, &channelOrDirect, &subscribeAmount, &subscribeFeeRate,
			&productName, &customerName, &actualBuyer, &amount,
			&subscribeFeeRatio, &managementFeeRatio, &performanceFeeRatio,
			&rebateTarget, &flightDate, &holdingStatus, &completeDate,
			&underlying, &structureType, &lockPeriod, &dividendBarrier, &monthlyCoupon, &coupon1st, &raw, &storedOrderID,
		); err != nil {
			return nil, err
		}
		row.TransactionDate = nullString(transactionDate)
		row.FlightID = nullString(flightID)
		row.Counterparty = nullString(counterparty)
		row.ChannelOrDirect = nullString(channelOrDirect)
		row.SubscribeAmount = floatPtr(subscribeAmount)
		row.SubscribeFeeRate = floatPtr(subscribeFeeRate)
		row.ProductName = nullString(productName)
		row.CustomerName = nullString(customerName)
		row.ActualBuyer = nullString(actualBuyer)
		row.Amount = floatPtr(amount)
		row.SubscribeFeeRatio = floatPtr(subscribeFeeRatio)
		row.ManagementFeeRatio = floatPtr(managementFeeRatio)
		row.PerformanceFeeRatio = floatPtr(performanceFeeRatio)
		row.RebateTarget = nullString(rebateTarget)
		row.FlightDate = nullString(flightDate)
		row.HoldingStatus = nullString(holdingStatus)
		row.CompleteDate = nullString(completeDate)
		row.Underlying = nullString(underlying)
		row.StructureType = nullString(structureType)
		row.LockPeriod = nullString(lockPeriod)
		row.DividendBarrier = floatPtr(dividendBarrier)
		row.MonthlyCoupon = floatPtr(monthlyCoupon)
		row.Coupon1st = floatPtr(coupon1st)
		row.Raw = nullString(raw)
		row.OrderID = nullString(storedOrderID)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *Store) GetProductByID(id string) (*model.Product, error) {
	products, err := s.scanProducts("SELECT * FROM products WHERE id = ? LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, nil
	}
	return &products[0], nil
}

func (s *Store) UpsertRebateStatus(orderID string, fields map[string]any) error {
	setClauses := []string{}
	args := []any{}
	insertCols := []string{"order_id"}
	insertPlaceholders := []string{"?"}
	insertArgs := []any{orderID}

	if v, ok := fields["is_returnable"]; ok {
		setClauses = append(setClauses, "is_returnable = ?")
		args = append(args, v)
		insertCols = append(insertCols, "is_returnable")
		insertPlaceholders = append(insertPlaceholders, "?")
		insertArgs = append(insertArgs, v)
	}
	if v, ok := fields["plan_subscribe"]; ok {
		setClauses = append(setClauses, "plan_subscribe = ?")
		args = append(args, v)
		insertCols = append(insertCols, "plan_subscribe")
		insertPlaceholders = append(insertPlaceholders, "?")
		insertArgs = append(insertArgs, v)
	}
	if v, ok := fields["plan_management"]; ok {
		setClauses = append(setClauses, "plan_management = ?")
		args = append(args, v)
		insertCols = append(insertCols, "plan_management")
		insertPlaceholders = append(insertPlaceholders, "?")
		insertArgs = append(insertArgs, v)
	}
	if v, ok := fields["plan_performance"]; ok {
		setClauses = append(setClauses, "plan_performance = ?")
		args = append(args, v)
		insertCols = append(insertCols, "plan_performance")
		insertPlaceholders = append(insertPlaceholders, "?")
		insertArgs = append(insertArgs, v)
	}
	setClauses = append(setClauses, "updated_at = datetime('now')")

	query := fmt.Sprintf(
		"INSERT INTO rebate_status (%s) VALUES (%s) ON CONFLICT(order_id) DO UPDATE SET %s",
		strings.Join(insertCols, ", "),
		strings.Join(insertPlaceholders, ", "),
		strings.Join(setClauses, ", "),
	)
	allArgs := append(insertArgs, args...)
	_, err := s.DB.Exec(query, allArgs...)
	return err
}

func (s *Store) QueryRebateCompleted(filters map[string]string) ([]model.RebateCompleted, error) {
	query := `
		SELECT id, order_id, flight_id, product_name, customer_name, channel_or_direct,
		       principal, margin_ratio, business_type, subscribe_date, order_status,
		       rebate_target, channel_subscribe_ratio, channel_management_ratio,
		       channel_performance_ratio, expense_category, expense_amount,
		       payment_time, payment_year, payment_month, payment_day, ignored_conflicts,
		       source, created_at, updated_at
		FROM rebate_completed WHERE 1=1
	`
	args := []any{}
	if v, ok := filters["order_id"]; ok && v != "" {
		query += " AND order_id LIKE ?"
		args = append(args, LikeContains(v))
	}
	if v, ok := filters["customer_name"]; ok && v != "" {
		query += " AND customer_name LIKE ?"
		args = append(args, LikeContains(v))
	}
	if v, ok := filters["product_name"]; ok && v != "" {
		query += " AND product_name LIKE ?"
		args = append(args, LikeContains(v))
	}
	if v, ok := filters["expense_category"]; ok && v != "" {
		query += " AND expense_category = ?"
		args = append(args, v)
	}
	if v, ok := filters["source"]; ok && v != "" {
		query += " AND source = ?"
		args = append(args, v)
	}
	if v, ok := filters["flight_id"]; ok && v != "" {
		query += " AND flight_id LIKE ?"
		args = append(args, LikeContains(v))
	}
	if v, ok := filters["rebate_target"]; ok && v != "" {
		query += " AND rebate_target LIKE ?"
		args = append(args, LikeContains(v))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.RebateCompleted
	for rows.Next() {
		var row model.RebateCompleted
		var principal, marginRatio, chSubRatio, chMgmtRatio, chPerfRatio, expAmount sql.NullFloat64
		var orderID, flightID, productName, customerName, channelOrDirect sql.NullString
		var businessType, subscribeDate, orderStatus, rebateTarget sql.NullString
		var expCategory, paymentTime, paymentYear, paymentMonth, paymentDay, ignoredConflicts sql.NullString
		var source, createdAt, updatedAt sql.NullString
		if err := rows.Scan(
			&row.ID, &orderID, &flightID, &productName, &customerName, &channelOrDirect,
			&principal, &marginRatio, &businessType, &subscribeDate, &orderStatus,
			&rebateTarget, &chSubRatio, &chMgmtRatio, &chPerfRatio,
			&expCategory, &expAmount,
			&paymentTime, &paymentYear, &paymentMonth, &paymentDay, &ignoredConflicts,
			&source, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		row.OrderID = nullString(orderID)
		row.FlightID = nullString(flightID)
		row.ProductName = nullString(productName)
		row.CustomerName = nullString(customerName)
		row.ChannelOrDirect = nullString(channelOrDirect)
		row.Principal = floatPtr(principal)
		row.MarginRatio = floatPtr(marginRatio)
		row.BusinessType = nullString(businessType)
		row.SubscribeDate = nullString(subscribeDate)
		row.OrderStatus = nullString(orderStatus)
		row.RebateTarget = nullString(rebateTarget)
		row.ChannelSubscribeRatio = floatPtr(chSubRatio)
		row.ChannelManagementRatio = floatPtr(chMgmtRatio)
		row.ChannelPerformanceRatio = floatPtr(chPerfRatio)
		row.ExpenseCategory = nullString(expCategory)
		row.ExpenseAmount = floatPtr(expAmount)
		row.PaymentTime = nullString(paymentTime)
		row.PaymentYear = nullString(paymentYear)
		row.PaymentMonth = nullString(paymentMonth)
		row.PaymentDay = nullString(paymentDay)
		row.IgnoredConflicts = parseJSONStringSlice(nullString(ignoredConflicts))
		row.Source = nullString(source)
		row.CreatedAt = nullString(createdAt)
		row.UpdatedAt = nullString(updatedAt)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *Store) InsertRebateCompleted(record model.RebateCompleted) (int64, error) {
	result, err := s.DB.Exec(`
		INSERT INTO rebate_completed
			(order_id, flight_id, product_name, customer_name, channel_or_direct,
			 principal, margin_ratio, business_type, subscribe_date, order_status,
			 rebate_target, channel_subscribe_ratio, channel_management_ratio,
			 channel_performance_ratio, expense_category, expense_amount,
			 payment_time, payment_year, payment_month, payment_day, ignored_conflicts, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		nullableDBString(record.OrderID), nullableDBString(record.FlightID),
		nullableDBString(record.ProductName), nullableDBString(record.CustomerName),
		nullableDBString(record.ChannelOrDirect),
		nullableFloat(record.Principal), nullableFloat(record.MarginRatio),
		nullableDBString(record.BusinessType), nullableDBString(record.SubscribeDate),
		nullableDBString(record.OrderStatus), nullableDBString(record.RebateTarget),
		nullableFloat(record.ChannelSubscribeRatio), nullableFloat(record.ChannelManagementRatio),
		nullableFloat(record.ChannelPerformanceRatio),
		nullableDBString(record.ExpenseCategory), nullableFloat(record.ExpenseAmount),
		nullableDBString(record.PaymentTime), nullableDBString(record.PaymentYear),
		nullableDBString(record.PaymentMonth), nullableDBString(record.PaymentDay),
		nullableDBString(mustJSONString(record.IgnoredConflicts)),
		nullableDBString(record.Source),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) BulkInsertRebateCompleted(records []model.RebateCompleted) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO rebate_completed
			(order_id, flight_id, product_name, customer_name, channel_or_direct,
			 principal, margin_ratio, business_type, subscribe_date, order_status,
			 rebate_target, channel_subscribe_ratio, channel_management_ratio,
			 channel_performance_ratio, expense_category, expense_amount,
			 payment_time, payment_year, payment_month, payment_day, ignored_conflicts, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, r := range records {
		if _, err := stmt.Exec(
			nullableDBString(r.OrderID), nullableDBString(r.FlightID),
			nullableDBString(r.ProductName), nullableDBString(r.CustomerName),
			nullableDBString(r.ChannelOrDirect),
			nullableFloat(r.Principal), nullableFloat(r.MarginRatio),
			nullableDBString(r.BusinessType), nullableDBString(r.SubscribeDate),
			nullableDBString(r.OrderStatus), nullableDBString(r.RebateTarget),
			nullableFloat(r.ChannelSubscribeRatio), nullableFloat(r.ChannelManagementRatio),
			nullableFloat(r.ChannelPerformanceRatio),
			nullableDBString(r.ExpenseCategory), nullableFloat(r.ExpenseAmount),
			nullableDBString(r.PaymentTime), nullableDBString(r.PaymentYear),
			nullableDBString(r.PaymentMonth), nullableDBString(r.PaymentDay),
			nullableDBString(mustJSONString(r.IgnoredConflicts)),
			nullableDBString(r.Source),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RebateCompletedSummary(orderID string) (map[string]float64, error) {
	rows, err := s.DB.Query(`
		SELECT expense_category, COALESCE(SUM(expense_amount), 0)
		FROM rebate_completed
		WHERE order_id = ?
		GROUP BY expense_category
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]float64{
		"申购费":  0,
		"管理费":  0,
		"业绩报酬": 0,
	}
	for rows.Next() {
		var category sql.NullString
		var amount float64
		if err := rows.Scan(&category, &amount); err != nil {
			return nil, err
		}
		if category.Valid {
			result[category.String] = amount
		}
	}
	return result, rows.Err()
}

func (s *Store) DeleteRebateCompleted(id int64) error {
	_, err := s.DB.Exec("DELETE FROM rebate_completed WHERE id = ?", id)
	return err
}

func (s *Store) QueryDistinctValues(table, column string) ([]string, error) {
	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL AND %s != '' ORDER BY %s", column, table, column, column, column)
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []string
	for rows.Next() {
		var raw any
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		val := strings.TrimSpace(fmt.Sprint(raw))
		if val == "" || val == "<nil>" {
			continue
		}
		results = append(results, val)
	}
	return results, nil
}

func (s *Store) SaveRebateDetail(rows []map[string]any, sheetName string, sheetToken string) error {
	rawJSON, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	now := isoNow()
	_, err = s.DB.Exec(`
		INSERT INTO rebate_detail_data (id, raw_json, sheet_name, sheet_token, updated_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			raw_json = excluded.raw_json,
			sheet_name = excluded.sheet_name,
			sheet_token = excluded.sheet_token,
			updated_at = excluded.updated_at
	`, string(rawJSON), sheetName, sheetToken, now)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`
		INSERT INTO rebate_detail_sync_log (synced_at, row_count, sheet_name, sheet_token)
		VALUES (?, ?, ?, ?)
	`, now, len(rows), sheetName, sheetToken)
	return err
}

func (s *Store) LastRebateDetailSync() (map[string]any, error) {
	row, err := s.queryOneMap(`
		SELECT synced_at, row_count, sheet_name, sheet_token
		FROM rebate_detail_sync_log ORDER BY synced_at DESC LIMIT 1
	`)
	if err != nil || row == nil {
		return nil, err
	}
	return row, nil
}

func (s *Store) RebateDetailData() ([]byte, error) {
	row, err := s.queryOneMap(`SELECT raw_json FROM rebate_detail_data WHERE id = 1`)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return []byte("[]"), nil
	}
	raw, ok := row["raw_json"].(string)
	if !ok {
		return []byte("[]"), nil
	}
	return []byte(raw), nil
}
