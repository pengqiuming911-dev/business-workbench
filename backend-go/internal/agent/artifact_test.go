package agent

import "testing"

func TestExtractArtifact(t *testing.T) {
	artifact := map[string]any{"product_name": "鹿*8号(三期)", "annualized_return": "15.96"}

	t.Run("present", func(t *testing.T) {
		result := map[string]any{
			"poster_artifact": artifact,
			"message":         "已生成",
		}
		got, ok := extractArtifact(result)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if got["product_name"] != "鹿*8号(三期)" {
			t.Errorf("got %v", got["product_name"])
		}
	})

	t.Run("absent", func(t *testing.T) {
		if _, ok := extractArtifact(map[string]any{"count": 0}); ok {
			t.Fatal("expected ok=false when no artifact")
		}
	})

	t.Run("wrong_type", func(t *testing.T) {
		if _, ok := extractArtifact(map[string]any{"poster_artifact": "not a map"}); ok {
			t.Fatal("expected ok=false when artifact is not a map")
		}
	})
}
