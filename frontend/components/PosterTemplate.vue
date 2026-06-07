<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import html2canvas from 'html2canvas'

const props = defineProps({
  posterType: { type: String, default: 'knockout' },
  data: { type: Object, required: true },
})

const emit = defineEmits(['generated'])

const posterRef = ref(null)
const canvasRef = ref(null)
const isGenerating = ref(false)

const templateSrc = computed(() => {
  const base = window.location.port === '5173' ? 'http://localhost:3001' : ''
  const path = props.posterType === 'knockout'
    ? '/public/喜报/敲出喜报.png'
    : '/public/喜报/分红喜报.png'
  return `${base}${path}`
})

function formatChineseDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (Number.isNaN(d.getTime())) return String(dateStr)
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
}

function trimFixed(value, decimals = 2) {
  const text = Number(value).toFixed(decimals)
  return text.replace(/\.?0+$/, '')
}

function formatPercent(value, decimals = 2) {
  if (value === null || value === undefined || value === '') return '--'
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return '--'
  return trimFixed(numeric * 100, decimals)
}

function formatPlainPercent(value) {
  if (value === null || value === undefined || value === '') return '--'
  const text = String(value).trim()
  return text.endsWith('%') ? text : `${text}%`
}

function formatKnockoutPercent(value) {
  if (value === null || value === undefined || value === '') return '--'
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return '--'
  return trimFixed(numeric, 2)
}

function maskProductName(name) {
  if (!name) return ''
  const text = String(name).trim()
  const match = text.match(/^(.)(.*?)(\d+号.*)$/)
  if (!match) return text
  return `${match[1]}*${match[3]}`
}

const productName = computed(() => maskProductName(props.data.product_name))
const displayDate = computed(() => props.data.date_display || formatChineseDate(props.data.observation_date))
const entryDate = computed(() => formatChineseDate(props.data.entry_date))
const durationMonths = computed(() => props.data.months_since_entry || props.data.duration_months || 0)
const dividendCount = computed(() => props.data.dividend_count || 0)

async function generateImage() {
  if (!posterRef.value) return
  isGenerating.value = true

  try {
    await nextTick()
    const canvas = await html2canvas(posterRef.value, {
      scale: 2,
      backgroundColor: null,
      logging: false,
      useCORS: true,
    })

    canvasRef.value = canvas

    const link = document.createElement('a')
    link.download = `${props.posterType === 'knockout' ? '敲出喜报' : '派息喜报'}_${props.data.product_name || '产品'}_${formatChineseDate(props.data.observation_date)}.png`
    link.href = canvas.toDataURL('image/png')
    link.click()

    emit('generated', canvas)
  } catch (error) {
    console.error('生成喜报图片失败:', error)
  } finally {
    isGenerating.value = false
  }
}

onMounted(() => {
  nextTick(() => {
    if (posterRef.value) generateImage().catch(() => {})
  })
})

defineExpose({
  generateImage,
  canvasRef,
})
</script>

