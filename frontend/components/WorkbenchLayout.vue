<template>
  <div class="workbench-shell">
    <header class="workbench-topbar">
      <RouterLink to="/" class="workbench-brand" aria-label="返回工作台首页" @click="closeSidebar">
        <span class="brand-mark">BW</span>
        <span>
          <strong>业务工作台</strong>
          <em>航班服务 · 数据分析平台</em>
        </span>
      </RouterLink>

      <nav class="topbar-links" aria-label="顶部导航">
        <RouterLink to="/" class="topbar-link">首页</RouterLink>
        <RouterLink to="/product-completion" class="topbar-link">观察</RouterLink>
        <RouterLink to="/product-report" class="topbar-link">报告</RouterLink>
      </nav>

      <div class="topbar-actions">
        <button class="btn btn-secondary btn-sm" type="button" @click="openFeishu">
          打开飞书总表
        </button>
        <button
          class="icon-menu"
          type="button"
          :aria-expanded="sidebarOpen"
          aria-controls="workbench-sidebar"
          aria-label="切换导航"
          @click="sidebarOpen = !sidebarOpen"
        >
          <span></span>
          <span></span>
          <span></span>
          <span class="menu-text">菜单</span>
        </button>
      </div>
    </header>

    <div class="workbench-body">
      <aside id="workbench-sidebar" class="workbench-sidebar" :class="{ open: sidebarOpen }">
        <div class="sidebar-section">
          <p class="sidebar-kicker">Modules</p>
          <RouterLink
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            class="sidebar-link"
            @click="closeSidebar"
          >
            <span class="sidebar-index">{{ item.index }}</span>
            <span>
              <strong>{{ item.title }}</strong>
              <em>{{ item.description }}</em>
            </span>
          </RouterLink>
        </div>
      </aside>

      <main class="workbench-main" :class="{ wide }">
        <div class="page-heading">
          <p class="page-kicker">Business Workbench</p>
          <h1>{{ title }}</h1>
          <p v-if="description" class="page-description">{{ description }}</p>
        </div>
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { RouterLink } from 'vue-router'

defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
  wide: { type: Boolean, default: false },
})

const sidebarOpen = ref(false)

const navItems = [
  { index: '01', title: '数据准备', description: '同步飞书与本地数据', path: '/data-preparation' },
  { index: '02', title: '用户画像', description: '查询合投用户特征', path: '/user-profile' },
  { index: '03', title: '客户流失', description: '识别完结未复购客户', path: '/customer-churn' },
  { index: '04', title: '产品报告', description: '查看产品运行材料', path: '/product-report' },
  { index: '05', title: '派息/敲出观察', description: '跟踪存续产品观察日', path: '/product-completion' },
  { index: '06', title: '存续分析', description: '分析仍在持有产品', path: '/ongoing-product' },
  { index: '07', title: '渠道分析', description: '统计渠道成交表现', path: '/channel-analysis' },
  { index: '08', title: '名义购买人', description: '匹配私募管理人', path: '/nominal-buyer' },
]

function closeSidebar() {
  sidebarOpen.value = false
}

function openFeishu() {
  alert('请在飞书中打开“航班服务交易总表”。')
}
</script>
