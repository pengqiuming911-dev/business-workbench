<template>
  <SubPageLayout
    title="名义购买人 × 私募管理人"
    description="一次输入多个名义购买人，查看每个人分别在哪些私募管理人处购买过产品。"
  >
    <div class="section">
      <p class="desc">一次输入多个名义购买人，查看每个人分别在哪些私募管理人处买过产品，仅展示姓名与私募管理人名称。</p>

      <div class="panel">
        <h3 class="panel-title">输入查询</h3>
        <div class="form-row align-top">
          <label>名义购买人<br/><span class="hint">（每行一个）</span></label>
          <textarea v-model="names" rows="6" placeholder="请输入名义购买人姓名，每行一个" class="textarea" />
        </div>
        <div class="form-row">
          <label>数据文件</label>
          <input v-model="filePath" type="text" placeholder="请输入交易表_修正后.xlsx 路径" class="input" />
        </div>
        <button class="btn btn-primary" @click="query">查询</button>
      </div>

      <div v-if="result.length" class="result-panel">
        <h3 class="panel-title">查询结果</h3>
        <table class="table">
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
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

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
