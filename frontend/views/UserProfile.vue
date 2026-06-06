<template>
  <SubPageLayout
    title="用户画像"
    description="查询合投用户画像，支持按实际购买人、名义购买人、专户客户、竞品客户和行业筛选。"
    wide
  >
    <div class="section">
      <p class="desc">查询合投用户画像，支持按实际购买人、名义购买人、是否专户客户、客户是否竞品群、客户行业等条件筛选。</p>

      <!-- 搜索面板 -->
      <div class="panel">
        <h3 class="panel-title">搜索条件</h3>
        <div class="form-grid">
          <div class="form-row">
            <label>实际购买人</label>
            <input v-model="filters.actual_buyer" type="text" placeholder="模糊匹配" class="input" @keyup.enter="search" />
          </div>
          <div class="form-row">
            <label>名义购买人</label>
            <input v-model="filters.nominal_buyer" type="text" placeholder="模糊匹配" class="input" @keyup.enter="search" />
          </div>
          <div class="form-row">
            <label>是否专户客户</label>
            <select v-model="filters.is_dedicated" class="input">
              <option value="">全部</option>
              <option value="是">是</option>
              <option value="否">否</option>
            </select>
          </div>
          <div class="form-row">
            <label>客户是否竞品群</label>
            <select v-model="filters.is_competitor" class="input">
              <option value="">全部</option>
              <option value="是">是</option>
              <option value="否">否</option>
            </select>
          </div>
          <div class="form-row">
            <label>客户行业</label>
            <select v-model="filters.industry" class="input">
              <option value="">全部</option>
              <option v-for="ind in industries" :key="ind" :value="ind">{{ ind }}</option>
            </select>
          </div>
        </div>
        <div class="search-actions">
          <button class="btn btn-primary" :disabled="loading" @click="search">
            {{ loading ? '查询中...' : '搜索' }}
          </button>
          <button class="btn btn-outline" @click="reset">重置</button>
          <span v-if="errorMsg" class="error">{{ errorMsg }}</span>
          <span class="result-count" v-if="!loading && rows.length > 0">共 {{ rows.length }} 条</span>
        </div>
      </div>

      <!-- 结果表格 -->
      <div v-if="rows.length" class="panel table-panel">
        <table class="table">
          <thead>
            <tr>
              <th>实际购买人</th>
              <th>名义购买人</th>
              <th>客户是否竞品群</th>
              <th>是否专户客户</th>
              <th>衍选成交前购买过结构化产品</th>
              <th>境内资产规模区间/万RMB</th>
              <th>微信昵称</th>
              <th>手机号</th>
              <th>风险承受</th>
              <th>历史存量峰值</th>
              <th>峰值差额</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in rows" :key="i">
              <td>{{ row.actual_buyer || '-' }}</td>
              <td>{{ row.nominal_buyer || '-' }}</td>
              <td>
                <span class="badge" :class="row.is_competitor === '是' ? 'badge-red' : 'badge-green'">
                  {{ row.is_competitor || '-' }}
                </span>
              </td>
              <td>
                <span class="badge" :class="row.is_dedicated_account === '是' ? 'badge-red' : 'badge-green'">
                  {{ row.is_dedicated_account || '-' }}
                </span>
              </td>
              <td>{{ row.bought_before_yanxuan || '-' }}</td>
              <td>{{ row.asset_range || '-' }}</td>
              <td>{{ row.wechat || '-' }}</td>
              <td>{{ row.phone || '-' }}</td>
              <td>{{ row.risk_tolerance || '-' }}</td>
              <td>
                <span v-if="row.peak_balance != null">{{ fmtAmt(row.peak_balance) }}</span>
                <span v-else class="pending">待接入</span>
              </td>
              <td>
                <span v-if="row.peak_diff != null" :class="row.peak_diff >= 0 ? 'positive' : 'negative'">
                  {{ row.peak_diff >= 0 ? '+' : '' }}{{ fmtAmt(row.peak_diff) }}
                </span>
                <span v-else class="pending">待接入</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else-if="!loading && searched" class="empty">
        未找到匹配的用户，请调整搜索条件
      </div>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

const filters = ref({
  actual_buyer: '',
  nominal_buyer: '',
  is_dedicated: '',
  is_competitor: '',
  industry: ''
})

const rows = ref([])
const industries = ref([])
const loading = ref(false)
const errorMsg = ref('')
const searched = ref(false)

onMounted(async () => {
  await loadIndustries()
  await search()
})

async function loadIndustries() {
  try {
    const res = await fetch('/api/db/industries')
    const data = await res.json()
    industries.value = data.rows || []
  } catch {}
}

async function search() {
  loading.value = true
  errorMsg.value = ''
  try {
    const params = new URLSearchParams()
    if (filters.value.actual_buyer) params.set('actual_buyer', filters.value.actual_buyer)
    if (filters.value.nominal_buyer) params.set('nominal_buyer', filters.value.nominal_buyer)
    if (filters.value.is_dedicated) params.set('is_dedicated', filters.value.is_dedicated)
    if (filters.value.is_competitor) params.set('is_competitor', filters.value.is_competitor)
    if (filters.value.industry) params.set('industry', filters.value.industry)

    const res = await fetch(`/api/db/user-profiles?${params}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '查询失败')
    rows.value = data.rows || []
    searched.value = true
  } catch (e) {
    errorMsg.value = e.message
  } finally {
    loading.value = false
  }
}

function reset() {
  filters.value = { actual_buyer: '', nominal_buyer: '', is_dedicated: '', is_competitor: '', industry: '' }
  search()
}

function fmtAmt(v) {
  if (v == null) return '-'
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
</script>

<style scoped>
.table-panel {
  overflow-x: auto;
}

.positive {
  color: var(--success);
  font-weight: 700;
}

.negative {
  color: var(--danger);
  font-weight: 700;
}
</style>
