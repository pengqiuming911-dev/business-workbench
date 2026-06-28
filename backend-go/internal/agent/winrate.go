package agent

import (
	"strings"

	"business-workbench/backend-go/internal/config"
)

// formField 是通毓回测表单的一个填写项：按标签文本定位的输入框 + 要填的值。
type formField struct {
	Label string
	Value string
}

// paramToFormFields 把产品参数(中文键)映射成通毓"结构化产品回测"表单的字段标签 + 值。
// 映射来源：skills/structured-product-copywriter/references/tongyu-winrate.md 的字段映射表。
// 空值字段省略(如派息线/锁定期不适用)。标的与结构类型不在此映射——它们是搜索框/单选，由 chromedp 流程单独处理。
func paramToFormFields(params map[string]any) []formField {
	get := func(k string) string { return strings.TrimSpace(stringArg(params, k)) }
	var out []formField
	add := func(label, key string) {
		if v := get(key); v != "" {
			out = append(out, formField{Label: label, Value: v})
		}
	}
	add("期限(月)", "期限")
	add("锁定期(月)", "锁定期")
	add("首次观察敲出价(%)", "期初敲出线")
	add("敲出价递减步长(%)", "降敲")
	add("期末障碍价(%)", "降落伞")
	add("派息障碍价(%)", "派息线")
	add("每月或有派息(%)", "费后派息")
	add("保证金水平(%)", "保证金")
	add("是否追保", "是否追保")
	return out
}

// FetchWinrate 取真实回测胜率。
// - WINRATE_DRY_RUN=true：返回 canned 98.17%（source=dry-run），仅用于测试 agent 流程，生产不开。
// - 无 TONGYU_USER/PASS：返回 [胜率待补] + reason，不外泄凭证状态。
// - 否则：调 runTongyuBacktest（Task 2）。chromedp 失败/验证码/站点不可达 → [胜率待补] + reason。
func FetchWinrate(params map[string]any, cfg config.Config) map[string]any {
	if strings.EqualFold(cfg.WinrateDryRun, "true") {
		return map[string]any{"胜率": "98.17%", "source": "dry-run"}
	}
	if cfg.TongyuUser == "" || cfg.TongyuPass == "" {
		return map[string]any{"胜率": "[胜率待补]", "reason": "未配置 TONGYU_USER/TONGYU_PASS，请手动提供胜率"}
	}
	winrate, reason, err := runTongyuBacktest(params, tongyuCreds{User: cfg.TongyuUser, Pass: cfg.TongyuPass}, cfg.ChromePath)
	if err != nil || winrate == "" {
		r := reason
		if r == "" {
			r = err.Error()
		}
		return map[string]any{"胜率": "[胜率待补]", "reason": r}
	}
	return map[string]any{"胜率": winrate, "source": "tongyu"}
}

// fetchWinrate 是 agent 工具入口：取回测胜率或 [胜率待补] 占位。
func (s *Service) fetchWinrate(args map[string]any) map[string]any {
	out := FetchWinrate(args, s.cfg)
	out["标的"] = stringArg(args, "标的")
	return out
}

// tongyuCreds 与 runTongyuBacktest 的凭证契约（Task 2 定义 runTongyuBacktest）。
type tongyuCreds struct {
	User string
	Pass string
}
