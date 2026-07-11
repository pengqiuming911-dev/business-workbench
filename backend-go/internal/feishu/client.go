package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// feishuBase 是飞书开放平台 base URL，默认官方；测试可覆盖以指向 httptest。
var feishuBase = "https://open.feishu.cn"

type Client struct {
	AppID       string
	AppSecret   string
	RedirectURI string
	HTTP        *http.Client

	mu           sync.RWMutex
	userToken    string
	refreshToken string
	expiresAt    time.Time
	tokenPath    string // 非空时 user token 持久化到此文件，重启不丢
}

// tokenData 用于 token 持久化的 JSON 结构
type tokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type DriveFile struct {
	Token       string
	Name        string
	Type        string
	ParentToken string
	ParentPath  string
}

type DriveWalkResult struct {
	Files       []DriveFile
	FolderCount int
}

type CurrentUser struct {
	Name      string `json:"name"`
	EnName    string `json:"en_name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
	OpenID    string `json:"open_id"`
	UnionID   string `json:"union_id"`
	UserID    string `json:"user_id"`
}

type CreatedDocx struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
}

type DocxBlock struct {
	BlockID   string `json:"block_id"`
	BlockType int    `json:"block_type"`
}

func New(appID, appSecret, redirectURI string) *Client {
	return &Client{
		AppID:       appID,
		AppSecret:   appSecret,
		RedirectURI: redirectURI,
		HTTP:        &http.Client{Timeout: 15 * time.Second},
	}
}

// SetTokenPersistPath 设置 user token 持久化文件路径，并尝试从文件加载已存在的 token。
func (c *Client) SetTokenPersistPath(path string) {
	c.mu.Lock()
	c.tokenPath = path
	c.mu.Unlock()
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := strings.TrimSpace(string(b))
	if content == "" {
		return
	}
	// 尝试 JSON 格式（新格式）
	var td tokenData
	if err := json.Unmarshal(b, &td); err == nil && td.AccessToken != "" {
		c.mu.Lock()
		c.userToken = td.AccessToken
		c.refreshToken = td.RefreshToken
		c.expiresAt = td.ExpiresAt
		c.mu.Unlock()
		return
	}
	// 兼容旧的纯文本格式（仅 access_token）
	c.mu.Lock()
	c.userToken = content
	c.mu.Unlock()
}

func (c *Client) persistToken() {
	c.mu.RLock()
	path := c.tokenPath
	tok := c.userToken
	td := tokenData{
		AccessToken:  c.userToken,
		RefreshToken: c.refreshToken,
		ExpiresAt:    c.expiresAt,
	}
	c.mu.RUnlock()
	if path == "" {
		return
	}
	if tok == "" {
		_ = os.Remove(path)
		return
	}
	data, err := json.Marshal(td)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

func (c *Client) Authorized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userToken != ""
}

func (c *Client) UserToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userToken
}

func (c *Client) ClearUserToken() {
	c.mu.Lock()
	c.userToken = ""
	c.refreshToken = ""
	c.expiresAt = time.Time{}
	c.mu.Unlock()
	c.persistToken()
}

func (c *Client) ExchangeCode(ctx context.Context, code string) error {
	appToken, err := c.appAccessToken(ctx)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": c.RedirectURI,
	}
	body, err := c.post(ctx, "https://open.feishu.cn/open-apis/authen/v1/oidc/access_token", payload, appToken)
	if err != nil {
		return err
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken      string `json:"access_token"`
			RefreshToken     string `json:"refresh_token"`
			ExpiresIn        int    `json:"expires_in"`
			RefreshExpiresIn int    `json:"refresh_expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("exchange user token failed (%d): %s", result.Code, result.Msg)
	}
	if result.Data.AccessToken == "" {
		return fmt.Errorf("exchange user token failed: empty access_token")
	}
	expiresAt := time.Now().Add(time.Duration(result.Data.ExpiresIn) * time.Second)
	c.mu.Lock()
	c.userToken = result.Data.AccessToken
	c.refreshToken = result.Data.RefreshToken
	c.expiresAt = expiresAt
	c.mu.Unlock()
	c.persistToken()
	return nil
}

