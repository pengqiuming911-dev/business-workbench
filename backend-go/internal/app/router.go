package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"business-workbench/backend-go/internal/agent"
	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/db"
	"business-workbench/backend-go/internal/email"
	"business-workbench/backend-go/internal/feishu"
	"business-workbench/backend-go/internal/model"
	"business-workbench/backend-go/internal/observations"
	"business-workbench/backend-go/internal/posters"
	"business-workbench/backend-go/internal/prices"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type Server struct {
	cfg      config.Config
	store    *db.Store
	agentSvc *agent.Service
	feishu   *feishu.Client
	Cron     *cron.Cron
}

const (
	mainSheetToken      = "HdxnsNXSQhKoSItKiLwcnEmjn8b"
	productSheetID      = "3JiyjX"
	transactionSheetID  = "0PZFXO"
	coInvestAppToken    = "G1sbbVhL2awTltsOoi8cqci4nJh"
	coInvestTableID     = "tbl5mm7sQ001Z7p1"
	productLibraryToken = "W9OGfnjzQl8dOOdqPFwcL6gEnkf"
)

func NewRouter(cfg config.Config, store *db.Store) *gin.Engine {
	location, _ := time.LoadLocation(cfg.CronTimezone)
	if location == nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	server := &Server{
		cfg:      cfg,
		store:    store,
		agentSvc: agent.NewService(cfg, store),
		feishu:   feishu.New(cfg.FeishuAppID, cfg.FeishuAppSecret, cfg.FeishuRedirectURI),
		Cron:     cron.New(cron.WithLocation(location)),
	}

	router := gin.Default()
	router.Static("/public", "public")
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/api/health", server.health)
	router.GET("/api/auth/status", server.authStatus)
	router.GET("/api/auth/login", server.authLogin)
	router.GET("/api/auth/callback", server.authCallback)
	router.DELETE("/api/auth/logout", server.authLogout)
	router.GET("/api/auth/logout", server.authLogout)
	router.GET("/api/db/sync-status", server.syncStatus)
	router.GET("/api/db/sync-coinvest-status", server.coInvestSyncStatus)
	router.GET("/api/db/products", server.products)
	router.GET("/api/db/industries", server.industries)
	router.GET("/api/db/user-profiles", server.userProfiles)
	router.POST("/api/db/sync", server.syncDatabase)
	router.POST("/api/db/sync-coinvest", server.syncCoInvest)
	router.GET("/api/drive/shared-with-me", server.driveSharedWithMe)
	router.GET("/api/drive/shared-spaces", server.driveSharedSpaces)
	router.GET("/api/drive/shared-files", server.driveSharedFiles)
	router.GET("/api/drive/files", server.driveFiles)
	router.GET("/api/drive/download", server.driveDownload)
	router.GET("/api/drive/sheet-data", server.driveSheetData)
	router.GET("/api/drive/doc-content", server.driveDocContent)
	router.GET("/api/drive/export-sheet", server.driveExportSheet)
	router.GET("/api/drive/product-docs", server.productDocs)
	router.GET("/api/drive/product-docs/sync-status", server.productDocsSyncStatus)
	router.POST("/api/drive/sync-product-docs", server.syncProductDocs)
	router.GET("/api/dashboard/stats", server.dashboardStats)
	router.GET("/api/dashboard/charts", server.dashboardCharts)
	router.GET("/api/observations/calendar", server.observationCalendar)
	router.GET("/api/observations/products", server.observationProducts)
	router.GET("/api/observations", server.observations)
	router.GET("/api/observations/today", server.todayObservations)
	router.POST("/api/observations/generate", server.generateObservations)
	router.POST("/api/observations/refresh-prices", server.refreshPrices)
	router.GET("/api/posters/today", server.postersToday)
	router.GET("/api/posters", server.posters)
	router.POST("/api/posters/generate", server.generatePosters)
	router.GET("/api/push-config", server.getPushConfig)
	router.PUT("/api/push-config", server.putPushConfig)
	router.POST("/api/push/test", server.testPush)
	router.GET("/api/activity-logs", server.activityLogs)
	router.GET("/api/search", server.search)
	router.GET("/api/agent/conversations", server.agentConversations)
	router.POST("/api/agent/conversations", server.createAgentConversation)
	router.DELETE("/api/agent/conversations/:id", server.deleteAgentConversation)
	router.GET("/api/agent/conversations/:id/messages", server.agentMessages)
	router.POST("/api/agent/chat", server.agentChat)

	router.GET("/api/holding/products", server.holdingProducts)
	router.GET("/api/holding/transactions", server.holdingTransactions)
	router.GET("/api/holding/filter-options", server.holdingFilterOptions)
	router.POST("/api/holding/refresh-price", server.holdingRefreshPrice)

	server.startScheduler()
	schedulerInstance = server.Cron
	server.Cron.Start()
	return router
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"service": "business-workbench-go",
	})
}

func (s *Server) authStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"authorized": s.feishu.Authorized()})
}

func (s *Server) authLogin(c *gin.Context) {
	if strings.TrimSpace(s.cfg.FeishuAppID) == "" || strings.TrimSpace(s.cfg.FeishuRedirectURI) == "" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Feishu OAuth is not configured in Go backend"})
		return
	}
	scope := "drive:drive drive:file drive:export:readonly space:document:retrieve bitable:app:readonly bitable:app docx:document docx:document:readonly"
	url := "https://open.feishu.cn/open-apis/authen/v1/authorize" +
		"?app_id=" + s.cfg.FeishuAppID +
		"&redirect_uri=" + urlQueryEscape(s.cfg.FeishuRedirectURI) +
		"&scope=" + urlQueryEscape(scope) +
		"&response_type=code"
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (s *Server) authCallback(c *gin.Context) {
	target := strings.TrimRight(s.cfg.FrontendURL, "/")
	if target == "" {
		target = "http://localhost:5173"
	}
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.Redirect(http.StatusFound, target+"/data-preparation?auth=error&msg=missing_code")
		return
	}
	if err := s.feishu.ExchangeCode(c.Request.Context(), code); err != nil {
		c.Redirect(http.StatusFound, target+"/data-preparation?auth=error&msg="+urlQueryEscape(err.Error()))
		return
	}
	c.Redirect(http.StatusFound, target+"/data-preparation?auth=success")
}

func (s *Server) authLogout(c *gin.Context) {
	s.feishu.ClearUserToken()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) syncStatus(c *gin.Context) {
	row, err := s.store.LastSync()
	if err != nil {
		writeError(c, err)
		return
	}
	if row == nil {
		c.JSON(http.StatusOK, gin.H{"message": "从未同步"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) coInvestSyncStatus(c *gin.Context) {
	row, err := s.store.LastCoInvestSync()
	if err != nil {
		writeError(c, err)
		return
	}
	if row == nil {
		c.JSON(http.StatusOK, gin.H{"message": "从未同步"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) products(c *gin.Context) {
	products, err := s.store.QueryProducts(c.Query("startDate"), c.Query("endDate"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (s *Server) dashboardStats(c *gin.Context) {
	stats, err := s.store.DashboardStats()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (s *Server) dashboardCharts(c *gin.Context) {
	trend, err := s.store.MonthlyTrend()
	if err != nil {
		writeError(c, err)
		return
	}
	channels, err := s.store.ChannelDistribution()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"monthlyTrend":        trend,
		"channelDistribution": channels,
	})
}

func (s *Server) observationCalendar(c *gin.Context) {
	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	if len(month) != 7 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "月份格式应为 YYYY-MM"})
		return
	}

	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"month":    month,
		"calendar": observations.CalendarForMonth(products, month),
	})
}

func (s *Server) observationProducts(c *gin.Context) {
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (s *Server) industries(c *gin.Context) {
	rows, err := s.store.DistinctIndustries()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows})
}

func (s *Server) userProfiles(c *gin.Context) {
	rows, err := s.store.UserProfiles(
		c.Query("actual_buyer"),
		c.Query("nominal_buyer"),
		c.Query("is_dedicated"),
		c.Query("is_competitor"),
		c.Query("industry"),
	)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows})
}

