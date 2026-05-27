<template>
  <div>
    <h1 class="page-title">PHP 扩展</h1>
    <div class="card" v-if="!status.php.binInstalled">
      <p>PHP 未安装. 请先到首页安装 PHP.</p>
    </div>
    <template v-else>
      <div class="actions">
        <input v-model="filter" placeholder="筛选扩展名..." style="width: 220px" />
        <button class="btn" @click="refresh">刷新</button>
        <button class="btn primary" @click="restartPhp">应用 (重启 PHP-CGI)</button>
        <span class="muted">总 {{ exts.length }} 个, 已启用 {{ enabledCount }}</span>
      </div>
      <div class="ext-grid">
        <label v-for="e in filtered" :key="e.name" class="ext-item" :class="{ enabled: e.enabled }">
          <input type="checkbox" :checked="e.enabled" @change="toggle(e, $event.target.checked)" />
          <span class="name">{{ e.name }}</span>
          <span v-if="e.enabled" class="badge">已启用</span>
        </label>
      </div>
    </template>
  </div>
</template>

<script setup>
import { inject, ref, onMounted, computed } from 'vue'
const status = inject('status')
const api = inject('api')

const exts = ref([])
const filter = ref('')

async function refresh() {
  exts.value = await api.PhpExtensions() || []
}
onMounted(refresh)

const filtered = computed(() => {
  const f = filter.value.toLowerCase()
  if (!f) return exts.value
  return exts.value.filter(e => e.name.includes(f))
})

const enabledCount = computed(() => exts.value.filter(e => e.enabled).length)

async function toggle(ext, enabled) {
  try {
    await api.PhpSetExtension(ext.name, enabled)
    ext.enabled = enabled
  } catch (e) {
    alert('修改失败: ' + e)
    ext.enabled = !enabled
  }
}

async function restartPhp() {
  await api.RestartService('php')
}
</script>

<style scoped>
.page-title {
  font-size: 22px; font-weight: 700; margin: 0 0 18px;
  background: var(--header-grad);
  -webkit-background-clip: text; background-clip: text; color: transparent;
}
.actions { display: flex; gap: 8px; align-items: center; margin-bottom: 16px; }
.muted { color: var(--text-secondary); font-size: 12px; margin-left: auto; }
.ext-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 8px; }
.ext-item {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 12px; background: #fff; border: 1px solid var(--border);
  border-radius: 6px; cursor: pointer; transition: all 0.15s;
}
.ext-item:hover { border-color: var(--primary); }
.ext-item.enabled { background: rgba(60,170,60,0.05); border-color: rgba(60,170,60,0.3); }
.ext-item .name { font-family: Consolas, monospace; font-size: 13px; flex: 1; }
.badge { font-size: 10px; color: var(--success); padding: 2px 6px; background: rgba(60,170,60,0.1); border-radius: 8px; }
</style>
