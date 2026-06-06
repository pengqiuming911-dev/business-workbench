<template>
  <SubPageLayout
    title="客户流失分析"
    description="生成客户存量峰值分析报告，识别已完结未复购客户，并为后续 Excel 导出预留入口。"
  >
    <div class="section">
      <p class="desc">生成客户存量峰值分析报告，识别已完结未复购客户，输出 Excel；报告可直接在页面查看。</p>

      <div class="panel">
        <h3 class="panel-title">生成分析报告</h3>
        <div class="form-row">
          <label>分析截止日期</label>
          <input v-model="endDate" type="date" class="input" />
        </div>
        <div class="form-row">
          <label>数据文件路径</label>
          <input v-model="filePath" type="text" placeholder="请输入交易表_修正后.xlsx 路径" class="input" />
        </div>
        <div class="actions">
          <button class="btn btn-primary" @click="generate">生成报告</button>
          <button class="btn btn-outline" @click="download">下载 Excel</button>
          <button class="btn btn-outline" @click="showReport = !showReport">{{ showReport ? '隐藏报告' : '查看报告' }}</button>
        </div>
      </div>

      <div v-if="showReport" class="report-panel">
        <h3 class="panel-title">报告预览</h3>
        <p class="placeholder">报告内容将在接入后端后展示。</p>
      </div>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

const endDate = ref('')
const filePath = ref('')
const showReport = ref(false)

function generate() { alert('生成报告功能尚未接入后端。') }
function download() { alert('下载 Excel 功能尚未接入后端。') }
</script>

<style scoped>
.placeholder {
  padding: 40px 0;
  text-align: center;
}
</style>
