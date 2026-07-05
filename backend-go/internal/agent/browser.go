package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
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
	// 关掉可能遮挡表单的"快捷查询"对话框（若存在，2s 内点不到就跳过——不阻塞主流程）
	closeCtx, closeCancel := context.WithTimeout(ctx, 2*time.Second)
	_ = chromedp.Run(closeCtx, chromedp.Click("//button[contains(@class,'close')]", chromedp.BySearch))
	closeCancel()

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
	userDataDir, _ := os.MkdirTemp("", "tongyu-chrome-profile-")
	if userDataDir == "" {
		userDataDir = "tongyu-chrome-profile"
	}
	opts := append([]chromedp.ExecAllocatorOption{},
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("lang", "zh-CN"),
		// 每次用独立 profile，避免连续/并发工具调用时 Chromium profile lock 卡住。
		chromedp.UserDataDir(userDataDir),
	)
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(timeoutCtx, opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	// 合并 cancel：取消时先 ctx 再 allocCtx 再 timeoutCtx
	return ctx, func() { cancel(); cancelAlloc(); cancelTimeout(); _ = os.RemoveAll(userDataDir) }
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
		chromedp.Click(`//button[contains(.,'登录账号') or contains(.,'Login')][1]`, chromedp.BySearch),
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
	// DEBUG: 在点结构前先抓页面 URL + body（click 超时后 ctx 已死，事后抓为空）
	var debugURL string
	_ = chromedp.Run(ctx, chromedp.Location(&debugURL))
	var body string
	_ = chromedp.Run(ctx, chromedp.Text("body", &body, chromedp.ByQuery))
	var innerHTML string
	_ = chromedp.Run(ctx, chromedp.Evaluate(`document.body.innerHTML`, &innerHTML))
	var shot []byte
	_ = chromedp.Run(ctx, chromedp.FullScreenshot(&shot, 90))
	_ = os.MkdirAll("public/poster-artifacts", 0o755)
	_ = os.WriteFile("public/poster-artifacts/tongyu-structure-debug.txt", []byte("URL: "+debugURL+"\n\nBODY:\n"+body), 0o644)
	_ = os.WriteFile("public/poster-artifacts/tongyu-structure-debug.html", []byte(innerHTML), 0o644)
	_ = os.WriteFile("public/poster-artifacts/tongyu-structure-debug.png", shot, 0o644)
	targets := structureTargets(structure)
	if len(targets) == 0 {
		targets = []string{"经典结构"}
	}
	// 先确保进入"组合结构类型"，否则基础结构/叠加条款可能处于 disabled。
	_ = chromedp.Run(ctx, chromedp.Evaluate(`(function(){
		var labels = Array.from(document.querySelectorAll('label.ant-radio-wrapper'));
		var combo = labels.find(function(el){ return (el.textContent || '').indexOf('组合结构类型') >= 0; });
		if (combo) combo.click();
	})()`, nil))
	time.Sleep(300 * time.Millisecond)
	for _, target := range targets {
		js := fmt.Sprintf(`(function(){
			var want = %q;
			var spans = Array.from(document.querySelectorAll('span.spantxt'));
			for (var i=0;i<spans.length;i++){
				var t = (spans[i].textContent || '').replace(/\s+/g, '');
				if (t === want.replace(/\s+/g, '') || t.indexOf(want.replace(/\s+/g, '')) >= 0){
					spans[i].click(); return 'ok';
				}
			}
			return 'notfound:' + want;
		})()`, target)
		var res string
		if err := chromedp.Run(ctx, chromedp.Evaluate(js, &res)); err != nil {
			return fmt.Errorf("点结构 %q 失败: %w", target, err)
		}
		if res != "ok" {
			return fmt.Errorf("点结构 %q 失败: 未找到 span.spantxt", target)
		}
		// FIX: 原简报中此处为裸 chromedp.Sleep(...)，Action 未经 chromedp.Run 执行是 no-op。
		// 改用 time.Sleep 让等待真正生效。
		time.Sleep(300 * time.Millisecond)
	}
	return nil
}

