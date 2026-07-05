package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/db"
	"business-workbench/backend-go/internal/feishu"
	"business-workbench/backend-go/internal/model"
	"business-workbench/backend-go/internal/observations"
	"business-workbench/backend-go/internal/posters"
	"business-workbench/backend-go/internal/retriever"
)

const maxToolRounds = 25

const systemPrompt = `你是一个专业的金融结构化产品业务助手，服务于业务工作台系统。请使用中文回答，优先基于系统内已有业务数据和用户问题给出简洁、准确的回复。需要查询产品、客户、交易、观察日历、投顾材料或业务统计时，主动调用可用工具。

搜索产品时请注意：产品名称通常是「航班服务XX号」这样的格式，标的指数或挂钩标的可能在 code 字段中。如果按产品名称搜索未果，请尝试用「中证1000」「沪深300」「恒科」「中证500」等标的关键词搜索。

当用户想生成「喜报」「分红喜报」「分红观察喜报」时：先调用 search_products 找到 product_id，再调用 generate_poster(product_id, observation_date)。喜报里的数字全部由系统计算，绝不可编造、估算或改写，也不可在 generate_poster 参数里传任何数字。

当用户给出完整结构化产品参数（如 DCN/雪球/降敲雪球 + 标的 + 期限 + 保证金 + 敲出/降落伞/派息等，常带产品名/管理人/打款日/入场时间），默认就是要生成推介文案与飞书原生云文档材料：先调 load_skill("structured-product-copywriter")，再核对参数、取当前点位、做点位换算、产出长版+短版文案，最后调 create_docx_material 创建飞书 /docx/ 原生云文档。不要先问用户是否要生成。仅在用户明确说「只查询/核对/先不生成」时才不生成。

文案里的当前点位必须调 fetch_quote 取真实值；绝对点位（降落伞/敲出/派息触发）必须调 calc_points，禁止自己手算。胜率：用户给了就直接用；否则调 fetch_winrate，返回 [胜率待补] 就用占位继续生成，绝不编胜率，也不要因此停下问用户。历史参考底部：用户给了就用；没给就用 [历史底部待补] 占位继续生成。

当用户要出推介材料/销售物料/飞书文档时：按 references/docx-template.md 的 11 个 H2 章节顺序组装 manifest，调用 create_docx_material。该工具创建飞书原生 /docx/ 云文档，不是 Word 附件。文案用 fetch_quote/calc_points/fetch_winrate 已算的真实数字，create_docx_material 只装配不改数字。管理人/产品公示图调 screenshot_amac，产品点位卡调 screenshot_product_card。一页通/托管募集账户/暂未取得的截图留 image 占位，缺图不报错。

只有用户明确要求 Word、.docx 附件、下载 Word 文件时，才调用旧版 build_docx。普通飞书机器人场景一律不要默认调用 build_docx。`

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

// extractArtifact 浠庡伐鍏疯繑鍥炵粨鏋滈噷瀹夊叏鍙栧嚭 poster_artifact 杞借嵎銆?
// 杩斿洖 false 琛ㄧず璇ュ伐鍏风粨鏋滀笉鍚枩鎶?artifact(鏅€氬伐鍏疯皟鐢?銆?
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
	case "screenshot_amac":
		return s.screenshotAMACTool(args)
	case "screenshot_product_card":
		return s.screenshotProductCardTool(args)
	case "create_docx_material":
		return s.createDocxMaterialTool(args)
	case "build_docx":
		return s.buildDocxTool(args)
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

// generatePoster 鏄?agent 鐨勫枩鎶ョ敓鎴愬伐鍏凤細鎸変骇鍝?ID + 瑙傚療鏃ヤ粠 DB 鎷夌湡瀹炴暟鎹紝
// 缁?posters.GenerateData / BuildArtifact 缁勮鎴愬睍绀哄瓧娈碉紝浠?poster_artifact 杩斿洖銆?
// 鏁板瓧鍏ㄩ儴鏉ヨ嚜 DB锛屾湰鍑芥暟涓嶄骇鐢熴€佷笉鎺ュ彈浠讳綍鏁板瓧鍏ュ弬銆?
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
		"message":          fmt.Sprintf("已生成 %s（%s）的分红观察喜报，请在下方查看并下载。", product.Name, observationDate),
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