// ensureValidToken 确保当前 access_token 有效，如果即将过期则自动刷新。
// 返回有效的 access_token，如果无法获取则返回错误。
func (c *Client) ensureValidToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	token := c.userToken
	refresh := c.refreshToken
	expiresAt := c.expiresAt
	c.mu.RUnlock()

	if token == "" {
		return "", fmt.Errorf("未授权，请先登录飞书")
	}

	// 如果没有 refresh_token 或未过期，直接返回当前 token
	// 提前 5 分钟刷新，避免边界情况
	if refresh == "" || time.Now().Before(expiresAt.Add(-5*time.Minute)) {
		return token, nil
	}

	// 需要刷新 token
	appToken, err := c.appAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("刷新 token 失败: %w", err)
	}

	payload := map[string]any{
		"grant_type":    "refresh_token",
		"refresh_token": refresh,
	}
	body, err := c.post(ctx, "https://open.feishu.cn/open-apis/authen/v1/oidc/refresh_access_token", payload, appToken)
	if err != nil {
		return "", fmt.Errorf("刷新 token 请求失败: %w", err)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken      string `json:"access_token"`
			RefreshToken     string `json:"refresh_token"`
			ExpiresIn        int    `json:"expires_in"`
			RefreshExpiresIn int    `json:"refresh_expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("刷新 token 响应解析失败: %w", err)
	}
	if result.Code != 0 {
		// refresh_token 已过期或无效，清除 token 要求重新登录
		c.ClearUserToken()
		return "", fmt.Errorf("登录已过期，请重新登录飞书 (%d): %s", result.Code, result.Msg)
	}
	if result.Data.AccessToken == "" {
		return "", fmt.Errorf("刷新 token 返回空 access_token")
	}

	newExpiresAt := time.Now().Add(time.Duration(result.Data.ExpiresIn) * time.Second)
	c.mu.Lock()
	c.userToken = result.Data.AccessToken
	c.refreshToken = result.Data.RefreshToken
	c.expiresAt = newExpiresAt
	c.mu.Unlock()
	c.persistToken()

	return result.Data.AccessToken, nil
}

func (c *Client) CurrentUser(ctx context.Context) (*CurrentUser, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}
	body, err := c.get(ctx, "https://open.feishu.cn/open-apis/authen/v1/user_info", token)
	if err != nil {
		return nil, err
	}
	var result struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data CurrentUser `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("fetch user info failed (%d): %s", result.Code, result.Msg)
	}
	return &result.Data, nil
}

func (c *Client) ReadSheetRows(ctx context.Context, spreadsheetToken string, sheetID string, colCount int) ([]map[string]any, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}
	const batch = 500
	const maxRows = 10000

	var allValues [][]any
	for startRow := 1; startRow <= maxRows; startRow += batch {
		endRow := startRow + batch - 1
		if endRow > maxRows {
			endRow = maxRows
		}
		sheetRange := fmt.Sprintf("%s!A%d:%s%d", sheetID, startRow, colLetter(colCount), endRow)
		endpoint := fmt.Sprintf(
			"https://open.feishu.cn/open-apis/sheets/v2/spreadsheets/%s/values/%s?valueRenderOption=UnformattedValue",
			spreadsheetToken,
			url.PathEscape(sheetRange),
		)
		body, err := c.get(ctx, endpoint, token)
		if err != nil {
			return nil, err
		}
		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				ValueRange struct {
					Values [][]any `json:"values"`
				} `json:"valueRange"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		if result.Code != 0 {
			return nil, fmt.Errorf("read sheet failed (%d): %s", result.Code, result.Msg)
		}
		values := result.Data.ValueRange.Values
		if len(values) == 0 {
			break
		}
		allValues = append(allValues, values...)
		if len(values) < batch {
			break
		}
	}
	return valuesToRows(allValues), nil
}

// ReadSheetRaw 读取 sheet 原始二维值（保留行顺序），用于需要按行定位（如分界行）的非标准布局。
func (c *Client) ReadSheetRaw(ctx context.Context, spreadsheetToken string, sheetID string, colCount int) ([][]any, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}
	const batch = 500
	const maxRows = 10000

	var allValues [][]any
	for startRow := 1; startRow <= maxRows; startRow += batch {
		endRow := startRow + batch - 1
		if endRow > maxRows {
			endRow = maxRows
		}
		sheetRange := fmt.Sprintf("%s!A%d:%s%d", sheetID, startRow, colLetter(colCount), endRow)
		endpoint := fmt.Sprintf(
			"https://open.feishu.cn/open-apis/sheets/v2/spreadsheets/%s/values/%s?valueRenderOption=UnformattedValue",
			spreadsheetToken,
			url.PathEscape(sheetRange),
		)
		body, err := c.get(ctx, endpoint, token)
		if err != nil {
			return nil, err
		}
		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				ValueRange struct {
					Values [][]any `json:"values"`
				} `json:"valueRange"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		if result.Code != 0 {
			return nil, fmt.Errorf("read sheet failed (%d): %s", result.Code, result.Msg)
		}
		values := result.Data.ValueRange.Values
		if len(values) == 0 {
			break
		}
		allValues = append(allValues, values...)
		if len(values) < batch {
			break
		}
	}
	return allValues, nil
}