func structureTargets(structure string) []string {
	s := strings.TrimSpace(structure)
	if s == "" {
		return nil
	}
	var targets []string
	add := func(v string) {
		for _, existing := range targets {
			if existing == v {
				return
			}
		}
		targets = append(targets, v)
	}
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "dcn"):
		add("DCN")
	case strings.Contains(s, "凤凰"):
		add("凤凰结构")
	case strings.Contains(s, "早利"):
		add("早利结构")
	case strings.Contains(s, "蝶变"):
		add("蝶变结构")
	case strings.Contains(s, "保本"):
		add("保本结构")
	case strings.Contains(s, "彩虹"):
		add("彩虹结构")
	case strings.Contains(s, "FCN") || strings.Contains(lower, "fcn"):
		add("FCN")
	default:
		// 业务口径里的"雪球/降敲雪球"在通毓页面对应基础结构"经典结构"。
		add("经典结构")
	}
	if strings.Contains(s, "降敲") || strings.Contains(s, "降KO") || strings.Contains(lower, "step") {
		add("降敲结构")
	}
	if strings.Contains(s, "降落伞") || strings.Contains(lower, "airbag") {
		add("降落伞结构")
	}
	return targets
}

// fillUnderlying 在"挂钩标的"搜索框输标的，选第一条结果。
func fillUnderlying(ctx context.Context, underlying string) error {
	js := fmt.Sprintf(`(function(){
		var want = %q;
		function norm(s){ return (s || '').replace(/\s+/g, '').toLowerCase(); }
		var labels = Array.from(document.querySelectorAll('label'));
		var label = labels.find(function(el){ return el.getAttribute('title') === '挂钩标的' || (el.textContent || '').trim() === '挂钩标的'; });
		if (!label) return 'label-not-found';
		var item = label.closest('.ant-form-item');
		if (!item) return 'form-item-not-found';
		var selected = item.querySelector('.ant-select-selection-item-content, .ant-select-selection-item');
		if (selected && (norm(want).indexOf(norm(selected.textContent)) >= 0 || norm(selected.textContent).indexOf(norm(want)) >= 0)) {
			return 'ok-existing';
		}
		var selector = item.querySelector('.ant-select-selector');
		if (!selector) return 'selector-not-found';
		selector.click();
		var input = item.querySelector('input[role="combobox"], input.ant-select-selection-search-input');
		if (!input) return 'input-not-found';
		input.removeAttribute('readonly');
		input.focus();
		var setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
		setter.call(input, want);
		input.dispatchEvent(new Event('input', {bubbles:true}));
		input.dispatchEvent(new Event('change', {bubbles:true}));
		return 'searching';
	})()`, underlying)
	var res string
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &res)); err != nil {
		return err
	}
	if res == "ok-existing" {
		return nil
	}
	if res != "searching" {
		return fmt.Errorf("填挂钩标的失败: %s", res)
	}
	time.Sleep(1500 * time.Millisecond)
	clickJS := fmt.Sprintf(`(function(){
		var want = %q;
		function norm(s){ return (s || '').replace(/\s+/g, '').toLowerCase(); }
		var options = Array.from(document.querySelectorAll('.ant-select-item-option, [role="option"], li'));
		var exact = options.find(function(el){ return norm(el.textContent).indexOf(norm(want)) >= 0 || norm(want).indexOf(norm(el.textContent)) >= 0; });
		var option = exact || options.find(function(el){ return (el.textContent || '').trim() !== ''; });
		if (!option) return 'option-not-found';
		option.click();
		return 'ok';
	})()`, underlying)
	if err := chromedp.Run(ctx, chromedp.Evaluate(clickJS, &res)); err != nil {
		return err
	}
	if res != "ok" {
		return fmt.Errorf("填挂钩标的失败: %s", res)
	}
	time.Sleep(300 * time.Millisecond)
	return nil
}

