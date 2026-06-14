package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/db"
	"business-workbench/backend-go/internal/model"
	"business-workbench/backend-go/internal/observations"
	"business-workbench/backend-go/internal/retriever"
)

const maxToolRounds = 12

const systemPrompt = "你是一个专业的金融结构化产品业务助手，服务于业务工作台系统。请使用中文回答，优先基于系统内已有业务数据和用户问题给出简洁、准确的回复。需要查询产品、客户、交易、观察日历、投顾材料或业务统计时，主动调用可用工具。\n\n搜索产品时请注意：产品名称（name）通常是「航班服务XX号」这样的格式，标的指数或挂钩标的可能在标的代码（code）字段中。如果按产品名称搜索未果，请尝试用标的关键词搜索，例如用「中证1000」「沪深300」「恒科」「中证500」等关键词。也可以先调用 get_product_analytics 查看有哪些不同的标的和结构类型，再针对性搜索。"

type Service struct {
	cfg    config.Config
	store  *db.Store
	client *http.Client
}

type StreamCallbacks struct {
	OnReasoning func(string)
	OnDelta     func(string)
	OnToolCall  func(string)
	OnToolDone  func(string)
}

func NewService(cfg config.Config, store *db.Store) *Service {
	return &Service{cfg: cfg, store: store, client: http.DefaultClient}
}

func (s *Service) StreamChat(ctx context.Context, history []model.AgentMessage, userMessage string, callbacks StreamCallbacks) (string, error) {
	if strings.TrimSpace(s.cfg.DeepSeekAPIKey) == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY not configured")
	}

	docContext := ""
	allDocs, err := s.store.AllProductDocs()
	if err == nil && len(allDocs) > 0 {
		scored := retriever.SearchDocs(allDocs, userMessage, 5)
		docContext = retriever.BuildDocContext(scored)
	}
	messages := buildMessages(history, userMessage, docContext)
	var finalContent string

	for round := 0; round < maxToolRounds; round++ {
		result, err := s.callModel(ctx, messages, callbacks)
		if err != nil {
			return finalContent, err
		}

		if result.Content != "" {
			finalContent += result.Content
		}

		if result.FinishReason != "tool_calls" || len(result.ToolCalls) == 0 {
			return finalContent, nil
		}

		messages = append(messages, chatMessage{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		for _, toolCall := range result.ToolCalls {
			if callbacks.OnToolCall != nil {
				callbacks.OnToolCall(toolCall.Function.Name)
			}
			toolResult := s.executeTool(toolCall.Function.Name, toolCall.Function.Arguments)
			if callbacks.OnToolDone != nil {
				callbacks.OnToolDone(toolCall.Function.Name)
			}
			resultJSON, _ := json.Marshal(toolResult)
			messages = append(messages, chatMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: toolCall.ID,
			})
		}
	}

	return finalContent, fmt.Errorf("max tool call rounds exceeded")
}