func (s *Server) productDocs(c *gin.Context) {
	rows, err := s.store.ProductDocs(c.Query("month"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (s *Server) productDocsSyncStatus(c *gin.Context) {
	row, err := s.store.LastProductDocsSync()
	if err != nil {
		writeError(c, err)
		return
	}
	if row == nil {
		c.JSON(http.StatusOK, gin.H{"synced_at": nil, "doc_count": 0, "folder_count": 0})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) syncDatabase(c *gin.Context) {
	if !s.feishu.Authorized() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权，请先登录飞书"})
		return
	}

	productRows, err := s.feishu.ReadSheetRows(c.Request.Context(), mainSheetToken, productSheetID, 58)
	if err != nil {
		writeError(c, err)
		return
	}
	products := mapProductSheetRows(productRows)
	if err := s.store.ImportProducts(products); err != nil {
		writeError(c, err)
		return
	}

	transactionRows, err := s.feishu.ReadSheetRows(c.Request.Context(), mainSheetToken, transactionSheetID, 500)
	if err != nil {
		writeError(c, err)
		return
	}
	transactions := mapTransactionSheetRows(transactionRows)
	if err := s.store.ImportTransactions(transactions); err != nil {
		writeError(c, err)
		return
	}

	total := len(products) + len(transactions)
	if err := s.store.LogSync(total); err != nil {
		writeError(c, err)
		return
	}
	_ = s.store.LogActivity("sync", "Transaction table synced", fmt.Sprintf("%d rows", total))

	ongoing, err := s.store.QueryOngoingProducts()
	if err == nil {
		priceResult := prices.FetchAll(c.Request.Context(), uniqueProductCodes(ongoing))
		today := time.Now().Format("2006-01-02")
		for code, price := range priceResult.Prices {
			_ = s.store.UpsertPrice(code, today, price)
		}
		_ = s.updateTodayObservationRecords(ongoing, today, priceResult.Prices)
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":               true,
		"rowCount":         total,
		"productCount":     len(products),
		"transactionCount": len(transactions),
	})
}

func (s *Server) syncCoInvest(c *gin.Context) {
	if !s.feishu.Authorized() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权，请先登录飞书"})
		return
	}

	records, err := s.feishu.ReadBitableRecords(c.Request.Context(), coInvestAppToken, coInvestTableID)
	if err != nil {
		writeError(c, err)
		return
	}
	rows := mapCoInvestRecords(records)
	if err := s.store.ImportCoInvestUsers(rows); err != nil {
		writeError(c, err)
		return
	}
	if err := s.store.LogCoInvestSync(len(rows)); err != nil {
		writeError(c, err)
		return
	}
	_ = s.store.LogActivity("sync", "Co-invest users synced", fmt.Sprintf("%d rows", len(rows)))
	c.JSON(http.StatusOK, gin.H{"ok": true, "rowCount": len(rows)})
}

func (s *Server) driveSharedWithMe(c *gin.Context) {
	folderType := ""
	if c.Query("folder_token") == "" {
		folderType = "share_with_me"
	}
	data, err := s.feishu.DriveFiles(c.Request.Context(), c.Query("folder_token"), c.Query("page_token"), folderType)
	if err != nil {
		writeDriveError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) driveSharedSpaces(c *gin.Context) {
	data, err := s.feishu.SharedSpaces(c.Request.Context())
	if err != nil {
		writeDriveError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) driveSharedFiles(c *gin.Context) {
	spaceID := strings.TrimSpace(c.Query("space_id"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 space_id 参数"})
		return
	}
	data, err := s.feishu.SharedFiles(c.Request.Context(), spaceID, c.Query("folder_token"), c.Query("page_token"))
	if err != nil {
		writeDriveError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) driveFiles(c *gin.Context) {
	data, err := s.feishu.DriveFiles(c.Request.Context(), c.Query("folder_token"), c.Query("page_token"), "")
	if err != nil {
		writeDriveError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) driveDownload(c *gin.Context) {
	fileToken := strings.TrimSpace(c.Query("file_token"))
	if fileToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 file_token 参数"})
		return
	}
	data, contentType, err := s.feishu.DownloadFile(c.Request.Context(), "https://open.feishu.cn/open-apis/drive/v1/files/"+url.PathEscape(fileToken)+"/download")
	if err != nil {
		writeDriveError(c, err)
		return
	}
	fileName := c.DefaultQuery("file_name", "download.xlsx")
	c.Header("Content-Disposition", `attachment; filename="`+url.QueryEscape(fileName)+`"`)
	if contentType == "" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	c.Data(http.StatusOK, contentType, data)
}

func (s *Server) driveSheetData(c *gin.Context) {
	sheetToken := strings.TrimSpace(c.Query("sheet_token"))
	sheetID := strings.TrimSpace(c.Query("sheet_id"))
	if sheetToken == "" || sheetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 sheet_token 或 sheet_id 参数"})
		return
	}
	rows, err := s.feishu.ReadSheetRows(c.Request.Context(), sheetToken, sheetID, 702)
	if err != nil {
		writeDriveError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows})
}

func (s *Server) driveDocContent(c *gin.Context) {
	docToken := strings.TrimSpace(c.Query("doc_token"))
	if docToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 doc_token 参数"})
		return
	}
	text, err := s.feishu.ReadDocRawContent(c.Request.Context(), docToken)
	if err != nil {
		writeDriveError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"text": text})
}

func (s *Server) driveExportSheet(c *gin.Context) {
	sheetToken := strings.TrimSpace(c.Query("sheet_token"))
	if sheetToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 sheet_token 参数"})
		return
	}
	data, contentType, err := s.feishu.ExportSheet(c.Request.Context(), sheetToken)
	if err != nil {
		writeDriveError(c, err)
		return
	}
	if contentType == "" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	c.Header("Content-Disposition", `attachment; filename="export.xlsx"`)
	c.Data(http.StatusOK, contentType, data)
}

func (s *Server) syncProductDocs(c *gin.Context) {
	if !s.feishu.Authorized() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权，请先登录飞书"})
		return
	}

	walk, err := s.feishu.WalkDriveFolder(c.Request.Context(), productLibraryToken)
	if err != nil {
		writeError(c, err)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rows := []map[string]any{}
	for _, file := range walk.Files {
		rawContent, err := s.feishu.ReadDocRawContent(c.Request.Context(), file.Token)
		if err != nil {
			rawContent = ""
		}
		structured := parseProductStructure(rawContent)
		structureJSON := ""
		if structured != nil {
			data, _ := json.Marshal(structured)
			structureJSON = string(data)
		}
		rows = append(rows, map[string]any{
			"doc_token":      file.Token,
			"doc_name":       file.Name,
			"parent_path":    file.ParentPath,
			"folder_token":   file.ParentToken,
			"raw_content":    rawContent,
			"structure_json": structureJSON,
			"synced_at":      now,
		})
	}
	if err := s.store.ImportProductDocs(rows); err != nil {
		writeError(c, err)
		return
	}
	if err := s.store.LogProductDocsSync(len(rows), walk.FolderCount); err != nil {
		writeError(c, err)
		return
	}
	_ = s.store.LogActivity("sync", "Product docs synced", fmt.Sprintf("%d docs, %d folders", len(rows), walk.FolderCount))
	c.JSON(http.StatusOK, gin.H{
		"ok":           true,
		"message":      fmt.Sprintf("同步成功，共 %d 个文档", len(rows)),
		"doc_count":    len(rows),
		"folder_count": walk.FolderCount,
		"last_sync":    now,
	})
}

func (s *Server) observations(c *gin.Context) {
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		writeError(c, err)
		return
	}
	products = filterObservationProducts(products, c.Query("search"), c.Query("code"))
	payload, err := s.buildObservationProducts(products, time.Now().Format("2006-01-02"), false)
	if err != nil {
		writeError(c, err)
		return
	}
	lastUpdated, err := s.store.LastObservationUpdate()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": payload, "lastUpdated": nullableString(lastUpdated)})
}

func (s *Server) todayObservations(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		writeError(c, err)
		return
	}
	payload, err := s.buildObservationProducts(products, today, true)
	if err != nil {
		writeError(c, err)
		return
	}
	lastUpdated, err := s.store.LastObservationUpdate()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": payload, "today": today, "lastUpdated": nullableString(lastUpdated)})
}

