# 观察日历卡片紧凑显示 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge the separate knockout and spot price rows into a single compact line in the observation calendar cards.

**Architecture:** Pure frontend template + CSS change in `ProductCompletion.vue`. No backend changes — the API already returns all needed fields (`knockout_price`, `spot_price`).

**Tech Stack:** Vue 3 template, scoped CSS

## Global Constraints

- Only modify `frontend/views/ProductCompletion.vue`
- Preserve existing dividend row as-is (independent row)
- Maintain existing color scheme: knockout blue (#2563a8/#edf6ff), spot purple (#6b5b95/#f3f0fb)
- Label "今日" for ongoing, "当日" for completed (existing logic)

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `frontend/views/ProductCompletion.vue:170-183` | Merge knockout + spot template into one row |
| Modify | `frontend/views/ProductCompletion.vue:888-936` | Add CSS for combined row, remove unused `.cal-detail-spot` class |

---

### Task 1: Merge knockout and spot rows in template + add CSS

**Files:**
- Modify: `frontend/views/ProductCompletion.vue:170-183` (template)
- Modify: `frontend/views/ProductCompletion.vue:888-936` (CSS)

**Step 1: Replace the 3 separate `cal-detail-row` divs with merged knockout+spot row + independent dividend row**

Replace lines 172–183 with:

```html
                  <div v-if="(product.is_knockout_observable && product.knockout_price != null) || product.spot_price != null" class="cal-detail-row cal-detail-knockout-spot">
                    <template v-if="product.is_knockout_observable && product.knockout_price != null">
                      <span class="cal-detail-label">敲出</span>
                      <strong>{{ fmtCalPrice(product.knockout_price) }}</strong>
                    </template>
                    <template v-if="product.spot_price != null">
                      <span class="cal-detail-label">{{ calendarStatus === 'completed' ? '当日' : '今日' }}</span>
                      <strong>{{ fmtCalPrice(product.spot_price) }}</strong>
                    </template>
                  </div>
                  <div v-if="product.has_dividend_observation && product.dividend_line != null" class="cal-detail-row cal-detail-dividend">
                    <span class="cal-detail-label">派息</span>
                    <strong>{{ fmtCalPrice(product.dividend_line) }}</strong>
                  </div>
```

**Step 2: Add CSS for combined row and clean up unused classes**

Replace lines 888–936 (`.cal-detail-row` through `.cal-detail-spot strong`) with:

```css
.cal-detail-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  min-height: 24px;
  padding: 3px 7px;
  border-radius: 6px;
  font-size: 11px;
}

.cal-detail-label {
  white-space: nowrap;
  font-weight: 700;
}

.cal-detail-row strong {
  font-weight: 700;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.cal-detail-knockout-spot {
  color: #2563a8;
  background: #edf6ff;
  flex-wrap: wrap;
}

.cal-detail-knockout-spot strong {
  color: #1d4f8a;
}

.cal-detail-knockout-spot .cal-detail-label + strong + .cal-detail-label {
  margin-left: 4px;
  color: #6b5b95;
}

.cal-detail-knockout-spot .cal-detail-label + strong + .cal-detail-label + strong {
  color: #4c3f73;
}

.cal-detail-dividend {
  color: #16806a;
  background: #edfbf7;
}

.cal-detail-dividend strong {
  color: #116451;
}
```

**Step 3: Run typecheck to verify no template errors**

Run: `npm run typecheck`
Expected: No errors

**Step 4: Visual verification**

Run: `npm run dev`
Expected:
- Open `/product-completion` → 观察日历 tab
- 存续: Each card shows `敲出 <price> 今日 <price>` on one line, `派息 <price>` on its own line below
- 已完结: Each card shows `敲出 <price> 当日 <price>` on one line, `派息 <price>` on its own line below
- Cards with only knockout (no spot) or only spot (no knockout) still render correctly

**Step 5: Commit**

```bash
git add frontend/views/ProductCompletion.vue
git commit -m "feat(observations): 日历卡片敲出+点位紧凑合并显示"
```