func (s *Service) callModel(ctx context.Context, messages []chatMessage, callbacks StreamCallbacks) (streamResult, error) {
	body := chatRequest{
		Model:         s.cfg.DeepSeekModel,
		Messages:      messages,
		Tools:         toolDefinitions(),
		Stream:        true,
		StreamOptions: map[string]any{"include_usage": true},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return streamResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.cfg.DeepSeekAPIURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return streamResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.DeepSeekAPIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return streamResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return streamResult{}, fmt.Errorf("model API error %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	return readStream(resp.Body, callbacks)
}

func buildMessages(history []model.AgentMessage, userMessage string, docContext string) []chatMessage {
	prompt := systemPrompt
	if docContext != "" {
		prompt += "\n" + docContext
	}
	messages := []chatMessage{{Role: "system", Content: prompt}}
	for _, item := range history {
		msg := chatMessage{Role: item.Role, Content: item.Content}
		if item.ToolCallID != "" {
			msg.ToolCallID = item.ToolCallID
		}
		if item.ToolCalls != "" {
			_ = json.Unmarshal([]byte(item.ToolCalls), &msg.ToolCalls)
		}
		messages = append(messages, msg)
	}
	if len(history) == 0 || history[len(history)-1].Role != "user" || history[len(history)-1].Content != userMessage {
		messages = append(messages, chatMessage{Role: "user", Content: userMessage})
	}
	return messages
}

func readStream(body io.Reader, callbacks StreamCallbacks) (streamResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	result := streamResult{ToolCalls: []toolCall{}}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if data == "" || data == "[DONE]" {
			continue
		}

		var event chatStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Choices) == 0 {
			continue
		}

		choice := event.Choices[0]
		if choice.FinishReason != "" {
			result.FinishReason = choice.FinishReason
		}

		delta := choice.Delta
		if delta.ReasoningContent != "" && callbacks.OnReasoning != nil {
			callbacks.OnReasoning(delta.ReasoningContent)
		}
		if delta.Content != "" {
			result.Content += delta.Content
			if callbacks.OnDelta != nil {
				callbacks.OnDelta(delta.Content)
			}
		}
		for _, tc := range delta.ToolCalls {
			result.mergeToolCall(tc)
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func (r *streamResult) mergeToolCall(delta toolCallDelta) {
	for len(r.ToolCalls) <= delta.Index {
		r.ToolCalls = append(r.ToolCalls, toolCall{Type: "function"})
	}
	target := &r.ToolCalls[delta.Index]
	if delta.ID != "" {
		target.ID = delta.ID
	}
	if delta.Type != "" {
		target.Type = delta.Type
	}
	if delta.Function.Name != "" {
		target.Function.Name += delta.Function.Name
	}
	if delta.Function.Arguments != "" {
		target.Function.Arguments += delta.Function.Arguments
	}
}

func (s *Service) executeTool(name string, rawArgs string) map[string]any {
	var args map[string]any
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return map[string]any{"error": "failed to parse tool arguments: " + rawArgs}
	}

	switch name {
	case "search_products":
		return s.searchProducts(args)
	case "get_product_detail":
		return s.getProductDetail(args)
	case "get_observations":
		return s.getObservations(args)
	case "get_price":
		return s.getPrice(args)
	case "get_dashboard_stats":
		return s.getDashboardStats()
	case "get_observation_calendar":
		return s.getObservationCalendar(args)
	case "search_customers":
		return s.searchCustomers(args)
	case "get_customer_products":
		return s.getCustomerProducts(args)
	case "get_customer_peak_analysis":
		return s.getCustomerPeakAnalysis(args)
	case "query_transactions":
		return s.queryTransactions(args)
	case "get_product_analytics":
		return s.getProductAnalytics(args)
	case "get_posters":
		return s.getPosters(args)
	case "search_product_docs":
		return s.searchProductDocs(args)
	case "get_channels_summary":
		return s.getChannelsSummary()
	case "get_sync_status":
		return s.getSyncStatus()
	case "get_activity_logs":
		return s.getActivityLogs(args)
	default:
		return map[string]any{"error": "unknown tool: " + name}
	}
}

func (s *Service) searchProducts(args map[string]any) map[string]any {
	keyword := stringArg(args, "keyword")
	products, err := s.store.SearchProductsForAgent(keyword)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"count": len(products), "products": products}
}

func (s *Service) getProductDetail(args map[string]any) map[string]any {
	productID := stringArg(args, "product_id")
	product, err := s.store.ProductByID(productID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if product == nil {
		return map[string]any{"error": "product " + productID + " not found"}
	}
	return map[string]any{"product": product}
}

func (s *Service) getObservations(args map[string]any) map[string]any {
	productID := stringArg(args, "product_id")
	rows, err := s.store.QueryObservationsByProduct(productID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"product_id": productID, "count": len(rows), "observations": rows}
}

func (s *Service) getPrice(args map[string]any) map[string]any {
	code := stringArg(args, "code")
	price, err := s.store.LatestPrice(code)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if price == nil {
		return map[string]any{"code": code, "price": nil, "source": "cache", "message": "no cached price found"}
	}
	price["source"] = "cache"
	return price
}

func (s *Service) getDashboardStats() map[string]any {
	stats, err := s.store.DashboardStats()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return stats
}

func (s *Service) getObservationCalendar(args map[string]any) map[string]any {
	query, errMsg := normalizeCalendarQuery(args)
	if errMsg != "" {
		return map[string]any{"error": errMsg}
	}

	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	products = filterCalendarProducts(products, query)

	calendar := calendarForQuery(products, query)
	total := 0
	for _, item := range calendar {
		total += len(item.Products)
	}

	return map[string]any{
		"query": query,
		"summary": map[string]any{
			"date_count":    len(calendar),
			"product_count": total,
		},
		"calendar": calendar,
	}
}

func (s *Service) searchCustomers(args map[string]any) map[string]any {
	rows, err := s.store.SearchCustomersForAgent(
		stringArg(args, "keyword"),
		stringArg(args, "industry"),
		stringArg(args, "is_dedicated_account"),
		stringArg(args, "is_competitor"),
		intArg(args, "limit", 20),
	)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"count": len(rows), "customers": rows}
}

func (s *Service) getCustomerProducts(args map[string]any) map[string]any {
	customerName := stringArg(args, "customer_name")
	rows, err := s.store.CustomerProductsForAgent(customerName)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"customer_name": customerName, "count": len(rows), "products": rows}
}

