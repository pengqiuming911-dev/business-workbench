package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
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
// 包裹 90s 超时：executeTool 同步无 per-tool 超时，通毓站点挂起会让 SSE 流无限阻塞；
// 超时后 chromedp 取消 → chromedp.Run 返回 err → runTongyuBacktest 退回 [胜率待补]。
func newBrowserContext(parent context.Context, chromePath string) (context.Context, context.CancelFunc) {
	timeoutCtx, cancelTimeout := context.WithTimeout(parent, 90*time.Second)
	opts := append([]chromedp.ExecAllocatorOption{},
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("lang", "zh-CN"),
		// 持久化 profile 降低验证码触发概率（参考 tongyu-winrate.md）；固定相对路径，未来可改配置项。
		chromedp.UserDataDir("tongyu-chrome-profile"),
	)
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(timeoutCtx, opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	// 合并 cancel：取消时先 ctx 再 allocCtx 再 timeoutCtx
	return ctx, func() { cancel(); cancelAlloc(); cancelTimeout() }
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
		// DEBUG: 截图 + body 片段供选择器调参（live tuning）
		var shot []byte
		_ = chromedp.Run(ctx, chromedp.FullScreenshot(&shot, 90))
		_ = os.MkdirAll("public/poster-artifacts", 0o755)
		_ = os.WriteFile("public/poster-artifacts/tongyu-login-debug.png", shot, 0o644)
		_ = os.WriteFile("public/poster-artifacts/tongyu-login-debug.txt", []byte("URL: "+url+"\n\nBODY:\n"+body), 0o644)
		snippet := body
		if len(snippet) > 500 {
			snippet = snippet[:500]
		}
		return fmt.Errorf("登录后未跳转主页 (url=%s); body=%s; screenshot=public/poster-artifacts/tongyu-login-debug.png", url, snippet)
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
		// 捕获点击错误：基础结构（DCN/雪球）若点空，页面保留默认结构，后续字段对错表单、
		// 立即分析提交，readWinrate 会返回"真实但属错结构"的胜率——agent 会自信地引用错数。
		if err := chromedp.Run(ctx, chromedp.Click(fmt.Sprintf(`//label[contains(.,'%s')]|//span[contains(.,'%s')]`, part, part), chromedp.BySearch)); err != nil {
			return fmt.Errorf("点结构 %q 失败: %w", part, err)
		}
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

// screenshotAMAC 打开 AMAC 详情页（JS 异步加载值），等值出现后整页截图存 outPath。
// url 用 references/amac-manager.md 的模板（type=1 管理人 / type=2 产品）。
func screenshotAMAC(url, outPath string) error {
	ctx, cancel := newBrowserContext(context.Background(), "")
	defer cancel()
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second), // 等 JS 填值
	); err != nil {
		return err
	}
	var buf []byte
	// FullScreenshot（整页）而非 CaptureScreenshot（视口）：AMAC 公示页要整页截图作官方来源凭证（references/amac-manager.md 要求 fullPage）。
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 100)); err != nil {
		return err
	}
	return os.WriteFile(outPath, buf, 0o644)
}

