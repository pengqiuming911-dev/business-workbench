package db

import (
	"path/filepath"
	"testing"
)

func TestSaveAndQueryPosterArtifact(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.InitSchema(); err != nil {
		t.Fatalf("schema: %v", err)
	}

	fieldsJSON := `{"product_name":"鹿*8号(三期)","annualized_return":"15.96"}`
	id, err := store.SavePosterArtifact("P001", "2026-04-30", fieldsJSON, "poster-artifacts/1.png", "abc123hash")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	got, err := store.QueryPosterArtifact(id)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if got.ProductID != "P001" || got.ObservationDate != "2026-04-30" {
		t.Errorf("got %+v", got)
	}
	if got.ContentHash != "abc123hash" || got.PngPath != "poster-artifacts/1.png" {
		t.Errorf("hash/path = %q / %q", got.ContentHash, got.PngPath)
	}
	if got.FieldsJSON != fieldsJSON {
		t.Errorf("fields_json = %q", got.FieldsJSON)
	}
	_ = filepath.Separator // silence unused import on non-cross-platform builds
}
