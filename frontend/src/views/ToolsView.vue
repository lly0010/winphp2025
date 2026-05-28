<template>
  <div>
    <h1 class="page-title">工具</h1>
    <div class="tools-grid">
      <button class="tool-btn" @click="api.OpenInBrowser('http://localhost')">
        <div class="t1">浏览 localhost</div>
        <div class="t2">http://localhost</div>
      </button>
      <button class="tool-btn" @click="api.OpenInBrowser('http://localhost/phpinfo.php')">
        <div class="t1">phpinfo</div>
        <div class="t2">查看 PHP 配置</div>
      </button>
      <button class="tool-btn" @click="openFolder('www')">
        <div class="t1">www 目录</div>
        <div class="t2">网站文件根目录</div>
      </button>
      <button class="tool-btn" @click="openFolder('logs')">
        <div class="t1">日志目录</div>
        <div class="t2">面板 / 服务日志</div>
      </button>
      <button class="tool-btn" @click="openFolder('bin')">
        <div class="t1">bin 目录</div>
        <div class="t2">组件二进制</div>
      </button>
      <button class="tool-btn" @click="openFolder('root')">
        <div class="t1">面板根目录</div>
        <div class="t2">整体可移动</div>
      </button>
      <button class="tool-btn" @click="hostsOpen = true">
        <div class="t1">编辑 hosts</div>
        <div class="t2">添加本地域名解析</div>
      </button>
      <button class="tool-btn" @click="checkPort(80)">
        <div class="t1">诊断端口 80</div>
        <div class="t2">谁占用 / 是否被 Win 预留</div>
      </button>
      <button class="tool-btn" @click="checkPort(3306)">
        <div class="t1">诊断端口 3306</div>
        <div class="t2">MySQL</div>
      </button>
      <button class="tool-btn" @click="checkPort(5432)">
        <div class="t1">诊断端口 5432</div>
        <div class="t2">PostgreSQL</div>
      </button>
      <button class="tool-btn" @click="checkPort(6379)">
        <div class="t1">诊断端口 6379</div>
        <div class="t2">Redis</div>
      </button>
      <button class="tool-btn" @click="checkPort(9000)">
        <div class="t1">诊断端口 9000</div>
        <div class="t2">PHP-CGI</div>
      </button>
      <button class="tool-btn" @click="api.NginxReload()">
        <div class="t1">Nginx reload</div>
        <div class="t2">重载配置, 不停服务</div>
      </button>
      <button class="tool-btn wallpaper-btn" @click="pickWallpaper">
        <div class="t1">🌸 自定义壁纸</div>
        <div class="t2">{{ hasWallpaper ? '已设置, 点击更换' : '让面板更萌一点' }}</div>
      </button>
      <button v-if="hasWallpaper" class="tool-btn" @click="clearWallpaper">
        <div class="t1">移除壁纸</div>
        <div class="t2">恢复默认背景</div>
      </button>
      <button class="tool-btn theme-btn" @click="themeOpen = true">
        <div class="t1">🎨 切换主题</div>
        <div class="t2">内置粉紫/蓝色, 支持第三方主题包</div>
      </button>
    </div>
    <ConfigEditor v-if="hostsOpen" ckey="hosts" title="hosts" @close="hostsOpen = false" />
    <ThemeDialog v-if="themeOpen" @close="themeOpen = false" />
  </div>
</template>

<script setup>
import { inject, ref, computed } from 'vue'
import ConfigEditor from '../components/ConfigEditor.vue'
import ThemeDialog from '../components/ThemeDialog.vue'
const api = inject('api')
const setWallpaperUrl = inject('setWallpaperUrl', () => {})
const wallpaperUrl = inject('wallpaperUrl', ref(''))
const hostsOpen = ref(false)
const themeOpen = ref(false)
const hasWallpaper = computed(() => !!wallpaperUrl.value)

async function pickWallpaper() {
  try {
    const wp = await api.PickAndSetWallpaper()
    if (wp && !wp.empty && wp.dataUrl) {
      setWallpaperUrl(wp.dataUrl)
    }
  } catch (e) {
    alert('设置壁纸失败: ' + e)
  }
}
async function clearWallpaper() {
  if (!confirm('确定移除当前壁纸?')) return
  try {
    await api.ClearWallpaper()
    setWallpaperUrl('')
  } catch (e) {
    alert('移除失败: ' + e)
  }
}

async function openFolder(key) {
  const p = await api.GetPaths()
  const map = { www: p.wwwDir, logs: p.logsDir, root: p.root, bin: p.binDir }
  if (map[key]) await api.OpenFolder(map[key])
}

async function checkPort(n) {
  try {
    const info = await api.DiagnosePort(n)
    alert(info.diagnosis || ('端口 ' + n + ' 状态未知'))
  } catch (e) {
    // 兜底: 老的 PortInUse
    const inUse = await api.PortInUse(n)
    alert('端口 ' + n + (inUse ? ' 已被占用' : ' 空闲'))
  }
}
</script>

<style scoped>
.page-title {
  font-size: 22px; font-weight: 700; margin: 0 0 18px;
  background: var(--header-grad);
  -webkit-background-clip: text; background-clip: text; color: transparent;
}
.tools-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}
.tool-btn {
  background: var(--bg-card); border: 1px solid var(--border-soft);
  border-radius: var(--radius); padding: 16px;
  cursor: pointer; text-align: left;
  transition: all 0.2s cubic-bezier(.4,.2,.2,1);
  backdrop-filter: blur(8px);
}
.tool-btn:hover {
  border-color: var(--primary);
  background: var(--primary-light);
  transform: translateY(-2px);
  box-shadow: var(--shadow-hover);
}
.tool-btn.wallpaper-btn {
  background: linear-gradient(135deg, #ffe5ef, #f3e8ff);
  border-color: rgba(255, 111, 158, 0.35);
}
.tool-btn.wallpaper-btn:hover {
  background: linear-gradient(135deg, #ffd6e6, #ead4ff);
}
.tool-btn.theme-btn {
  background: linear-gradient(135deg, #e8f1fa, #f3e8ff);
  border-color: rgba(176, 111, 255, 0.30);
}
.tool-btn.theme-btn:hover {
  background: linear-gradient(135deg, #d6e7f5, #ead4ff);
}
.t1 { font-weight: 600; color: var(--text); margin-bottom: 4px; }
.t2 { font-size: 12px; color: var(--text-secondary); }
</style>
