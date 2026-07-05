# AMAC 管理人/产品公示图

用于飞书原生云文档第 8 节和第 9 节。

必须截图 AMAC 详情页，不要截图搜索结果页。

URL 规则：

- 管理人详情页：`https://www.amac.org.cn/index/qzss/details/?type=1&name=<管理人全称>&code=<登记编号>`
- 产品详情页：`https://www.amac.org.cn/index/qzss/details/?type=2&code=<产品编码>&ctype=P`

调用 `screenshot_amac`：

```json
{"url":"https://www.amac.org.cn/index/qzss/details/?type=1&name=...&code=..."}
```

截图要求：
- 等待 JS 异步数据加载完成后再截。
- 定位详情页实际内容容器，只截有文字、表格、图片的有效区域。
- 不按浏览器视口整页截图。
- 截图后 trim/crop，裁掉底部和四周空白，底部距离最后一行内容不超过 20px。

飞书章节写法：

```json
{"type":"heading","text":"管理人-基金业协会公示图"},
{"type":"body","text":"北京泰创投资管理有限公司\nhttps://www.amac.org.cn/index/qzss/details/?type=1&name=...&code=..."},
{"type":"image","path":"public/poster-artifacts/amac-manager.png","caption":"管理人基金业协会公示图"}
```