// screenshotAMACTool 鏄?agent 宸ュ叿鍏ュ彛锛氭寜 AMAC URL 鎴浘锛岃繑鍥?/public URL + path銆?
func (s *Service) screenshotAMACTool(args map[string]any) map[string]any {
	url := stringArg(args, "url")
	if url == "" {
		return map[string]any{"error": "url is required"}
	}
	id := nextPublicID()
	outPath := fmt.Sprintf("public/poster-artifacts/%s.png", id)
	_ = os.MkdirAll("public/poster-artifacts", 0o755)
	if err := screenshotAMAC(url, outPath); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"url": "/public/poster-artifacts/" + id + ".png", "path": outPath}
}

// screenshotProductCardTool 鏄?agent 宸ュ叿鍏ュ彛锛氭寜浜у搧鍙傛暟鐢熸垚閫氭瘬浜у搧鐐逛綅鍗″浘銆?
func (s *Service) screenshotProductCardTool(args map[string]any) map[string]any {
	if s.cfg.TongyuUser == "" || s.cfg.TongyuPass == "" {
		return map[string]any{"error": "鏈厤缃?TONGYU_USER/TONGYU_PASS锛屾棤娉曞彇浜у搧鍗″浘"}
	}
	id := nextPublicID()
	outPath := fmt.Sprintf("public/poster-artifacts/%s.png", id)
	_ = os.MkdirAll("public/poster-artifacts", 0o755)
	if err := screenshotProductCard(args, tongyuCreds{User: s.cfg.TongyuUser, Pass: s.cfg.TongyuPass}, s.cfg.ChromePath, outPath); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"url": "/public/poster-artifacts/" + id + ".png", "path": outPath}
}