func (s *Service) getCustomerPeakAnalysis(args map[string]any) map[string]any {
	customerName := stringArg(args, "customer_name")
	rows, err := s.store.CustomerPeakAnalysis(customerName)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"customer_name": customerName, "count": len(rows), "items": rows}
}

func (s *Service) queryTransactions(args map[string]any) map[string]any {
	rows, err := s.store.QueryTransactionsForAgent(
		stringArg(args, "product_id"),
		stringArg(args, "counterparty"),
		stringArg(args, "start_date"),
		stringArg(args, "end_date"),
		intArg(args, "limit", 100),
	)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	var total float64
	for _, row := range rows {
		total += numericArg(row, "subscribe_amount")
	}
	return map[string]any{"count": len(rows), "total_subscribe_amount": total, "transactions": rows}
}

func (s *Service) getProductAnalytics(args map[string]any) map[string]any {
	groupBy := stringArg(args, "group_by")
	rows, err := s.store.ProductAnalytics(groupBy)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if groupBy == "" {
		groupBy = "manager"
	}
	return map[string]any{"group_by": groupBy, "count": len(rows), "items": rows}
}

func (s *Service) getPosters(args map[string]any) map[string]any {
	date := stringArg(args, "date")
	productID := stringArg(args, "product_id")
	var (
		posters []model.Poster
		err     error
	)
	switch {
	case date != "":
		posters, err = s.store.QueryPostersByDate(date)
	case productID != "":
		posters, err = s.store.QueryPostersByProduct(productID)
	default:
		posters, err = s.store.QueryAllPosters()
	}
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"count": len(posters), "posters": posters}
}

func (s *Service) searchProductDocs(args map[string]any) map[string]any {
	rows, err := s.store.SearchProductDocs(
		stringArg(args, "keyword"),
		stringArg(args, "month"),
		intArg(args, "limit", 20),
	)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"count": len(rows), "documents": rows}
}

func (s *Service) getChannelsSummary() map[string]any {
	channels, sources, err := s.store.ChannelsForAgent()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{
		"channels":                map[string]any{"count": len(channels), "items": channels},
		"direct_customer_sources": map[string]any{"count": len(sources), "items": sources},
	}
}

func (s *Service) getSyncStatus() map[string]any {
	status, err := s.store.SyncStatusForAgent()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return status
}

func (s *Service) getActivityLogs(args map[string]any) map[string]any {
	rows, err := s.store.QueryActivityLogs(stringArg(args, "type"), intArg(args, "limit", 50))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"count": len(rows), "logs": rows}
}

func filterCalendarProducts(products []model.Product, query map[string]string) []model.Product {
	keyword := strings.ToLower(query["product_keyword"])
	manager := strings.ToLower(query["manager"])
	result := []model.Product{}
	for _, product := range products {
		if keyword != "" && !strings.Contains(strings.ToLower(product.Name), keyword) {
			continue
		}
		if manager != "" && !strings.Contains(strings.ToLower(product.Manager), manager) {
			continue
		}
		result = append(result, product)
	}
	return result
}

