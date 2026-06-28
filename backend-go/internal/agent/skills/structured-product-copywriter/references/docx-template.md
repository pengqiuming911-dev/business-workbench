# Word 推介材料模板

用户最终要的输出是 **Word(.docx)** 推介材料,不是 PDF。本文档定义标准章节结构和 manifest 写法,供 agent 的 `build_docx` 工具装配(并上传飞书 Drive)。

## 标准章节顺序

按用户确认的模板,一份完整推介材料按以下顺序组织(每节一个 section):

1. **产品结构文字版(长版)** — `copy_file` 载入长版文案文件(🚀标题+副标题+正文+参数块)
2. **产品结构文字版(短版)** — `copy_file` 载入短版文案文件(参数块+打款日/入场时间)
3. **产品公告群通知** — `subheading` "稳定版接龙通知" + `body`(接龙通知内容;用户有模板就填,没有留占位)
4. **产品派息与敲出观察点位表** — `image` 产品卡(通毓产品点位页生成的 `product-card.png`)
5. **产品胜率数据** — `body`(回测区间+胜率,如"回测2016-06-26至2026-06-25,胜率98.14%")+ `image` 胜率结果截图(通毓回测页 `winrate-result.png`)
6. **一页通** — `image`(用户手动贴,留占位 `onepager.png`)
7. **管理人-基金业协会公示图** — `body`(管理人全称+AMAC URL)+ `image` 管理人公示截图(`amac-manager-fullpage.png`,见 `amac-manager.md`)
8. **产品-基金业协会公示图** — `body`(产品全称+AMAC URL)+ `image` 产品公示截图(`amac-product-fullpage.png`)
9. **托管募集账户核对** — `params` 或 `body`(账户/账号/银行 标签;内容用户后填,可留空)+ `image`(用户手动贴,留占位 `custody-account.png`)
10. **销售常见问题** — `link_list`,每个分类一项 `{label, url}`,挂飞书文档超链接(可点击)。分类:管理人相关/交易台相关/托管相关/申购赎回流程/衍生品设计/衍选公司/销售沟通(发行计划无链接,已删)。`link_list` 示例:
    ```json
    {"type":"link_list","items":[
      {"label":"管理人相关常见问题","url":"https://.../docx/Qsi..."},
      {"label":"交易台相关问题","url":"https://.../docx/JVs..."}
    ]}
    ```

## manifest JSON 结构

`build_docx` 工具(传 sections 数组 manifest,工具自动装配 .docx 并上传飞书 Drive「某年某月产品」文件夹,返回飞书链接)。manifest 里图片路径找不到会自动插"[图片待补:xxx]"红字占位,不报错——所以一页通/托管截图这类用户手动贴的,留个不存在的路径即可。

section 类型:`heading`(一级标题)、`subheading`(二级)、`body`(正文,按 `\n` 分段)、`params`(参数块等宽小字)、`image`(图+caption,等比缩放、高度上限 20cm)、`separator`(分隔线)、`copy_file`(读文案文件,自动按行判别 标题/参数/正文/分隔)、`link_list`(带超链接的列表,每项 `{label,url}`)。

参考实例见工作区 `manifest.json`。换产品时,改 manifest 里的文案文件路径、图片路径、管理人/产品名称和 URL、托管账户文字即可。

## 字体

中文用 Windows 自带微软雅黑(Normal 样式已设 eastAsia);参数块用 Consolas+雅黑。emoji(🚀)Word 雅黑不含,build_docx 已自动剥离 BMP 外字符,标题文字照常显示。

## 注意

- 文案里的数字要和实际抓取的一致(当前点位用 `fetch_quote` 工具、胜率用 `fetch_winrate` 工具、管理人/产品信息用 AMAC via `screenshot_amac`);用户若给了定稿文案原文(含特定数字),按用户给的原文用,不要擅自改。
- AMAC 截图偏高,build_docx 高度上限 20cm,会自动按高度缩放;不要手动改图。