func (c *Client) ReadBitableRecords(ctx context.Context, appToken string, tableID string) ([]map[string]any, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}

	result := []map[string]any{}
	pageToken := ""
	for {
		endpoint := fmt.Sprintf(
			"https://open.feishu.cn/open-apis/bitable/v1/apps/%s/tables/%s/records?page_size=500",
			appToken,
			tableID,
		)
		if pageToken != "" {
			endpoint += "&page_token=" + url.QueryEscape(pageToken)
		}
		body, err := c.get(ctx, endpoint, token)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Items []struct {
					Fields map[string]any `json:"fields"`
				} `json:"items"`
				PageToken string `json:"page_token"`
				HasMore   bool   `json:"has_more"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		if payload.Code != 0 {
			return nil, fmt.Errorf("read bitable records failed (%d): %s", payload.Code, payload.Msg)
		}
		for _, item := range payload.Data.Items {
			result = append(result, item.Fields)
		}
		if !payload.Data.HasMore || payload.Data.PageToken == "" {
			break
		}
		pageToken = payload.Data.PageToken
	}
	return result, nil
}

func (c *Client) WalkDriveFolder(ctx context.Context, folderToken string) (DriveWalkResult, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return DriveWalkResult{}, err
	}
	result := DriveWalkResult{Files: []DriveFile{}}
	if err := c.walkDriveFolder(ctx, token, folderToken, "", &result); err != nil {
		return DriveWalkResult{}, err
	}
	return result, nil
}

func (c *Client) ReadDocRawContent(ctx context.Context, docToken string) (string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("https://open.feishu.cn/open-apis/docx/v1/documents/%s/raw_content", docToken)
	body, err := c.get(ctx, endpoint, token)
	if err != nil {
		return "", err
	}
	var payload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.Code != 0 {
		return "", fmt.Errorf("read doc raw content failed (%d): %s", payload.Code, payload.Msg)
	}
	return payload.Data.Content, nil
}

func (c *Client) DriveFiles(ctx context.Context, folderToken string, pageToken string, folderType string) (map[string]any, error) {
	params := url.Values{}
	params.Set("page_size", "50")
	if folderToken != "" {
		params.Set("folder_token", folderToken)
	}
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}
	if folderType != "" {
		params.Set("folder_type", folderType)
	}
	return c.getData(ctx, feishuBase+"/open-apis/drive/v1/files?"+params.Encode())
}

func (c *Client) SharedSpaces(ctx context.Context) (map[string]any, error) {
	return c.getData(ctx, "https://open.feishu.cn/open-apis/drive/v1/shared_spaces?page_size=50")
}

func (c *Client) SharedFiles(ctx context.Context, spaceID string, folderToken string, pageToken string) (map[string]any, error) {
	params := url.Values{}
	params.Set("page_size", "50")
	if folderToken != "" {
		params.Set("folder_token", folderToken)
	}
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}
	endpoint := fmt.Sprintf("https://open.feishu.cn/open-apis/drive/v1/shared_spaces/%s/files?%s", spaceID, params.Encode())
	return c.getData(ctx, endpoint)
}

func (c *Client) DownloadFile(ctx context.Context, endpoint string) ([]byte, string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("feishu download status %d: %s", resp.StatusCode, string(data))
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func (c *Client) ExportSheet(ctx context.Context, sheetToken string) ([]byte, string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, "", err
	}
	createBody, err := c.post(ctx, "https://open.feishu.cn/open-apis/drive/v1/export_tasks", map[string]any{
		"file_extension": "xlsx",
		"token":          sheetToken,
		"type":           "sheet",
	}, token)
	if err != nil {
		return nil, "", err
	}
	var created struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Ticket string `json:"ticket"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createBody, &created); err != nil {
		return nil, "", err
	}
	if created.Code != 0 {
		return nil, "", fmt.Errorf("create export task failed (%d): %s", created.Code, created.Msg)
	}
	for i := 0; i < 15; i++ {
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, "", ctx.Err()
		case <-timer.C:
		}
		pollBody, err := c.get(ctx, "https://open.feishu.cn/open-apis/drive/v1/export_tasks/"+url.PathEscape(created.Data.Ticket)+"?token="+url.QueryEscape(sheetToken), token)
		if err != nil {
			return nil, "", err
		}
		var polled struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Result struct {
					JobStatus   int    `json:"job_status"`
					JobErrorMsg string `json:"job_error_msg"`
					FileToken   string `json:"file_token"`
				} `json:"result"`
			} `json:"data"`
		}
		if err := json.Unmarshal(pollBody, &polled); err != nil {
			return nil, "", err
		}
		if polled.Code != 0 {
			return nil, "", fmt.Errorf("poll export task failed (%d): %s", polled.Code, polled.Msg)
		}
		switch polled.Data.Result.JobStatus {
		case 0:
			return c.DownloadFile(ctx, "https://open.feishu.cn/open-apis/drive/v1/export_tasks/file/"+url.PathEscape(polled.Data.Result.FileToken)+"/download")
		case 2:
			return nil, "", fmt.Errorf("export failed: %s", polled.Data.Result.JobErrorMsg)
		}
	}
	return nil, "", fmt.Errorf("导出超时，请稍后重试")
}

