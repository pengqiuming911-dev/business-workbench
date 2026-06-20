package prices

import (
	"strings"
	"testing"
)

func TestDecodeTencentKlines(t *testing.T) {
	body := `{
		"code": 0,
		"data": {
			"sh000852": {
				"day": [
					["2026-06-17", "8581.650", "8704.470", "8705.400", "8581.650", "282227937.000"],
					["2026-06-18", "8666.640", "8771.020", "8796.480", "8664.280", "290062435.000"]
				]
			}
		}
	}`

	points, err := decodeTencentKlines(strings.NewReader(body), "sh000852")
	if err != nil {
		t.Fatalf("decodeTencentKlines returned error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2", len(points))
	}
	if points[1].Date != "2026-06-18" || points[1].Close != 8771.02 {
		t.Fatalf("unexpected final point: %+v", points[1])
	}
}

func TestDecodeTencentKlinesUsesQfqDayFallback(t *testing.T) {
	body := `{
		"code": 0,
		"data": {
			"sh513180": {
				"qfqday": [["2026-06-18", "0.586", "0.580", "0.588", "0.578", "49719239.000"]]
			}
		}
	}`

	points, err := decodeTencentKlines(strings.NewReader(body), "sh513180")
	if err != nil {
		t.Fatalf("decodeTencentKlines returned error: %v", err)
	}
	if len(points) != 1 || points[0].Close != 0.58 {
		t.Fatalf("unexpected points: %+v", points)
	}
}