// screenshotProductCard 进通毓"产品点位"小工具，按参数填表→提交→复制为图片→取图存 outPath。
// 剪贴板读不到时回退元素截图。流程见 references/product-position-card.md。
func screenshotProductCard(params map[string]any, creds tongyuCreds, chromePath, outPath string) error {
	ctx, cancel := newBrowserContext(context.Background(), chromePath)
	defer cancel()
	if err := loginTongyu(ctx, creds); err != nil {
		return err
	}
	if hasCaptcha(ctx) {
		return fmt.Errorf("遇验证码，产品卡截图失败")
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://terminal.tongyu-quant.com/smallTool/index.html#/product-position"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	// 选产品类型（原生 select）
	// 适配：chromedp v0.15.1 无 SetAttribute；改用 Evaluate 设 select.selectedIndex 并派发
	// change/input 事件（Vue/原生均监听 change）。保持简报意图：按 structure_type 选 DCN/锁盈。
	if pt := stringArg(params, "structure_type"); pt != "" {
		js := fmt.Sprintf(`(function(){
			var s = document.querySelector('select');
			if (!s) return;
			var v = %q;
			for (var i = 0; i < s.options.length; i++) {
				if (s.options[i].value === v || s.options[i].text === v) {
					s.selectedIndex = i;
					break;
				}
			}
			s.dispatchEvent(new Event('change', {bubbles: true}));
			s.dispatchEvent(new Event('input', {bubbles: true}));
		})()`, selectProductType(pt))
		_ = chromedp.Run(ctx, chromedp.Evaluate(js, nil))
	}
	// 按标签填数值字段
	for _, f := range productCardFields(params) {
		_ = fillByLabel(ctx, f.Label, f.Value)
	}
	// 点提交
	if err := chromedp.Run(ctx, chromedp.Click(`//button/span[contains(text(),'提交')]`, chromedp.BySearch)); err != nil {
		return err
	}
	// FIX: 简报此处为裸 chromedp.Sleep(...)，Action 未经 chromedp.Run 是 no-op；改用 time.Sleep。
	time.Sleep(1 * time.Second)
	// 点"复制为图片"
	_ = chromedp.Run(ctx, chromedp.Click(`//button/span[contains(text(),'复制为图片')]`, chromedp.BySearch))
	// 优先：剪贴板读 PNG
	if png, err := readClipboardPNG(ctx); err == nil && len(png) > 0 {
		return os.WriteFile(outPath, png, 0o644)
	}
	// 兜底：结果卡元素截图
	var buf []byte
	shotCtx, cancelShot := context.WithTimeout(ctx, 10*time.Second)
	defer cancelShot()
	if err := chromedp.Run(shotCtx, chromedp.Screenshot(`//*[contains(@class,'product-result')]`, &buf, chromedp.BySearch)); err != nil {
		return fmt.Errorf("剪贴板与元素截图均失败: %w", err)
	}
	return os.WriteFile(outPath, buf, 0o644)
}

// selectProductType 把 structure_type 映射成通毓产品点位页 select 的值（DCN/锁盈）。
func selectProductType(structure string) string {
	s := strings.ToLower(structure)
	if strings.Contains(s, "锁盈") || strings.Contains(s, "经典") {
		return "锁盈"
	}
	return "DCN"
}

// productCardFields 把产品参数映射成产品点位页表单字段标签+值（references/product-position-card.md 表）。
func productCardFields(params map[string]any) []formField {
	get := func(k string) string { return stringArg(params, k) }
	var out []formField
	add := func(label, key string) {
		if v := get(key); v != "" {
			out = append(out, formField{Label: label, Value: v})
		}
	}
	add("期限(月)", "期限")
	add("锁定期(月)", "锁定期")
	add("保证金(%)", "保证金")
	add("敲出线(%)", "期初敲出线")
	add("降敲(每月)", "降敲")
	add("降落伞(%)", "降落伞")
	add("每月或有派息(%)", "费后派息")
	add("派息线(%)", "派息线")
	add("入场点位", "current_price")
	return out
}

// readClipboardPNG 用浏览器 evaluate 调 navigator.clipboard.read 取 image/png，返回 PNG 字节。
func readClipboardPNG(ctx context.Context) ([]byte, error) {
	var b64 string
	js := `(async () => {
		const items = await navigator.clipboard.read();
		for (const it of items) {
			for (const type of it.types) {
				if (type === 'image/png') {
					const blob = await it.getType(type);
					const arr = new Uint8Array(await blob.arrayBuffer());
					let s = ''; for (let i = 0; i < arr.length; i++) s += String.fromCharCode(arr[i]);
					return btoa(s);
				}
			}
		}
		return '';
	})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &b64)); err != nil {
		return nil, err
	}
	if b64 == "" {
		return nil, fmt.Errorf("clipboard 无 image/png")
	}
	return base64.StdEncoding.DecodeString(b64)
}