func (c *Client) GetSheetMetaData(ctx context.Context, spreadsheetToken string) ([]SheetMeta, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("https://open.feishu.cn/open-apis/sheets/v3/spreadsheets/%s/sheets/query", spreadsheetToken)
	body, err := c.get(ctx, endpoint, token)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Sheets []SheetMeta `json:"sheets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("get sheet meta failed (%d): %s", payload.Code, payload.Msg)
	}
	return payload.Data.Sheets, nil
}

type SheetMeta struct {
	SheetID   string `json:"sheet_id"`
	Title     string `json:"title"`
	GridProps struct {
		ColumnCount int `json:"column_count"`
		RowCount    int `json:"row_count"`
	} `json:"grid_properties"`
}

func (c *Client) appAccessToken(ctx context.Context) (string, error) {
	payload := map[string]any{"app_id": c.AppID, "app_secret": c.AppSecret}
	body, err := c.post(ctx, "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal", payload, "")
	if err != nil {
		return "", err
	}
	var result struct {
		Code           int    `json:"code"`
		Msg            string `json:"msg"`
		AppAccessToken string `json:"app_access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", fmt.Errorf("get app_access_token failed (%d): %s", result.Code, result.Msg)
	}
	return result.AppAccessToken, nil
}

func (c *Client) walkDriveFolder(ctx context.Context, bearer string, folderToken string, parentPath string, result *DriveWalkResult) error {
	pageToken := ""
	for {
		endpoint := "https://open.feishu.cn/open-apis/drive/v1/files?page_size=200&folder_token=" + url.QueryEscape(folderToken)
		if pageToken != "" {
			endpoint += "&page_token=" + url.QueryEscape(pageToken)
		}
		body, err := c.get(ctx, endpoint, bearer)
		if err != nil {
			return err
		}
		var payload struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Files []struct {
					Token       string `json:"token"`
					Name        string `json:"name"`
					Type        string `json:"type"`
					ParentToken string `json:"parent_token"`
				} `json:"files"`
				PageToken string `json:"page_token"`
				HasMore   bool   `json:"has_more"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return err
		}
		if payload.Code != 0 {
			return fmt.Errorf("read drive folder failed (%d): %s", payload.Code, payload.Msg)
		}
		for _, file := range payload.Data.Files {
			currentPath := file.Name
			if parentPath != "" {
				currentPath = parentPath + " / " + file.Name
			}
			if file.Type == "folder" {
				result.FolderCount++
				if err := c.walkDriveFolder(ctx, bearer, file.Token, currentPath, result); err != nil {
					return err
				}
				continue
			}
			result.Files = append(result.Files, DriveFile{
				Token:       file.Token,
				Name:        file.Name,
				Type:        file.Type,
				ParentToken: file.ParentToken,
				ParentPath:  currentPath,
			})
		}
		if !payload.Data.HasMore || payload.Data.PageToken == "" {
			break
		}
		pageToken = payload.Data.PageToken
	}
	return nil
}

func (c *Client) post(ctx context.Context, endpoint string, payload map[string]any, bearer string) ([]byte, error) {
	data, _ := json.Marshal(payload)
	var lastStatus int
	var lastBody string
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			if err := sleepWithContext(ctx, time.Duration(attempt)*1500*time.Millisecond); err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if _, err := out.ReadFrom(resp.Body); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return out.Bytes(), nil
		}
		lastStatus = resp.StatusCode
		lastBody = out.String()
		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
	}
	return nil, fmt.Errorf("feishu API status %d: %s", lastStatus, lastBody)
}

func (c *Client) patch(ctx context.Context, endpoint string, payload map[string]any, bearer string) ([]byte, error) {
	data, _ := json.Marshal(payload)
	var lastStatus int
	var lastBody string
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			if err := sleepWithContext(ctx, time.Duration(attempt)*1500*time.Millisecond); err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if _, err := out.ReadFrom(resp.Body); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return out.Bytes(), nil
		}
		lastStatus = resp.StatusCode
		lastBody = out.String()
		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
	}
	return nil, fmt.Errorf("feishu API status %d: %s", lastStatus, lastBody)
}

func (c *Client) get(ctx context.Context, endpoint string, bearer string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out bytes.Buffer
	if _, err := out.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("feishu API status %d: %s", resp.StatusCode, out.String())
	}
	return out.Bytes(), nil
}

func (c *Client) getData(ctx context.Context, endpoint string) (map[string]any, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}
	body, err := c.get(ctx, endpoint, token)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Code int            `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu API failed (%d): %s", payload.Code, payload.Msg)
	}
	return payload.Data, nil
}

