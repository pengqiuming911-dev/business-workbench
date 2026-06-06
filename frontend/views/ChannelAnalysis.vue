<template>
  <SubPageLayout
    title="渠道分析"
    description="按最终渠道统计各渠道的成交人数、成交金额与复购表现。"
  >
    <div class="section">
      <p class="desc">按「最终渠道」（交易表「渠道」列 / 客户表渠道名称）统计各渠道的成交人数、成交金额与复购表现。</p>

      <div class="panel">
        <h3 class="panel-title">参数设置</h3>
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
      </div>

      <div class="report-panel">
        <h3 class="panel-title">分析结果</h3>
        <p class="placeholder">各渠道数据将在接入后端后展示。</p>
      </div>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

const startMonth = ref('')
const endMonth = ref('')
const filePath = ref('')

function run() { alert('渠道分析功能尚未接入后端。') }
</script>

<style scoped>
.placeholder {
  padding: 40px 0;
  text-align: center;
}
</style>
