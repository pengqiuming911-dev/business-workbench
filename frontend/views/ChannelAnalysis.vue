<template>
  <SubPageLayout title="渠道分析">
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
.desc { color: #6B5C4E; font-size: 14px; line-height: 1.8; margin-bottom: 24px; }
.panel { background: #fff; border-radius: 12px; padding: 24px; margin-bottom: 20px; border: 1px solid #E8DDD0; }
.panel-title { font-size: 16px; font-weight: 600; margin-bottom: 16px; }
.form-row { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.form-row label { font-size: 13px; white-space: nowrap; width: 90px; }
.input { flex: 1; border: 1px solid #E8DDD0; border-radius: 6px; padding: 8px 12px; font-size: 13px; outline: none; background: #fff; }
.input:focus { border-color: #D97757; }
.btn { padding: 8px 20px; border-radius: 6px; font-size: 13px; cursor: pointer; border: none; }
.btn-primary { background: #D97757; color: #fff; }
.report-panel { background: #fff; border-radius: 12px; padding: 24px; border: 1px solid #E8DDD0; }
.placeholder { color: #aaa; font-size: 14px; padding: 40px 0; text-align: center; }
</style>
