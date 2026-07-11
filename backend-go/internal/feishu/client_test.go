package feishu

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDocxImageDisplaySizeScalesHeight(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 875, 1280))
	for y := 0; y < 1280; y++ {
		for x := 0; x < 875; x++ {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	// 自然宽高：飞书按正文内容宽度等比缩放显示，不再 cap 成 600 居中缩小图。
	width, height := DocxImageDisplaySize(buf.Bytes(), 600)
	if width != 875 || height != 1280 {
		t.Fatalf("size = %dx%d, want 875x1280 (natural)", width, height)
	}
}

func TestDocxImageMaxWidthDefaultAndOverride(t *testing.T) {
	// 默认正文内容宽度 686（仅作 undecodable 兜底），环境变量为空时回退此值。
	t.Setenv("FEISHU_DOCX_CONTENT_WIDTH", "")
	if got := DocxImageMaxWidth(); got != 686 {
		t.Fatalf("default DocxImageMaxWidth = %d, want 686", got)
	}
	// 环境变量覆盖兜底值。
	t.Setenv("FEISHU_DOCX_CONTENT_WIDTH", "860")
	if got := DocxImageMaxWidth(); got != 860 {
		t.Fatalf("overridden DocxImageMaxWidth = %d, want 860", got)
	}
	// 可解码图片用自然宽高（宽图 ≥ 正文宽度 → 飞书 clamp 撑满两边），与 maxWidth 无关。
	img := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	width, height := DocxImageDisplaySize(buf.Bytes(), DocxImageMaxWidth())
	if width != 1000 || height != 500 {
		t.Fatalf("size = %dx%d, want 1000x500 (natural)", width, height)
	}
}

// getTimeFuture 返回一个远未来时间，用于让 ensureValidToken 跳过刷新。
func getTimeFuture() time.Time { return time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC) }

// newAuthedClient 建一个带未过期 user token 的 client（ensureValidToken 不触发刷新）。
func newAuthedClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c := New("app", "secret", "http://redirect")
	c.HTTP = srv.Client()
	c.userToken = "test-token"
	c.expiresAt = getTimeFuture() // 远未来，确保不刷新
	return c
}

