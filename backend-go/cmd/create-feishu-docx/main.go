package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/feishu"
)

func main() {
	title := flag.String("title", "", "document title")
	content := flag.String("content", "", "markdown content")
	contentFile := flag.String("content-file", "", "path to markdown content file")
	folderToken := flag.String("folder-token", "", "target Feishu folder token")
	folderName := flag.String("folder-name", "", "month folder name under FEISHU_PITCH_FOLDER_TOKEN")
	tokenPath := flag.String("token-path", ".feishu-user-token", "persisted Feishu user token path")
	flag.Parse()

	if strings.TrimSpace(*title) == "" {
		fail("missing -title")
	}

	body := *content
	if *contentFile != "" {
		data, err := os.ReadFile(*contentFile)
		if err != nil {
			fail("read content file: %v", err)
		}
		body = string(data)
	}

	cfg := config.Load()
	client := feishu.New(cfg.FeishuAppID, cfg.FeishuAppSecret, cfg.FeishuRedirectURI)
	client.SetTokenPersistPath(*tokenPath)

	ctx := context.Background()
	targetFolder := strings.TrimSpace(*folderToken)
	resolvedFolderName := strings.TrimSpace(*folderName)
	if targetFolder == "" {
		if resolvedFolderName == "" {
			resolvedFolderName = time.Now().Format("2006年1月产品")
		}
		token, found, err := client.FindSubfolder(ctx, cfg.FeishuPitchFolderToken, resolvedFolderName)
		if err != nil {
			fail("find folder: %v", err)
		}
		if !found {
			token, err = client.CreateFolder(ctx, cfg.FeishuPitchFolderToken, resolvedFolderName)
			if err != nil {
				fail("create folder: %v", err)
			}
		}
		targetFolder = token
	}

	doc, err := client.CreateDocx(ctx, strings.TrimSpace(*title), targetFolder, cfg.FeishuDriveDomain)
	if err != nil {
		fail("create docx: %v", err)
	}
	blocksAdded := 0
	if strings.TrimSpace(body) != "" {
		blocksAdded, err = client.WriteDocxMarkdown(ctx, doc.DocumentID, body)
		if err != nil {
			fail("write docx: %v", err)
		}
	}

	fmt.Printf("url=%s\n", doc.URL)
	fmt.Printf("doc_token=%s\n", doc.DocumentID)
	fmt.Printf("folder_token=%s\n", targetFolder)
	fmt.Printf("folder=%s\n", resolvedFolderName)
	fmt.Printf("blocks_added=%d\n", blocksAdded)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
