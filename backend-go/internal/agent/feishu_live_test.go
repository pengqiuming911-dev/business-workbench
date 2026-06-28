//go:build feishu_live

// Live integration test for the build_docx → Feishu Drive upload path.
// Skipped in the normal suite (build tag feishu_live). Run with:
//   set -a; source backend-go/.env; set +a
//   FEISHU_TOKEN_PATH=$(pwd)/backend-go/.feishu-user-token \
//     go test -tags=feishu_live -run TestLiveBuildDocxFeishuUpload ./internal/agent/ -v
// It creates a real .docx in the user's Feishu year-month folder (the feature working for real).

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/feishu"
)

func TestLiveBuildDocxFeishuUpload(t *testing.T) {
	cfg := config.Load()
	if cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" {
		t.Skip("FEISHU_APP_ID/SECRET not set (source backend-go/.env)")
	}
	tokenPath := os.Getenv("FEISHU_TOKEN_PATH")
	if tokenPath == "" {
		tokenPath = ".feishu-user-token"
	}

	fc := feishu.New(cfg.FeishuAppID, cfg.FeishuAppSecret, cfg.FeishuRedirectURI)
	fc.SetTokenPersistPath(tokenPath)
	ctx := context.Background()

	// 1. Build a small real .docx via the S3 writer.
	sections := []docxSection{
		{Type: "heading", Text: "飞书上传 live 测试"},
		{Type: "body", Text: "本文件由 TestLiveBuildDocxFeishuUpload 生成，验证 build_docx→飞书 Drive 上传链路真实可用。\n当前点位/胜率等数字在此测试中为占位。"},
		{Type: "params", Text: "标的：中证1000\n胜率：97.88%"},
	}
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "live-test.docx")
	if err := BuildDocx(sections, tmpPath); err != nil {
		t.Fatalf("BuildDocx: %v", err)
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("read tmp docx: %v", err)
	}

	// 2. Find-or-create the year-month subfolder under the base folder.
	folderName := time.Now().Format("2006年1月")
	subToken, found, err := fc.FindSubfolder(ctx, cfg.FeishuPitchFolderToken, folderName)
	if err != nil {
		t.Fatalf("FindSubfolder: %v", err)
	}
	if !found {
		subToken, err = fc.CreateFolder(ctx, cfg.FeishuPitchFolderToken, folderName)
		if err != nil {
			t.Fatalf("CreateFolder: %v", err)
		}
		t.Logf("created subfolder %q -> %s", folderName, subToken)
	} else {
		t.Logf("found existing subfolder %q -> %s", folderName, subToken)
	}

	// 3. Upload.
	fileName := fmt.Sprintf("live-test-%s.docx", time.Now().Format("20060102_150405"))
	fileToken, err := fc.UploadDocx(ctx, subToken, fileName, data)
	if err != nil {
		t.Fatalf("UploadDocx: %v", err)
	}
	url := "https://" + cfg.FeishuDriveDomain + "/file/" + fileToken
	t.Logf("✅ uploaded %s (%d bytes) to folder %q", url, len(data), folderName)
	t.Logf("   file_token=%s subfolder=%s", fileToken, subToken)
}