<template>
  <div class="poster-wrapper">
    <div
      ref="posterRef"
      class="poster"
      :class="posterType === 'knockout' ? 'knockout-poster' : 'dividend-poster'"
    >
      <img class="template-image" :src="templateSrc" alt="" />

      <template v-if="posterType === 'knockout'">
        <div class="patch knockout-product-patch"></div>
        <div class="overlay knockout-product">{{ productName }}</div>

        <div class="patch knockout-date-patch"></div>
        <div class="overlay knockout-date">{{ displayDate }} 顺利敲出！</div>

        <div class="patch knockout-absolute-patch"></div>
        <div class="overlay knockout-absolute">
          <span class="big-number absolute-number">{{ formatPercent(data.absolute_return, 2) }}</span>
          <span class="absolute-percent">%</span>
        </div>

        <div class="patch knockout-annual-patch"></div>
        <div class="overlay knockout-annual">
          <span class="annual-number">{{ formatPercent(data.annualized_return, 2) }}</span>
          <span class="annual-percent">%</span>
        </div>

        <div class="patch knockout-duration-patch"></div>
        <div class="overlay knockout-duration">{{ durationMonths }}</div>

        <div class="patch knockout-underlying-patch"></div>
        <div class="overlay knockout-underlying">挂钩标的：{{ data.underlying_name || '--' }}</div>

        <div class="patch knockout-parachute-patch"></div>
        <div class="overlay knockout-parachute">下跌保护界限：{{ formatPlainPercent(data.parachute_value) }}</div>

        <div class="patch knockout-profit-patch"></div>
        <div class="overlay knockout-profit">止盈界限：{{ formatKnockoutPercent(data.knockout_value) }}%</div>

        <div class="patch knockout-entry-patch"></div>
        <div class="overlay knockout-entry">入场时间：{{ entryDate }}</div>
      </template>

      <template v-else>
        <div class="patch dividend-product-patch"></div>
        <div class="overlay dividend-product">热烈祝贺 {{ productName }}</div>

        <div class="patch dividend-date-patch"></div>
        <div class="overlay dividend-date">派息观察日:{{ displayDate }}</div>

        <div class="patch dividend-annual-patch"></div>
        <div class="overlay dividend-annual">
          <span class="dividend-main-number">{{ formatPercent(data.annualized_return, 2) }}</span>
          <span class="dividend-main-percent">%</span>
        </div>

        <div class="patch dividend-count-patch"></div>
        <div class="overlay dividend-count">累计分红 {{ dividendCount }} 次:</div>

        <div class="patch dividend-cumulative-patch"></div>
        <div class="overlay dividend-cumulative">
          <span>{{ formatPercent(data.cumulative_rate, 2) }}</span><span class="dividend-sub-percent">%</span>
        </div>

        <div class="patch dividend-month-patch"></div>
        <div class="overlay dividend-month">
          <span>{{ formatPercent(data.monthly_coupon, 2) }}</span><span class="dividend-sub-percent">%</span>
        </div>

        <div class="patch dividend-underlying-patch"></div>
        <div class="overlay dividend-underlying">挂钩标的: {{ data.underlying_name || '--' }}</div>

        <div class="patch dividend-barrier-patch"></div>
        <div class="overlay dividend-barrier">派息界限: {{ data.dividend_barrier_value || '--' }}</div>

        <div class="patch dividend-profit-patch"></div>
        <div class="overlay dividend-profit">止盈界限: {{ formatKnockoutPercent(data.knockout_value) }}%</div>

        <div class="patch dividend-parachute-patch"></div>
        <div class="overlay dividend-parachute">月末降至: {{ formatPlainPercent(data.parachute_value) }}</div>

        <div class="patch dividend-entry-patch"></div>
        <div class="overlay dividend-entry">入场时间: {{ entryDate }}</div>
      </template>
    </div>

    <div class="poster-actions">
      <button class="btn-download" :disabled="isGenerating" @click="generateImage">
        {{ isGenerating ? '生成中...' : '下载图片' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.poster-wrapper {
  display: inline-block;
  margin: 20px;
}

.poster {
  position: relative;
  width: min(576px, 78vw);
  aspect-ratio: 9 / 16;
  overflow: hidden;
  background: #b50000;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  font-family: "Microsoft YaHei", "SimHei", Arial, sans-serif;
}

.template-image {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.overlay,
.patch {
  position: absolute;
  z-index: 2;
}

.patch {
  z-index: 1;
}

.knockout-poster {
  color: #ffe7a5;
}

.knockout-poster .patch {
  background: rgba(174, 0, 0, 0.92);
}

.knockout-product-patch {
  left: 8%;
  top: 28.1%;
  width: 84%;
  height: 6.8%;
}

.knockout-product {
  left: 8%;
  top: 29.2%;
  width: 84%;
  text-align: center;
  font-size: clamp(25px, 5.2vw, 38px);
  color: #ffe7a5;
  line-height: 1.15;
}

.knockout-date-patch {
  left: 14%;
  top: 35.3%;
  width: 72%;
  height: 4.1%;
}

.knockout-date {
  left: 14%;
  top: 35.7%;
  width: 72%;
  text-align: center;
  font-size: clamp(21px, 4.2vw, 31px);
  color: #ffe7a5;
}

.knockout-absolute-patch {
  left: 12%;
  top: 46.5%;
  width: 48%;
  height: 4.7%;
}

.knockout-absolute {
  left: 28%;
  top: 45.3%;
  width: 34%;
  color: #ffe7a5;
  white-space: nowrap;
}

.absolute-number {
  font-size: clamp(46px, 8.5vw, 72px);
  line-height: 1;
}

.absolute-percent {
  font-size: clamp(9px, 1.6vw, 15px);
  margin-left: 0.18em;
}

.knockout-annual-patch {
  left: 42%;
  top: 53.8%;
  width: 25%;
  height: 3.8%;
}

.knockout-annual {
  left: 42%;
  top: 52.8%;
  width: 25%;
  color: #ffe7a5;
  white-space: nowrap;
}

.annual-number {
  font-size: clamp(38px, 7vw, 58px);
  line-height: 1;
}

.annual-percent {
  font-size: clamp(14px, 2.8vw, 21px);
  margin-left: 0.18em;
}

.knockout-duration-patch {
  left: 51.5%;
  top: 59.8%;
  width: 13%;
  height: 4.7%;
}

.knockout-duration {
  left: 52%;
  top: 59.1%;
  width: 12%;
  text-align: center;
  font-size: clamp(40px, 8.8vw, 70px);
  color: #ffe7a5;
  line-height: 1;
}

.knockout-underlying-patch,
.knockout-parachute-patch,
.knockout-profit-patch,
.knockout-entry-patch {
  left: 9%;
  width: 56%;
  height: 3.1%;
  background: rgba(104, 0, 0, 0.9) !important;
}

.knockout-underlying,
.knockout-parachute,
.knockout-profit,
.knockout-entry {
  left: 10%;
  width: 55%;
  color: #ffe7a5;
  font-size: clamp(17px, 3.4vw, 26px);
  line-height: 1.2;
}

.knockout-underlying-patch { top: 82.4%; }
.knockout-underlying { top: 82.6%; }
.knockout-parachute-patch { top: 86.2%; }
.knockout-parachute { top: 86.4%; }
.knockout-profit-patch { top: 90.1%; }
.knockout-profit { top: 90.3%; }
.knockout-entry-patch { top: 94%; }
.knockout-entry { top: 94.2%; }

.dividend-poster {
  color: #d60000;
}

.dividend-poster .patch {
  background: #f5f1e9;
}

.dividend-product-patch {
  left: 12%;
  top: 27.2%;
  width: 77%;
  height: 5.9%;
}

.dividend-product {
  left: 12%;
  top: 27.7%;
  width: 77%;
  text-align: center;
  font-size: clamp(25px, 5.2vw, 39px);
  color: #000;
  line-height: 1.2;
}

.dividend-date-patch {
  left: 25%;
  top: 34.1%;
  width: 63%;
  height: 3.7%;
}

.dividend-date {
  left: 25%;
  top: 34.5%;
  width: 63%;
  font-size: clamp(20px, 4vw, 31px);
  color: #000;
  line-height: 1.2;
}

.dividend-annual-patch {
  left: 25%;
  top: 48.2%;
  width: 60%;
  height: 12%;
}

.dividend-annual {
  left: 25%;
  top: 50%;
  width: 60%;
  color: #d60000;
  text-align: center;
  white-space: nowrap;
}

.dividend-main-number {
  font-size: clamp(76px, 16vw, 126px);
  line-height: 1;
}

.dividend-main-percent {
  font-size: clamp(27px, 5.8vw, 45px);
  margin-left: 0.12em;
}

.dividend-count-patch {
  left: 10%;
  top: 67.4%;
  width: 37%;
  height: 4.1%;
}

.dividend-count {
  left: 10%;
  top: 67.8%;
  width: 37%;
  font-size: clamp(18px, 3.8vw, 29px);
  color: #000;
}

.dividend-cumulative-patch {
  left: 8%;
  top: 73%;
  width: 38%;
  height: 7.4%;
}

.dividend-cumulative {
  left: 8%;
  top: 73.2%;
  width: 38%;
  color: #d60000;
  font-size: clamp(64px, 13vw, 104px);
  line-height: 1;
  white-space: nowrap;
}

.dividend-month-patch {
  left: 57%;
  top: 73%;
  width: 34%;
  height: 7.4%;
}

.dividend-month {
  left: 57%;
  top: 73.2%;
  width: 34%;
  color: #d60000;
  font-size: clamp(64px, 13vw, 104px);
  line-height: 1;
  white-space: nowrap;
}

.dividend-sub-percent {
  font-size: clamp(22px, 4.6vw, 36px);
  margin-left: 0.1em;
}

.dividend-underlying-patch,
.dividend-barrier-patch,
.dividend-profit-patch,
.dividend-parachute-patch,
.dividend-entry-patch {
  left: 10%;
  width: 55%;
  height: 2.5%;
}

.dividend-underlying,
.dividend-barrier,
.dividend-profit,
.dividend-parachute,
.dividend-entry {
  left: 10%;
  width: 55%;
  color: #000;
  font-size: clamp(14px, 3vw, 22px);
  line-height: 1.2;
}

.dividend-underlying-patch { top: 81%; }
.dividend-underlying { top: 81.1%; }
.dividend-barrier-patch { top: 83.2%; }
.dividend-barrier { top: 83.3%; }
.dividend-profit-patch { top: 85.5%; }
.dividend-profit { top: 85.6%; }
.dividend-parachute-patch { top: 87.8%; }
.dividend-parachute { top: 87.9%; }
.dividend-entry-patch { top: 90.1%; }
.dividend-entry { top: 90.2%; }

.poster-actions {
  margin-top: 12px;
  text-align: center;
}

.btn-download {
  padding: 10px 22px;
  background: #b91c1c;
  color: #fff;
  border: 1px solid #991b1b;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.btn-download:hover:not(:disabled) {
  background: #991b1b;
}

.btn-download:disabled {
  background: #9ca3af;
  border-color: #9ca3af;
  cursor: not-allowed;
}
</style>
