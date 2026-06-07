<template>
  <Teleport to="body">
    <div v-if="open" class="search-overlay" @mousedown.self="close" @keydown.esc="close">
      <div class="search-modal">
        <p>Search modal placeholder (full implementation in Task 3)</p>
        <button @click="close">Close</button>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
defineProps({
  open: { type: Boolean, default: false }
})
const emit = defineEmits(['update:open'])
function close() {
  emit('update:open', false)
}
</script>

<style scoped>
.search-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
  z-index: 200;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 20vh;
}
.search-modal {
  background: white;
  padding: 20px;
  border-radius: 10px;
  min-width: 400px;
}
</style>