// fillByLabel 按字段标签文本定位 antd 输入框并填值。
// 选择器策略：找含标签文本的 span/label，向上找 ant-form-item，取其下 input。
// 首次 live 运行需按真实 DOM 结构调（参考文档要求按字段名定位当前 ref）。
func fillByLabel(ctx context.Context, label, value string) error {
	js := fmt.Sprintf(`(function(){
		var labelText = %q;
		var value = %q;
		function norm(s){ return (s || '').replace(/\s+/g, ''); }
		var labels = Array.from(document.querySelectorAll('label'));
		var label = labels.find(function(el){
			return norm(el.getAttribute('title')) === norm(labelText) || norm(el.textContent) === norm(labelText);
		}) || labels.find(function(el){
			return norm(el.getAttribute('title')).indexOf(norm(labelText)) >= 0 || norm(el.textContent).indexOf(norm(labelText)) >= 0;
		});
		if (!label) return 'label-not-found';
		var item = label.closest('.ant-form-item') || label.parentElement;
		if (!item) return 'form-item-not-found';
		if (labelText === '是否追保') {
			var radioLabel = Array.from(item.querySelectorAll('label')).find(function(el){ return norm(el.textContent).indexOf(norm(value)) >= 0; });
			if (radioLabel) { radioLabel.click(); return 'ok-radio'; }
		}
		var input = item.querySelector('input');
		if (!input) return 'input-not-found';
		input.focus();
		var setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
		setter.call(input, value);
		input.dispatchEvent(new Event('input', {bubbles:true}));
		input.dispatchEvent(new Event('change', {bubbles:true}));
		input.dispatchEvent(new Event('blur', {bubbles:true}));
		return 'ok';
	})()`, label, value)
	var res string
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &res)); err != nil {
		return err
	}
	if res != "ok" && res != "ok-radio" {
		return fmt.Errorf("填字段 %q 失败: %s", label, res)
	}
	time.Sleep(150 * time.Millisecond)
	return nil
}

// readWinrate 从结果区读胜率百分比。结果区标题含"买入一份该SNOWBALL合约"。
// 胜率形如 "98.14%"。按结果区文本正则取首个百分比。
func readWinrate(ctx context.Context) (string, error) {
	if err := chromedp.Run(ctx, chromedp.WaitVisible(`//*[contains(.,'买入一份') or contains(.,'胜率')]`, chromedp.BySearch)); err != nil {
		return "", err
	}
	deadline := time.Now().Add(60 * time.Second)
	var lastText string
	for time.Now().Before(deadline) {
		var text string
		err := chromedp.Run(ctx, chromedp.Evaluate(`(function(){
			var rows = Array.from(document.querySelectorAll('.chart-tip-box .ant-row, .chart-tip-box'));
			var row = rows.find(function(el){ return (el.textContent || '').indexOf('胜率') >= 0; });
			return row ? row.textContent : document.body.innerText;
		})()`, &text))
		if err != nil {
			return "", err
		}
		lastText = text
		if wr := firstPercent(text); wr != "" {
			return wr, nil
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("结果区未找到胜率百分比: %s", lastText)
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

// screenshotAMAC 打开 AMAC 详情页（JS 异步加载值），等值出现后截 .content 内容卡存 outPath。
// url 用 references/amac-manager.md 的模板（type=1 管理人 / type=2 产品）。
// 截 .content 内容卡（非整页 FullScreenshot），匹配管理人公示模板：无 AMAC 官网头/页脚的下方留白。
// .content 取不到或截图空才回退整页（保旧行为兜底）。
func screenshotAMAC(url, outPath string) error {
	ctx, cancel := newBrowserContext(context.Background(), "")
	defer cancel()
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1280, 900), // 让 .content 排到模板宽度，截图与管理人公示模板一致
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second), // 等 JS 初载填值
	); err != nil {
		return err
	}
	// 等 .content 值加载完：轮询直到文本出现连续数字（登记编号/日期）且 "--" 占位消失；超时 10s 放行不阻塞。
	waitDeadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(waitDeadline) {
		var ready bool
		_ = chromedp.Run(ctx, chromedp.Evaluate(`(function(){
			var el = document.querySelector('.content');
			if (!el) return false;
			var txt = el.innerText || '';
			return /\d{4,}/.test(txt) && txt.indexOf('--') < 0;
		})()`, &ready))
		if ready {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	var buf []byte
	// 截 .content 内容卡：无下方留白，对齐管理人公示模板
	if err := chromedp.Run(ctx, chromedp.Screenshot(`.content`, &buf, chromedp.ByQuery)); err == nil && len(buf) > 0 {
		return os.WriteFile(outPath, buf, 0o644)
	}
	// 兜底：.content 取不到或截图空 → 整页截图
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 100)); err != nil {
		return err
	}
	return os.WriteFile(outPath, buf, 0o644)
}

