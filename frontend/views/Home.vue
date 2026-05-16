<template>
  <div class="page">
    <!-- Header with nav -->
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
        <button class="btn-feishu" @click="openExternal">打开飞书总表</button>
        <button class="logout-btn">退出登录</button>
      </div>
    </header>

    <!-- Home landing -->
    <main class="main">
      <div class="hero">
        <p class="hero-label">航班服务数据分析平台</p>
        <h2 class="hero-title">业务工作台</h2>
        <p class="hero-desc">通过上方导航进入各分析模块，按以下流程完成数据分析工作。</p>
      </div>

      <div class="flow">
        <div class="flow-step" v-for="(s, i) in steps" :key="i">
          <router-link :to="s.path" class="flow-card">
            <span class="flow-num">0{{ i + 1 }}</span>
            <span class="flow-name">{{ s.name }}</span>
          </router-link>
          <span class="flow-arrow" v-if="i < steps.length - 1">→</span>
        </div>
      </div>

      <p class="feishu-hint">
        开始前，请先点击右上角「打开飞书总表」，在飞书中导出「航班服务交易总表」并保存至项目目录。
      </p>
    </main>
  </div>
</template>

<script setup>
const steps = [
  { name: '数据准备', path: '/data-preparation' },
  { name: '用户画像', path: '/user-profile' },
  { name: '客户流失', path: '/customer-churn' },
  { name: '产品报告', path: '/product-report' },
  { name: '完结与发行', path: '/product-completion' },
  { name: '存续分析', path: '/ongoing-product' },
  { name: '渠道分析', path: '/channel-analysis' },
]

function openExternal() {
  alert('请在飞书中打开「航班服务交易总表」')
}
</script>

<style scoped>
.page {
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
  gap: 0;
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
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  margin-left: 16px;
}

.btn-feishu {
  background: #D97757;
  color: #fff;
  border: none;
  padding: 7px 14px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: opacity 0.15s;
  white-space: nowrap;
}

.btn-feishu:hover { opacity: 0.85; }

.logout-btn {
  background: transparent;
  color: #8C7B6E;
  border: 1px solid #D6C9BB;
  padding: 6px 13px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.15s;
  white-space: nowrap;
}

.logout-btn:hover { background: #EFE9DF; }

/* ─── Landing ─── */
.main {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80px 32px 60px;
  gap: 40px;
}

.hero {
  text-align: center;
}

.hero-label {
  font-size: 12px;
  font-weight: 600;
  color: #D97757;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-bottom: 14px;
}

.hero-title {
  font-size: 36px;
  font-weight: 700;
  color: #1A1109;
  margin-bottom: 14px;
  letter-spacing: -0.5px;
}

.hero-desc {
  font-size: 15px;
  color: #6B5C4E;
  line-height: 1.6;
}

/* ─── Flow ─── */
.flow {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: center;
}

.flow-step {
  display: flex;
  align-items: center;
  gap: 8px;
}

.flow-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  background: #fff;
  border: 1px solid #E8DDD0;
  border-radius: 10px;
  padding: 14px 20px;
  transition: border-color 0.15s, box-shadow 0.15s;
  text-align: center;
  min-width: 88px;
}

.flow-card:hover {
  border-color: #D97757;
  box-shadow: 0 2px 10px rgba(217, 119, 87, 0.12);
}

.flow-num {
  font-size: 10px;
  font-weight: 700;
  color: #D97757;
  letter-spacing: 0.5px;
}

.flow-name {
  font-size: 13px;
  font-weight: 600;
  color: #1A1109;
}

.flow-arrow {
  font-size: 16px;
  color: #C4B5A5;
}

/* ─── Hint ─── */
.feishu-hint {
  font-size: 13px;
  color: #8C7B6E;
  background: #fff;
  border: 1px solid #E8DDD0;
  border-radius: 8px;
  padding: 14px 20px;
  max-width: 560px;
  text-align: center;
  line-height: 1.6;
}

@media (max-width: 1100px) {
  .nav-link { font-size: 12.5px; padding: 6px 9px; }
}

@media (max-width: 860px) {
  .nav { display: none; }
}
</style>
