<template>
  <Teleport to="body">
    <Transition name="fade">
      <div v-if="open" class="search-overlay" @mousedown.self="close">
        <div class="search-modal" role="dialog" aria-label="全局搜索">
          <div class="search-input-wrap">
            <Search :size="18" class="search-icon" />
            <input
              ref="inputRef"
              v-model="query"
              type="text"
              class="search-input"
              placeholder="搜索客户、产品..."
              @keydown.escape="close"
              @keydown.down.prevent="moveDown"
              @keydown.up.prevent="moveUp"
              @keydown.enter.prevent="selectCurrent"
            />
            <kbd class="search-esc">ESC</kbd>
          </div>

          <div v-if="loading" class="search-status">搜索中...</div>
          <div v-else-if="query && results.length === 0" class="search-status">暂无结果</div>

          <div v-else-if="!query && recents.length > 0" class="search-section">
            <p class="section-label">最近访问</p>
            <div
              v-for="(r, i) in recents"
              :key="r.path"
              class="search-item"
              :class="{ active: activeIndex === i }"
              @click="navigateTo(r)"
            >
              <span class="item-type">最近</span>
              <span class="item-name">{{ r.title }}</span>
            </div>
          </div>

          <div v-else-if="query" class="search-results">
            <template v-for="(items, type) in groupedResults" :key="type">
              <p class="section-label">{{ typeLabel(type) }}</p>
              <div
                v-for="r in items"
                :key="r.type + r.id"
                class="search-item"
                :class="{ active: activeIndex === flatIndex(r) }"
                @click="navigateTo(r)"
              >
                <span class="item-type" :class="'type-' + r.type">{{ typeLabel(r.type) }}</span>
                <span class="item-name">{{ r.name }}</span>
              </div>
            </template>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { Search } from '@lucide/vue'

const props = defineProps({
  open: { type: Boolean, default: false }
})
const emit = defineEmits(['update:open'])

const router = useRouter()
const inputRef = ref(null)
const query = ref('')
const results = ref([])
const loading = ref(false)
const activeIndex = ref(0)
const recents = ref([])
let debounceTimer = null
let abortController = null

const groupedResults = computed(() => {
  const groups = {}
  for (const r of results.value) {
    if (!groups[r.type]) groups[r.type] = []
    groups[r.type].push(r)
  }
  return groups
})

function flatIndex(item) {
  return results.value.findIndex(r => r.type === item.type && r.id === item.id)
}

function close() {
  emit('update:open', false)
  query.value = ''
  results.value = []
  activeIndex.value = 0
}

function moveDown() {
  const max = query.value ? results.value.length : recents.value.length
  if (max > 0) activeIndex.value = Math.min(activeIndex.value + 1, max - 1)
}

function moveUp() {
  activeIndex.value = Math.max(activeIndex.value - 1, 0)
}

function selectCurrent() {
  const list = query.value ? results.value : recents.value
  const item = list[activeIndex.value]
  if (item) navigateTo(item)
}

function navigateTo(item) {
  const path = item.path || '/'
  router.push(path)
  saveRecent(item)
  close()
}

function typeLabel(t) {
  return { customer: '客户', product: '产品', channel: '渠道' }[t] || t
}

function saveRecent(item) {
  const entry = { path: item.path || '/', title: item.name || item.title || '' }
  const list = recents.value.filter(r => r.path !== entry.path)
  list.unshift(entry)
  const trimmed = list.slice(0, 5)
  recents.value = trimmed
  try { localStorage.setItem('bw-search-recents', JSON.stringify(trimmed)) } catch {}
}

function loadRecents() {
  try {
    const raw = localStorage.getItem('bw-search-recents')
    recents.value = raw ? JSON.parse(raw) : []
  } catch {
    recents.value = []
  }
}

watch(query, (q) => {
  activeIndex.value = 0
  if (debounceTimer) clearTimeout(debounceTimer)
  if (abortController) abortController.abort()
  if (!q || q.trim().length === 0) {
    results.value = []
    loading.value = false
    return
  }
  loading.value = true
  debounceTimer = setTimeout(async () => {
    abortController = new AbortController()
    try {
      const res = await fetch(`/api/search?q=${encodeURIComponent(q.trim())}`, {
        signal: abortController.signal
      })
      const data = await res.json()
      results.value = data.results || []
    } catch (err) {
      if (err.name !== 'AbortError') {
        results.value = []
      }
    } finally {
      loading.value = false
    }
  }, 300)
})

watch(() => props.open, async (isOpen) => {
  if (isOpen) {
    loadRecents()
    await nextTick()
    inputRef.value?.focus()
  }
})

onMounted(loadRecents)

onBeforeUnmount(() => {
  if (debounceTimer) clearTimeout(debounceTimer)
  if (abortController) abortController.abort()
})
</script>

<style scoped>
.search-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 15vh;
}

.search-modal {
  width: 100%;
  max-width: 520px;
  background: #fff;
  border-radius: 14px;
  box-shadow: var(--shadow-lg);
  overflow: hidden;
  border: 1px solid var(--border-soft);
}

.search-input-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-soft);
}

.search-icon { color: var(--ink-soft); flex-shrink: 0; }

.search-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 16px;
  color: var(--ink-strong);
  background: transparent;
}
.search-input::placeholder { color: var(--ink-faint); }

.search-esc {
  font-family: var(--font-sans);
  font-size: 11px;
  font-weight: 600;
  padding: 2px 6px;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--ink-soft);
  background: var(--bg-hover);
}

.search-status {
  padding: 24px;
  text-align: center;
  color: var(--ink-soft);
  font-size: 13px;
}

.search-section { padding: 8px 0; }

.section-label {
  padding: 6px 16px;
  font-size: 11px;
  font-weight: 700;
  color: var(--ink-soft);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.search-results { padding: 8px 0; max-height: 320px; overflow-y: auto; }

.search-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 150ms;
}
.search-item:hover, .search-item.active { background: var(--bg-hover); }

.item-type {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 20px;
  background: var(--bg-active);
  color: var(--ink-soft);
  flex-shrink: 0;
}
.item-type.type-customer { background: var(--success-soft); color: var(--success); }
.item-type.type-product { background: var(--brand-soft); color: var(--brand); }
.item-type.type-channel { background: var(--warning-soft); color: #92400e; }

.item-name {
  font-size: 14px;
  color: var(--ink-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
