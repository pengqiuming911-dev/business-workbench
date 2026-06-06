<template>
  <SubPageLayout
    title="存续产品分析"
    description="指定开始和结束年月，查看进场时间、金额、人次、人数及新客/增购/复购分布。"
    wide
  >
    <div class="section">
      <p class="desc">指定开始/结束年月，查看进场时间分布、每月进场金额、交易人次与人数、新客/增购/复购分布。</p>

      <div class="panel">
        <h3 class="panel-title">参数设置</h3>
        <div class="form-row">
          <label>数据文件</label>
          <label class="file-label">
            <span>{{ fileName || '点击选择航班交易服务总表.xlsx' }}</span>
            <input type="file" accept=".xlsx,.xls" @change="onFileChange" class="file-input" />
          </label>
        </div>
        <div class="form-row">
          <label>开始年月</label>
          <input v-model="startMonth" type="month" class="input" />
        </div>
        <div class="form-row">
          <label>结束年月</label>
          <input v-model="endMonth" type="month" class="input" />
        </div>
        <button class="btn btn-primary" :disabled="!fileLoaded || loading" @click="run">
          {{ loading ? '计算中...' : '生成分析' }}
        </button>
        <span v-if="errorMsg" class="error">{{ errorMsg }}</span>
      </div>

      <template v-if="result">
        <!-- 总览 -->
        <div class="report-panel">
          <h3 class="section-title">总览</h3>
          <div class="summary-grid">
            <div class="s-card">
              <div class="s-val">{{ result.totProds }}</div>
              <div class="s-lbl">产品数量（航班）</div>
            </div>
            <div class="s-card">
              <div class="s-val">{{ fmt(result.totAmt) }}</div>
              <div class="s-lbl">总金额（万元）</div>
            </div>
            <div class="s-card">
              <div class="s-val">{{ result.totVisits }}</div>
              <div class="s-lbl">交易人次</div>
            </div>
            <div class="s-card">
              <div class="s-val">{{ result.totPeople }}</div>
              <div class="s-lbl">客户人数</div>
            </div>
          </div>
        </div>

        <!-- 1. 进场时间分布 -->
        <div class="report-panel">
          <h3 class="section-title">1. 进场时间分布</h3>
          <p class="section-desc">统计区间内按进场（航班日期）年月的产品数量、金额、人次。</p>
          <table class="data-table">
            <thead>
              <tr>
                <th>年月</th>
                <th class="num">产品数量</th>
                <th class="num">金额（万元）</th>
                <th class="num">人次</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in result.monthRows" :key="row.ym">
                <td>{{ row.ym }}</td>
                <td class="num">{{ row.prods }}</td>
                <td class="num">{{ fmt(row.amt) }}</td>
                <td class="num">{{ row.visits }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="total-row">
                <td>合计</td>
                <td class="num">{{ result.totProds }}</td>
                <td class="num">{{ fmt(result.totAmt) }}</td>
                <td class="num">{{ result.totVisits }}</td>
              </tr>
            </tfoot>
          </table>
        </div>

        <!-- 2. 交易人次与人数 -->
        <div class="report-panel">
          <h3 class="section-title">2. 交易人次与人数</h3>
          <p class="section-desc">人次为每月交易笔数，人数为每月参与的唯一客户数（按姓名去重）。</p>
          <table class="data-table">
            <thead>
              <tr>
                <th>年月</th>
                <th class="num">人次</th>
                <th class="num">人数（去重）</th>
                <th class="num">人均笔数</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in result.monthRows" :key="row.ym">
                <td>{{ row.ym }}</td>
                <td class="num">{{ row.visits }}</td>
                <td class="num">{{ row.people }}</td>
                <td class="num">{{ row.people ? (row.visits / row.people).toFixed(1) : '-' }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="total-row">
                <td>合计</td>
                <td class="num">{{ result.totVisits }}</td>
                <td class="num">{{ result.totPeople }}</td>
                <td class="num">{{ result.totPeople ? (result.totVisits / result.totPeople).toFixed(1) : '-' }}</td>
              </tr>
            </tfoot>
          </table>
        </div>

        <!-- 3. 新客/增购/复购分布 -->
        <div class="report-panel">
          <h3 class="section-title">3. 新客 / 增购 / 复购分布</h3>
          <p class="section-desc">按交易类型统计金额与笔数。</p>
          <table class="data-table">
            <thead>
              <tr>
                <th>类型</th>
                <th class="num">笔数</th>
                <th class="num">金额（万元）</th>
                <th class="num">金额占比</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in result.typeRows" :key="row.type">
                <td>{{ row.type }}</td>
                <td class="num">{{ row.cnt }}</td>
                <td class="num">{{ fmt(row.amt) }}</td>
                <td class="num">{{ result.totAmt ? (row.amt / result.totAmt * 100).toFixed(1) + '%' : '-' }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="total-row">
                <td>合计</td>
                <td class="num">{{ result.totVisits }}</td>
                <td class="num">{{ fmt(result.totAmt) }}</td>
                <td class="num">100%</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </template>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref } from 'vue'
import * as XLSX from 'xlsx'
import SubPageLayout from '../components/SubPageLayout.vue'

const startMonth = ref('')
const endMonth = ref('')
const fileName = ref('')
const fileLoaded = ref(false)
const loading = ref(false)
const errorMsg = ref('')
const result = ref(null)

// 原始行数据 [航班编号, 姓名, 金额, 类型, 存续状态, year, month]
let rows = []

function onFileChange(e) {
  const file = e.target.files[0]
  if (!file) return
  fileName.value = file.name
  errorMsg.value = ''
  result.value = null
  const reader = new FileReader()
  reader.onload = (ev) => {
    try {
      const wb = XLSX.read(ev.target.result, { type: 'array' })
      // 优先读 '交易表' sheet，否则取第一个
      const sheetName = wb.SheetNames.includes('交易表') ? '交易表' : wb.SheetNames[0]
      const raw = XLSX.utils.sheet_to_json(wb.Sheets[sheetName], { header: 1, defval: null })
      rows = []
      for (let i = 1; i < raw.length; i++) {
        const r = raw[i]
        if (!r[0] || !r[1]) continue
        const flightNum = String(r[1])
        let year = null, month = null
        const m = flightNum.match(/-(\d{2})(\d{2})\d{2}$/)
        if (m) {
          year = 2000 + parseInt(m[1])
          month = parseInt(m[2])
        }
        rows.push([
          r[1] || '',    // 0: 航班编号
          r[3] || '',    // 1: 姓名
          r[6] || 0,     // 2: 金额
          r[7] || '',    // 3: 类型
          r[21] || '',   // 4: 存续状态
          year,          // 5: year
          month,         // 6: month
        ])
      }
      fileLoaded.value = true
    } catch (err) {
      errorMsg.value = '文件解析失败：' + err.message
    }
  }
  reader.readAsArrayBuffer(file)
}

function ymNum(y, m) { return y * 100 + m }
function pad(n) { return n < 10 ? '0' + n : String(n) }

function run() {
  errorMsg.value = ''
  result.value = null
  if (!startMonth.value || !endMonth.value) {
    errorMsg.value = '请选择开始和结束年月'
    return
  }
  loading.value = true

  const [sy, sm] = startMonth.value.split('-').map(Number)
  const [ey, em] = endMonth.value.split('-').map(Number)
  const ymS = ymNum(sy, sm)
  const ymE = ymNum(ey, em)

  const filtered = rows.filter(r =>
    r[4] !== '完结' && r[5] && r[6] && ymNum(r[5], r[6]) >= ymS && ymNum(r[5], r[6]) <= ymE
  )

  if (filtered.length === 0) {
    errorMsg.value = '所选年月范围内无存续数据'
    loading.value = false
    return
  }

  // 按年月聚合
  const MM = {}
  for (const r of filtered) {
    const k = `${r[5]}-${pad(r[6])}`
    if (!MM[k]) MM[k] = { prods: new Set(), amt: 0, visits: 0, names: new Set() }
    MM[k].prods.add(r[0])
    MM[k].amt += Number(r[2])
    MM[k].visits++
    if (r[1]) MM[k].names.add(r[1])
  }
  const monthRows = Object.keys(MM).sort().map(k => ({
    ym: k,
    prods: MM[k].prods.size,
    amt: MM[k].amt,
    visits: MM[k].visits,
    people: MM[k].names.size,
  }))

  // 按类型聚合
  const TM = {}
  for (const r of filtered) {
    const t = r[3] || '未知'
    if (!TM[t]) TM[t] = { cnt: 0, amt: 0 }
    TM[t].cnt++
    TM[t].amt += Number(r[2])
  }
  const typeRows = Object.keys(TM).sort().map(t => ({ type: t, cnt: TM[t].cnt, amt: TM[t].amt }))

  result.value = {
    monthRows,
    typeRows,
    totProds: new Set(filtered.map(r => r[0])).size,
    totAmt: filtered.reduce((s, r) => s + Number(r[2]), 0),
    totVisits: filtered.length,
    totPeople: new Set(filtered.map(r => r[1])).size,
  }

  loading.value = false
}

function fmt(val) {
  return typeof val === 'number'
    ? val.toLocaleString('zh-CN', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
    : val
}
</script>

<style scoped>
.desc { color: #6B5C4E; font-size: 14px; line-height: 1.8; margin-bottom: 24px; }

.panel {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  border: 1px solid #E8DDD0;
}

.panel-title { font-size: 16px; font-weight: 600; margin-bottom: 16px; }

.form-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.form-row > label:first-child {
  font-size: 13px;
  white-space: nowrap;
  width: 90px;
  flex-shrink: 0;
}

.input {
  flex: 1;
  border: 1px solid #E8DDD0;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 13px;
  outline: none;
  background: #fff;
}

.input:focus { border-color: #D97757; }

.file-label {
  flex: 1;
  display: flex;
  align-items: center;
  border: 1px solid #E8DDD0;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 13px;
  cursor: pointer;
  color: #6B5C4E;
  background: #fff;
  transition: border-color 0.2s;
}

.file-label:hover { border-color: #D97757; }
.file-input { display: none; }

.btn {
  padding: 8px 20px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  border: none;
}

.btn-primary { background: #D97757; color: #fff; }
.btn-primary:disabled { background: #E9B99A; cursor: not-allowed; }

.error { margin-left: 12px; color: #e53935; font-size: 13px; }

.report-panel {
  background: #fff;
  border-radius: 12px;
  padding: 28px 28px 20px;
  margin-bottom: 20px;
  border: 1px solid #E8DDD0;
}

.section-title {
  font-size: 17px;
  font-weight: 700;
  color: #D97757;
  margin-bottom: 8px;
}

.section-desc { font-size: 13px; color: #6B5C4E; margin-bottom: 14px; }

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-top: 14px;
}

.s-card {
  background: #F5F0E8;
  border: 1px solid #E8DDD0;
  border-radius: 6px;
  padding: 14px 16px;
  text-align: center;
}

.s-val { font-size: 24px; font-weight: 700; color: #D97757; }
.s-lbl { font-size: 12px; color: #6B5C4E; margin-top: 4px; }

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.data-table th {
  padding: 10px 14px;
  border-bottom: 2px solid #E8DDD0;
  color: #D97757;
  font-weight: 600;
  background: #fff;
  text-align: left;
}

.data-table td {
  padding: 10px 14px;
  border-bottom: 1px solid #EDE5DA;
  color: #D97757;
}

.num { text-align: right; }

.total-row td {
  font-weight: 700;
  border-top: 2px solid #E8DDD0;
  border-bottom: none;
  background: #F5F0E8;
}

/* Workbench theme overrides */
.desc,
.section-desc,
.s-lbl {
  color: var(--ink-soft);
}

.panel,
.report-panel,
.s-card {
  border-color: var(--border-soft);
  border-radius: var(--radius);
  background: var(--surface);
}

.panel-title,
.data-table td {
  color: var(--ink-strong);
}

.section-title,
.s-val,
.data-table th {
  color: var(--brand);
}

.input,
.file-label {
  border-color: var(--border);
  border-radius: var(--radius);
  color: var(--ink);
}

.input:focus,
.file-label:hover {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.btn-primary {
  background: var(--brand);
  color: #fff;
}

.btn-primary:hover:not(:disabled) {
  background: var(--brand-hover);
}

.btn-primary:disabled {
  background: var(--brand);
}

.error {
  color: var(--danger);
}

.s-card,
.total-row td {
  background: var(--surface-muted);
}

.data-table th,
.data-table td,
.total-row td {
  border-color: var(--border-soft);
}
</style>
