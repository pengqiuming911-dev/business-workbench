<template>
  <div class="subpage">
    <header class="header">
      <div class="header-brand">
        <h1 class="title">业务工作台</h1>
        <p class="subtitle">航班服务 · 数据分析平台</p>
      </div>

      <nav class="nav">
        <router-link to="/data-preparation" class="nav-link">数据准备</router-link>
        <router-link to="/user-profile" class="nav-link">用户画像</router-link>
        <router-link to="/customer-churn" class="nav-link">客户流失</router-link>
        <router-link to="/product-report" class="nav-link">产品报告</router-link>
        <router-link to="/product-completion" class="nav-link">完结与发行</router-link>
        <router-link to="/ongoing-product" class="nav-link">存续分析</router-link>
        <router-link to="/channel-analysis" class="nav-link">渠道分析</router-link>
        <router-link to="/nominal-buyer" class="nav-link">名义购买人</router-link>
      </nav>

      <div class="header-actions">
        <button class="logout-btn">退出登录</button>
      </div>
    </header>

    <div class="breadcrumb">
      <router-link to="/" class="back-link">← 返回工作台</router-link>
      <span class="sep">/</span>
      <span class="current">{{ title }}</span>
    </div>

    <main class="content">
      <h2 class="page-title">{{ title }}</h2>
      <slot />
    </main>
  </div>
</template>

<script setup>
defineProps({ title: String })
</script>

<style scoped>
.subpage {
  min-height: 100vh;
  background: #F5F0E8;
}

/* ─── Header ─── */
.header {
  background: #FEFCFA;
  border-bottom: 1px solid #E8DDD0;
  padding: 0 28px;
  height: 58px;
  display: flex;
  align-items: center;
}

.header-brand {
  flex-shrink: 0;
  margin-right: 28px;
}

.title {
  font-size: 16px;
  font-weight: 700;
  color: #1A1109;
  line-height: 1;
}

.subtitle {
  font-size: 11px;
  color: #A8967E;
  margin-top: 3px;
}

/* ─── Nav ─── */
.nav {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: 1;
}

.nav-link {
  font-size: 13.5px;
  color: #6B5C4E;
  padding: 6px 11px;
  border-radius: 6px;
  transition: color 0.15s, background 0.15s;
  white-space: nowrap;
}

.nav-link:hover {
  color: #1A1109;
  background: #EFE9DF;
}

.nav-link.router-link-exact-active {
  color: #D97757;
  background: #FDF0E8;
  font-weight: 500;
}

/* ─── Header Actions ─── */
.header-actions {
  flex-shrink: 0;
  margin-left: 16px;
}

.logout-btn {
  background: transparent;
  color: #8C7B6E;
  border: 1px solid #D6C9BB;
  padding: 6px 13px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.15s;
}

.logout-btn:hover { background: #EFE9DF; }

/* ─── Breadcrumb ─── */
.breadcrumb {
  background: #F5F0E8;
  border-bottom: 1px solid #E8DDD0;
  padding: 0 28px;
  height: 38px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.back-link {
  color: #D97757;
  font-weight: 500;
}

.back-link:hover { text-decoration: underline; }

.sep { color: #C4B5A5; }

.current { color: #1A1109; }

/* ─── Content ─── */
.content {
  padding: 32px;
  max-width: 1100px;
}

.page-title {
  font-size: 22px;
  font-weight: 700;
  color: #1A1109;
  margin-bottom: 24px;
}

@media (max-width: 1100px) {
  .nav-link { font-size: 12.5px; padding: 6px 9px; }
}

@media (max-width: 860px) {
  .nav { display: none; }
}
</style>