func TestCreateFolder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/drive/v1/files/create_folder" {
			t.Errorf("path = %s, want create_folder", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth = %q, want Bearer test-token", got)
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"name":"2026年6月"`) || !strings.Contains(string(b), `"folder_token":"parentT"`) {
			t.Errorf("body = %s, want name+folder_token", b)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"token":"fldNew","url":"https://x.feishu.cn/drive/folder/fldNew"}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	tok, err := c.CreateFolder(context.Background(), "parentT", "2026年6月")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if tok != "fldNew" {
		t.Errorf("token = %q, want fldNew", tok)
	}
}

func TestFindSubfolder_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/open-apis/drive/v1/files") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"files":[
			{"token":"fldA","name":"2026年5月","type":"folder"},
			{"token":"fldB","name":"2026年6月","type":"folder"},
			{"token":"filC","name":"some.docx","type":"docx"}
		],"has_more":false}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	tok, found, err := c.FindSubfolder(context.Background(), "parentT", "2026年6月")
	if err != nil {
		t.Fatalf("FindSubfolder: %v", err)
	}
	if !found || tok != "fldB" {
		t.Errorf("got tok=%q found=%v, want fldB/true", tok, found)
	}
}

func TestFindSubfolder_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"files":[{"token":"fldA","name":"2026年5月","type":"folder"}],"has_more":false}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	tok, found, err := c.FindSubfolder(context.Background(), "parentT", "2026年6月")
	if err != nil || found {
		t.Fatalf("got tok=%q found=%v err=%v, want false/nil", tok, found, err)
	}
}

func TestUploadDocx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/drive/v1/files/upload_all" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth = %q", got)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("content-type = %q, want multipart", ct)
		}
		b, _ := io.ReadAll(r.Body)
		body := string(b)
		for _, want := range []string{`name="file_name"`, "pitch.docx", `name="parent_type"`, "explorer", `name="parent_node"`, "fldMonth", `name="size"`, `filename="pitch.docx"`} {
			if !strings.Contains(body, want) {
				t.Errorf("multipart body missing %q", want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"file_token":"filNew"}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	ft, err := c.UploadDocx(context.Background(), "fldMonth", "pitch.docx", []byte("fake docx bytes"))
	if err != nil {
		t.Fatalf("UploadDocx: %v", err)
	}
	if ft != "filNew" {
		t.Errorf("file_token = %q, want filNew", ft)
	}
}

func TestCreateDocx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/docx/v1/documents" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth = %q", got)
		}
		b, _ := io.ReadAll(r.Body)
		body := string(b)
		if !strings.Contains(body, `"title":"Pitch"`) || !strings.Contains(body, `"folder_token":"fldMonth"`) {
			t.Errorf("body = %s, want title+folder_token", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"document":{"document_id":"docNew","title":"Pitch"}}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	doc, err := c.CreateDocx(context.Background(), "Pitch", "fldMonth", "kcngap16uccc.feishu.cn")
	if err != nil {
		t.Fatalf("CreateDocx: %v", err)
	}
	if doc.DocumentID != "docNew" || doc.URL != "https://kcngap16uccc.feishu.cn/docx/docNew" {
		t.Errorf("doc = %+v", doc)
	}
}

func TestUploadDocxImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/drive/v1/medias/upload_all" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth = %q", got)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("content-type = %q, want multipart", ct)
		}
		b, _ := io.ReadAll(r.Body)
		body := string(b)
		for _, want := range []string{`name="file_name"`, "chart.png", `name="parent_type"`, "docx_image", `name="parent_node"`, "docNew", `name="size"`, `filename="chart.png"`} {
			if !strings.Contains(body, want) {
				t.Errorf("multipart body missing %q", want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"file_token":"imgTok"}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	tok, err := c.UploadDocxImage(context.Background(), "docNew", "chart.png", []byte("png"))
	if err != nil {
		t.Fatalf("UploadDocxImage: %v", err)
	}
	if tok != "imgTok" {
		t.Errorf("token = %q, want imgTok", tok)
	}
}

func TestInsertDocxImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/docx/v1/documents/docNew/blocks/docNew/children" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth = %q", got)
		}
		b, _ := io.ReadAll(r.Body)
		body := string(b)
		for _, want := range []string{`"block_type":27`, `"image"`, `"file_token":"imgTok"`} {
			if !strings.Contains(body, want) {
				t.Errorf("body missing %q: %s", want, body)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"children":[{"block_id":"imgBlk"}]}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	blockID, err := c.InsertDocxImage(context.Background(), "docNew", "imgTok")
	if err != nil {
		t.Fatalf("InsertDocxImage: %v", err)
	}
	if blockID != "imgBlk" {
		t.Fatalf("blockID = %q, want imgBlk", blockID)
	}
}

func TestAddDocxImage(t *testing.T) {
	var uploads []string
	seenInsert := false
	seenReplace := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/open-apis/drive/v1/medias/upload_all":
			b, _ := io.ReadAll(r.Body)
			body := string(b)
			if strings.Contains(body, `name="parent_node"`) && strings.Contains(body, "docNew") {
				uploads = append(uploads, "docNew")
				w.Write([]byte(`{"code":0,"data":{"file_token":"seedTok"}}`))
				return
			}
			if strings.Contains(body, `name="parent_node"`) && strings.Contains(body, "imgBlk") {
				uploads = append(uploads, "imgBlk")
				w.Write([]byte(`{"code":0,"data":{"file_token":"blockTok"}}`))
				return
			}
			t.Errorf("unexpected upload body = %s", body)
			w.Write([]byte(`{"code":999,"msg":"bad upload"}`))
		case "/open-apis/docx/v1/documents/docNew/blocks/docNew/children":
			seenInsert = true
			b, _ := io.ReadAll(r.Body)
			body := string(b)
			if !strings.Contains(body, `"file_token":"seedTok"`) {
				t.Errorf("insert body = %s", body)
			}
			w.Write([]byte(`{"code":0,"data":{"children":[{"block_id":"imgBlk"}]}}`))
		case "/open-apis/docx/v1/documents/docNew/blocks/batch_update":
			seenReplace = true
			if r.Method != http.MethodPatch {
				t.Errorf("method = %s, want PATCH", r.Method)
			}
			b, _ := io.ReadAll(r.Body)
			body := string(b)
			for _, want := range []string{`"block_id":"imgBlk"`, `"replace_image"`, `"token":"blockTok"`, `"width":600`} {
				if !strings.Contains(body, want) {
					t.Errorf("replace body missing %q: %s", want, body)
				}
			}
			w.Write([]byte(`{"code":0,"data":{"blocks":[{"block_id":"imgBlk"}]}}`))
		default:
			t.Errorf("unexpected path = %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	if err := c.AddDocxImage(context.Background(), "docNew", "chart.png", []byte("png")); err != nil {
		t.Fatalf("AddDocxImage: %v", err)
	}
	if !reflect.DeepEqual(uploads, []string{"docNew", "imgBlk"}) || !seenInsert || !seenReplace {
		t.Fatalf("uploads=%v seenInsert=%v seenReplace=%v", uploads, seenInsert, seenReplace)
	}
}

func TestWriteDocxMarkdown(t *testing.T) {
	seenConvert := false
	seenInsert := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/open-apis/docx/v1/documents/blocks/convert":
			seenConvert = true
			b, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(b), `"content_type":"markdown"`) || !strings.Contains(string(b), `# Title`) {
				t.Errorf("convert body = %s", b)
			}
			w.Write([]byte(`{"code":0,"data":{"blocks":[{"block_type":2,"text":{"elements":[{"text_run":{"content":"Title"}}]}}]}}`))
		case "/open-apis/docx/v1/documents/docNew/blocks/docNew/children":
			seenInsert = true
			b, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(b), `"children"`) || !strings.Contains(string(b), `"block_type":2`) {
				t.Errorf("insert body = %s", b)
			}
			w.Write([]byte(`{"code":0,"data":{"children":[{"block_id":"blk1"}]}}`))
		default:
			t.Errorf("unexpected path = %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	added, err := c.WriteDocxMarkdown(context.Background(), "docNew", "# Title")
	if err != nil {
		t.Fatalf("WriteDocxMarkdown: %v", err)
	}
	if added != 1 || !seenConvert || !seenInsert {
		t.Fatalf("added=%d seenConvert=%v seenInsert=%v", added, seenConvert, seenInsert)
	}
}