// buildDocxTool 鏄?agent 宸ュ叿鍏ュ彛锛氳閰?.docx 骞朵笂浼犲埌椋炰功 Drive 褰撳勾褰撴湀瀛愭枃浠跺す锛岃繑鍥為涔?URL銆?
func (s *Service) buildDocxTool(args map[string]any) map[string]any {
	sections, err := parseManifest(args)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	// 1. 鍐欎复鏃舵枃浠?
	tmpDir, err := os.MkdirTemp("", "docx-")
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := filepath.Join(tmpDir, "鎺ㄤ粙鏉愭枡.docx")
	if err := BuildDocx(sections, tmpPath); err != nil {
		return map[string]any{"error": err.Error()}
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	// 2. 椋炰功 client锛堝鐢ㄦ寔涔呭寲 user token锛?
	fc := feishu.New(s.cfg.FeishuAppID, s.cfg.FeishuAppSecret, s.cfg.FeishuRedirectURI)
	fc.SetTokenPersistPath(".feishu-user-token")
	// 3. find-or-create 褰撳勾褰撴湀瀛愭枃浠跺す锛氭ā绯婂尮閰嶅惈"鏌愬勾鏌愭湀"鐨勬棦鏈夋枃浠跺す锛堝吋瀹?2026骞?鏈堜骇鍝?绛夊甫鍚庣紑鍚嶏級锛?
	//    閮戒笉鍛戒腑鍒欐柊寤猴紝鍛藉悕銆屾煇骞存煇鏈堜骇鍝併€嶅榻愭棦鏈変骇鍝佹潗鏂欐枃浠跺す绾﹀畾銆?
	yearMonth := time.Now().Format("2006年1月")
	folderName := yearMonth + "产品"
	ctx := context.Background()
	subToken, found, err := fc.FindSubfolderFuzzy(ctx, s.cfg.FeishuPitchFolderToken, yearMonth)
	if err != nil {
		return map[string]any{"error": "鏌ユ壘椋炰功鏂囦欢澶瑰け璐ワ細" + err.Error()}
	}
	if !found {
		subToken, err = fc.CreateFolder(ctx, s.cfg.FeishuPitchFolderToken, folderName)
		if err != nil {
			return map[string]any{"error": "鍒涘缓椋炰功鏂囦欢澶瑰け璐ワ細" + err.Error()}
		}
	}
	// 4. 涓婁紶锛堝彲閫?title 浣滀负鏂囦欢鍚嶏紝缂虹渷鍥為€€甯︽椂闂存埑鐨勯粯璁ゅ悕锛?
	fileName := fmt.Sprintf("鎺ㄤ粙鏉愭枡_%s.docx", time.Now().Format("20060102_150405"))
	if t := stringArg(args, "title"); t != "" {
		fileName = SanitizeFileName(t) + ".docx"
	}
	fileToken, err := fc.UploadDocx(ctx, subToken, fileName, data)
	if err != nil {
		return map[string]any{"error": "上传飞书失败：" + err.Error()}
	}
	url := "https://" + s.cfg.FeishuDriveDomain + "/file/" + fileToken
	return map[string]any{"url": url, "file_token": fileToken, "folder": folderName}
}

// createDocxMaterialTool creates a Feishu native /docx/ material document.
// It accepts the same manifest shape as build_docx, but inserts image sections
// as native Feishu image blocks instead of packaging a Word file.
func (s *Service) createDocxMaterialTool(args map[string]any) map[string]any {
	sections, err := parseManifest(args)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	fc := feishu.New(s.cfg.FeishuAppID, s.cfg.FeishuAppSecret, s.cfg.FeishuRedirectURI)
	fc.SetTokenPersistPath(".feishu-user-token")

	yearMonth := time.Now().Format("2006年1月")
	folderName := yearMonth + "产品"
	ctx := context.Background()
	subToken, found, err := fc.FindSubfolderFuzzy(ctx, s.cfg.FeishuPitchFolderToken, yearMonth)
	if err != nil {
		return map[string]any{"error": "查找飞书文件夹失败：" + err.Error()}
	}
	if !found {
		subToken, err = fc.CreateFolder(ctx, s.cfg.FeishuPitchFolderToken, folderName)
		if err != nil {
			return map[string]any{"error": "创建飞书文件夹失败：" + err.Error()}
		}
	}

	title := strings.TrimSpace(stringArg(args, "title"))
	if title == "" {
		title = "推介材料_" + time.Now().Format("20060102_150405")
	}
	doc, err := fc.CreateDocx(ctx, title, subToken, s.cfg.FeishuDriveDomain)
	if err != nil {
		return map[string]any{"error": "创建飞书云文档失败：" + err.Error()}
	}

	blocks := []any{}
	type pendingImage struct {
		BlockIndex int
		FileName   string
		Data       []byte
	}
	pendingImages := []pendingImage{}
	missingImages := []string{}
	appendMarkdown := func(content string) error {
		content = strings.TrimSpace(content)
		if content == "" {
			return nil
		}
		converted, err := fc.MarkdownBlocks(ctx, content)
		if err != nil {
			return err
		}
		blocks = append(blocks, converted...)
		return nil
	}

	for _, section := range sections {
		switch strings.ToLower(strings.TrimSpace(section.Type)) {
		case "heading", "subheading":
			if err := appendMarkdown("## " + strings.TrimSpace(section.Text)); err != nil {
				return map[string]any{"error": "转换标题失败：" + err.Error()}
			}
		case "body", "copy_file":
			if err := appendMarkdown(strings.TrimSpace(section.Text)); err != nil {
				return map[string]any{"error": "转换正文失败：" + err.Error()}
			}
		case "params":
			if strings.TrimSpace(section.Text) != "" {
				if err := appendMarkdown("```\n" + strings.TrimSpace(section.Text) + "\n```"); err != nil {
					return map[string]any{"error": "转换参数块失败：" + err.Error()}
				}
			}
		case "separator":
			// Keep the native material compact; visible separators are not required.
		case "link_list":
			lines := []string{}
			for _, item := range section.Items {
				label := strings.TrimSpace(item.Label)
				if label == "" {
					label = "链接"
				}
				if strings.TrimSpace(item.URL) != "" {
					lines = append(lines, fmt.Sprintf("- [%s](%s)", label, strings.TrimSpace(item.URL)))
				} else {
					lines = append(lines, "- "+label)
				}
			}
			if err := appendMarkdown(strings.Join(lines, "\n")); err != nil {
				return map[string]any{"error": "转换链接列表失败：" + err.Error()}
			}
		case "image":
			path := strings.TrimSpace(section.Path)
			data, fileName, ok := materialImageData(path)
			if !ok {
				missingImages = append(missingImages, path)
				caption := strings.TrimSpace(section.Caption)
				if caption == "" {
					caption = path
				}
				if err := appendMarkdown("[图片待补: " + caption + "]"); err != nil {
					return map[string]any{"error": "转换图片占位失败：" + err.Error()}
				}
				continue
			}
			imageToken, err := fc.UploadDocxImage(ctx, doc.DocumentID, fileName, data)
			if err != nil {
				return map[string]any{"error": "上传飞书图片失败：" + err.Error()}
			}
			blocks = append(blocks, map[string]any{
				"block_type": 27,
				"image": map[string]any{
					"file_token": imageToken,
				},
			})
			pendingImages = append(pendingImages, pendingImage{
				BlockIndex: len(blocks) - 1,
				FileName:   fileName,
				Data:       data,
			})
		}
	}

	inserted, err := fc.InsertDocxBlocks(ctx, doc.DocumentID, blocks)
	if err != nil {
		return map[string]any{"error": "写入飞书云文档失败：" + err.Error()}
	}
	imagesAdded := 0
	for _, image := range pendingImages {
		if image.BlockIndex < 0 || image.BlockIndex >= len(inserted) || strings.TrimSpace(inserted[image.BlockIndex].BlockID) == "" {
			return map[string]any{"error": "飞书图片块返回缺少 block_id"}
		}
		blockID := inserted[image.BlockIndex].BlockID
		blockToken, err := fc.UploadDocxImage(ctx, blockID, image.FileName, image.Data)
		if err != nil {
			return map[string]any{"error": "绑定飞书图片失败：" + err.Error()}
		}
		width, height := feishu.DocxImageDisplaySize(image.Data, 600)
		if err := fc.ReplaceDocxImage(ctx, doc.DocumentID, blockID, blockToken, width, height); err != nil {
			return map[string]any{"error": "替换飞书图片失败：" + err.Error()}
		}
		imagesAdded++
	}

	return map[string]any{
		"url":            doc.URL,
		"doc_token":      doc.DocumentID,
		"folder":         folderName,
		"blocks_added":   len(inserted),
		"images_added":   imagesAdded,
		"missing_images": missingImages,
	}
}

func materialImageData(path string) ([]byte, string, bool) {
	if path == "" {
		return nil, "", false
	}
	candidates := []string{path}
	if strings.HasPrefix(path, "/") {
		candidates = append(candidates, strings.TrimPrefix(path, "/"))
	}
	if strings.HasPrefix(path, "/public/") {
		candidates = append(candidates, strings.TrimPrefix(path, "/"))
	}
	for _, candidate := range candidates {
		p := filepath.Clean(candidate)
		data, err := os.ReadFile(p)
		if err != nil || len(data) == 0 {
			continue
		}
		name := filepath.Base(p)
		if name == "." || name == "" {
			name = "image.png"
		}
		return data, name, true
	}
	return nil, "", false
}

// BuildAndUploadDocx 鏄?HTTP 鍏ュ彛锛圥OST /api/drive/build-docx锛夌殑鍏紑灏佽锛?// 瑁呴厤 .docx 骞朵笂浼犲埌椋炰功 Drive 褰撳勾褰撴湀瀛愭枃浠跺す锛岃繑鍥?{url, file_token, folder}銆?// 渚?openclaw 绛夊閮?agent 澶嶇敤 business-workbench 鐨勯涔?OAuth token 涓庝笂浼犻€昏緫锛?
// 閬垮厤姣忎釜 agent 鍚勮嚜閰?token / 鍒嗕韩 folder銆俛rgs 涓?build_docx 宸ュ叿涓€鑷达細
// {title?, sections:[{type,text,path?,caption?,items?}]}銆?
func (s *Service) BuildAndUploadDocx(args map[string]any) map[string]any {
	return s.buildDocxTool(args)
}

// SanitizeFileName 鎶婁换鎰忓瓧绗︿覆娓呮垚瀹夊叏鐨勬枃浠跺悕锛堝幓鎺夎矾寰勫垎闅旂涓?Windows/椋炰功闈炴硶瀛楃锛夈€?
func SanitizeFileName(s string) string {
	repl := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "|", "_", "<", "_", ">", "_", "\n", "_", "\r", "_")
	out := strings.TrimSpace(repl.Replace(s))
	if out == "" {
		out = "鎺ㄤ粙鏉愭枡"
	}
	return out
}

