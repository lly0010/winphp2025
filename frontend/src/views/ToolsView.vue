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
      <button class="tool-btn data-btn" @click="dataDirOpen = true">
        <div class="t1">📦 数据目录</div>
        <div class="t2 t2-data">{{ dataDirShort }}</div>
      </button>
    </div>

    <!-- 数据目录对话框 -->
    <div v-if="dataDirOpen" class="modal-mask" @click.self="dataDirOpen = false">
      <div class="modal" style="width: 600px">
        <div class="modal-header">数据目录设置</div>
        <div class="modal-body">
          <div class="info-row"><span class="lbl">当前数据目录</span><code>{{ dataInfo.current }}</code></div>
          <div class="info-row"><span class="lbl">EXE 所在</span><code>{{ dataInfo.exeDir }}</code></div>
          <div class="info-row"><span class="lbl">指针文件</span><code>{{ dataInfo.pointerPath }}</code> {{ dataInfo.pointerExist ? '✓ 已设置' : '(未设置)' }}</div>

          <div class="tip">
            指针文件 <code>data-dir.txt</code> 放在 EXE 旁, 内容是一行路径.
            <br>设置后所有 <code>bin/ www/ config/ logs/</code> 都放进新路径,
            <strong>更新 EXE 时数据完全独立不丢</strong>.
            <br>修改后需要重启面板才生效.
          </div>

          <div v-if="msg" :class="['msg', msgKind]">{{ msg }}</div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="dataDirOpen = false">关闭</button>
          <button class="btn" :disabled="!dataInfo.pointerExist" @click="resetDataDir">恢复默认 (回 EXE 目录)</button>
          <button class="btn primary" @click="pickDataDir">选择新目录...</button>
        </div>
      </div>
    </div>

    <ConfigEditor v-if="hostsOpen" ckey="hosts" title="hosts" @close="hostsOpen = false" />
    <ThemeDialog v-if="themeOpen" @close="themeOpen = false" />
  </div>
</template>

<script setup>
import { inject, ref, computed, onMounted, watch } from 'vue'
import ConfigEditor from '../components/ConfigEditor.vue'
import ThemeDialog from '../components/ThemeDialog.vue'
const api = inject('api')
const setWallpaperUrl = inject('setWallpaperUrl', () => {})
const wallpaperUrl = inject('wallpaperUrl', ref(''))
const hostsOpen = ref(false)
const themeOpen = ref(false)
const dataDirOpen = ref(false)
const dataInfo = ref({ current: '', exeDir: '', pointerExist: false, pointerPath: '' })
const msg = ref('')
const msgKind = ref('ok')
const hasWallpaper = computed(() => !!wallpaperUrl.value)
const dataDirShort = computed(() => {
  const p = dataInfo.value.current || ''
  if (!p) return '加载中...'
  // 太长截短显示, 保留头尾
  if (p.length > 36) return p.slice(0, 14) + '...' + p.slice(-18)
  return p
})

async function loadDataInfo() {
  try { dataInfo.value = await api.GetDataDirInfo() || dataInfo.value }
  catch { /* ignore */ }
}
onMounted(loadDataInfo)
watch(dataDirOpen, v => { if (v) { msg.value = ''; loadDataInfo() } })

async function pickDataDir() {
  msg.value = ''
  try {
    const info = await api.PickAndSetDataDir()
    if (info) dataInfo.value = info
    if (info && info.pointerExist) {
      msg.value = '✓ 已保存. 请关闭面板后重新打开, 新数据目录才生效.'
      msgKind.value = 'ok'
    }
  } catch (e) {
    msg.value = '失败: ' + e
    msgKind.value = 'err'
  }
}

async function resetDataDir() {
  if (!confirm('恢复默认会删除指针文件 data-dir.txt, 重启后数据回到 EXE 同目录. 继续?')) return
  msg.value = ''
  try {
    const info = await api.ResetDataDir()
    if (info) dataInfo.value = info
    msg.value = '✓ 已恢复默认. 重启面板生效.'
    msgKind.value = 'ok'
  } catch (e) {
    msg.value = '失败: ' + e
    msgKind.value = 'err'
  }
}

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
.tool-btn.data-btn {
  background: linear-gradient(135deg, #fffbeb, #fff0d6);
  border-color: rgba(255, 183, 77, 0.40);
}
.tool-btn.data-btn:hover {
  background: linear-gradient(135deg, #fff3cf, #ffe0a8);
}
.t2-data { font-family: Consolas, monospace; font-size: 11px; word-break: break-all; }

.info-row { display: flex; align-items: baseline; gap: 8px; padding: 6px 0; }
.info-row .lbl { width: 110px; flex-shrink: 0; color: var(--text-secondary); font-size: 12px; }
.info-row code { font-family: Consolas, monospace; font-size: 12px; word-break: break-all; color: var(--primary-dark); }
.tip {
  margin-top: 14px; padding: 12px 14px;
  background: linear-gradient(135deg, #fffbeb, #fff0d6);
  border-left: 3px solid var(--warning); border-radius: 6px;
  font-size: 12px; color: var(--text-secondary); line-height: 1.7;
}
.tip code { background: #fff3cd; padding: 1px 5px; border-radius: 3px; font-family: Consolas, monospace; }
.tip strong { color: var(--danger); }
.msg { margin-top: 12px; padding: 8px 12px; border-radius: 6px; font-size: 12px; }
.msg.ok  { color: #2d7a2d; background: rgba(95,203,111,0.10); }
.msg.err { color: var(--danger); background: #fff5f5; }
.t1 { font-weight: 600; color: var(--text); margin-bottom: 4px; }
.t2 { font-size: 12px; color: var(--text-secondary); }
</style>
