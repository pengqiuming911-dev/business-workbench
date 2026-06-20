# 观察日历卡片紧凑显示设计

**日期**: 2026-06-20  
**状态**: 已确认  
**范围**: 前端 UI 微调，无后端改动

## 目标

在观察日历页面（`/product-completion` → 观察日历 tab）中，将日历卡片内原本分开显示的"敲出"行和"今日/当日点位"行合并为一行紧凑展示。

## 当前行为

每个 `cal-card` 内最多渲染 3 个独立行：

```
敲出 1.025
派息 0.950
今日 1.038
```

- 存续（ongoing）：标签为"今日"，显示实时点位
- 已完结（completed）：标签为"当日"，显示观察日收盘价

敲出和点位各占一行，浪费卡片垂直空间。

## 目标行为

敲出与点位合并为一行，派息保持独立行：

**存续（ongoing）**：
```
敲出 1.025 今日 1.038
派息 0.950
```

**已完结（completed）**：
```
敲出 1.025 当日 1.038
派息 0.950
```

### 显示规则

| 条件 | 敲出+点位行 | 派息行 |
|------|------------|--------|
| 有敲出价且有可观察敲出 | 显示 | 有派息观察时显示 |
| 无敲出价或不可观察 | 仅显示点位（今日/当日） | 有派息观察时显示 |
| 无点位数据 | 仅显示敲出 | 有派息观察时显示 |
| 两者都无 | 不显示该行 | 不显示 |

## 改动范围

### 前端

**文件**: `frontend/views/ProductCompletion.vue`

**模板变更** — 将 cal-card 中原来独立的 `cal-detail-knockout` 和 `cal-detail-spot` 两个 div 合并为一个 `cal-detail-row`：

```html
<!-- 敲出 + 点位合并行 -->
<div v-if="(product.is_knockout_observable && product.knockout_price != null) || product.spot_price != null"
     class="cal-detail-row cal-detail-knockout-spot">
  <template v-if="product.is_knockout_observable && product.knockout_price != null">
    <span class="cal-detail-label">敲出</span>
    <strong>{{ fmtCalPrice(product.knockout_price) }}</strong>
  </template>
  <template v-if="product.spot_price != null">
    <span class="cal-detail-label">{{ calendarStatus === 'completed' ? '当日' : '今日' }}</span>
    <strong>{{ fmtCalPrice(product.spot_price) }}</strong>
  </template>
</div>

<!-- 派息保持独立行（不变） -->
<div v-if="product.has_dividend_observation && product.dividend_line != null"
     class="cal-detail-row cal-detail-dividend">
  <span class="cal-detail-label">派息</span>
  <strong>{{ fmtCalPrice(product.dividend_line) }}</strong>
</div>
```

**样式**: 为 `cal-detail-knockout-spot` 添加少量间距，确保两组 label-value 之间有适当间隔：

```css
.cal-detail-knockout-spot strong + .cal-detail-label {
  margin-left: 0.5em;
}
```

### 后端

无需改动。`/api/observations/calendar` 已返回 `knockout_price` 和 `spot_price` 字段。

## 不做的事

- 不修改后端 API 或数据模型
- 不改变状态筛选逻辑（已实现）
- 不修改全量 tab、今日观察 tab、喜报 tab
- 不改变点位数据源（存续用实时价格，完结用历史收盘价）