// screenshotProductCard 进通毓"产品点位"小工具，按参数填表→提交→复制为图片→取图存 outPath。
// 剪贴板读不到时回退元素截图。流程见 references/product-position-card.md。
func screenshotProductCard(params map[string]any, creds tongyuCreds, chromePath, outPath string) error {
	parent, cancelParent := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancelParent()
	ctx, cancel := newBrowserContext(parent, chromePath)
	defer cancel()
	chromedp.ListenTarget(ctx, func(ev any) {
		if _, ok := ev.(*page.EventJavascriptDialogOpening); ok {
			go func() {
				_ = chromedp.Run(ctx, page.HandleJavaScriptDialog(true))
			}()
		}
	})
	if err := loginTongyu(ctx, creds); err != nil {
		return err
	}
	if hasCaptcha(ctx) {
		return fmt.Errorf("遇验证码，产品卡截图失败")
	}
	debugProductCard(ctx, "after-login")
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://terminal.tongyu-quant.com/smallTool/index.html#/product-position"),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	debugProductCard(ctx, "after-navigate")
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
		time.Sleep(500 * time.Millisecond)
	}
	// 按标签填数值字段
	cardName := productCardName(params)
	for _, f := range productCardFields(params, cardName) {
		if err := fillByLabel(ctx, f.Label, f.Value); err != nil {
			debugProductCard(ctx, "fill-failed")
			return err
		}
	}
	debugProductCard(ctx, "after-fill")
	// 点提交
	submitCtx, cancelSubmit := context.WithTimeout(ctx, 8*time.Second)
	var submitRes string
	if err := chromedp.Run(submitCtx, chromedp.Evaluate(`(function(){
		var buttons = Array.from(document.querySelectorAll('button'));
		var btn = buttons.find(function(el){ return (el.textContent || '').indexOf('提交') >= 0; });
		if (!btn) return 'submit-not-found';
		btn.scrollIntoView({block:'center'});
		btn.click();
		return 'ok';
	})()`, &submitRes)); err != nil {
		cancelSubmit()
		return err
	}
	cancelSubmit()
	if submitRes != "ok" {
		debugProductCard(ctx, "submit-click-failed")
		return fmt.Errorf("产品卡提交按钮点击失败: %s", submitRes)
	}
	debugProductCard(ctx, "after-submit-click")
	if err := waitProductCardResult(ctx, cardName); err != nil {
		debugProductCard(ctx, "submit-failed")
		return err
	}
	debugProductCard(ctx, "after-submit")

	var buf []byte
	shotCtx, cancelShot := context.WithTimeout(ctx, 12*time.Second)
	defer cancelShot()
	markJS := fmt.Sprintf(`(function(){
		var cardName = %q;
		var candidates = Array.from(document.querySelectorAll('div')).filter(function(el){
			var t = el.textContent || '';
			return t.indexOf(cardName) >= 0 && t.indexOf('结构解析') >= 0 && t.indexOf('敲出参考图') >= 0;
		}).filter(function(el){
			var r = el.getBoundingClientRect();
			return r.width > 300 && r.height > 300;
		});
		candidates.sort(function(a,b){
			var ar = a.getBoundingClientRect(), br = b.getBoundingClientRect();
			return (ar.width * ar.height) - (br.width * br.height);
		});
		if (!candidates.length) return 'not-found';
		document.querySelectorAll('[data-capture-product-card]').forEach(function(el){ el.removeAttribute('data-capture-product-card'); });
		candidates[0].setAttribute('data-capture-product-card', 'true');
		candidates[0].scrollIntoView({block:'center'});
		return 'ok';
	})()`, cardName)
	var markRes string
	if err := chromedp.Run(shotCtx, chromedp.Evaluate(markJS, &markRes)); err == nil && markRes == "ok" {
		_ = chromedp.Run(shotCtx, chromedp.Screenshot(`[data-capture-product-card="true"]`, &buf, chromedp.ByQuery))
	}
	if len(buf) > 0 {
		return os.WriteFile(outPath, buf, 0o644)
	}

	copyCtx, cancelCopy := context.WithTimeout(ctx, 8*time.Second)
	defer cancelCopy()
	_ = chromedp.Run(copyCtx, chromedp.Click(`//button[contains(.,'复制为图片') or .//span[contains(.,'复制为图片')]]`, chromedp.BySearch))
	if png, err := readClipboardPNGWithTimeout(ctx, 5*time.Second); err == nil && len(png) > 0 {
		return os.WriteFile(outPath, png, 0o644)
	}
	debugProductCard(ctx, "capture-failed")
	return fmt.Errorf("产品卡结果区域截图失败")
}

