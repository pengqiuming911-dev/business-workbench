package agent

import (
	"embed"
	"fmt"
)

//go:embed skills
var skillFS embed.FS

// copywriterSkillDir 是 embed.FS 中 skills 根目录；具体 skill 名通过参数拼接。
// 当前 skills/ 下仅嵌入 "structured-product-copywriter"。
const copywriterSkillDir = "skills"

// LoadSkillContent 返回指定 skill 的 verbatim SKILL.md。
// 当前仅嵌入 "structured-product-copywriter"。
func LoadSkillContent(name string) (string, error) {
	path := fmt.Sprintf("%s/%s/SKILL.md", copywriterSkillDir, name)
	b, err := skillFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("skill %q not found: %w", name, err)
	}
	return string(b), nil
}

// GetSkillReference 返回某 skill 下 references/ 里的参考文档（refName 不带 .md）。
func GetSkillReference(skillName, refName string) (string, error) {
	path := fmt.Sprintf("%s/%s/references/%s.md", copywriterSkillDir, skillName, refName)
	b, err := skillFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reference %q of skill %q not found: %w", refName, skillName, err)
	}
	return string(b), nil
}

// loadSkill 是 agent 工具入口：返回 skill 原文供 agent 按步执行。
func (s *Service) loadSkill(args map[string]any) map[string]any {
	name := stringArg(args, "name")
	if name == "" {
		name = "structured-product-copywriter"
	}
	content, err := LoadSkillContent(name)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"name": name, "content": content}
}

// getSkillReference 是 agent 工具入口：按需返回重步骤参考文档。
func (s *Service) getSkillReference(args map[string]any) map[string]any {
	skillName := stringArg(args, "skill")
	if skillName == "" {
		skillName = "structured-product-copywriter"
	}
	refName := stringArg(args, "name")
	if refName == "" {
		return map[string]any{"error": "name is required"}
	}
	content, err := GetSkillReference(skillName, refName)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"skill": skillName, "name": refName, "content": content}
}