func (s *Server) generateObservations(c *gin.Context) {
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		writeError(c, err)
		return
	}
	codes := uniqueProductCodes(products)
	priceResult := prices.FetchAll(c.Request.Context(), codes)
	today := time.Now().Format("2006-01-02")
	for code, price := range priceResult.Prices {
		if err := s.store.UpsertPrice(code, today, price); err != nil {
			writeError(c, err)
			return
		}
	}

	generated := 0
	recalculatedExisting := 0
	skippedNoCode := 0
	skippedNoPrice := 0
	skippedNoDates := 0

	for _, product := range products {
		if product.Code == "" || product.IssueDate == "" || product.EntryPrice == nil {
			skippedNoCode++
			continue
		}
		obsDates := observations.DatesUntil(product, today)
		if len(obsDates) == 0 {
			skippedNoDates++
			continue
		}
		existingRows, err := s.store.QueryObservationsByProduct(product.ID)
		if err != nil {
			writeError(c, err)
			return
		}
		existing := map[string]model.Observation{}
		for _, row := range existingRows {
			existing[row.ObservationDate] = row
		}
		for _, obs := range obsDates {
			price, ok := priceForObservation(s.store, product, obs.Date, priceResult.Prices)
			if !ok {
				if existingRow, exists := existing[obs.Date]; exists && existingRow.UnderlyingPrice != nil {
					price = *existingRow.UnderlyingPrice
					ok = true
				}
			}
			if !ok {
				skippedNoPrice++
				continue
			}
			eval := observations.EvaluateObservation(product, obs.Date, price, obs.MonthsSinceEntry)
			if err := s.store.UpsertObservation(product.ID, observationEvalMap(eval)); err != nil {
				writeError(c, err)
				return
			}
			if _, exists := existing[obs.Date]; exists {
				recalculatedExisting++
			} else {
				generated++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":                   true,
		"generated":            generated,
		"recalculatedExisting": recalculatedExisting,
		"priceRefreshed":       len(priceResult.Prices),
		"priceFailed":          len(priceResult.Failed),
		"skippedNoCode":        skippedNoCode,
		"skippedNoPrice":       skippedNoPrice,
		"skippedNoDates":       skippedNoDates,
	})
}

func (s *Server) refreshPrices(c *gin.Context) {
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		writeError(c, err)
		return
	}
	codes := uniqueProductCodes(products)
	priceResult := prices.FetchAll(c.Request.Context(), codes)
	today := time.Now().Format("2006-01-02")
	for code, price := range priceResult.Prices {
		if err := s.store.UpsertPrice(code, today, price); err != nil {
			writeError(c, err)
			return
		}
	}

	updated := 0
	for _, product := range products {
		if product.Code == "" {
			continue
		}
		records, err := s.store.QueryObservationsByProduct(product.ID)
		if err != nil {
			writeError(c, err)
			return
		}
		for _, obs := range records {
			if obs.MonthsSinceEntry == nil {
				continue
			}
			price, ok := priceForObservation(s.store, product, obs.ObservationDate, priceResult.Prices)
			if !ok {
				continue
			}
			eval := observations.EvaluateObservation(product, obs.ObservationDate, price, *obs.MonthsSinceEntry)
			if err := s.store.UpsertObservation(product.ID, observationEvalMap(eval)); err != nil {
				writeError(c, err)
				return
			}
			updated++
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "refreshed": len(priceResult.Prices), "updated": updated, "failed": len(priceResult.Failed)})
}

func (s *Server) postersToday(c *gin.Context) {
	date := c.Query("date")
	today := time.Now().Format("2006-01-02")
	if date == "" {
		date = today
	}
	posters, err := s.store.QueryPostersByDate(date)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"posters": posters, "today": today, "queryDate": date})
}

func (s *Server) posters(c *gin.Context) {
	productID := c.Query("product_id")
	var (
		posters []model.Poster
		err     error
	)
	if productID != "" {
		posters, err = s.store.QueryPostersByProduct(productID)
	} else {
		posters, err = s.store.QueryAllPosters()
	}
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"posters": posters})
}

func (s *Server) generatePosters(c *gin.Context) {
	var req struct {
		Date string `json:"date"`
	}
	_ = c.ShouldBindJSON(&req)
	targetDate := strings.TrimSpace(req.Date)
	if targetDate == "" {
		targetDate = time.Now().Format("2006-01-02")
	}

	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		writeError(c, err)
		return
	}

	knockoutCount := 0
	dividendCount := 0
	matched := 0
	for _, product := range products {
		targetObsInfo, ok := observationInfoForDate(product, targetDate)
		if !ok {
			continue
		}
		matched++
		records, err := s.store.QueryObservationsByProduct(product.ID)
		if err != nil {
			writeError(c, err)
			return
		}
		var targetRecord *model.Observation
		for i := range records {
			if records[i].ObservationDate == targetDate {
				targetRecord = &records[i]
				break
			}
		}
		if targetRecord == nil {
			continue
		}
		data := posters.GenerateData(product, targetDate, targetObsInfo.MonthsSinceEntry)
		isKnockout := data.KnockoutValue != "" && targetRecord.IsKnockedOut == "是"
		isDividend := data.HasDividendObservation && data.DividendBarrierValue != "" && targetRecord.IsDividend == "是"

		if isKnockout {
			knockoutCount++
			row := posterRow(product, targetDate, targetObsInfo.MonthsSinceEntry, data, "knockout")
			row.AbsoluteReturn = floatPtr(data.AbsoluteReturn)
			row.AnnualizedReturn = floatPtr(data.AnnualizedReturn)
			row.DurationMonths = intPtr(targetObsInfo.MonthsSinceEntry)
			row.DividendBarrierValue = ""
			row.DividendCount = intPtr(0)
			row.CumulativeRate = floatPtr(0)
			if err := s.store.UpsertPoster(row); err != nil {
				writeError(c, err)
				return
			}
		}

		if isDividend && !isKnockout {
			dividendCount++
			row := posterRow(product, targetDate, targetObsInfo.MonthsSinceEntry, data, "dividend")
			row.AbsoluteReturn = floatPtr(0)
			row.DurationMonths = intPtr(0)
			row.DividendCount = intPtr(data.DividendCount)
			row.CumulativeRate = floatPtr(data.CumulativeDividendRate)
			if err := s.store.UpsertPoster(row); err != nil {
				writeError(c, err)
				return
			}
		}
	}

	generated := knockoutCount + dividendCount
	message := ""
	if matched == 0 {
		message = targetDate + " 无产品需要观察"
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"generated": generated,
		"knockout":  knockoutCount,
		"dividend":  dividendCount,
		"today":     targetDate,
		"message":   message,
	})
}

