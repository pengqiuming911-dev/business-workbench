# Agent Tools 扩展设计 - 全量业务数据覆盖

**日期**: 2026-06-13  
**状态**: 待审批  
**技术栈**: 当前栈（Node.js/Express + SQLite/sql.js）

---

## 背景

业务工作台 Agent 已有 6 个工具（`search_products`, `get_product_detail`, `get_observations`, `get_price`, `get_dashboard_stats`, `get_observation_calendar`），覆盖了产品和观察记录领域。但数据库中还有大量业务实体（客户、交易、渠道、喜报、文档、运维日志）没有工具覆盖，导致 Agent 无法回答这些领域的查询。

本次设计新增 10 个工具，实现全业务数据覆盖。

## 设计原则

1. **单一职责**：每个工具只做一件事，降低 LLM 调用歧义
2. **只读查询**：所有工具为只读操作，不修改数据
3. **统一错误格式**：失败时返回 `{ error: "描述" }`，不抛异常
4. **复用已有 DB 函数**：尽量使用 `db.js` 已有的查询函数，减少新增代码

## 工具详细设计

### 1. `search_customers` - 搜索合投客户

**用途**: 搜索潜在客户/合投用户资料

**参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 搜索姓名、实际购买人、微信昵称 |
| industry | string | 否 | 行业过滤 |
| is_dedicated | string | 否 | 是否专户客户："是"/"否"/"all" |
| is_competitor | string | 否 | 是否竞品群："是"/"否"/"all" |

**返回**:

```json
{
  "count": 5,
  "customers": [
    {
      "user_name": "张三",
      "actual_buyer": "张三",
      "phone": "138...",
      "wechat": "zhangsan",
      "total_assets": 500,
      "risk_tolerance": "积极型",
      "industry": "互联网",
      "is_actual_deal": "是",
      "lead_source": "朋友推荐",
      "is_dedicated_account": "否",
      "is_competitor": "否"
    }
  ]
}
```

**实现**: 扩展 `db.js` 的 `queryCoInvestUsers`，增加 keyword 在 user_name/actual_buyer/wechat 上的模糊搜索

---

### 2. `get_customer_products` - 客户持仓查询

**用途**: 查询指定客户关联的所有产品及持仓状态

**参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| customer_name | string | 是 | 实际购买人姓名 |

**返回**:

```json
{
  "customer_name": "张三",
  "count": 3,
  "products": [
    {
      "product_id": "A001",
      "product_name": "XX 1号",
      "subscribe_amount": 100,
      "outstanding_amount": 80,
      "holding_status": "存续",
      "issue_date": "2025-01-15",
      "manager": "某私募"
    }
  ]
}
```

**实现**: 关联 `customer_product_link` 和 `products` 表查询

---

### 3. `get_customer_peak_analysis` - 客户峰值分析

**用途**: 分析客户的存续余额峰值与当前差额

**参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| customer_name | string | 否 | 实际购买人，不传返回全部 |

**返回**:

```json
{
  "count": 1,
  "analyses": [
    {
      "customer_name": "张三",
      "peak_balance": 500,
      "current_outstanding": 300,
      "peak_diff": 200
    }
  ]
}
```

**实现**: 调用已有 `computeUserPeakBalances()`，按 customer_name 过滤

---

### 4. `query_transactions` - 交易记录查询

**用途**: 查询交易流水，支持按产品、对手、日期过滤

**参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| product_id | string | 否 | 航班编号过滤 |
| counterparty | string | 否 | 交易对手关键词 |
| start_date | string | 否 | 开始日期 YYYY-MM-DD |
| end_date | string | 否 | 结束日期 YYYY-MM-DD |

**返回**:

```json
{
  "count": 10,
  "total_amount": 5000,
  "transactions": [
    {
      "transaction_date": "2026-03-15",
      "flight_id": "A001",
      "counterparty": "XX证券",
      "subscribe_amount": 500
    }
  ]
}
```

**实现**: 新增 `queryTransactions` 函数到 `db.js`

---

### 5. `get_product_analytics` - 产品聚合统计

**用途**: 按维度聚合分析产品分布

**参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_by | string | 是 | 分组维度: manager / holding_status / structure_type / issue_month |

**返回**:

```json
{
  "group_by": "manager",
  "groups": [
    {
      "key": "某私募",
      "count": 15,
      "total_subscribe": 3000,
      "total_outstanding": 1200,
      "active_count": 8
    }
  ]
}
```

