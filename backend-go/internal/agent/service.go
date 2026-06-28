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
	"business-workbench/backend-go/internal/posters"
	"business-workbench/backend-go/internal/retriever"
)

const maxToolRounds = 12

const systemPrompt = "你是一个专业的金融结构化产品业务助手，服务于业务工作台系统。请使用中文回答，优先基于系统内已有业务数据和用户问题给出简洁、准确的回复。需要查询产品、客户、交易、观察日历、投顾材料或业务统计时，主动调用可用工具。\n\n搜索产品时请注意：产品名称（name）通常是「航班服务XX号」这样的格式，标的指数或挂钩标的可能在标的代码（code）字段中。如果按产品名称搜索未果，请尝试用标的关键词搜索，例如用「中证1000」「沪深300」「恒科」「中证500」等关键词。也可以先调用 get_product_analytics 查看有哪些不同的标的和结构类型，再针对性搜索。\n\n当用户想要生成、制作、下载「喜报」「分红喜报」「分红观察喜报」时：先调用 search_products 找到目标产品的 product_id，再调用 generate_poster(product_id, observation_date) 生成。喜报里的所有数字（年化收益、本月分红、累计分红率、累计分红次数、派息界限、止盈界限、末月降至、挂钩标的、入场时间）都由系统从真实数据计算，你绝不可在对话中编造、估算或改写这些数字，也不可在 generate_poster 参数里传任何数字。若系统返回错误（如无该观察日记录），如实告知用户，不要自行补数。\n\n当用户想要生成结构化产品推介文案/材料（雪球/降敲雪球/DCN/FCN/限亏雪球等）时：先调用 load_skill('structured-product-copywriter') 加载完整工作流，再严格按其步骤执行——核对 10 项参数（缺了一次性问全）、取当前点位、做点位换算、产出长版+短版文案。文案里的「当前点位」必须调 fetch_quote 取真实值，绝对点位（降落伞/敲出/派息触发）必须调 calc_points 计算，你绝不可在对话中编造点位、胜率或自行做小数换算。胜率步骤：调用 fetch_winrate 取真实回测胜率（structure_type 由你判断传入）；若 fetch_winrate 返回 [胜率待补]（无凭证/遇验证码/站点不可达），如实告知并请用户手动提供，绝不编一个胜率数字。历史参考底部属判断性数据，问用户要，不要自己填。若 fetch_quote 失败，如实告知并让用户手动提供点位。走到通毓胜率/AMAC/Word 等重步骤时，可调 get_skill_reference 取对应参考文档。"

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
	OnArtifact  func(map[string]any)
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
			if art, ok := extractArtifact(toolResult); ok && callbacks.OnArtifact != nil {
				callbacks.OnArtifact(art)
			}
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