func productCardName(params map[string]any) string {
	for _, key := range []string{"产品名称", "product_name", "name"} {
		if v := strings.TrimSpace(stringArg(params, key)); v != "" {
			return v
		}
	}
	if u := strings.TrimSpace(stringArg(params, "标的")); u != "" {
		return u + "产品点位卡"
	}
	return "产品点位卡"
}

func waitProductCardResult(ctx context.Context, cardName string) error {
	deadline := time.Now().Add(12 * time.Second)
	var body string
	for time.Now().Before(deadline) {
		body = ""
		readCtx, cancel := context.WithTimeout(ctx, time.Second)
		_ = chromedp.Run(readCtx, chromedp.Evaluate(`document.body ? document.body.innerText : ''`, &body))
		cancel()
		if strings.Contains(body, cardName) && strings.Contains(body, "结构解析") && strings.Contains(body, "敲出参考图") {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	if strings.Contains(body, "Please fill out this field") || strings.Contains(body, "请填写") {
		return fmt.Errorf("产品卡提交失败，表单仍有必填项未通过")
	}
	return fmt.Errorf("产品卡提交后未看到新结果: %s", cardName)
}

func debugProductCard(ctx context.Context, stage string) {
	_ = os.MkdirAll("public/poster-artifacts", 0o755)
	var url string
	_ = chromedp.Run(ctx, chromedp.Location(&url))
	var body string
	_ = chromedp.Run(ctx, chromedp.Text("body", &body, chromedp.ByQuery))
	if len(body) > 4000 {
		body = body[:4000]
	}
	var shot []byte
	_ = chromedp.Run(ctx, chromedp.FullScreenshot(&shot, 70))
	_ = os.WriteFile("public/poster-artifacts/product-card-debug-"+stage+".txt", []byte("URL: "+url+"\n\nBODY:\n"+body), 0o644)
	if len(shot) > 0 {
		_ = os.WriteFile("public/poster-artifacts/product-card-debug-"+stage+".png", shot, 0o644)
	}
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
func productCardFields(params map[string]any, cardName string) []formField {
	get := func(k string) string { return stringArg(params, k) }
	out := []formField{{Label: "产品名称", Value: cardName}}
	add := func(label, key string) {
		if v := get(key); v != "" {
			out = append(out, formField{Label: label, Value: v})
		}
	}
	add("期限", "期限")
	add("锁定期", "锁定期")
	add("保证金", "保证金")
	add("敲出线", "期初敲出线")
	add("降敲", "降敲")
	add("降落伞", "降落伞")
	add("每月或有派息", "费后派息")
	add("派息线", "派息线")
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

func readClipboardPNGWithTimeout(ctx context.Context, timeout time.Duration) ([]byte, error) {
	clipCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return readClipboardPNG(clipCtx)
}
