<template>
  <div class="logs">
    <h1 class="page-title">日志</h1>
    <div class="actions">
      <input v-model="filter" placeholder="筛选..." />
      <select v-model="level">
        <option value="">全部级别</option>
        <option>INFO</option><option>WARN</option><option>ERROR</option>
      </select>
      <button class="btn" @click="logs.list.length = 0">清空显示</button>
      <button class="btn" :class="{ primary: follow }" @click="follow = !follow">{{ follow ? '✓ 自动跟随' : '自动跟随' }}</button>
    </div>
    <div class="log-pane" ref="pane">
      <div v-for="(e, i) in filtered" :key="i" :class="['line', 'lv-' + e.level]">
        <span class="t">{{ e.time }}</span>
        <span class="l">{{ e.level }}</span>
        <span class="m">{{ e.msg }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, computed, watch, nextTick } from 'vue'
const logs = inject('logs')
const filter = ref('')
const level = ref('')
const follow = ref(true)
const pane = ref(null)

const filtered = computed(() => {
  return logs.list.filter(e =>
    (!level.value || e.level === level.value) &&
    (!filter.value || e.msg.toLowerCase().includes(filter.value.toLowerCase()))
  )
})

watch(() => logs.list.length, async () => {
  if (!follow.value) return
  await nextTick()
  if (pane.value) pane.value.scrollTop = pane.value.scrollHeight
})
</script>

<style scoped>
.page-title {
  font-size: 22px; font-weight: 700; margin: 0 0 18px;
  background: var(--header-grad);
  -webkit-background-clip: text; background-clip: text; color: transparent;
}
.actions { display: flex; gap: 8px; margin-bottom: 10px; }
.actions input { width: 220px; }
.log-pane {
  background: #1e1e1e; color: #d4d4d4;
  font-family: Consolas, monospace; font-size: 12px;
  padding: 12px;
  border-radius: var(--radius);
  height: calc(100vh - 180px);
  overflow: auto;
  user-select: text;
}
.line { padding: 1px 0; white-space: pre-wrap; word-break: break-all; }
.line .t { color: #6a9955; margin-right: 6px; }
.line .l { color: #569cd6; margin-right: 6px; font-weight: bold; min-width: 50px; display: inline-block; }
.line.lv-WARN .l { color: #d7ba7d; }
.line.lv-ERROR .l { color: #f48771; }
.line.lv-ERROR .m { color: #f48771; }
</style>