**实现**: 新增聚合 SQL 查询

---

### 6. `get_posters` - 喜报查询

**用途**: 按产品或日期查询喜报数据

**参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| product_id | string | 否 | 产品 ID 过滤 |
| observation_date | string | 否 | 观察日 YYYY-MM-DD |

**返回**:

```json
{
  "count": 3,
  "posters": [
    {
      "product_id": "A001",
      "product_name": "XX 1号",
      "poster_type": "knockout",
      "observation_date": "2026-06-23",
      "absolute_return": 0.15,
      "annualized_return": 0.22,
      "dividend_count": 3,
      "cumulative_rate": 0.045
    }
  ]
}
```

**实现**: 基于已有 `queryPostersByDate`/`queryPostersByProduct`/`queryAllPosters`

---

### 7. `search_product_docs` - 产品文档搜索

**用途**: 搜索产品库文档内容

**参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 文档名/内容关键词 |
| month | string | 否 | 月份过滤（如 "6月"、"202606"） |

**返回**:

```json
{
  "count": 2,
  "docs": [
    {
      "doc_name": "XX产品说明",
      "parent_path": "产品库 / 2026年 / 6月",
      "raw_content": "结构：FCN...",
      "structure": {
        "结构": "FCN",
        "标的": "沪深300",
        "期限": "12M"
      }
    }
  ]
}
```

**实现**: 基于 `getAllProductDocs`/`getProductDocsByMonth` + 关键词过滤（不依赖 RAG 的 searchDocs，而是直接返回文档给 LLM）

---

### 8. `get_channels_summary` - 渠道来源分布

**用途**: 查看客户来源渠道的分布情况

**参数**: 无

**返回**:

```json
{
  "channels": {
    "count": 12,
    "items": [
      { "channel_name": "XX银行", "id": 1 }
    ]
  },
  "direct_customer_sources": {
    "count": 8,
    "items": [
      { "source_name": "某券商渠道", "id": 1 }
    ]
  }
}
```

**实现**: 新增 `getAllChannels` 和 `getAllDirectCustomerSources` 函数到 `db.js`

---

### 9. `get_sync_status` - 数据同步状态

**用途**: 查看各数据源的最近同步时间和状态

**参数**: 无

**返回**:

```json
{
  "sources": [
    {
      "name": "产品+交易",
      "last_sync": "2026-06-13T08:00:00Z",
      "row_count": 500
    },
    {
      "name": "合投用户",
      "last_sync": "2026-06-12T10:00:00Z",
      "row_count": 120
    },
    {
      "name": "产品文档",
      "last_sync": "2026-06-10T15:00:00Z",
      "doc_count": 30,
      "folder_count": 5
    }
  ]
}
```

**实现**: 组合调用 `getLastSync`, `getLastCoInvestSync`, `getLastProductDocsSync`

---

### 10. `get_activity_logs` - 操作日志查询

**用途**: 查看系统操作历史

**参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 否 | 日志类型过滤（如 "sync"） |
| limit | integer | 否 | 返回条数，默认 20 |

**返回**:

```json
{
  "count": 5,
  "logs": [
    {
      "id": 100,
      "type": "sync",
      "action": "Transaction table synced",
      "detail": "500 rows",
      "createdAt": "2026-06-13T08:00:00Z"
    }
  ]
}
```

**实现**: 调用已有 `queryActivityLogs`

---

## 实现约束

### 文件修改范围

| 文件 | 修改内容 |
|------|---------|
| `backend/services/agentTools.js` | 新增 10 个工具定义 + 执行函数 |
| `backend/db.js` | 新增 `queryTransactions`, `getAllChannels`, `getAllDirectCustomerSources` |

### 不需要修改的部分

- `agentService.js` - 无需修改，自动从 `TOOL_DEFINITIONS` 读取工具
- `documentRetriever.js` - RAG 逻辑保持不变
- 前端 - 无需修改（Agent 聊天 UI 已通用）

### 测试策略

扩展现有 `agentTools.test.js`：
- 每个工具添加单元测试
- 覆盖正常路径 + 空结果 + 错误输入

---

## 后续规划

当前设计基于现有技术栈（Node.js/Express + SQLite）实现。待功能稳定后，计划进行技术栈迁移：
- 前端：Vue 3 + TypeScript
- 后端：Go（独立服务或 gRPC 架构）

迁移时这些工具的定义和 API 契约可直接作为跨语言接口规范使用。
