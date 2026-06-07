<template>
  <WorkbenchLayout>
    <h1 class="text-page-title">存续产品分析</h1>
    <p class="text-body" style="margin-bottom: 24px;">指定开始/结束年月，查看进场时间分布、每月进场金额、交易人次与人数、新客/增购/复购分布。</p>

    <PanelCard title="参数设置">
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
      <span v-if="errorMsg" class="error-msg" style="margin-left: 12px;">{{ errorMsg }}</span>
    </PanelCard>

    <template v-if="result">
      <PanelCard title="总览">
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
      </PanelCard>

      <PanelCard title="1. 进场时间分布">
        <p class="text-label" style="margin-bottom: 14px;">统计区间内按进场（航班日期）年月的产品数量、金额、人次。</p>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>年月</th>
                <th style="text-align: right;">产品数量</th>
                <th style="text-align: right;">金额（万元）</th>
                <th style="text-align: right;">人次</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in result.monthRows" :key="row.ym">
                <td>{{ row.ym }}</td>
                <td style="text-align: right;">{{ row.prods }}</td>
                <td style="text-align: right;">{{ fmt(row.amt) }}</td>
                <td style="text-align: right;">{{ row.visits }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="total-row">
                <td>合计</td>
                <td style="text-align: right;">{{ result.totProds }}</td>
                <td style="text-align: right;">{{ fmt(result.totAmt) }}</td>
                <td style="text-align: right;">{{ result.totVisits }}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </PanelCard>

      <PanelCard title="2. 交易人次与人数">
        <p class="text-label" style="margin-bottom: 14px;">人次为每月交易笔数，人数为每月参与的唯一客户数（按姓名去重）。</p>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>年月</th>
                <th style="text-align: right;">人次</th>
                <th style="text-align: right;">人数（去重）</th>
                <th style="text-align: right;">人均笔数</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in result.monthRows" :key="row.ym">
                <td>{{ row.ym }}</td>
                <td style="text-align: right;">{{ row.visits }}</td>
                <td style="text-align: right;">{{ row.people }}</td>
                <td style="text-align: right;">{{ row.people ? (row.visits / row.people).toFixed(1) : '-' }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="total-row">
                <td>合计</td>
                <td style="text-align: right;">{{ result.totVisits }}</td>
                <td style="text-align: right;">{{ result.totPeople }}</td>
                <td style="text-align: right;">{{ result.totPeople ? (result.totVisits / result.totPeople).toFixed(1) : '-' }}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </PanelCard>

      <PanelCard title="3. 新客 / 增购 / 复购分布">
        <p class="text-label" style="margin-bottom: 14px;">按交易类型统计金额与笔数。</p>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>类型</th>
                <th style="text-align: right;">笔数</th>
                <th style="text-align: right;">金额（万元）</th>
                <th style="text-align: right;">金额占比</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in result.typeRows" :key="row.type">
                <td>{{ row.type }}</td>
                <td style="text-align: right;">{{ row.cnt }}</td>
                <td style="text-align: right;">{{ fmt(row.amt) }}</td>
                <td style="text-align: right;">{{ result.totAmt ? (row.amt / result.totAmt * 100).toFixed(1) + '%' : '-' }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="total-row">
                <td>合计</td>
                <td style="text-align: right;">{{ result.totVisits }}</td>
                <td style="text-align: right;">{{ fmt(result.totAmt) }}</td>
                <td style="text-align: right;">100%</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </PanelCard>
    </template>
  </WorkbenchLayout>
</template>

<script setup>
import { ref } from 'vue'
import * as XLSX from 'xlsx'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import PanelCard from '../components/PanelCard.vue'

const startMonth = ref('')
const endMonth = ref('')
const fileName = ref('')
const fileLoaded = ref(false)
const loading = ref(false)
const errorMsg = ref('')
const result = ref(null)

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
      const sheetName = wb.SheetNames.includes('交易表') ? '交易表' : wb.SheetNames[0]
      const raw = XLSX.utils.sheetTo_json(wb.Sheets[sheetName], { header: 1, defval: null })
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
          r[1] || '',
          r[3] || '',
          r[6] || 0,
          r[7] || '',
          r[21] || '',
          year,
          month,
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
:deep(.workbench-main) {
  max-width: none;
}

.file-label {
  flex: 1;
  display: flex;
  align-items: center;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 12px;
  font-size: 13px;
  cursor: pointer;
  color: var(--ink-soft);
  background: #fff;
  transition: border-color 0.2s;
}

.file-label:hover {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.file-input { display: none; }

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-top: 14px;
}

.s-card {
  background: var(--surface-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  padding: 14px 16px;
  text-align: center;
}

.s-val { font-size: 24px; font-weight: 700; color: var(--brand); }
.s-lbl { font-size: 12px; color: var(--ink-soft); margin-top: 4px; }

.total-row td {
  font-weight: 700;
  border-top: 2px solid var(--border-soft);
  border-bottom: none;
  background: var(--surface-muted);
}

@media (max-width: 720px) {
  .summary-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