func valuesToRows(values [][]any) []map[string]any {
	if len(values) == 0 {
		return nil
	}
	headers := make([]string, len(values[0]))
	for i, header := range values[0] {
		headers[i] = cellToString(header)
	}
	rows := []map[string]any{}
	for _, valueRow := range values[1:] {
		empty := true
		row := map[string]any{}
		for i, header := range headers {
			if header == "" {
				continue
			}
			var value any
			if i < len(valueRow) {
				value = valueRow[i]
			}
			if cellToString(value) != "" {
				empty = false
			}
			row[header] = value
		}
		if !empty {
			rows = append(rows, row)
		}
	}
	return rows
}

func cellToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return fmt.Sprint(v)
	case bool:
		return fmt.Sprint(v)
	case []any:
		out := ""
		for _, item := range v {
			out += cellToString(item)
		}
		return out
	case map[string]any:
		if text, ok := v["text"]; ok {
			return cellToString(text)
		}
		if elements, ok := v["elements"]; ok {
			return cellToString(elements)
		}
		data, _ := json.Marshal(v)
		return string(data)
	default:
		return fmt.Sprint(v)
	}
}

func colLetter(n int) string {
	if n <= 0 {
		return "A"
	}
	out := ""
	for n > 0 {
		n--
		out = string(rune('A'+n%26)) + out
		n /= 26
	}
	return out
}

