# AMAC 管理人/产品公示信息抓取

本文档说明如何从中国证券投资基金业协会(AMAC,amac.org.cn)抓取**私募管理人**和**基金产品**的公示信息(登记编号、成立时间、法定代表人、托管人、运作状态等),用于推介材料的"管理人公示""产品公示"两节,通常配截图作为官方来源凭证。

## 何时用

用户要出 Word 推介材料时,材料模板里有"管理人-基金业协会公示图""产品-基金业协会公示图"两节,走本流程抓数据 + 截图。用户没要出材料时不跑。

## URL 规则

AMAC 全站搜索详情页,同一个 URL 模板,`type` 区分管理人/产品:

- **管理人**:`https://www.amac.org.cn/index/qzss/details/?type=1&name=<管理人全称>&code=<登记编号,如 101000008864>`
- **产品**:`https://www.amac.org.cn/index/qzss/details/?type=2&code=<产品编码,如 2105110905109934>&ctype=P`

管理人 URL 带 `name`(可省,有 `code` 即可);产品 URL 带 `code` + `ctype=P`。

## 关键坑:数据是 JS 异步加载的

AMAC 详情页的初始 HTML **只有字段名**(登记编号/成立时间等),**值**(P1006143/上官永强等)是 JS 异步加载的。所以**纯 urllib GET 抓不到值**,必须用浏览器渲染后取。用 Playwright 导航到 URL,等 1~2 秒让 JS 填值,再截图或读 DOM。

## 取数(两种输出)

1. **整页截图**(做公示凭证图,放 Word 里):
   ```js
   await page.goto(url); await page.waitForTimeout(2000);
   await page.screenshot({path: 'amac-xxx-fullpage.png', fullPage: true});
   ```
   fullPage 截图带 AMAC 官网头 + 详情,做公示凭证最合适。图偏高,build_docx 已做高度上限(20cm)防撑爆页。

2. **结构化字段**(做文字表格,可选):用 `browser_evaluate` 读 DOM。页面字段是"key: value"成对,但有些挤在一行,用按行 `innerText` 解析更稳:
   ```js
   () => {
     const lines = (document.querySelector('main')||document.body).innerText.split('\n').map(s=>s.trim()).filter(Boolean);
     return lines;
   }
   ```
   拿到行数组后,自己挑要的字段(管理人名称、登记编号、成立时间、机构类型、注册资本、实缴资本、法定代表人、办公地址、运作产品数等;产品:产品名称、产品编码、基金类型、成立/备案时间、到期日、管理人名称、托管人、运作状态)。

## 管理人 vs 产品 字段

管理人页(type=1):登记编号(P 开头)、机构类型、成立时间、登记时间、注册资本、实缴资本、法定代表人、实际控制人、注册/办公地址、会员类型、高管、正在运作/提前清算产品数等。

产品页(type=2):产品名称、产品编码(S 开头)、产品类型、基金类型、成立时间、备案时间、到期日、币种、基金管理人名称、托管人名称、管理类型、运作状态等。

## 字段会变怎么办

AMAC 页结构稳定(不像通毓是动态表单),但字段值 JS 加载。执行时导航后等够时间(1~2 秒)再取;截图前确认值已出现(如 `page.waitForFunction(() => document.body.innerText.includes('P1006143'))` 或直接等 2 秒)。字段定位用文本,不记死 ref。
