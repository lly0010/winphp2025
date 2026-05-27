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
    </div>
    <ConfigEditor v-if="hostsOpen" ckey="hosts" title="hosts" @close="hostsOpen = false" />
  </div>
</template>

<script setup>
import { inject, ref } from 'vue'
import ConfigEditor from '../components/ConfigEditor.vue'
const api = inject('api')
const hostsOpen = ref(false)

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
.page-title { font-size: 20px; font-weight: 600; margin: 0 0 16px; }
.tools-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}
.tool-btn {
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 16px;
  cursor: pointer; text-align: left;
  transition: all 0.15s;
}
.tool-btn:hover { border-color: var(--primary); background: var(--primary-light); }
.t1 { font-weight: 500; color: var(--text); margin-bottom: 4px; }
.t2 { font-size: 12px; color: var(--text-secondary); }
</style>
