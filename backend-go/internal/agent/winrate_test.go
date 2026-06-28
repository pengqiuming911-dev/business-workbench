package agent

import (
	"testing"

	"business-workbench/backend-go/internal/config"
)

func TestParamToFormFields_FullDCN(t *testing.T) {
	params := map[string]any{
		"期限":    "36",
		"锁定期":   "3",
		"期初敲出线": "101",
		"降敲":    "0.5",
		"降落伞":   "60",
		"派息线":   "78",
		"费后派息":  "1.39",
		"保证金":   "50",
		"是否追保":  "不追保",
	}
	fields := paramToFormFields(params)
	byLabel := map[string]string{}
	for _, f := range fields {
		byLabel[f.Label] = f.Value
	}
	cases := map[string]string{
		"期限(月)":      "36",
		"锁定期(月)":     "3",
		"首次观察敲出价(%)": "101",
		"敲出价递减步长(%)": "0.5",
		"期末障碍价(%)":   "60",
		"派息障碍价(%)":   "78",
		"每月或有派息(%)":  "1.39",
		"保证金水平(%)":   "50",
		"是否追保":       "不追保",
	}
	for label, want := range cases {
		if got := byLabel[label]; got != want {
			t.Errorf("field %q = %q, want %q", label, got, want)
		}
	}
}

func TestParamToFormFields_OmitsEmpty(t *testing.T) {
	// 派息线/锁定期 不适用时不出现
	params := map[string]any{
		"期限":   "24",
		"降落伞":  "60",
		"费后派息": "1.2",
		"保证金":  "100",
		"是否追保": "不追保",
	}
	fields := paramToFormFields(params)
	for _, f := range fields {
		if f.Label == "派息障碍价(%)" {
			t.Error("派息线为空时不应出现 派息障碍价(%)")
		}
		if f.Label == "锁定期(月)" {
			t.Error("锁定期为空时不应出现 锁定期(月)")
		}
		if f.Value == "" {
			t.Errorf("field %q value 不应为空", f.Label)
		}
	}
}

func TestFetchWinrate_DryRun(t *testing.T) {
	s := &Service{cfg: config.Config{WinrateDryRun: "true"}}
	out := s.fetchWinrate(map[string]any{"标的": "中证1000", "structure_type": "DCN"})
	if got := out["胜率"]; got != "98.17%" {
		t.Errorf("dry-run 胜率 = %v, want 98.17%%", got)
	}
	if got := out["source"]; got != "dry-run" {
		t.Errorf("source = %v, want dry-run", got)
	}
}

func TestFetchWinrate_NoCreds(t *testing.T) {
	s := &Service{cfg: config.Config{WinrateDryRun: "false"}} // 无 TongyuUser/Pass
	out := s.fetchWinrate(map[string]any{"标的": "中证1000"})
	if got := out["胜率"]; got != "[胜率待补]" {
		t.Errorf("无凭证时胜率 = %v, want [胜率待补]", got)
	}
	if _, ok := out["reason"]; !ok {
		t.Error("无凭证时应带 reason")
	}
}