func (s *Server) getPushConfig(c *gin.Context) {
	config, err := s.store.GetPushConfig(s.cfg.FeishuPushWebhook)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (s *Server) putPushConfig(c *gin.Context) {
	var config model.PushConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if config.CronHour < 0 || config.CronHour > 23 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cron_hour must be integer 0-23"})
		return
	}
	if config.CronMinute < 0 || config.CronMinute > 59 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cron_minute must be integer 0-59"})
		return
	}
	if err := s.store.UpsertPushConfig(config); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) testPush(c *gin.Context) {
	config, err := s.store.GetPushConfig(s.cfg.FeishuPushWebhook)
	if err != nil {
		writeError(c, err)
		return
	}
	if strings.TrimSpace(config.WebhookURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "webhook URL not configured"})
		return
	}

	today := time.Now().Format("2006-01-02")
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		writeError(c, err)
		return
	}
	priceResult := prices.FetchAll(c.Request.Context(), uniqueProductCodes(products))
	for code, price := range priceResult.Prices {
		if err := s.store.UpsertPrice(code, today, price); err != nil {
			writeError(c, err)
			return
		}
	}
	if err := s.updateTodayObservationRecords(products, today, priceResult.Prices); err != nil {
		writeError(c, err)
		return
	}

	payload, err := s.buildObservationProducts(products, today, true)
	if err != nil {
		writeError(c, err)
		return
	}
	text := buildFeishuPushText(today, payload)
	if len(payload) == 0 {
		text = "今日产品派息/敲出观察提醒\n观察日期：" + today + "\n\n今日无产品需要观察。"
	}

	result := gin.H{"sent": false, "count": len(payload)}
	if err := sendFeishuWebhook(c.Request.Context(), config.WebhookURL, text); err != nil {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_ = s.store.UpdatePushResult(now, "error: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	status := fmt.Sprintf("success (%d products)", len(payload))
	if err := s.store.UpdatePushResult(now, status); err != nil {
		writeError(c, err)
		return
	}
	result["sent"] = true
	if len(payload) == 0 {
		result["reason"] = "no-observation-today"
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) executeScheduledObservationPush(ctx context.Context, webhookURL string) (int, error) {
	today := time.Now().Format("2006-01-02")
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		return 0, err
	}
	priceResult := prices.FetchAll(ctx, uniqueProductCodes(products))
	for code, price := range priceResult.Prices {
		if err := s.store.UpsertPrice(code, today, price); err != nil {
			return 0, err
		}
	}
	if err := s.updateTodayObservationRecords(products, today, priceResult.Prices); err != nil {
		return 0, err
	}
	payload, err := s.buildObservationProducts(products, today, true)
	if err != nil {
		return 0, err
	}
	text := buildFeishuPushText(today, payload)
	if len(payload) == 0 {
		text = "今日产品派息/敲出观察提醒\n观察日期：" + today + "\n\n今日无产品需要观察。"
	}
	if err := sendFeishuWebhook(ctx, webhookURL, text); err != nil {
		return 0, err
	}
	return len(payload), nil
}

func (s *Server) startScheduler() {
	s.Cron.AddFunc("30 11 * * 1-5", s.scheduledPriceUpdate)
	s.Cron.AddFunc("0 15 * * 1-5", s.scheduledPriceUpdate)
	s.Cron.AddFunc("30 15 * * 1-5", s.scheduledPriceUpdate)
	s.Cron.AddFunc("5 15 * * 1-5", s.generateAutoPosters)
	s.Cron.AddFunc("0 10 * * *", s.scheduledObservationEmail)
	s.Cron.AddFunc("10 15 * * *", s.scheduledObservationEmail)
	s.Cron.AddFunc("* * * * *", s.handleFeishuPushMinute)
}

var feishuLastRunKey string

func (s *Server) handleFeishuPushMinute() {
	now := time.Now()
	cfg, err := s.store.GetPushConfig(s.cfg.FeishuPushWebhook)
	if err != nil || !cfg.Enabled || strings.TrimSpace(cfg.WebhookURL) == "" {
		return
	}
	if now.Hour() != cfg.CronHour || now.Minute() != cfg.CronMinute {
		return
	}
	runKey := now.Format("2006-01-02 15:04")
	if runKey == feishuLastRunKey {
		return
	}
	feishuLastRunKey = runKey
	count, pushErr := s.executeScheduledObservationPush(context.Background(), cfg.WebhookURL)
	result := fmt.Sprintf("success (%d products)", count)
	if pushErr != nil {
		result = "error: " + pushErr.Error()
	}
	_ = s.store.UpdatePushResult(time.Now().UTC().Format(time.RFC3339Nano), result)
}

func (s *Server) scheduledPriceUpdate() {
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		fmt.Printf("[定时任务] 获取进行中产品失败: %v\n", err)
		return
	}
	codes := uniqueProductCodes(products)
	if len(codes) == 0 {
		return
	}
	fmt.Printf("[定时任务] 开始更新 %d 个标的价格...\n", len(codes))
	priceResult := prices.FetchAll(context.Background(), codes)
	today := time.Now().Format("2006-01-02")
	for code, price := range priceResult.Prices {
		if err := s.store.UpsertPrice(code, today, price); err != nil {
			fmt.Printf("[定时任务] 写入价格 %s 失败: %v\n", code, err)
			return
		}
	}
	updatedObs, err := s.updateTodayObservations(products, today, priceResult.Prices)
	if err != nil {
		fmt.Printf("[定时任务] 更新今日观察记录失败: %v\n", err)
		return
	}
	failed := len(priceResult.Failed)
	fmt.Printf("[定时任务] 完成: 价格更新 %d/%d, 观察记录更新 %d 条\n", len(codes)-failed, len(codes), updatedObs)
}

func (s *Server) updateTodayObservations(products []model.Product, today string, latest map[string]float64) (int, error) {
	count := 0
	for _, product := range products {
		obs, ok := observationInfoForDate(product, today)
		if !ok {
			continue
		}
		price, ok := priceForObservation(s.store, product, today, latest)
		if !ok {
			continue
		}
		eval := observations.EvaluateObservation(product, today, price, obs.MonthsSinceEntry)
		if err := s.store.UpsertObservation(product.ID, observationEvalMap(eval)); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *Server) generateAutoPosters() {
	today := time.Now().Format("2006-01-02")
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		fmt.Printf("[喜报生成] 获取产品失败: %v\n", err)
		return
	}
	knockoutCount, dividendCount := 0, 0
	for _, product := range products {
		if product.Code == "" {
			continue
		}
		targetObsInfo, ok := observationInfoForDate(product, today)
		if !ok {
			continue
		}
		records, err := s.store.QueryObservationsByProduct(product.ID)
		if err != nil {
			continue
		}
		var targetRecord *model.Observation
		for i := range records {
			if records[i].ObservationDate == today {
				targetRecord = &records[i]
				break
			}
		}
		if targetRecord == nil {
			continue
		}
		data := posters.GenerateData(product, today, targetObsInfo.MonthsSinceEntry)
		isKnockout := data.KnockoutValue != "" && targetRecord.IsKnockedOut == "是"
		isDividend := data.HasDividendObservation && data.DividendBarrierValue != "" && targetRecord.IsDividend == "是"

		if isKnockout {
			knockoutCount++
			row := posterRow(product, today, targetObsInfo.MonthsSinceEntry, data, "knockout")
			row.AbsoluteReturn = floatPtr(data.AbsoluteReturn)
			row.DurationMonths = intPtr(targetObsInfo.MonthsSinceEntry)
			row.DividendBarrierValue = ""
			row.DividendCount = intPtr(0)
			row.CumulativeRate = floatPtr(0)
			_ = s.store.UpsertPoster(row)
			if !isDividend {
				continue
			}
		}
		if isDividend && !isKnockout {
			dividendCount++
			row := posterRow(product, today, targetObsInfo.MonthsSinceEntry, data, "dividend")
			row.AbsoluteReturn = floatPtr(0)
			row.DurationMonths = intPtr(0)
			row.DividendCount = intPtr(data.DividendCount)
			row.CumulativeRate = floatPtr(data.CumulativeDividendRate)
			_ = s.store.UpsertPoster(row)
		}
	}
	fmt.Printf("[喜报生成] 今日自动生成：敲出喜报 %d 张，派息喜报 %d 张\n", knockoutCount, dividendCount)
}

func (s *Server) scheduledObservationEmail() {
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		fmt.Printf("[邮件提醒] 获取产品失败: %v\n", err)
		return
	}
	codes := uniqueProductCodes(products)
	if len(codes) == 0 {
		return
	}
	priceResult := prices.FetchAll(context.Background(), codes)
	today := time.Now().Format("2006-01-02")
	for code, price := range priceResult.Prices {
		_ = s.store.UpsertPrice(code, today, price)
	}

	emailCfg := email.Config{
		SMTPHost:   s.cfg.SMTPHost,
		SMTPPort:   s.cfg.SMTPPort,
		SMTPSecure: s.cfg.SMTPSecure,
		SMTPUser:   s.cfg.SMTPUser,
		SMTPPass:   s.cfg.SMTPPass,
		SMTPFrom:   s.cfg.SMTPFrom,
	}
	notification := email.BuildTodayNotification(products, priceResult.Prices, today, emailCfg)
	sent, reason := email.SendObservationEmail(emailCfg, notification)
	if sent {
		fmt.Printf("[邮件提醒] 已发送今日观察提醒: %d 个产品\n", len(notification.Products))
	} else {
		fmt.Printf("[邮件提醒] 未发送: %s\n", reason)
	}
}

func (s *Server) activityLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	logs, err := s.store.QueryActivityLogs(c.Query("type"), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (s *Server) search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"results": []any{}})
		return
	}
	results, err := s.store.SearchProducts(q)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (s *Server) agentConversations(c *gin.Context) {
	conversations, err := s.store.AgentConversations()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, conversations)
}

