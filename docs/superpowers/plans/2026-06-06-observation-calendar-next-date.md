# Observation Calendar Next Date Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a next observation date field to the all-products table and add a month-filterable observation calendar tab.

**Architecture:** Reuse the existing observation schedule calculation in `backend/services/observationService.js`. The API adds `next_observation_date` per product; the Vue page renders that field and aggregates product observations into a month calendar.

**Tech Stack:** Node.js, Express, Vue 3, Vite.

---

### Task 1: Backend Schedule Helper

**Files:**
- Modify: `backend/services/observationService.js`
- Test: `backend/services/observationService.test.js`

- [ ] Add tests for `getNextObservationDate(product, todayOverride)`.
- [ ] Verify tests fail before implementation.
- [ ] Implement `getNextObservationDate` by checking future monthly schedule dates after `todayOverride`.
- [ ] Run `node backend/services/observationService.test.js`.

### Task 2: API Field

**Files:**
- Modify: `backend/index.js`

- [ ] Import `getNextObservationDate`.
- [ ] Include `next_observation_date` in `/api/observations` and `/api/observations/today` product payloads.
- [ ] Run `node --check backend/index.js`.

### Task 3: Frontend Calendar

**Files:**
- Modify: `frontend/views/ProductCompletion.vue`

- [ ] Add `观察日历` tab between `全量` and `今日观察`.
- [ ] Add `下个观察日` column after `最近观察日`.
- [ ] Add month input and calendar grid computed from `products`.
- [ ] Include historical observations and `next_observation_date` in month aggregation.
- [ ] Run `npm run build` in `frontend`.

### Self-Review

- The plan covers the requested next date field and the observation calendar tab.
- No new database table is needed.
- No placeholder tasks remain.