func nextPublicID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// parseManifest 鎶?agent 浼犲叆鐨?manifest锛坅rgs["sections"] 鏁扮粍锛夎В鏋愭垚 docxSection 鍒囩墖銆?
// 姣忎釜 section: {type, text, path, caption, items:[{label,url}]}銆?
func parseManifest(args map[string]any) ([]docxSection, error) {
	raw, ok := args["sections"]
	if !ok {
		return nil, fmt.Errorf("sections is required")
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("sections must be an array")
	}
	var out []docxSection
	for _, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		sec := docxSection{
			Type:    stringArg(m, "type"),
			Text:    stringArg(m, "text"),
			Path:    stringArg(m, "path"),
			Caption: stringArg(m, "caption"),
		}
		if items, ok := m["items"].([]any); ok {
			for _, it := range items {
				im, ok := it.(map[string]any)
				if !ok {
					continue
				}
				sec.Items = append(sec.Items, docxLink{Label: stringArg(im, "label"), URL: stringArg(im, "url")})
			}
		}
		out = append(out, sec)
	}
	return out, nil
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
				"description": "根据关键词搜索产品，同时匹配产品名称和标的代码，返回匹配产品的 id、名称、存续状态和标的代码。",
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
				"description": "为指定产品在指定观察日生成分红观察喜报 PNG。所有数字均由系统从真实观察数据计算；先用 search_products 拿 product_id。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"product_id":       map[string]any{"type": "string", "description": "产品 ID，由 search_products 返回"},
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
				"description": "加载结构化产品推介文案生成 skill 的完整工作流。生成雪球、降敲雪球、DCN、FCN 等推介文案或材料时先调用本工具。",
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
				"description": "按需获取 skill 的参考文档，例如 tongyu-winrate、amac-manager、product-position-card、docx-template。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"skill": map[string]any{"type": "string", "description": "skill 名，默认 structured-product-copywriter"},
						"name":  map[string]any{"type": "string", "description": "参考文档名，不带 .md，例如 tongyu-winrate"},
					},
					"required": []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "fetch_quote",
				"description": "获取指数或个股当前实时点位。文案里的当前点位必须来自本工具，失败时如实告知，不要编造。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"标的":   map[string]any{"type": "string", "description": "标的名，如 中证1000/沪深300/创业板指，或代码如 sh000852"},
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
				"description": "按当前点位和百分比换算降落伞、期初敲出线、派息线的绝对点位。文案里的绝对点位必须来自本工具。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"降落伞":           map[string]any{"type": "string", "description": "如 60%"},
						"期初敲出线":         map[string]any{"type": "string", "description": "如 101%"},
						"派息线":           map[string]any{"type": "string", "description": "如 78%；不适用时留空"},
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
				"description": "通过通毓终端结构化产品回测计算真实胜率。失败时返回占位并如实告知，不要编造胜率。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"structure_type": map[string]any{"type": "string", "description": "结构类型，如 DCN、雪球；可含附加条款，如 DCN+降敲+降落伞"},
						"标的":             map[string]any{"type": "string", "description": "标的名如 中证1000，或代码"},
						"期限":             map[string]any{"type": "string", "description": "如 36"},
						"锁定期":            map[string]any{"type": "string", "description": "如 3；无则不传"},
						"期初敲出线":          map[string]any{"type": "string", "description": "如 101"},
						"降敲":             map[string]any{"type": "string", "description": "如 0.5"},
						"降落伞":            map[string]any{"type": "string", "description": "如 60"},
						"派息线":            map[string]any{"type": "string", "description": "如 78；不适用不传"},
						"费后派息":           map[string]any{"type": "string", "description": "如 1.39"},
						"保证金":            map[string]any{"type": "string", "description": "如 50"},
						"是否追保":           map[string]any{"type": "string", "description": "不追保/追保"},
					},
					"required": []string{"structure_type", "标的", "期限", "期初敲出线", "降落伞", "费后派息", "保证金"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "screenshot_amac",
				"description": "截取 AMAC 管理人或产品公示详情页图片。URL 应为 amac.org.cn 的 details 页面，用于推介材料公示图。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{"type": "string", "description": "AMAC 详情页 URL"},
					},
					"required": []string{"url"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "screenshot_product_card",
				"description": "用通毓终端产品点位小工具按产品参数生成产品结构解析卡图，用于推介材料的派息与敲出观察点位表图片。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"structure_type": map[string]any{"type": "string"},
						"标的":             map[string]any{"type": "string"},
						"期限":             map[string]any{"type": "string"},
						"锁定期":            map[string]any{"type": "string"},
						"期初敲出线":          map[string]any{"type": "string"},
						"降敲":             map[string]any{"type": "string"},
						"降落伞":            map[string]any{"type": "string"},
						"派息线":            map[string]any{"type": "string"},
						"费后派息":           map[string]any{"type": "string"},
						"保证金":            map[string]any{"type": "string"},
						"current_price":  map[string]any{"type": "string"},
					},
					"required": []string{"structure_type", "标的", "期限", "期初敲出线", "降落伞", "费后派息", "保证金", "current_price"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "create_docx_material",
				"description": "按 manifest 创建飞书原生 /docx/ 云文档，不是 Word 附件。用户要出推介材料、销售物料或飞书文档时优先调用本工具。sections 每项支持 type,text,path,caption,items；image 的 path 使用截图工具返回的本地 path。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title":    map[string]any{"type": "string", "description": "飞书云文档标题，如 产品简称-结构名称"},
						"sections": map[string]any{"type": "array", "description": "章节数组，严格按 docx-template.md 的 11 个 H2 顺序。"},
					},
					"required": []string{"sections"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "build_docx",
				"description": "旧版 Word 附件工具，仅当用户明确要求 Word 或 .docx 附件时使用。普通推介材料和飞书机器人场景必须优先使用 create_docx_material。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"sections": map[string]any{"type": "array", "description": "章节数组。"},
					},
					"required": []string{"sections"},
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
