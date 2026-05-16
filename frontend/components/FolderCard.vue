<template>
  <div class="folder-card-wrapper">
    <div
      class="folder-card"
      :class="{ 'is-folder': item.type === 'folder', 'is-file': item.type !== 'folder' }"
      :style="{ marginLeft: depth * 20 + 'px' }"
    >
      <div class="card-icon">
        <span v-if="item.type === 'folder'" class="icon-folder">📁</span>
        <span v-else class="icon-file">📄</span>
      </div>
      <div class="card-content">
        <div class="card-name">{{ item.name }}</div>
        <div class="card-meta">
          <span v-if="item.type !== 'folder'" class="file-type">{{ getFileType(item.name) }}</span>
          <span v-if="item.type === 'folder'" class="item-count">{{ item.count || 0 }} 项</span>
        </div>
      </div>
      <div class="card-actions" v-if="item.type === 'folder'">
        <button
          v-if="!expanded || loadingChildren"
          class="btn-expand"
          @click="expand"
          :disabled="loadingChildren"
        >
          {{ loadingChildren ? '加载中...' : '展开' }}
        </button>
        <button v-else class="btn-expand" @click="collapse">收起</button>
      </div>
    </div>

    <div v-if="expanded && childItems.length > 0" class="children">
      <FolderCard
        v-for="child in childItems"
        :key="child.token"
        :item="child"
        :depth="depth + 1"
      />
    </div>

    <div v-if="expanded && childItems.length === 0 && !loadingChildren && item.type === 'folder'" class="empty-children" :style="{ marginLeft: (depth + 1) * 20 + 'px' }">
      文件夹为空
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  item: {
    type: Object,
    required: true
  },
  depth: {
    type: Number,
    default: 0
  }
})

const expanded = ref(false)
const loadingChildren = ref(false)
const childItems = ref([])

function getFileType(name) {
  if (!name) return ''
  const ext = name.split('.').pop()?.toLowerCase()
  const typeMap = {
    'xlsx': 'Excel',
    'xls': 'Excel',
    'csv': 'CSV',
    'pdf': 'PDF',
    'doc': 'Word',
    'docx': 'Word',
    'ppt': 'PPT',
    'pptx': 'PPT',
    'txt': '文本'
  }
  return typeMap[ext] || ext
}

async function expand() {
  if (childItems.value.length > 0) {
    expanded.value = true
    return
  }
  loadingChildren.value = true
  try {
    const res = await fetch(`/api/drive/files?folder_token=${props.item.token}`)
    if (!res.ok) throw new Error('获取失败')
    const data = await res.json()
    childItems.value = data.files || []
    expanded.value = true
  } catch (e) {
    console.error('加载子文件夹失败:', e)
    childItems.value = []
    expanded.value = true
  } finally {
    loadingChildren.value = false
  }
}

function collapse() {
  expanded.value = false
}
</script>

<style scoped>
.folder-card-wrapper {
  margin-bottom: 4px;
}

.folder-card {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border: 1px solid #E8DDD0;
  border-radius: 8px;
  transition: box-shadow 0.2s, border-color 0.2s;
}

.folder-card:hover {
  border-color: #D97757;
  box-shadow: 0 2px 8px rgba(217, 119, 87, 0.1);
}

.folder-card.is-folder {
  background: #FFFAF7;
}

.card-icon {
  font-size: 20px;
  margin-right: 12px;
  flex-shrink: 0;
}

.card-content {
  flex: 1;
  min-width: 0;
}

.card-name {
  font-size: 14px;
  font-weight: 500;
  color: #1A1109;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  display: flex;
  gap: 8px;
  margin-top: 4px;
  font-size: 12px;
  color: #8C7B6E;
}

.file-type {
  background: #F5F0E8;
  padding: 1px 6px;
  border-radius: 4px;
}

.item-count {
  color: #A8967E;
}

.card-actions {
  margin-left: 12px;
  flex-shrink: 0;
}

.btn-expand {
  background: transparent;
  border: 1px solid #D6C9BB;
  color: #6B5C4E;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-expand:hover:not(:disabled) {
  background: #EFE9DF;
  border-color: #D97757;
  color: #D97757;
}

.btn-expand:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.children {
  margin-top: 4px;
}

.empty-children {
  color: #A8967E;
  font-size: 13px;
  padding: 8px 16px;
}
</style>
