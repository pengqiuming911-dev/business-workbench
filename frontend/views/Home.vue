<template>
  <WorkbenchLayout
    title="业务工作台"
    description="统一管理航班服务数据准备、客户分析、产品报告、存续观察和渠道统计。"
  >
    <section class="home-hero">
      <div>
        <p class="home-label">Operational Console</p>
        <h2>从数据同步到观察报告，一屏进入核心流程。</h2>
        <p>
          先完成飞书数据同步，再进入用户画像、产品观察、存续分析和渠道分析等业务模块。
        </p>
      </div>
      <button class="btn btn-primary" type="button" @click="openExternal">打开飞书总表</button>
    </section>

    <section class="module-grid" aria-label="业务模块">
      <RouterLink v-for="item in modules" :key="item.path" :to="item.path" class="module-card">
        <span class="module-index">{{ item.index }}</span>
        <strong>{{ item.title }}</strong>
        <em>{{ item.description }}</em>
      </RouterLink>
    </section>

    <section class="report-panel">
      <h3 class="panel-title">建议流程</h3>
      <div class="flow">
        <RouterLink v-for="item in workflow" :key="item.path" :to="item.path" class="flow-card">
          <span class="flow-num">{{ item.index }}</span>
          <span class="flow-name">{{ item.title }}</span>
        </RouterLink>
      </div>
    </section>
  </WorkbenchLayout>
</template>

<script setup>
import { RouterLink } from 'vue-router'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'

const modules = [
  { index: '01', title: '数据准备', description: '同步飞书总表和合投用户表', path: '/data-preparation' },
  { index: '02', title: '用户画像', description: '按购买人、专户、竞品和行业筛选用户', path: '/user-profile' },
  { index: '03', title: '客户流失', description: '生成完结未复购客户分析', path: '/customer-churn' },
  { index: '04', title: '产品报告', description: '同步和查看产品运行材料', path: '/product-report' },
  { index: '05', title: '派息/敲出观察', description: '跟踪存续产品观察日和喜报', path: '/product-completion' },
  { index: '06', title: '存续分析', description: '分析存续产品金额、人数和类型', path: '/ongoing-product' },
  { index: '07', title: '渠道分析', description: '统计渠道成交人数、金额和复购表现', path: '/channel-analysis' },
  { index: '08', title: '名义购买人', description: '查询名义购买人与私募管理人关系', path: '/nominal-buyer' },
]

const workflow = modules.slice(0, 7)

function openExternal() {
  alert('请在飞书中打开“航班服务交易总表”，导出或同步后再继续分析。')
}
</script>

<style scoped>
.home-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 24px;
  align-items: end;
  margin-bottom: 24px;
  padding: 28px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--surface);
}

.home-label {
  margin: 0 0 12px;
  color: var(--brand);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.home-hero h2 {
  max-width: 760px;
  margin: 0;
  color: var(--ink-strong);
  font-size: clamp(28px, 3.2vw, 44px);
  font-weight: 800;
  line-height: 1.12;
}

.home-hero p:last-child {
  max-width: 720px;
  margin: 14px 0 0;
  color: var(--ink-soft);
  line-height: 1.8;
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 16px;
}

.module-card {
  display: grid;
  gap: 8px;
  min-height: 150px;
  padding: 18px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--surface);
  transition: border-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
}

.module-card:hover {
  transform: translateY(-2px);
  border-color: var(--brand);
  box-shadow: var(--shadow-soft);
}

.module-index {
  color: var(--brand);
  font-family: var(--mono);
  font-size: 12px;
  font-weight: 800;
}

.module-card strong {
  color: var(--ink-strong);
  font-size: 18px;
}

.module-card em {
  color: var(--ink-soft);
  font-size: 13px;
  font-style: normal;
  line-height: 1.6;
}

.flow {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.flow-card {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--ink);
  background: #ffffff;
}

.flow-card:hover {
  border-color: var(--brand);
  color: var(--brand);
}

.flow-num {
  color: var(--brand);
  font-size: 11px;
  font-weight: 800;
}

.flow-name {
  font-size: 13px;
  font-weight: 700;
}

@media (max-width: 720px) {
  .home-hero {
    grid-template-columns: 1fr;
    padding: 20px;
  }
}
</style>
