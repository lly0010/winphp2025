<template>
  <!-- 壁纸层 (有自定义壁纸时显示) -->
  <div v-if="wallpaperUrl" class="wallpaper" :style="{ backgroundImage: `url(${wallpaperUrl})` }"></div>

  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-title">
          <span class="brand-icon">✿</span>
          WinPHP <span class="brand-accent">2025</span>
        </div>
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
      <HomeView v-if="view === 'home'" :status="status" @goto="(k) => view = k" />
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
  { key: 'home',       label: '首页',     icon: '🏠' },
  { key: 'sites',      label: '网站',     icon: '🌸' },
  { key: 'database',   label: '数据库',   icon: '💾' },
  { key: 'extensions', label: 'PHP 扩展', icon: '🧩' },
  { key: 'autostart',  label: '自启动',   icon: '⚡' },
  { key: 'tools',      label: '工具',     icon: '🛠' },
  { key: 'logs',       label: '日志',     icon: '📋' }
]

const wallpaperUrl = ref('')
provide('wallpaperUrl', wallpaperUrl)
provide('setWallpaperUrl', (u) => { wallpaperUrl.value = u || '' })

// 通过 window.go.main.App.* 调用后端
const api = window.go?.main?.App || {}
provide('api', api)
provide('runtime', window.runtime || {})

const status = reactive({
  nginx: { running: false, version: '', port: 80 },
  php: { running: false, version: '', port: 9000 },
  mysql: { running: false, version: '', port: 3306 },
  postgres: { running: false, version: '', port: 5432 },
  redis: { running: false, version: '', port: 6379 },
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
  // 加载自定义壁纸 (如果有)
  try {
    if (api.GetWallpaper) {
      const wp = await api.GetWallpaper()
      if (wp && !wp.empty && wp.dataUrl) wallpaperUrl.value = wp.dataUrl
    }
  } catch (e) { /* ignore */ }

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
  position: relative; z-index: 1;
}
.sidebar {
  width: 220px;
  background: var(--sidebar-bg);
  display: flex; flex-direction: column;
  color: #e6e9ee;
  position: relative;
  box-shadow: 4px 0 24px rgba(45, 31, 77, 0.18);
}
/* 侧边栏顶部装饰光晕 */
.sidebar::before {
  content: ''; position: absolute;
  top: -40px; left: -40px; right: -40px; height: 160px;
  background: radial-gradient(ellipse at top, rgba(255,111,158,0.35), transparent 70%);
  pointer-events: none;
}

.brand {
  padding: 22px 20px 18px;
  border-bottom: 1px solid rgba(255,255,255,0.08);
  position: relative;
}
.brand-title {
  font-size: 19px; font-weight: 700;
  display: flex; align-items: center; gap: 6px;
  color: #fff;
  text-shadow: 0 2px 8px rgba(255, 111, 158, 0.4);
}
.brand-icon {
  display: inline-block; color: #ffb1cf;
  animation: spin-slow 6s linear infinite;
  font-size: 18px;
}
@keyframes spin-slow { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.brand-accent { color: #ffb1cf; }
.brand-sub { font-size: 11px; color: #b8a3d4; margin-top: 4px; letter-spacing: 0.3px; }

.nav { flex: 1; padding: 14px 10px; position: relative; z-index: 1; }
.nav a {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 14px; border-radius: 10px;
  color: #c8bedb; cursor: pointer;
  margin-bottom: 3px;
  font-size: 14px;
  transition: all 0.18s;
  position: relative;
}
.nav a:hover { background: rgba(255,255,255,0.08); color: #fff; transform: translateX(2px); }
.nav a.active {
  background: linear-gradient(135deg, var(--primary), var(--accent));
  color: #fff;
  box-shadow: 0 4px 16px rgba(255, 111, 158, 0.45);
}
.nav .ico { width: 20px; text-align: center; font-size: 15px; }

.sidebar-foot { padding: 14px; border-top: 1px solid rgba(255,255,255,0.08); position: relative; z-index: 1; }
.bulk-btns { display: flex; gap: 6px; margin-bottom: 10px; }
.bulk-btns .btn { flex: 1; padding: 8px 0; font-size: 12px; border-radius: 10px; }
.bulk-btns .btn:not(.primary) {
  background: rgba(255,255,255,0.08);
  border-color: rgba(255,255,255,0.15); color: #d8cee8;
  backdrop-filter: none;
}
.bulk-btns .btn:not(.primary):hover { background: rgba(255,255,255,0.16); color: #fff; }
.admin-tag {
  font-size: 11px; color: #ffd28a;
  text-align: center;
  padding: 5px 8px; background: rgba(255,183,77,0.15); border-radius: 8px;
}
.admin-tag.ok { color: #b6f5be; background: rgba(95, 203, 111, 0.15); }

.content {
  flex: 1; overflow: auto;
  padding: 24px;
  position: relative;
  z-index: 1;
}
</style>
