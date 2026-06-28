package agent

import (
	"strings"
	"testing"
)

func TestLoadSkillContent_Copywriter(t *testing.T) {
	content, err := LoadSkillContent("structured-product-copywriter")
	if err != nil {
		t.Fatalf("LoadSkillContent: %v", err)
	}
	if !strings.Contains(content, "结构化产品推介文案") {
		t.Error("缺 skill 标题「结构化产品推介文案」")
	}
	if !strings.Contains(content, "降落伞") {
		t.Error("缺 10 项参数表内容（降落伞）")
	}
}

func TestLoadSkillContent_NotFound(t *testing.T) {
	if _, err := LoadSkillContent("nonexistent-skill"); err == nil {
		t.Error("未知名应返回 error")
	}
}

func TestGetSkillReference_TongyuWinrate(t *testing.T) {
	content, err := GetSkillReference("structured-product-copywriter", "tongyu-winrate")
	if err != nil {
		t.Fatalf("GetSkillReference: %v", err)
	}
	if !strings.Contains(content, "胜率") {
		t.Error("缺「胜率」内容")
	}
	if !strings.Contains(content, "通毓") {
		t.Error("缺「通毓」内容")
	}
}

func TestGetSkillReference_NotFound(t *testing.T) {
	if _, err := GetSkillReference("structured-product-copywriter", "nope"); err == nil {
		t.Error("未知名应返回 error")
	}
}
