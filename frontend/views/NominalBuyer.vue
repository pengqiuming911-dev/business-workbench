<template>
  <WorkbenchLayout>
    <h1 class="text-page-title">名义购买人 × 私募管理人</h1>
    <p class="text-body">一次输入多个名义购买人，查看每个人分别在哪些私募管理人处购买过产品。</p>

    <PanelCard title="输入查询">
      <div class="form-row align-top">
        <label>名义购买人<br/><span class="hint">（每行一个）</span></label>
        <textarea v-model="names" rows="6" placeholder="请输入名义购买人姓名，每行一个" class="textarea" />
      </div>
      <div class="form-row">
        <label>数据文件</label>
        <input v-model="filePath" type="text" placeholder="请输入交易表_修正后.xlsx 路径" class="input" />
      </div>
      <button class="btn btn-primary" @click="query">查询</button>
    </PanelCard>

    <PanelCard v-if="result.length" title="查询结果">
      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>名义购买人</th>
              <th>私募管理人</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in result" :key="i">
              <td>{{ row.buyer }}</td>
              <td>{{ row.manager }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </PanelCard>
  </WorkbenchLayout>
</template>

<script setup>
import { ref } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import PanelCard from '../components/PanelCard.vue'

const names = ref('')
const filePath = ref('')
const result = ref([])

function query() {
  alert('查询功能尚未接入后端。')
}
</script>

<style scoped>
.hint {
  color: var(--ink-soft);
  font-size: 11px;
}
</style>
