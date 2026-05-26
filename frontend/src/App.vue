<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-title">WinPHP <span class="brand-accent">2025</span></div>
        <div class="brand-sub">PHP / MySQL / Nginx / PG</div>
      </div>

      <nav class="nav">
        <a v-for="item in navItems" :key="item.key"
           :class="{ active: view === item.key }"
           @click="view = item.key">
          <span class="ico">{{ item.icon }}</span>
          <span>{{ item.label }}</span>
        </a>
      </nav>

      <div class="sidebar-foot">
        <div class="bulk-btns">
          <button class="btn primary" @click="api.StartAll()">全部启动</button>
          <button class="btn" @click="api.StopAll()">全部停止</button>
        </div>
        <div class="admin-tag" :class="{ ok: status.isAdmin }">
          {{ status.isAdmin ? '✓ 管理员' : '⚠ 非管理员 (部分功能受限)' }}
        </div>
      </div>
    </aside>

    <main class="content">
      <HomeView v-if="view === 'home'" :status="status" />
      <SitesView v-if="view === 'sites'" />
      <DatabaseView v-if="view === 'database'" :status="status" />
      <ExtensionsView v-if="view === 'extensions'" :status="status" />
      <AutoStartView v-if="view === 'autostart'" />
      <ToolsView v-if="view === 'tools'" />
      <LogsView v-if="view === 'logs'" />
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, reactive, provide } from 'vue'
import HomeView from './views/HomeView.vue'
import SitesView from './views/SitesView.vue'
import DatabaseView from './views/DatabaseView.vue'
import ExtensionsView from './views/ExtensionsView.vue'
import AutoStartView from './views/AutoStartView.vue'
import ToolsView from './views/ToolsView.vue'
import LogsView from './views/LogsView.vue'

const view = ref('home')
const navItems = [
  { key: 'home',       label: '首页',     icon: '⌂' },
  { key: 'sites',      label: '网站',     icon: '⌥' },
  { key: 'database',   label: '数据库',   icon: '⛁' },
  { key: 'extensions', label: 'PHP 扩展', icon: '⚙' },
  { key: 'autostart',  label: '自启动',   icon: '⚡' },
  { key: 'tools',      label: '工具',     icon: '⚒' },
  { key: 'logs',       label: '日志',     icon: '📋' }
]

// 通过 window.go.main.App.* 调用后端
const api = window.go?.main?.App || {}
provide('api', api)
provide('runtime', window.runtime || {})

const status = reactive({
  nginx: { running: false, version: '', port: 80 },
  php: { running: false, version: '', port: 9000 },
  mysql: { running: false, version: '', port: 3306 },
  postgres: { running: false, version: '', port: 5432 },
  isAdmin: false,
  panelAutoStart: false
})
provide('status', status)

let unsubStatus
let unsubLog
const logs = reactive({ list: [] })
provide('logs', logs)

async function refreshAll() {
  if (!api.AllStatus) return
  const r = await api.AllStatus()
  Object.assign(status, r)
}

onMounted(async () => {
  await refreshAll()
  // 推送事件: 状态 + 日志
  if (window.runtime?.EventsOn) {
    unsubStatus = window.runtime.EventsOn('status', (s) => {
      Object.assign(status, s)
    })
    unsubLog = window.runtime.EventsOn('log', (e) => {
      logs.list.push(e)
      if (logs.list.length > 800) logs.list.splice(0, logs.list.length - 800)
    })
  }
  if (api.LogTail) {
    const tail = await api.LogTail(200)
    if (tail) logs.list = tail
  }
})

onUnmounted(() => {
  if (unsubStatus) unsubStatus()
  if (unsubLog) unsubLog()
})
</script>

<style>
.layout {
  display: flex; height: 100vh; overflow: hidden;
}
.sidebar {
  width: 220px; background: #1f2937;
  display: flex; flex-direction: column;
  color: #e6e9ee;
}
.brand {
  padding: 22px 20px 18px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}
.brand-title { font-size: 18px; font-weight: 600; }
.brand-accent { color: #5fa9ff; }
.brand-sub { font-size: 11px; color: #8b95a3; margin-top: 4px; }

.nav { flex: 1; padding: 12px 8px; }
.nav a {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 14px; border-radius: 6px;
  color: #c1c7d0; cursor: pointer;
  margin-bottom: 2px;
  font-size: 14px;
}
.nav a:hover { background: rgba(255,255,255,0.05); color: #fff; }
.nav a.active { background: var(--primary); color: #fff; }
.nav .ico { width: 18px; text-align: center; font-size: 15px; }

.sidebar-foot { padding: 14px; border-top: 1px solid rgba(255,255,255,0.06); }
.bulk-btns { display: flex; gap: 6px; margin-bottom: 10px; }
.bulk-btns .btn { flex: 1; padding: 7px 0; font-size: 12px; }
.bulk-btns .btn:not(.primary) { background: rgba(255,255,255,0.05); border-color: rgba(255,255,255,0.12); color: #c1c7d0; }
.bulk-btns .btn:not(.primary):hover { background: rgba(255,255,255,0.1); color: #fff; }
.admin-tag {
  font-size: 11px; color: #e8a83c;
  text-align: center;
  padding: 4px 6px; background: rgba(232,168,60,0.1); border-radius: 4px;
}
.admin-tag.ok { color: var(--success); background: rgba(60,170,60,0.1); }

.content {
  flex: 1; overflow: auto;
  padding: 24px;
}
</style>
