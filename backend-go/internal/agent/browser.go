package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// runTongyuBacktest 驱动通毓终端取真实回测胜率。
// 流程（按 references/tongyu-winrate.md）：登录 → 验证码检测(遇则退回) → 进回测页 →
// 关快捷查询 → 选结构类型 → 按标签填数值字段 → 点立即分析 → 读胜率。
// 选择器按字段标签文本定位（Vue SPA ref 每次变，不能记死）。首次 live 运行需按真实 DOM 调选择器。
func runTongyuBacktest(params map[string]any, creds tongyuCreds, chromePath string) (winrate, reason string, err error) {
	ctx, cancel := newBrowserContext(context.Background(), chromePath)
	defer cancel()

	if err := loginTongyu(ctx, creds); err != nil {
		return "", "登录失败：" + err.Error(), err
	}
	if hasCaptcha(ctx) {
		return "", "遇滑块验证码，请在浏览器手动跑一次取胜率", errors.New("captcha")
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://terminal.tongyu-quant.com/#/investmentAnalysis/InvestmentAnalysis"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return "", "进入回测页失败：" + err.Error(), err
	}
	// 关掉可能遮挡表单的"快捷查询"对话框（若存在）
	_ = chromedp.Run(ctx, chromedp.Click("//button[contains(@class,'close')]", chromedp.BySearch))

	// 选结构类型：按 structure_type 点对应单选/多选。structure_type 由 agent 从对话判断传入（如 "DCN+降敲+降落伞"）。
	structure := stringArg(params, "structure_type")
	if err := selectStructure(ctx, structure); err != nil {
		return "", "选结构类型失败：" + err.Error(), err
	}

	// 挂钩标的：搜索框输代码/名称（标的由 agent 传入）
	if u := stringArg(params, "标的"); u != "" {
		if err := fillUnderlying(ctx, u); err != nil {
			return "", "填挂钩标的失败：" + err.Error(), err
		}
	}

	// 填数值字段（paramToFormFields 给出 label+value）
	for _, f := range paramToFormFields(params) {
		if err := fillByLabel(ctx, f.Label, f.Value); err != nil {
			return "", fmt.Sprintf("填字段 %q 失败：%v", f.Label, err), err
		}
	}

	// 点"立即分析"
	if err := chromedp.Run(ctx, chromedp.Click("//button/span[contains(text(),'立即分析')]", chromedp.BySearch)); err != nil {
		return "", "点立即分析失败：" + err.Error(), err
	}

	// 等结果加载，读胜率
	wr, err := readWinrate(ctx)
	if err != nil {
		return "", "读胜率失败：" + err.Error(), err
	}
	return wr, "", nil
}

// newBrowserContext 起无头 Chrome。chromePath 空时让 chromedp 自动找 Chrome。
// 用持久化 userDataDir 降低验证码触发概率（参考文档建议）。
func newBrowserContext(parent context.Context, chromePath string) (context.Context, context.CancelFunc) {
	opts := append([]chromedp.ExecAllocatorOption{},
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("lang", "zh-CN"),
	)
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	// 合并 cancel：取消时先 ctx 再 allocCtx
	return ctx, func() { cancel(); cancelAlloc() }
}

// loginTongyu 登录通毓。用页面中央账号密码框（不是顶部"管理员登录"）。
func loginTongyu(ctx context.Context, creds tongyuCreds) error {
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://terminal.tongyu-quant.com/#/login"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	// 用户名/密码输入框：按 placeholder 或 type 定位（首次 live 运行按真实 DOM 调）
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`//input[@type='text' or @name='username' or contains(@placeholder,'账号') or contains(@placeholder,'用户')][1]`, chromedp.BySearch),
		chromedp.SendKeys(`//input[@type='text' or @name='username' or contains(@placeholder,'账号') or contains(@placeholder,'用户')][1]`, creds.User, chromedp.BySearch),
		chromedp.SendKeys(`//input[@type='password'][1]`, creds.Pass, chromedp.BySearch),
		chromedp.Click(`//button[contains(.,'登录') or contains(.,'Login')][1]`, chromedp.BySearch),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	// 登录成功后 URL 跳 #/；若停在 #/login 且有"用户名或密码错误"则凭证错（不重试，防风控）
	var url string
	_ = chromedp.Run(ctx, chromedp.Location(&url))
	if strings.Contains(url, "/login") {
		var body string
		_ = chromedp.Run(ctx, chromedp.Text("body", &body, chromedp.ByQuery))
		if strings.Contains(body, "用户名或密码错误") {
			return errors.New("用户名或密码错误（凭证错，不重试）")
		}
		return errors.New("登录后未跳转主页")
	}
	return nil
}