// CreateFolder 在 parentToken 下创建名为 name 的文件夹，返回新文件夹 token。
func (c *Client) CreateFolder(ctx context.Context, parentToken, name string) (string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return "", err
	}
	body, err := c.post(ctx, feishuBase+"/open-apis/drive/v1/files/create_folder", map[string]any{
		"name":         name,
		"folder_token": parentToken,
	}, token)
	if err != nil {
		return "", err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("create_folder parse: %w", err)
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("create_folder code %d: %s", resp.Code, resp.Msg)
	}
	if resp.Data.Token == "" {
		return "", fmt.Errorf("create_folder returned empty token")
	}
	return resp.Data.Token, nil
}

// FindSubfolder 在 parentToken 下找名为 name 的子文件夹，返回其 token + 是否找到。
func (c *Client) FindSubfolder(ctx context.Context, parentToken, name string) (string, bool, error) {
	raw, err := c.DriveFiles(ctx, parentToken, "", "")
	if err != nil {
		return "", false, err
	}
	// DriveFiles -> getData 已解包到 data 层，raw 即 data map（含 "files" 键）。
	files, _ := raw["files"].([]any)
	for _, f := range files {
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		n, _ := m["name"].(string)
		ty, _ := m["type"].(string)
		if n == name && (ty == "folder" || ty == "") {
			if tok, _ := m["token"].(string); tok != "" {
				return tok, true, nil
			}
		}
	}
	return "", false, nil
}

// FindSubfolderFuzzy 在 parentToken 下找 name 包含 substr 的子文件夹（模糊匹配，
// 兼容"2026年7月"命中"2026年7月产品"等带后缀的既有文件夹）。多个命中取第一个；
// 都不命中返回 ("", false, nil)。
func (c *Client) FindSubfolderFuzzy(ctx context.Context, parentToken, substr string) (string, bool, error) {
	raw, err := c.DriveFiles(ctx, parentToken, "", "")
	if err != nil {
		return "", false, err
	}
	files, _ := raw["files"].([]any)
	for _, f := range files {
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		n, _ := m["name"].(string)
		ty, _ := m["type"].(string)
		if strings.Contains(n, substr) && (ty == "folder" || ty == "") {
			if tok, _ := m["token"].(string); tok != "" {
				return tok, true, nil
			}
		}
	}
	return "", false, nil
}

// UploadDocx 把 data 作为 fileName 上传到 parentFolderToken，返回 file_token。
// URL 由调用方（buildDocxTool）用 config.FeishuDriveDomain 构造，避免 feishu 包依赖 config。
// 用 upload_all（单次，< 20MB）。
func (c *Client) UploadDocx(ctx context.Context, parentFolderToken, fileName string, data []byte) (string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return "", err
	}
	body, err := c.postMultipart(ctx, feishuBase+"/open-apis/drive/v1/files/upload_all",
		map[string]string{
			"file_name":   fileName,
			"parent_type": "explorer",
			"parent_node": parentFolderToken,
			"size":        fmt.Sprintf("%d", len(data)),
		}, fileName, data, token)
	if err != nil {
		return "", err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			FileToken string `json:"file_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("upload_all parse: %w", err)
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("upload_all code %d: %s", resp.Code, resp.Msg)
	}
	if resp.Data.FileToken == "" {
		return "", fmt.Errorf("upload_all returned empty file_token")
	}
	return resp.Data.FileToken, nil
}

// UploadDocxImage uploads image data as a docx-scoped media resource.
// parentNode must match the target relation expected by Feishu: use the
// document ID for initially creating an image block, and the image block ID
// before calling replace_image.
func (c *Client) UploadDocxImage(ctx context.Context, parentNode, fileName string, data []byte) (string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return "", err
	}
	body, err := c.postMultipart(ctx, feishuBase+"/open-apis/drive/v1/medias/upload_all",
		map[string]string{
			"file_name":   fileName,
			"parent_type": "docx_image",
			"parent_node": parentNode,
			"size":        fmt.Sprintf("%d", len(data)),
		}, fileName, data, token)
	if err != nil {
		return "", err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			FileToken string `json:"file_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("upload docx image parse: %w", err)
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("upload docx image code %d: %s", resp.Code, resp.Msg)
	}
	if resp.Data.FileToken == "" {
		return "", fmt.Errorf("upload docx image returned empty file_token")
	}
	return resp.Data.FileToken, nil
}