func calendarForQuery(products []model.Product, query map[string]string) []model.CalendarDay {
	months := queryMonths(query)
	dates := map[string][]model.CalendarProduct{}
	for _, product := range products {
		for _, month := range months {
			for _, obs := range observations.DatesForMonth(product, month) {
				if !matchesCalendarQuery(obs.Date, query) {
					continue
				}
				knockoutPrice := observations.ComputeKnockoutPrice(product, obs.MonthsSinceEntry)
				dividendLine := observations.ComputeDividendLine(product)
				dates[obs.Date] = append(dates[obs.Date], model.CalendarProduct{
					ID:                     product.ID,
					Name:                   product.Name,
					Manager:                product.Manager,
					Code:                   product.Code,
					MonthsSinceEntry:       obs.MonthsSinceEntry,
					EntryPrice:             product.EntryPrice,
					KnockoutPrice:          knockoutPrice,
					DividendLine:           dividendLine,
					IsKnockoutObservable:   knockoutPrice != nil,
					HasDividendObservation: product.MonthlyCoupon != nil && *product.MonthlyCoupon > 0,
				})
			}
		}
	}

	result := make([]model.CalendarDay, 0, len(dates))
	for date, products := range dates {
		sort.Slice(products, func(i, j int) bool { return products[i].Name < products[j].Name })
		result = append(result, model.CalendarDay{Date: date, Products: products})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	return result
}

func normalizeCalendarQuery(args map[string]any) (map[string]string, string) {
	month := stringArg(args, "month")
	date := stringArg(args, "date")
	startDate := stringArg(args, "start_date")
	endDate := stringArg(args, "end_date")

	query := map[string]string{
		"product_keyword": stringArg(args, "product_keyword"),
		"manager":         stringArg(args, "manager"),
	}

	if date != "" {
		query["mode"] = "date"
		query["date"] = date
		return query, ""
	}
	if startDate != "" {
		query["mode"] = "range"
		query["start_date"] = startDate
		if endDate == "" {
			endDate = startDate
		}
		if endDate < startDate {
			return nil, "end_date must be greater than or equal to start_date"
		}
		query["end_date"] = endDate
		return query, ""
	}
	if endDate != "" {
		return nil, "start_date is required when end_date is provided"
	}
	if month == "" {
		month = currentMonth()
	}
	query["mode"] = "month"
	query["month"] = month
	return query, ""
}

func queryMonths(query map[string]string) []string {
	switch query["mode"] {
	case "date":
		return []string{query["date"][:7]}
	case "range":
		return listMonthsBetween(query["start_date"][:7], query["end_date"][:7])
	default:
		return []string{query["month"]}
	}
}

func matchesCalendarQuery(date string, query map[string]string) bool {
	switch query["mode"] {
	case "date":
		return date == query["date"]
	case "range":
		return date >= query["start_date"] && date <= query["end_date"]
	default:
		return strings.HasPrefix(date, query["month"]+"-")
	}
}

func listMonthsBetween(startMonth string, endMonth string) []string {
	if startMonth > endMonth {
		return nil
	}
	result := []string{startMonth}
	current := startMonth
	for current < endMonth {
		year := atoi(current[:4])
		month := atoi(current[5:7])
		month++
		if month > 12 {
			year++
			month = 1
		}
		current = fmt.Sprintf("%04d-%02d", year, month)
		result = append(result, current)
	}
	return result
}

func stringArg(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func intArg(args map[string]any, key string, fallback int) int {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i)
		}
	}
	var result int
	if _, err := fmt.Sscanf(fmt.Sprint(value), "%d", &result); err != nil {
		return fallback
	}
	return result
}

func numericArg(args map[string]any, key string) float64 {
	value, ok := args[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case json.Number:
		f, err := v.Float64()
		if err == nil {
			return f
		}
	}
	var result float64
	_, _ = fmt.Sscanf(fmt.Sprint(value), "%f", &result)
	return result
}

func atoi(value string) int {
	var result int
	_, _ = fmt.Sscanf(value, "%d", &result)
	return result
}

func currentMonth() string {
	return time.Now().Format("2006-01")
}

func toolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Type: "function",
			Function: map[string]any{
				"name":        "search_products",
				"description": "根据关键词搜索产品，同时匹配产品名称（name）和标的代码（code），返回匹配产品的 id、名称、存续状态和标的代码。标的关键词如「恒科」「中证1000」「沪深300」等应在此搜索。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"keyword": map[string]any{"type": "string", "description": "产品名称或标的代码关键词"},
					},
					"required": []string{"keyword"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_product_detail",
				"description": "获取指定产品的完整详情字段。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"product_id": map[string]any{"type": "string", "description": "产品 ID"},
					},
					"required": []string{"product_id"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_observations",
				"description": "获取指定产品的观察记录，包括观察日、敲出价、派息线、标的价格、是否敲出和是否派息。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"product_id": map[string]any{"type": "string", "description": "产品 ID"},
					},
					"required": []string{"product_id"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_price",
				"description": "获取标的的最新缓存价格；如果没有缓存，会返回空价格和说明。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code": map[string]any{"type": "string", "description": "标的代码或名称"},
					},
					"required": []string{"code"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_dashboard_stats",
				"description": "获取业务总览统计，包括产品总数、存续产品数、客户数和渠道数。",
				"parameters": map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_observation_calendar",
				"description": "查询观察日历，支持按月、按日或日期范围查看观察安排，并可按产品名或管理人过滤。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"month":           map[string]any{"type": "string", "description": "月份，格式 YYYY-MM"},
						"date":            map[string]any{"type": "string", "description": "单日，格式 YYYY-MM-DD"},
						"start_date":      map[string]any{"type": "string", "description": "开始日期，格式 YYYY-MM-DD"},
						"end_date":        map[string]any{"type": "string", "description": "结束日期，格式 YYYY-MM-DD"},
						"product_keyword": map[string]any{"type": "string", "description": "产品名称关键词"},
						"manager":         map[string]any{"type": "string", "description": "管理人名称关键词"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "search_customers",
				"description": "搜索共投客户，可按客户名、实际购买人、微信、行业、专户或竞品标记过滤。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"keyword":              map[string]any{"type": "string", "description": "客户名、实际购买人或微信关键词"},
						"industry":             map[string]any{"type": "string", "description": "行业关键词"},
						"is_dedicated_account": map[string]any{"type": "string", "description": "是否专户，如 是/否"},
						"is_competitor":        map[string]any{"type": "string", "description": "是否竞品，如 是/否"},
						"limit":                map[string]any{"type": "integer", "description": "返回数量，默认 20"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_customer_products",
				"description": "查询某个客户或实际购买人关联的产品列表。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"customer_name": map[string]any{"type": "string", "description": "客户名或实际购买人"},
					},
					"required": []string{"customer_name"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_customer_peak_analysis",
				"description": "按客户汇总关联产品数量、当前余额和单产品峰值余额。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"customer_name": map[string]any{"type": "string", "description": "客户名或实际购买人关键词"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "query_transactions",
				"description": "查询交易流水，支持按产品 flight_id、交易对手和日期范围过滤。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"product_id":   map[string]any{"type": "string", "description": "产品 ID 或 flight_id"},
						"counterparty": map[string]any{"type": "string", "description": "交易对手关键词"},
						"start_date":   map[string]any{"type": "string", "description": "开始日期 YYYY-MM-DD"},
						"end_date":     map[string]any{"type": "string", "description": "结束日期 YYYY-MM-DD"},
						"limit":        map[string]any{"type": "integer", "description": "返回数量，默认 100"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_product_analytics",
				"description": "按管理人、存续状态、结构类型或发行月份聚合产品数量和规模。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"group_by": map[string]any{
							"type":        "string",
							"description": "聚合维度：manager、holding_status、structure_type、issue_month",
							"enum":        []string{"manager", "holding_status", "structure_type", "issue_month"},
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_posters",
				"description": "查询已生成的观察海报，可按观察日期或产品 ID 过滤。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"date":       map[string]any{"type": "string", "description": "观察日期 YYYY-MM-DD"},
						"product_id": map[string]any{"type": "string", "description": "产品 ID"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "search_product_docs",
				"description": "搜索投顾文档内容或标题，返回文档名、目录和内容预览。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"keyword": map[string]any{"type": "string", "description": "标题、正文或目录关键词"},
						"month":   map[string]any{"type": "string", "description": "月份 YYYY-MM，用于过滤目录"},
						"limit":   map[string]any{"type": "integer", "description": "返回数量，默认 20"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_channels_summary",
				"description": "获取渠道和直客来源列表。",
				"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_sync_status",
				"description": "获取产品、共投客户和投顾文档的最近同步状态。",
				"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_activity_logs",
				"description": "查询系统活动日志，可按日志类型过滤。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"type":  map[string]any{"type": "string", "description": "日志类型"},
						"limit": map[string]any{"type": "integer", "description": "返回数量，默认 50"},
					},
				},
			},
		},
	}
}

type streamResult struct {
	Content      string
	FinishReason string
	ToolCalls    []toolCall
}

type chatRequest struct {
	Model         string           `json:"model"`
	Messages      []chatMessage    `json:"messages"`
	Tools         []toolDefinition `json:"tools,omitempty"`
	Stream        bool             `json:"stream"`
	StreamOptions map[string]any   `json:"stream_options,omitempty"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type chatStreamEvent struct {
	Choices []struct {
		Delta struct {
			Content          string          `json:"content"`
			ReasoningContent string          `json:"reasoning_content"`
			ToolCalls        []toolCallDelta `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type toolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type toolDefinition struct {
	Type     string `json:"type"`
	Function any    `json:"function"`
}