// extractArtifact 从工具返回结果里安全取出 poster_artifact 载荷。
// 返回 false 表示该工具结果不含喜报 artifact(普通工具调用)。
func extractArtifact(toolResult map[string]any) (map[string]any, bool) {
	raw, ok := toolResult["poster_artifact"]
	if !ok {
		return nil, false
	}
	m, ok := raw.(map[string]any)
	return m, ok
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
	case "generate_poster":
		return s.generatePoster(args)
	case "search_product_docs":
		return s.searchProductDocs(args)
	case "get_channels_summary":
		return s.getChannelsSummary()
	case "get_sync_status":
		return s.getSyncStatus()
	case "load_skill":
		return s.loadSkill(args)
	case "get_skill_reference":
		return s.getSkillReference(args)
	case "fetch_quote":
		return s.fetchQuote(args)
	case "calc_points":
		return s.calcPoints(args)
	case "fetch_winrate":
		return s.fetchWinrate(args)
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

// generatePoster 是 agent 的喜报生成工具：按产品 ID + 观察日从 DB 拉真实数据，
// 经 posters.GenerateData / BuildArtifact 组装成展示字段，以 poster_artifact 返回。
// 数字全部来自 DB，本函数不产生、不接受任何数字入参。
func (s *Service) generatePoster(args map[string]any) map[string]any {
	productID := stringArg(args, "product_id")
	observationDate := stringArg(args, "observation_date")
	if observationDate == "" {
		observationDate = time.Now().Format("2006-01-02")
	}
	if productID == "" {
		return map[string]any{"error": "product_id is required"}
	}

	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	var product *model.Product
	for i := range products {
		if products[i].ID == productID {
			product = &products[i]
			break
		}
	}
	if product == nil {
		return map[string]any{"error": "product not found or not ongoing: " + productID}
	}

	records, err := s.store.QueryObservationsByProduct(productID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	monthsSinceEntry := 0
	found := false
	for i := range records {
		if records[i].ObservationDate == observationDate && records[i].MonthsSinceEntry != nil {
			monthsSinceEntry = *records[i].MonthsSinceEntry
			found = true
			break
		}
	}
	if !found {
		return map[string]any{"error": "no observation record for " + observationDate + " on product " + productID}
	}

	data := posters.GenerateData(*product, observationDate, monthsSinceEntry)
	artifact := posters.BuildArtifact(*product, data, observationDate)
	return map[string]any{
		"poster_artifact":  artifact,
		"product_id":       productID,
		"observation_date": observationDate,
		"message":          "已生成「" + product.Name + "」(" + observationDate + ")的分红观察喜报，请在下方查看并下载。",
	}
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
				"name":        "generate_poster",
				"description": "为指定产品在指定观察日生成分红观察喜报（可下载的 PNG）。所有数字（年化、本月分红、累计分红率、界限、标的等）均由系统从该产品的真实观察数据计算，不要在参数里提供任何数字。先调用 search_products 拿到 product_id，再调用本工具。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"product_id":       map[string]any{"type": "string", "description": "产品 ID（由 search_products 返回）"},
						"observation_date": map[string]any{"type": "string", "description": "观察日 YYYY-MM-DD，默认今天"},
					},
					"required": []string{"product_id"},
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
		{
			Type: "function",
			Function: map[string]any{
				"name":        "load_skill",
				"description": "加载结构化产品推介文案生成 skill 的完整工作流（SKILL.md 原文）。当用户想要生成结构化产品推介文案/材料（雪球/降敲雪球/DCN/FCN/限亏雪球等）时，先调用本工具，再严格按返回的工作流步骤执行。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string", "description": "skill 名，默认 structured-product-copywriter"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_skill_reference",
				"description": "按需获取 skill 的某份重步骤参考文档（如 tongyu-winrate 通毓胜率流程、amac-manager AMAC 公示、product-position-card 产品点位卡、docx-template Word 模板）。走到该步骤时再调。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"skill": map[string]any{"type": "string", "description": "skill 名，默认 structured-product-copywriter"},
						"name":  map[string]any{"type": "string", "description": "参考文档名（不带 .md），如 tongyu-winrate"},
					},
					"required": []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "fetch_quote",
				"description": "获取指数/个股当前实时点位（腾讯→新浪→东财三源兜底）。文案里所有「当前点位」必须来自本工具，绝不编造。失败时如实告知并让用户手动提供，不要补数。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"标的":  map[string]any{"type": "string", "description": "标的名（如 中证1000/沪深300/创业板指）或代码（如 sh000852）"},
						"code": map[string]any{"type": "string", "description": "可选，直接给代码 sh000852/sz399006"},
					},
					"required": []string{"标的"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "calc_points",
				"description": "按当前点位 + 降落伞/期初敲出线/派息线百分比，机械换算绝对点位（降落伞点位、期初敲出点位、派息触发点位）+ 口语化约点。文案里的绝对点位必须来自本工具，绝不自己算小数。current_price 先用 fetch_quote 拿到。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"降落伞":         map[string]any{"type": "string", "description": "如 60%"},
						"期初敲出线":      map[string]any{"type": "string", "description": "如 101%"},
						"派息线":         map[string]any{"type": "string", "description": "如 78%；不适用时留空"},
						"current_price": map[string]any{"type": "number", "description": "fetch_quote 返回的当前点位"},
					},
					"required": []string{"降落伞", "期初敲出线", "current_price"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "fetch_winrate",
				"description": "通过通毓终端结构化产品回测算真实胜率。需要 TONGYU 凭证（服务端配置）；遇验证码/登录失败/站点不可达会返回 [胜率待补] 占位，届时请用户手动提供，绝不编造胜率。structure_type 由你从对话判断（如 DCN/雪球+降敲+降落伞）。标的用中文名或代码。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"structure_type": map[string]any{"type": "string", "description": "结构类型，如 DCN、雪球；含叠加条款用 + 连，如 DCN+降敲+降落伞"},
						"标的":           map[string]any{"type": "string", "description": "标的名（如 中证1000）或代码"},
						"期限":           map[string]any{"type": "string", "description": "如 36"},
						"锁定期":          map[string]any{"type": "string", "description": "如 3；无则不传"},
						"期初敲出线":        map[string]any{"type": "string", "description": "如 101"},
						"降敲":           map[string]any{"type": "string", "description": "如 0.5"},
						"降落伞":          map[string]any{"type": "string", "description": "如 60"},
						"派息线":          map[string]any{"type": "string", "description": "如 78；不适用不传"},
						"费后派息":         map[string]any{"type": "string", "description": "如 1.39"},
						"保证金":          map[string]any{"type": "string", "description": "如 50"},
						"是否追保":         map[string]any{"type": "string", "description": "不追保/追保"},
					},
					"required": []string{"structure_type", "标的", "期限", "期初敲出线", "降落伞", "费后派息", "保证金"},
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