// CreateDocx creates a Feishu native docx document, optionally under a folder.
func (c *Client) CreateDocx(ctx context.Context, title, folderToken, driveDomain string) (*CreatedDocx, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{"title": title}
	if folderToken != "" {
		payload["folder_token"] = folderToken
	}
	body, err := c.post(ctx, feishuBase+"/open-apis/docx/v1/documents", payload, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Document struct {
				DocumentID string `json:"document_id"`
				Title      string `json:"title"`
			} `json:"document"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("create docx parse: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("create docx code %d: %s", resp.Code, resp.Msg)
	}
	doc := resp.Data.Document
	if doc.DocumentID == "" {
		return nil, fmt.Errorf("create docx returned empty document_id")
	}
	if driveDomain == "" {
		driveDomain = "feishu.cn"
	}
	return &CreatedDocx{
		DocumentID: doc.DocumentID,
		Title:      doc.Title,
		URL:        "https://" + driveDomain + "/docx/" + doc.DocumentID,
	}, nil
}

// AddDocxImage appends an image block under the document root and binds image
// media to it. Feishu requires the media used by replace_image to belong to the
// image block itself, so this intentionally uploads twice.
func (c *Client) AddDocxImage(ctx context.Context, documentID, fileName string, data []byte) error {
	initialToken, err := c.UploadDocxImage(ctx, documentID, fileName, data)
	if err != nil {
		return err
	}
	blockID, err := c.InsertDocxImage(ctx, documentID, initialToken)
	if err != nil {
		return err
	}
	blockToken, err := c.UploadDocxImage(ctx, blockID, fileName, data)
	if err != nil {
		return err
	}
	width, height := DocxImageDisplaySize(data, DocxImageMaxWidth())
	return c.ReplaceDocxImage(ctx, documentID, blockID, blockToken, width, height)
}

// InsertDocxImage appends an image block under the document root and returns
// the created image block ID.
func (c *Client) InsertDocxImage(ctx context.Context, documentID, imageToken string) (string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return "", err
	}
	image := map[string]any{
		"file_token": imageToken,
	}
	block := map[string]any{
		"block_type": 27,
		"image":      image,
	}
	body, err := c.post(ctx,
		feishuBase+"/open-apis/docx/v1/documents/"+url.PathEscape(documentID)+"/blocks/"+url.PathEscape(documentID)+"/children",
		map[string]any{"children": []any{block}},
		token,
	)
	if err != nil {
		return "", err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Children []struct {
				BlockID string `json:"block_id"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("insert docx image parse: %w", err)
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("insert docx image code %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.Data.Children) == 0 || resp.Data.Children[0].BlockID == "" {
		return "", fmt.Errorf("insert docx image returned empty block_id")
	}
	return resp.Data.Children[0].BlockID, nil
}

// DocxImageMaxWidth 返回飞书 docx 正文内容宽度，用于让图片块像手工截图粘贴
// 那样“两边对齐”撑满正文宽度，而不是 600px 居中的缩小图。默认 686（飞书 docx
// 默认正文宽度），可用环境变量 FEISHU_DOCX_CONTENT_WIDTH 覆盖；宽图按
// min(原图宽, 内容宽度) 撑满，窄图保持原宽不放大。
func DocxImageMaxWidth() int {
	if v := strings.TrimSpace(os.Getenv("FEISHU_DOCX_CONTENT_WIDTH")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 686
}

// DocxImageDisplaySize returns a proportional display size for Feishu docx
// image blocks. Feishu may preserve the uploaded image's pixel height when only
// width is replaced, leaving a large blank area under tall images.
func DocxImageDisplaySize(data []byte, maxWidth int) (int, int) {
	if maxWidth <= 0 {
		maxWidth = DocxImageMaxWidth()
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return maxWidth, 0
	}
	width := maxWidth
	if cfg.Width < width {
		width = cfg.Width
	}
	height := int(math.Round(float64(cfg.Height) * float64(width) / float64(cfg.Width)))
	if height <= 0 {
		height = 1
	}
	return width, height
}

// ReplaceDocxImage binds a docx image media token to an existing image block.
func (c *Client) ReplaceDocxImage(ctx context.Context, documentID, blockID, imageToken string, width, height int) error {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return err
	}
	if width <= 0 {
		width = DocxImageMaxWidth()
	}
	replaceImage := map[string]any{
		"token": imageToken,
		"width": width,
		"align": 2,
		"scale": 1,
	}
	if height > 0 {
		replaceImage["height"] = height
	}
	payload := map[string]any{
		"requests": []any{
			map[string]any{
				"block_id":      blockID,
				"replace_image": replaceImage,
			},
		},
	}
	body, err := c.patch(ctx,
		feishuBase+"/open-apis/docx/v1/documents/"+url.PathEscape(documentID)+"/blocks/batch_update",
		payload,
		token,
	)
	if err != nil {
		return err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("replace docx image parse: %w", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("replace docx image code %d: %s", resp.Code, resp.Msg)
	}
	return nil
}

// WriteDocxMarkdown converts markdown to Feishu docx blocks and inserts them.
// It is intended for newly created blank documents.
func (c *Client) WriteDocxMarkdown(ctx context.Context, documentID, markdown string) (int, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return 0, err
	}
	blocks, err := c.convertMarkdown(ctx, token, markdown)
	if err != nil {
		return 0, err
	}
	if len(blocks) == 0 {
		return 0, nil
	}
	inserted, err := c.InsertDocxBlocks(ctx, documentID, blocks)
	if err != nil {
		return 0, err
	}
	return len(inserted), nil
}

func (c *Client) MarkdownBlocks(ctx context.Context, markdown string) ([]any, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}
	return c.convertMarkdown(ctx, token, markdown)
}