func (s *Server) createAgentConversation(c *gin.Context) {
	var req struct {
		Title string `json:"title"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Title == "" {
		req.Title = "新对话"
	}
	id, err := s.store.CreateAgentConversation(req.Title)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "title": req.Title, "created_at": time.Now().UTC().Format(time.RFC3339Nano)})
}

func (s *Server) deleteAgentConversation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := s.store.DeleteAgentConversation(id); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) agentMessages(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	messages, err := s.store.AgentMessages(id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, messages)
}

func (s *Server) agentChat(c *gin.Context) {
	var req struct {
		ConversationID int64  `json:"conversation_id"`
		Message        string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	conversationID := req.ConversationID
	if conversationID == 0 {
		title := req.Message
		if len([]rune(title)) > 30 {
			title = string([]rune(title)[:30])
		}
		id, err := s.store.CreateAgentConversation(title)
		if err != nil {
			writeError(c, err)
			return
		}
		conversationID = id
	}

	if err := s.store.AddAgentMessage(conversationID, "user", req.Message, "", ""); err != nil {
		writeError(c, err)
		return
	}
	count, err := s.store.AgentMessageCount(conversationID)
	if err != nil {
		writeError(c, err)
		return
	}
	if count <= 1 {
		title := req.Message
		if len([]rune(title)) > 30 {
			title = string([]rune(title)[:30])
		}
		_ = s.store.UpdateAgentConversationTitle(conversationID, title)
	}

	history, err := s.store.AgentMessages(conversationID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	writeSSE(c, gin.H{"type": "conversation_id", "conversation_id": conversationID})

	content, err := s.agentSvc.StreamChat(c.Request.Context(), history, req.Message, agent.StreamCallbacks{
		OnReasoning: func(text string) {
			writeSSE(c, gin.H{"type": "reasoning_delta", "text": text})
		},
		OnDelta: func(text string) {
			writeSSE(c, gin.H{"type": "delta", "text": text})
		},
		OnToolCall: func(name string) {
			writeSSE(c, gin.H{"type": "tool_call", "name": name})
		},
	})
	if err != nil {
		writeSSE(c, gin.H{"type": "error", "error": err.Error()})
		return
	}
	if err := s.store.AddAgentMessage(conversationID, "assistant", content, "", ""); err != nil {
		writeSSE(c, gin.H{"type": "error", "error": err.Error()})
		return
	}
	writeSSE(c, gin.H{"type": "done", "usage": nil})
}

func writeSSE(c *gin.Context, payload gin.H) {
	data, _ := json.Marshal(payload)
	_, _ = c.Writer.Write([]byte("data: "))
	_, _ = c.Writer.Write(data)
	_, _ = c.Writer.Write([]byte("\n\n"))
	c.Writer.Flush()
}

func (s *Server) notImplemented(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": message})
	}
}

func (s *Server) syncNotMigrated(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.feishu.Authorized() {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权，请先登录飞书"})
			return
		}
		c.JSON(http.StatusNotImplemented, gin.H{"error": message})
	}
}

func (s *Server) buildObservationProducts(products []model.Product, today string, onlyToday bool) ([]gin.H, error) {
	result := []gin.H{}
	for _, product := range products {
		scheduled := observations.DatesUntil(product, today)
		hasToday := false
		for _, item := range scheduled {
			if item.Date == today {
				hasToday = true
				break
			}
		}
		if onlyToday && !hasToday {
			continue
		}
		records, err := s.store.QueryObservationsByProduct(product.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, productObservationPayload(product, mergeObservations(product, scheduled, records, today)))
	}
	return result, nil
}

func (s *Server) updateTodayObservationRecords(products []model.Product, today string, latest map[string]float64) error {
	for _, product := range products {
		obs, ok := observationInfoForDate(product, today)
		if !ok {
			continue
		}
		price, ok := priceForObservation(s.store, product, today, latest)
		if !ok {
			continue
		}
		eval := observations.EvaluateObservation(product, today, price, obs.MonthsSinceEntry)
		if err := s.store.UpsertObservation(product.ID, observationEvalMap(eval)); err != nil {
			return err
		}
	}
	return nil
}

func productObservationPayload(product model.Product, merged []gin.H) gin.H {
	return gin.H{
		"id":                    product.ID,
		"name":                  product.Name,
		"manager":               product.Manager,
		"holding_status":        product.HoldingStatus,
		"code":                  product.Code,
		"entry_price":           product.EntryPrice,
		"first_knockout_ratio":  product.FirstKnockoutRatio,
		"lock_months":           product.LockMonths,
		"monthly_decrease":      product.MonthlyDecrease,
		"issue_date":            product.IssueDate,
		"subscribe_amount":      product.SubscribeAmount,
		"dividend_barrier":      product.DividendBarrier,
		"holiday_adjust":        product.HolidayAdjust,
		"lock_days":             product.LockDays,
		"duration_months":       product.DurationMonths,
		"next_observation_date": observations.NextObservationDate(product, time.Now().Format("2006-01-02")),
		"observations":          merged,
	}
}

func mergeObservations(product model.Product, scheduled []observations.ObservationDate, records []model.Observation, today string) []gin.H {
	merged := []gin.H{}
	seen := map[string]bool{}
	for _, record := range records {
		seen[record.ObservationDate] = true
		merged = append(merged, gin.H{
			"date":               record.ObservationDate,
			"knockout_price":     record.KnockoutPrice,
			"dividend_line":      record.DividendLine,
			"underlying_price":   record.UnderlyingPrice,
			"is_knocked_out":     record.IsKnockedOut,
			"is_dividend":        record.IsDividend,
			"months_since_entry": record.MonthsSinceEntry,
		})
	}
	for _, obs := range scheduled {
		if seen[obs.Date] {
			continue
		}
		knockoutPrice := observations.ComputeKnockoutPrice(product, obs.MonthsSinceEntry)
		dividendLine := observations.ComputeDividendLine(product)
		merged = append(merged, gin.H{
			"date":               obs.Date,
			"knockout_price":     knockoutPrice,
			"dividend_line":      dividendLine,
			"underlying_price":   nil,
			"is_knocked_out":     "--",
			"is_dividend":        "--",
			"months_since_entry": obs.MonthsSinceEntry,
		})
	}
	sortObservations(merged)
	return merged
}

func sortObservations(rows []gin.H) {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j]["date"].(string) < rows[i]["date"].(string) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

func filterObservationProducts(products []model.Product, search, code string) []model.Product {
	search = strings.ToLower(strings.TrimSpace(search))
	code = strings.ToLower(strings.TrimSpace(code))
	result := []model.Product{}
	for _, product := range products {
		if search != "" && !strings.Contains(strings.ToLower(product.Name), search) && !strings.Contains(strings.ToLower(product.ID), search) {
			continue
		}
		if code != "" && !strings.Contains(strings.ToLower(product.Code), code) {
			continue
		}
		result = append(result, product)
	}
	return result
}

func mapProductSheetRows(rows []map[string]any) []model.Product {
	products := []model.Product{}
	for _, row := range rows {
		flightID := strings.TrimSpace(cellString(row["航班编号"]))
		if flightID == "" {
			continue
		}
		lockDays := int(numberFromCell(row["锁定期"]))
		entryPrice := numberFromCell(row["入场价"])
		rawKO := cellString(row["敲出"])
		product := model.Product{
			ID:                 flightID,
			Name:               cellString(row["产品名称"]),
			IsMain:             intPtr(boolInt(strings.TrimSpace(cellString(firstValue(row, "是否主产品", " 是否主产品"))) == "是")),
			IssueDate:          excelDateToString(row["认购日"]),
			CompleteDate:       excelDateToString(row["完结时间"]),
			SubscribeAmount:    floatPtr(numberFromCell(row["认购金额"])),
			OutstandingAmount:  floatPtr(numberFromCell(row["存续金额"])),
			Manager:            cellString(row["私募管理人"]),
			HoldingStatus:      cellString(row["持有状态"]),
			StructureType:      cellString(row["结构类型"]),
			Code:               cellString(row["代码"]),
			LockDays:           intPtr(lockDays),
			LockMonths:         intPtr(lockDays / 30),
			FirstKnockoutRatio: floatPtr(parseFirstKnockoutRatio(rawKO, entryPrice)),
			EntryPrice:         floatPtr(entryPrice),
			MonthlyDecrease:    floatPtr(parseRatioValue(findSheetField(row, "每月递减"))),
			Term:               cellString(findSheetField(row, "期限")),
			Parachute:          cellString(row["降落伞"]),
			DividendBarrier:    floatPtr(parseRatioValue(findSheetField(row, "派息障碍"))),
			MonthlyCoupon:      floatPtr(parseRatioValue(findSheetField(row, "月票息"))),
			Coupon1st:          floatPtr(parseRatioValue(findSheetField(row, "第一段票息"))),
			Coupon2nd:          floatPtr(parseRatioValue(findSheetField(row, "第二段票息"))),
			Coupon3rd:          floatPtr(parseRatioValue(findSheetField(row, "第三段票息"))),
			DurationMonths:     floatPtr(numberFromCell(findSheetField(row, "存续时间"))),
			AbsoluteReturn:     floatPtr(numberFromCell(findSheetField(row, "绝对收益率"))),
			HolidayAdjust:      cellString(findSheetField(row, "观察日节假日")),
			DurationDays:       intPtr(int(numberFromCell(findSheetField(row, "存续天数")))),
			KnockedIn:          cellString(findSheetField(row, "是否敲入")),
			MarginRatio:        floatPtr(parseRatioValue(findSheetField(row, "保证金比例"))),
			Custodian:          cellString(findSheetField(row, "托管券商")),
			Counterparty:       cellString(findSheetField(row, "交易对手")),
		}
		product.KnockIn = cellString(findSheetField(row, "鏁插叆"))
		if strings.Contains(rawKO, "*") {
			product.FirstKnockoutRatio = floatPtr(0)
		}
		raw, _ := json.Marshal(row)
		product.Raw = string(raw)
		products = append(products, product)
	}
	return products
}

func mapTransactionSheetRows(rows []map[string]any) []map[string]any {
	transactions := []map[string]any{}
	for _, row := range rows {
		flightID := strings.TrimSpace(cellString(row["航班编号"]))
		if flightID == "" {
			continue
		}
		raw, _ := json.Marshal(row)
		transactions = append(transactions, map[string]any{
			"transaction_date":      excelDateToString(row["交易日期"]),
			"flight_id":             flightID,
			"counterparty":          cellString(firstValue(row, "交易对手", "对手方")),
			"subscribe_amount":      numberFromCell(row["认购金额"]),
			"product_name":          cellString(row["产品名字"]),
			"customer_name":         cellString(row["姓名"]),
			"actual_buyer":          cellString(row["实际申购人"]),
			"amount":                numberFromCell(row["金额/万"]),
			"subscribe_fee_ratio":   parseRatioValue(findSheetField(row, "申购费返还比例")),
			"management_fee_ratio":  parseRatioValue(findSheetField(row, "管理费返还比例")),
			"performance_fee_ratio": parseRatioValue(findSheetField(row, "业绩报酬返还比例")),
			"rebate_target":         cellString(row["返还对象"]),
			"flight_date":           excelDateToString(firstValue(row, "航班日期")),
			"holding_status":        cellString(row["存续状态"]),
			"complete_date":         excelDateToString(row["完结日期"]),
			"underlying":            cellString(row["挂钩标的"]),
			"structure_type":        cellString(row["结构类型"]),
			"lock_period":           cellString(row["锁定期"]),
			"dividend_barrier":      parseRatioValue(findSheetField(row, "派息障碍")),
			"monthly_coupon":        parseRatioValue(findSheetField(row, "月票息")),
			"coupon_1st":            parseRatioValue(findSheetField(row, "第一段票息")),
			"raw":                   string(raw),
		})
	}
	return transactions
}

func mapCoInvestRecords(records []map[string]any) []map[string]any {
	rows := []map[string]any{}
	for _, fields := range records {
		raw, _ := json.Marshal(fields)
		rows = append(rows, map[string]any{
			"user_name":            cellString(fields["名义购买人"]),
			"actual_buyer":         cellString(fields["实际购买人"]),
			"phone":                cellString(fields["手机号"]),
			"wechat":               cellString(fields["微信昵称"]),
			"total_assets":         numberFromCell(fields["资产总和/万"]),
			"risk_tolerance":       cellString(fields["风险承受"]),
			"industry":             firstNonEmptyString(cellString(fields["行业"]), cellString(fields["客户行业"])),
			"is_actual_deal":       cellString(fields["是否成交客户"]),
			"lead_source":          cellString(fields["进线来源分类"]),
			"asset_match":          cellString(fields["资产匹配度"]),
			"is_dedicated_account": cellString(fields["是否专户客户"]),
			"is_competitor":        cellString(fields["客户是否竞品群"]),
			"raw":                  string(raw),
		})
	}
	return rows
}

func parseProductStructure(text string) map[string]string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	patterns := map[string]string{
		"结构":      `结构[：:]\s*(.+)`,
		"标的":      `标的[：:]\s*(.+)`,
		"期限":      `期限[：:]\s*(.+)`,
		"保证金比例":   `保证金比例[：:]\s*(.+)`,
		"期初敲出线":   `(?:期初)?敲出线[：:]\s*(.+)`,
		"降敲":      `降敲[：:]\s*(.+)`,
		"降落伞":     `降落伞[：:]\s*(.+)`,
		"派息线":     `派息线[：:]\s*(.+)`,
		"票息（税费后）": `票息[（(]税费后[）)][：:]\s*(.+)`,
		"打款时间":    `打款时间[：:]\s*(.+)`,
		"入场时间":    `入场时间[：:]\s*(.+)`,
	}
	result := map[string]string{}
	for field, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			result[field] = strings.TrimSpace(match[1])
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func uniqueProductCodes(products []model.Product) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, product := range products {
		code := strings.TrimSpace(product.Code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		result = append(result, code)
	}
	return result
}

func priceForObservation(store *db.Store, product model.Product, date string, latest map[string]float64) (float64, bool) {
	if product.Code == "" {
		return 0, false
	}
	if cached, err := store.PriceByDate(product.Code, date); err == nil && cached != nil {
		if value, ok := numberFromAny(cached["price"]); ok {
			return value, true
		}
	}
	if value, ok := latest[product.Code]; ok {
		return value, true
	}
	return 0, false
}

func observationEvalMap(eval observations.Evaluation) map[string]any {
	return map[string]any{
		"observation_date":   eval.ObservationDate,
		"knockout_price":     nullableFloat(eval.KnockoutPrice),
		"dividend_line":      nullableFloat(eval.DividendLine),
		"underlying_price":   eval.UnderlyingPrice,
		"is_knocked_out":     eval.IsKnockedOut,
		"is_dividend":        eval.IsDividend,
		"months_since_entry": eval.MonthsSinceEntry,
	}
}

func observationInfoForDate(product model.Product, targetDate string) (observations.ObservationDate, bool) {
	if len(targetDate) < 7 {
		return observations.ObservationDate{}, false
	}
	for _, item := range observations.DatesForMonth(product, targetDate[:7]) {
		if item.Date == targetDate {
			return item, true
		}
	}
	return observations.ObservationDate{}, false
}

func posterRow(product model.Product, targetDate string, monthsSinceEntry int, data posters.Data, posterType string) model.Poster {
	return model.Poster{
		ProductID:            product.ID,
		PosterType:           posterType,
		ObservationDate:      targetDate,
		ProductName:          product.Name,
		DateDisplay:          posters.FormatChineseDate(targetDate),
		MonthsSinceEntry:     intPtr(monthsSinceEntry),
		UnderlyingName:       data.UnderlyingName,
		AnnualizedReturn:     floatPtr(data.AnnualizedReturn),
		ParachuteValue:       data.ParachuteValue,
		KnockoutValue:        data.KnockoutValue,
		DividendBarrierValue: data.DividendBarrierValue,
		MonthlyCoupon:        floatPtr(data.MonthlyCoupon),
		EntryDate:            product.IssueDate,
	}
}

func numberFromAny(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case []byte:
		var out float64
		_, err := fmt.Sscanf(string(v), "%f", &out)
		return out, err == nil
	case string:
		var out float64
		_, err := fmt.Sscanf(v, "%f", &out)
		return out, err == nil
	default:
		return 0, false
	}
}

func firstValue(row map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			return value
		}
	}
	return nil
}

func findSheetField(row map[string]any, patterns ...string) any {
	for _, pattern := range patterns {
		normalizedPattern := strings.ReplaceAll(pattern, " ", "")
		for key, value := range row {
			normalizedKey := strings.ReplaceAll(key, " ", "")
			if normalizedKey == normalizedPattern || strings.Contains(normalizedKey, normalizedPattern) {
				return value
			}
		}
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cellString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return strings.TrimSpace(fmt.Sprint(v))
	case bool:
		return fmt.Sprint(v)
	case []any:
		parts := []string{}
		for _, item := range v {
			if text := cellString(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "、"))
	case map[string]any:
		if text, ok := v["text"]; ok {
			return cellString(text)
		}
		if name, ok := v["name"]; ok {
			return cellString(name)
		}
		if elements, ok := v["elements"]; ok {
			return cellString(elements)
		}
		data, _ := json.Marshal(v)
		return string(data)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func numberFromCell(value any) float64 {
	switch v := value.(type) {
	case nil:
		return 0
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		var out float64
		_, _ = fmt.Sscanf(strings.ReplaceAll(strings.TrimSpace(v), ",", ""), "%f", &out)
		return out
	default:
		return numberFromCell(cellString(value))
	}
}

func excelDateToString(value any) string {
	if value == nil {
		return ""
	}
	if text := cellString(value); text != "" {
		if _, err := time.Parse("2006-01-02", text); err == nil {
			return text
		}
	}
	num := numberFromCell(value)
	if num < 1 {
		return cellString(value)
	}
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	return epoch.Add(time.Duration(num*86400) * time.Second).Format("2006-01-02")
}

func parseRatioValue(value any) float64 {
	text := cellString(value)
	if text == "" {
		return 0
	}
	cleaned := strings.ReplaceAll(text, "%", "")
	num := numberFromCell(cleaned)
	if strings.Contains(text, "%") {
		return num / 100
	}
	if num > 2 || num < -2 {
		return num / 100
	}
	return num
}

func parseFirstKnockoutRatio(raw string, entryPrice float64) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	price := numberFromCell(raw)
	if !strings.Contains(raw, "%") && entryPrice != 0 && price > 2 {
		return price / entryPrice
	}
	return parseRatioValue(raw)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}

func floatPtr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func buildFeishuPushText(today string, products []gin.H) string {
	lines := []string{
		"今日产品派息/敲出观察提醒",
		"观察日期：" + today,
		fmt.Sprintf("今日需要观察产品数量：%d", len(products)),
		"",
	}
	for _, product := range products {
		obs := latestObservation(product)
		lines = append(lines,
			"产品："+formatValue(product["name"]),
			"航班编号："+formatValue(product["id"]),
			"私募管理人："+formatValue(product["manager"]),
			"标的代码："+formatValue(product["code"]),
			"入场价："+formatValue(product["entry_price"]),
			"实时标的价格："+formatValue(obs["underlying_price"]),
			"敲出价："+formatValue(obs["knockout_price"]),
			"派息线："+formatValue(obs["dividend_line"]),
			"是否敲出："+formatValue(obs["is_knocked_out"]),
			"是否派息："+formatValue(obs["is_dividend"]),
			"",
		)
	}
	return strings.Join(lines, "\n")
}

func latestObservation(product gin.H) gin.H {
	rows, ok := product["observations"].([]gin.H)
	if !ok || len(rows) == 0 {
		return gin.H{}
	}
	return rows[len(rows)-1]
}

func formatValue(value any) string {
	if value == nil {
		return "--"
	}
	switch v := value.(type) {
	case *float64:
		if v == nil {
			return "--"
		}
		return fmt.Sprintf("%.2f", *v)
	case *int:
		if v == nil {
			return "--"
		}
		return fmt.Sprint(*v)
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" || text == "<nil>" {
			return "--"
		}
		return text
	}
}

func sendFeishuWebhook(ctx context.Context, webhookURL string, text string) error {
	payload, _ := json.Marshal(gin.H{"msg_type": "text", "content": gin.H{"text": text}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook status %d", resp.StatusCode)
	}
	if body.Code != 0 {
		return fmt.Errorf("feishu push failed (%d): %s", body.Code, body.Msg)
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func writeError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func writeDriveError(c *gin.Context, err error) {
	if strings.Contains(err.Error(), "未授权") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (s *Server) holdingProducts(c *gin.Context) {
	f := db.ProductFilter{
		IssueDateStart:    c.Query("issue_date_start"),
		IssueDateEnd:      c.Query("issue_date_end"),
		HoldingStatus:     c.Query("holding_status"),
		Manager:           c.Query("manager"),
		CompleteDateStart: c.Query("complete_date_start"),
		CompleteDateEnd:   c.Query("complete_date_end"),
		Code:              c.Query("code"),
		StructureType:     c.Query("structure_type"),
		LockMonths:        c.Query("lock_months"),
		MarginRatio:       c.Query("margin_ratio"),
	}
	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := parsePositiveInt(c.DefaultQuery("page_size", "50"), 50)
	products, err := s.store.QueryHoldingProducts(f)
	if err != nil {
		writeError(c, err)
		return
	}
	var result []map[string]any
	for _, p := range products {
		item := map[string]any{
			"id":                   p.ID,
			"name":                 p.Name,
			"manager":              p.Manager,
			"holding_status":       p.HoldingStatus,
			"issue_date":           p.IssueDate,
			"structure_type":       p.StructureType,
			"code":                 p.Code,
			"lock_days":            p.LockDays,
			"lock_months":          p.LockMonths,
			"entry_price":          p.EntryPrice,
			"first_knockout_ratio": p.FirstKnockoutRatio,
			"monthly_decrease":     p.MonthlyDecrease,
			"term":                 p.Term,
			"complete_date":        p.CompleteDate,
			"parachute":            p.Parachute,
			"dividend_barrier":     p.DividendBarrier,
			"monthly_coupon":       p.MonthlyCoupon,
			"coupon_1st":           p.Coupon1st,
			"coupon_2nd":           p.Coupon2nd,
			"coupon_3rd":           p.Coupon3rd,
			"absolute_return":      p.AbsoluteReturn,
			"knocked_in":           p.KnockedIn,
			"margin_ratio":         p.MarginRatio,
			"custodian":            p.Custodian,
			"counterparty":         p.Counterparty,
			"duration_months":      p.DurationMonths,
		}
		item["knock_in"] = p.KnockIn
		status := strings.TrimSpace(p.HoldingStatus)
		if strings.Contains(status, "存续") {
			if p.DurationDays != nil && *p.DurationDays > 0 {
				months := float64(*p.DurationDays) / 30.0
				item["duration_months"] = math.Round(months*10) / 10
			}
		}
		result = append(result, item)
	}
	if result == nil {
		result = []map[string]any{}
	}
	total := len(result)
	c.JSON(http.StatusOK, gin.H{
		"items": paginateMaps(result, page, pageSize),
		"total": total,
	})
}

func (s *Server) holdingTransactions(c *gin.Context) {
	f := db.TransactionFilter{
		CustomerName:      c.Query("customer_name"),
		ActualBuyer:       c.Query("actual_buyer"),
		RebateTarget:      c.Query("rebate_target"),
		HoldingStatus:     c.Query("holding_status"),
		FlightDateStart:   c.Query("flight_date_start"),
		FlightDateEnd:     c.Query("flight_date_end"),
		CompleteDateStart: c.Query("complete_date_start"),
		CompleteDateEnd:   c.Query("complete_date_end"),
		ProductName:       c.Query("product_name"),
	}
	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := parsePositiveInt(c.DefaultQuery("page_size", "50"), 50)
	obsDateStart := c.Query("obs_date_start")
	obsDateEnd := c.Query("obs_date_end")
	observeDividend := c.Query("observe_dividend") == "true"
	observeKnockout := c.Query("observe_knockout") == "true"
	nameCheck := c.DefaultQuery("match_name", "true")
	buyerCheck := c.DefaultQuery("match_buyer", "true")
	if nameCheck == "true" && buyerCheck == "true" {
		f.MatchType = "any"
	} else if nameCheck == "true" {
		f.MatchType = "name_only"
	} else {
		f.MatchType = "buyer_only"
	}
	if f.CustomerName == "" {
		f.MatchType = ""
	}

	transactions, err := s.store.QueryHoldingTransactions(f)
	if err != nil {
		writeError(c, err)
		return
	}

	allProducts, _ := s.store.QueryProducts("", "")
	productMap := map[string]model.Product{}
	for _, p := range allProducts {
		productMap[p.ID] = p
	}

	today := time.Now().Format("2006-01-02")
	var result []map[string]any
	for _, t := range transactions {
		product, hasProduct := productMap[t.FlightID]
		item := map[string]any{
			"id":                    t.ID,
			"product_name":          t.ProductName,
			"customer_name":         t.CustomerName,
			"actual_buyer":          t.ActualBuyer,
			"amount":                t.Amount,
			"subscribe_fee_ratio":   t.SubscribeFeeRatio,
			"management_fee_ratio":  t.ManagementFeeRatio,
			"performance_fee_ratio": t.PerformanceFeeRatio,
			"rebate_target":         t.RebateTarget,
			"flight_date":           t.FlightDate,
			"holding_status":        t.HoldingStatus,
			"complete_date":         t.CompleteDate,
			"flight_id":             t.FlightID,
			"underlying":            t.Underlying,
			"structure_type":        t.StructureType,
			"lock_period":           t.LockPeriod,
			"dividend_barrier":      t.DividendBarrier,
			"monthly_coupon":        t.MonthlyCoupon,
			"coupon_1st":            t.Coupon1st,
			"entry_price":           pNilOr(product.EntryPrice, hasProduct),
			"first_knockout_ratio":  pNilOrF(product.FirstKnockoutRatio, hasProduct),
			"monthly_decrease":      pNilOrF(product.MonthlyDecrease, hasProduct),
			"parachute":             pNilOrStr(product.Parachute, hasProduct),
			"coupon_2nd":            pNilOrF(product.Coupon2nd, hasProduct),
			"coupon_3rd":            pNilOrF(product.Coupon3rd, hasProduct),
		}

		item["observation_day"], item["observation_type"] = s.computeObservationDay(t, product, hasProduct, today)
		item["knockout_price"], item["today_price"], item["knockout_position"] = s.computeKnockoutAndPrice(t, product, hasProduct, today)

		if !matchesObservationFilters(item, obsDateStart, obsDateEnd, observeDividend, observeKnockout) {
			continue
		}

		result = append(result, item)
	}
	if result == nil {
		result = []map[string]any{}
	}
	total := len(result)
	c.JSON(http.StatusOK, gin.H{
		"items": paginateMaps(result, page, pageSize),
		"total": total,
	})
}

func (s *Server) holdingFilterOptions(c *gin.Context) {
	managers, _ := s.store.QueryDistinctValues("products", "manager")
	statuses, _ := s.store.QueryDistinctValues("products", "holding_status")
	structureTypes, _ := s.store.QueryDistinctValues("products", "structure_type")
	codes, _ := s.store.QueryDistinctValues("products", "code")
	lockMonths, _ := s.store.QueryDistinctValues("products", "lock_months")
	marginRatios, _ := s.store.QueryDistinctValues("products", "margin_ratio")

	txStatuses, _ := s.store.QueryDistinctValues("transactions", "holding_status")
	rebateTargets, _ := s.store.QueryDistinctValues("transactions", "rebate_target")
	underlyings, _ := s.store.QueryDistinctValues("transactions", "underlying")

	c.JSON(http.StatusOK, gin.H{
		"managers":         managers,
		"statuses":         statuses,
		"holding_statuses": uniqueStrings(statuses, txStatuses),
		"structure_types":  structureTypes,
		"codes":            codes,
		"lock_months":      lockMonths,
		"margin_ratios":    marginRatios,
		"tx_statuses":      txStatuses,
		"rebate_targets":   rebateTargets,
		"underlyings":      underlyings,
	})
}

func (s *Server) holdingRefreshPrice(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	result := prices.FetchAll(c.Request.Context(), []string{code})
	if price, ok := result.Prices[code]; ok {
		today := time.Now().Format("2006-01-02")
		_ = s.store.UpsertPrice(code, today, price)
		c.JSON(http.StatusOK, gin.H{"code": code, "price": price, "date": today})
	} else {
		today := time.Now().Format("2006-01-02")
		c.JSON(http.StatusOK, gin.H{"code": code, "price": nil, "date": today})
	}
}

func pNilOr(v *float64, ok bool) any {
	if !ok || v == nil {
		return nil
	}
	return *v
}

func pNilOrF(v *float64, ok bool) *float64 {
	if !ok {
		return nil
	}
	return v
}

func pNilOrStr(v string, ok bool) string {
	if !ok {
		return ""
	}
	return v
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func paginateMaps(items []map[string]any, page, pageSize int) []map[string]any {
	if len(items) == 0 {
		return []map[string]any{}
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []map[string]any{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func uniqueStrings(groups ...[]string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, group := range groups {
		for _, item := range group {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	sort.Strings(result)
	return result
}

func matchesObservationFilters(item map[string]any, start, end string, observeDividend, observeKnockout bool) bool {
	obsDay, _ := item["observation_day"].(string)
	obsType, _ := item["observation_type"].(string)
	obsDay = strings.TrimSpace(obsDay)
	obsType = strings.TrimSpace(obsType)

	if start != "" {
		if obsDay == "" || obsDay == "宸插畬缁?" || obsDay < start {
			return false
		}
	}
	if end != "" {
		if obsDay == "" || obsDay == "宸插畬缁?" || obsDay > end {
			return false
		}
	}
	if observeDividend || observeKnockout {
		if obsType == "" {
			return false
		}
		hasDividend := strings.Contains(obsType, "娲炬伅")
		hasKnockout := strings.Contains(obsType, "鏁插嚭")
		if observeDividend && observeKnockout {
			return hasDividend || hasKnockout
		}
		if observeDividend {
			return hasDividend
		}
		if observeKnockout {
			return hasKnockout
		}
	}
	return true
}

func (s *Server) computeObservationDay(t model.TransactionRow, product model.Product, hasProduct bool, today string) (string, string) {
	status := strings.TrimSpace(t.HoldingStatus)
	if strings.Contains(status, "完结") {
		return "已完结", ""
	}
	if !hasProduct || product.IssueDate == "" {
		return "", ""
	}

	hasDividend := t.MonthlyCoupon != nil && *t.MonthlyCoupon > 0
	lockMonths := 0
	if product.LockMonths != nil {
		lockMonths = *product.LockMonths
	}

	for months := 1; months < 600; months++ {
		rawDate := observations.AddMonths(product.IssueDate, months)
		adjusted := observations.AdjustForHoliday(rawDate, product.HolidayAdjust)
		if adjusted >= today {
			observeKnockout := months >= lockMonths
			if hasDividend && observeKnockout {
				return adjusted, "派息/敲出"
			} else if hasDividend {
				return adjusted, "派息"
			} else if observeKnockout {
				return adjusted, "敲出"
			}
			return adjusted, ""
		}
	}
	return "", ""
}

func (s *Server) computeKnockoutAndPrice(t model.TransactionRow, product model.Product, hasProduct bool, today string) (*float64, *float64, string) {
	if !hasProduct {
		return nil, nil, ""
	}

	status := strings.TrimSpace(t.HoldingStatus)
	if strings.Contains(status, "完结") {
		allObs := observations.DatesUntil(product, today)
		if len(allObs) > 0 {
			lastObs := allObs[len(allObs)-1]
			kp := observations.ComputeKnockoutPrice(product, lastObs.MonthsSinceEntry)
			return kp, nil, ""
		}
		return nil, nil, ""
	}

	obsDay, _ := s.computeObservationDay(t, product, hasProduct, today)
	if obsDay == "" || obsDay == "已完结" {
		var todayPrice *float64
		if product.Code != "" {
			if cached, err := s.store.LatestPrice(product.Code); err == nil && cached != nil {
				if v, ok := numberFromAny(cached["price"]); ok {
					todayPrice = &v
				}
			}
		}
		return nil, todayPrice, ""
	}

	entryDate, _ := time.Parse("2006-01-02", product.IssueDate)
	obsDate, _ := time.Parse("2006-01-02", obsDay)
	if entryDate.IsZero() || obsDate.IsZero() {
		return nil, nil, ""
	}
	monthsSinceEntry := monthsBetween(entryDate, obsDate)

	lockMonths := 0
	if product.LockMonths != nil {
		lockMonths = *product.LockMonths
	}
	if monthsSinceEntry < lockMonths {
		var todayPrice *float64
		if product.Code != "" {
			if cached, err := s.store.LatestPrice(product.Code); err == nil && cached != nil {
				if v, ok := numberFromAny(cached["price"]); ok {
					todayPrice = &v
				}
			}
		}
		return nil, todayPrice, ""
	}

	knockoutPrice := observations.ComputeKnockoutPrice(product, monthsSinceEntry)

	var todayPrice *float64
	if product.Code != "" {
		if cached, err := s.store.LatestPrice(product.Code); err == nil && cached != nil {
			if v, ok := numberFromAny(cached["price"]); ok {
				todayPrice = &v
			}
		}
	}

	position := ""
	if knockoutPrice != nil && todayPrice != nil {
		if *todayPrice >= *knockoutPrice {
			position = "以上"
		} else {
			position = "以下"
		}
	}
	return knockoutPrice, todayPrice, position
}

func monthsBetween(a, b time.Time) int {
	years := b.Year() - a.Year()
	months := int(b.Month()) - int(a.Month())
	total := years*12 + months
	if total < 0 {
		return 0
	}
	return total
}

var schedulerInstance *cron.Cron

func GetCron() *cron.Cron {
	return schedulerInstance
}
