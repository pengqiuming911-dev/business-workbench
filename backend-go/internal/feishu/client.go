package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Client struct {
	AppID       string
	AppSecret   string
	RedirectURI string
	HTTP        *http.Client

	mu        sync.RWMutex
	userToken string
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

func New(appID, appSecret, redirectURI string) *Client {
	return &Client{
		AppID:       appID,
		AppSecret:   appSecret,
		RedirectURI: redirectURI,
		HTTP:        &http.Client{Timeout: 15 * time.Second},
	}
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
	defer c.mu.Unlock()
	c.userToken = ""
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
			AccessToken string `json:"access_token"`
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
	c.mu.Lock()
	c.userToken = result.Data.AccessToken
	c.mu.Unlock()
	return nil
}

func (c *Client) ReadSheetRows(ctx context.Context, spreadsheetToken string, sheetID string, colCount int) ([]map[string]any, error) {
	token := c.UserToken()
	if token == "" {
		return nil, fmt.Errorf("未授权，请先登录飞书")
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

func (c *Client) ReadBitableRecords(ctx context.Context, appToken string, tableID string) ([]map[string]any, error) {
	token := c.UserToken()
	if token == "" {
		return nil, fmt.Errorf("未授权，请先登录飞书")
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
	token := c.UserToken()
	if token == "" {
		return DriveWalkResult{}, fmt.Errorf("未授权，请先登录飞书")
	}
	result := DriveWalkResult{Files: []DriveFile{}}
	if err := c.walkDriveFolder(ctx, token, folderToken, "", &result); err != nil {
		return DriveWalkResult{}, err
	}
	return result, nil
}

func (c *Client) ReadDocRawContent(ctx context.Context, docToken string) (string, error) {
	token := c.UserToken()
	if token == "" {
		return "", fmt.Errorf("未授权，请先登录飞书")
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
	return c.getData(ctx, "https://open.feishu.cn/open-apis/drive/v1/files?"+params.Encode())
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
	token := c.UserToken()
	if token == "" {
		return nil, "", fmt.Errorf("未授权，请先登录飞书")
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
	token := c.UserToken()
	if token == "" {
		return nil, "", fmt.Errorf("未授权，请先登录飞书")
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
	token := c.UserToken()
	if token == "" {
		return nil, fmt.Errorf("未授权，请先登录飞书")
	}
	endpoint := fmt.Sprintf("https://open.feishu.cn/open-apis/sheets/v3/spreadsheets/%s", spreadsheetToken)
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
	SheetID    string `json:"sheet_id"`
	Title      string `json:"title"`
	GridProps  struct {
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
	token := c.UserToken()
	if token == "" {
		return nil, fmt.Errorf("未授权，请先登录飞书")
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
