<template>
  <WorkbenchLayout>
    <h1 class="text-page-title">客户流失分析</h1>
    <p class="text-body">生成客户存量峰值分析报告，识别已完结未复购客户，并为后续 Excel 导出预留入口。</p>

    <PanelCard title="生成分析报告">
      <div class="form-row">
        <label>分析截止日期</label>
        <input v-model="endDate" type="date" class="input" />
      </div>
      <div class="form-row">
        <label>数据文件路径</label>
        <input v-model="filePath" type="text" placeholder="请输入交易表_修正后.xlsx 路径" class="input" />
      </div>
      <div class="toolbar actions">
        <button class="btn btn-primary" @click="generate">生成报告</button>
        <button class="btn btn-outline" @click="download">下载 Excel</button>
        <button class="btn btn-outline" @click="showReport = !showReport">{{ showReport ? '隐藏报告' : '查看报告' }}</button>
      </div>
    </PanelCard>

    <PanelCard v-if="showReport" title="报告预览">
      <p class="empty-state">报告内容将在接入后端后展示。</p>
    </PanelCard>
  </WorkbenchLayout>
</template>

<script setup>
import { ref } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import PanelCard from '../components/PanelCard.vue'

const endDate = ref('')
const filePath = ref('')
const showReport = ref(false)

function generate() { alert('生成报告功能尚未接入后端。') }
function download() { alert('下载 Excel 功能尚未接入后端。') }
</script>

<style scoped>
</style>
