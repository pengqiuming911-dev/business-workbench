package retriever

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ScoredDoc struct {
	DocName    string
	ParentPath string
	RawContent string
	Structure  any
	Score      int
}

func SearchDocs(docs []map[string]any, query string, limit int) []ScoredDoc {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	keywords := strings.Fields(query)

	var scored []ScoredDoc
	for _, doc := range docs {
		docName, _ := doc["doc_name"].(string)
		parentPath, _ := doc["parent_path"].(string)
		rawContent, _ := doc["raw_content"].(string)
		structureJSON, _ := doc["structure_json"].(string)

		text := strings.ToLower(docName + " " + parentPath + " " + rawContent)
		score := 0
		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			offset := 0
			for {
				pos := strings.Index(text[offset:], kwLower)
				if pos == -1 {
					break
				}
				score++
				offset += pos + len(kwLower)
			}
		}
		if score == 0 {
			continue
		}
		var structure any
		if structureJSON != "" {
			_ = json.Unmarshal([]byte(structureJSON), &structure)
		}
		scored = append(scored, ScoredDoc{
			DocName:    docName,
			ParentPath: parentPath,
			RawContent: rawContent,
			Structure:  structure,
			Score:      score,
		})
	}

	sortScored(scored)
	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	return scored
}

func sortScored(s []ScoredDoc) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j].Score > s[i].Score {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func BuildDocContext(scored []ScoredDoc) string {
	if len(scored) == 0 {
		return ""
	}
	var parts []string
	for i, d := range scored {
		structStr := ""
		if d.Structure != nil {
			j, _ := json.Marshal(d.Structure)
			structStr = fmt.Sprintf("\n结构信息：%s", string(j))
		}
		parts = append(parts, fmt.Sprintf("[文档%d] %s (%s)\n%s%s", i+1, d.DocName, d.ParentPath, d.RawContent, structStr))
	}
	return "\n\n以下是与用户问题相关的文档资料，请参考这些文档回答问题：\n\n" + strings.Join(parts, "\n\n---\n\n")
}