// hasCaptcha 检测滑块验证码（"请完成下列验证后继续"）。遇则 true，调用方退回 [胜率待补]。
func hasCaptcha(ctx context.Context) bool {
	var body string
	if err := chromedp.Run(ctx, chromedp.Text("body", &body, chromedp.ByQuery)); err != nil {
		return false
	}
	return strings.Contains(body, "请完成下列验证后继续")
}

// selectStructure 按 structure_type 字符串点结构单选 + 叠加条款多选。
// structure_type 形如 "DCN+降敲+降落伞"（agent 从对话判断传入）。
// 实现按标签文本点击；首次 live 运行按真实 DOM 调选择器。
func selectStructure(ctx context.Context, structure string) error {
	for _, part := range strings.Split(structure, "+") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// 点含该结构名的 label/span（antd 单选/多选用 label 包裹）
		_ = chromedp.Run(ctx, chromedp.Click(fmt.Sprintf(`//label[contains(.,'%s')]|//span[contains(.,'%s')]`, part, part), chromedp.BySearch))
		// FIX: 原简报中此处为裸 chromedp.Sleep(...)，Action 未经 chromedp.Run 执行是 no-op。
		// 改用 time.Sleep 让等待真正生效。
		time.Sleep(300 * time.Millisecond)
	}
	return nil
}

// fillUnderlying 在"挂钩标的"搜索框输标的，选第一条结果。
func fillUnderlying(ctx context.Context, underlying string) error {
	if err := chromedp.Run(ctx,
		chromedp.Click(`//input[@placeholder[contains(.,'标的') or contains(.,'代码') or contains(.,'名称')]][1]`, chromedp.BySearch),
		chromedp.SendKeys(`//input[@placeholder[contains(.,'标的') or contains(.,'代码') or contains(.,'名称')]][1]`, underlying, chromedp.BySearch),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`//li[contains(@class,'ant-select-item')][1]`, chromedp.BySearch),
	); err != nil {
		return err
	}
	return nil
}

// fillByLabel 按字段标签文本定位 antd 输入框并填值。
// 选择器策略：找含标签文本的 span/label，向上找 ant-form-item，取其下 input。
// 首次 live 运行需按真实 DOM 结构调（参考文档要求按字段名定位当前 ref）。
func fillByLabel(ctx context.Context, label, value string) error {
	xpath := fmt.Sprintf(`//span[contains(text(),'%s')]/ancestor::*[contains(@class,'ant-form-item')]//input | //label[contains(.,'%s')]/ancestor::*[contains(@class,'ant-form-item')]//input`, label, label)
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(xpath, chromedp.BySearch),
		chromedp.Click(xpath, chromedp.BySearch),
		chromedp.SetValue(xpath, value, chromedp.BySearch),
	); err != nil {
		// SetValue 失败时回退到 SendKeys 逐字输入（参考文档：fill 不生效改逐字）
		if e2 := chromedp.Run(ctx, chromedp.SendKeys(xpath, value, chromedp.BySearch)); e2 != nil {
			return err
		}
	}
	return nil
}

// readWinrate 从结果区读胜率百分比。结果区标题含"买入一份该SNOWBALL合约"。
// 胜率形如 "98.14%"。按结果区文本正则取首个百分比。
func readWinrate(ctx context.Context) (string, error) {
	// 等结果区出现
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`//*[contains(.,'买入一份') or contains(.,'胜率')]`, chromedp.BySearch),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		return "", err
	}
	var text string
	if err := chromedp.Run(ctx, chromedp.Text(`//*[contains(.,'胜率')]`, &text, chromedp.BySearch)); err != nil {
		return "", err
	}
	// 取首个百分比
	wr := firstPercent(text)
	if wr == "" {
		return "", errors.New("结果区未找到胜率百分比")
	}
	return wr, nil
}

// firstPercent 从文本里取第一个形如 12.34% 的百分比。
func firstPercent(s string) string {
	var b strings.Builder
	seen := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9' || r == '.':
			b.WriteRune(r)
			seen = true
		case r == '%' && seen:
			return b.String() + "%"
		default:
			if seen {
				b.Reset()
				seen = false
			}
		}
	}
	return ""
}