func (c *Client) InsertDocxBlocks(ctx context.Context, documentID string, blocks []any) ([]DocxBlock, error) {
	if len(blocks) == 0 {
		return nil, nil
	}
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, err
	}
	body, err := c.post(ctx,
		feishuBase+"/open-apis/docx/v1/documents/"+url.PathEscape(documentID)+"/blocks/"+url.PathEscape(documentID)+"/children",
		map[string]any{"children": blocks},
		token,
	)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Children []DocxBlock `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("insert docx blocks parse: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("insert docx blocks code %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.Data.Children) != len(blocks) {
		return nil, fmt.Errorf("insert docx blocks returned %d children, want %d", len(resp.Data.Children), len(blocks))
	}
	return resp.Data.Children, nil
}

func (c *Client) convertMarkdown(ctx context.Context, token, markdown string) ([]any, error) {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return nil, nil
	}
	body, err := c.post(ctx, feishuBase+"/open-apis/docx/v1/documents/blocks/convert", map[string]any{
		"content_type": "markdown",
		"content":      markdown,
	}, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Blocks []any `json:"blocks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("convert markdown parse: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("convert markdown code %d: %s", resp.Code, resp.Msg)
	}
	return resp.Data.Blocks, nil
}

// postMultipart 发 multipart/form-data 请求。fields 为普通字段，fileName+data 为文件字段。
func (c *Client) postMultipart(ctx context.Context, endpoint string, fields map[string]string, fileName string, data []byte, bearer string) ([]byte, error) {
	var lastStatus int
	var lastBody string
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			if err := sleepWithContext(ctx, time.Duration(attempt)*1500*time.Millisecond); err != nil {
				return nil, err
			}
		}
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		for k, v := range fields {
			if err := mw.WriteField(k, v); err != nil {
				return nil, err
			}
		}
		fw, err := mw.CreateFormFile("file", fileName)
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write(data); err != nil {
			return nil, err
		}
		if err := mw.Close(); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", mw.FormDataContentType())
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if _, err := out.ReadFrom(resp.Body); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return out.Bytes(), nil
		}
		lastStatus = resp.StatusCode
		lastBody = out.String()
		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
	}
	return nil, fmt.Errorf("feishu API status %d: %s", lastStatus, lastBody)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
