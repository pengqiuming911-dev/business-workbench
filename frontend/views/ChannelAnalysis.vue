<template>
  <WorkbenchLayout>
    <h1 class="text-page-title">渠道分析</h1>
    <p class="text-body">按最终渠道统计各渠道的成交人数、成交金额与复购表现。</p>

    <PanelCard title="参数设置">
      <div class="form-row">
        <label>开始年月</label>
        <input v-model="startMonth" type="month" class="input" />
      </div>
      <div class="form-row">
        <label>结束年月</label>
        <input v-model="endMonth" type="month" class="input" />
      </div>
      <div class="form-row">
        <label>数据文件</label>
        <input v-model="filePath" type="text" placeholder="请输入交易表_修正后.xlsx 路径" class="input" />
      </div>
      <button class="btn btn-primary" @click="run">生成渠道分析</button>
    </PanelCard>

    <PanelCard title="分析结果">
      <p class="empty-state">各渠道数据将在接入后端后展示。</p>
    </PanelCard>
  </WorkbenchLayout>
</template>

<script setup>
import { ref } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import PanelCard from '../components/PanelCard.vue'

const startMonth = ref('')
const endMonth = ref('')
const filePath = ref('')

function run() { alert('渠道分析功能尚未接入后端。') }
</script>

<style scoped>
</style>
