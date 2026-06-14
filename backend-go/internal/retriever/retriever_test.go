package retriever

import (
	"strings"
	"testing"
)

func TestSearchDocs_Empty(t *testing.T) {
	docs := []map[string]any{}
	result := SearchDocs(docs, "anything", 5)
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestSearchDocs_Score(t *testing.T) {
	docs := []map[string]any{
		{"doc_name": "产品月报", "parent_path": "/2024/01", "raw_content": "这是关于敲出产品的分析", "structure_json": ""},
		{"doc_name": "派息说明", "parent_path": "/2024/02", "raw_content": "派息线计算方式和派息条件", "structure_json": ""},
		{"doc_name": "无关文档", "parent_path": "/other", "raw_content": "其他内容", "structure_json": ""},
	}
	result := SearchDocs(docs, "派息", 5)
	if len(result) == 0 {
		t.Fatal("expected at least 1 result for keyword 派息")
	}
	if result[0].DocName != "派息说明" {
		t.Errorf("expected top result to be '派息说明', got %q", result[0].DocName)
	}
}

func TestSearchDocs_Limit(t *testing.T) {
	docs := []map[string]any{
		{"doc_name": "A", "parent_path": "/", "raw_content": "关键词 关键词 关键词", "structure_json": ""},
		{"doc_name": "B", "parent_path": "/", "raw_content": "关键词", "structure_json": ""},
		{"doc_name": "C", "parent_path": "/", "raw_content": "关键词 关键词", "structure_json": ""},
	}
	result := SearchDocs(docs, "关键词", 2)
	if len(result) != 2 {
		t.Errorf("expected limit 2 results, got %d", len(result))
	}
}

func TestSearchDocs_MultiKeyword(t *testing.T) {
	docs := []map[string]any{
		{"doc_name": "敲出分析", "parent_path": "/report", "raw_content": "这个产品已经敲出了", "structure_json": ""},
		{"doc_name": "派息报告", "parent_path": "/report", "raw_content": "这个产品派息了", "structure_json": ""},
	}
	result := SearchDocs(docs, "敲出 派息", 5)
	if len(result) != 2 {
		t.Errorf("expected 2 results matching either keyword, got %d", len(result))
	}
}

func TestBuildDocContext_Empty(t *testing.T) {
	ctx := BuildDocContext(nil)
	if ctx != "" {
		t.Errorf("expected empty string for nil docs, got %q", ctx)
	}
}

func TestBuildDocContext_Format(t *testing.T) {
	scored := []ScoredDoc{
		{DocName: "报告A", ParentPath: "/2024", RawContent: "内容A", Score: 3},
		{DocName: "报告B", ParentPath: "/2025", RawContent: "内容B", Score: 2},
	}
	ctx := BuildDocContext(scored)
	if !strings.Contains(ctx, "[文档1] 报告A") {
		t.Error("context should contain [文档1] 报告A")
	}
	if !strings.Contains(ctx, "[文档2] 报告B") {
		t.Error("context should contain [文档2] 报告B")
	}
	if !strings.Contains(ctx, "请参考这些文档回答问题") {
		t.Error("context should contain the intro sentence")
	}
}
