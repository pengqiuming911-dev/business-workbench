# 产品点位卡生成流程

使用通毓终端产品点位小工具生成“产品派息与敲出观察点位表图片”。

页面：
`https://terminal.tongyu-quant.com/smallTool/index.html#/product-position`

调用 `screenshot_product_card` 工具，传入产品参数即可。工具会登录通毓、填表、提交、优先使用页面“复制为图片”得到原图；失败时才按目标内容容器截图，并裁掉空白边缘。

生成结果作为飞书原生云文档第 5 节图片：

```json
{"type":"heading","text":"产品派息与敲出观察点位表图片"},
{"type":"image","path":"public/poster-artifacts/product-card.png","caption":"产品派息与敲出观察点位表图片"}
```

不要再使用 `build_docx`，除非用户明确要求 Word 附件。
