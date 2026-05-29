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
      <button class="tool-btn basedir-btn" @click="openBasedir">
        <div class="t1">🛡 PHP 防跨盘</div>
        <div class="t2">{{ basedirInfo.enabled ? '已启用, 点击调整' : '未开启 (可道云等会挂 C 盘)' }}</div>
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

    <!-- PHP 防跨盘对话框 -->
    <div v-if="basedirOpen" class="modal-mask" @click.self="basedirOpen = false">
      <div class="modal" style="width: 640px">
        <div class="modal-header">PHP 防跨盘访问 (open_basedir)</div>
        <div class="modal-body">
          <label class="basedir-toggle">
            <input type="checkbox" v-model="basedirEnabled" />
            <span>启用 PHP 目录访问限制</span>
          </label>
          <p class="hint">
            启用后 PHP 脚本 (如可道云 / 文件管理器) 只能访问下面列出的目录,
            不会自动挂 C 盘整盘. <strong>www 目录始终允许</strong> (网站在 C 盘也能正常跑).
            关掉后无限制.
          </p>
          <div class="paths-block">
            <div class="lbl">始终允许 (网站 + PHP 自身, 不可改)</div>
            <code v-for="p in basedirInfo.alwaysPaths" :key="p" class="path-row">{{ p }}</code>
          </div>
          <div class="paths-block">
            <div class="lbl-row">
              <div class="lbl">额外允许目录 (每行一个, 如 D:\ 或 E:\data\)</div>
              <button class="mini-btn" @click="addNonSystemDrives" :disabled="basedirSaving">+ 一键添加非 C 盘</button>
            </div>
            <textarea
              v-model="extraText"
              rows="5"
              placeholder="D:\&#10;E:\data\&#10;F:\backup\"
              :disabled="basedirSaving"
            ></textarea>
            <div class="tiny">路径结尾自动补斜杠. 写 <code>D:\</code> 等于允许整个 D 盘.</div>
          </div>
          <div v-if="basedirMsg" :class="['msg', basedirMsgErr ? 'err' : 'ok']">{{ basedirMsg }}</div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="basedirOpen = false" :disabled="basedirSaving">关闭</button>
          <button class="btn" @click="saveBasedir(false)" :disabled="basedirSaving">仅保存</button>
          <button class="btn primary" @click="saveBasedir(true)" :disabled="basedirSaving">
            {{ basedirSaving ? '应用中...' : '保存并重启 PHP-CGI' }}
          </button>
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
const basedirOpen = ref(false)
const basedirInfo = ref({ enabled: false, extra: [], effectivePaths: [], alwaysPaths: [] })
const basedirEnabled = ref(false)
const extraText = ref('')
const basedirSaving = ref(false)
const basedirMsg = ref('')
const basedirMsgErr = ref(false)
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
async function loadBasedir() {
  try {
    const info = await api.GetOpenBasedir()
    if (info) {
      basedirInfo.value = info
      basedirEnabled.value = !!info.enabled
      extraText.value = (info.extra || []).join('\n')
    }
  } catch { /* ignore */ }
}
onMounted(() => { loadDataInfo(); loadBasedir() })
watch(dataDirOpen, v => { if (v) { msg.value = ''; loadDataInfo() } })

async function openBasedir() {
  basedirMsg.value = ''
  basedirMsgErr.value = false
  await loadBasedir()
  basedirOpen.value = true
}

async function addNonSystemDrives() {
  try {
    const drives = await api.ListNonSystemDrives() || []
    if (!drives.length) {
      basedirMsg.value = '没检测到 C 盘以外的盘符'
      basedirMsgErr.value = true
      return
    }
    const cur = extraText.value.split('\n').map(s => s.trim()).filter(Boolean)
    const have = new Set(cur.map(s => s.toLowerCase().replace(/\\/g, '/')))
    for (const d of drives) {
      if (!have.has(d.toLowerCase().replace(/\\/g, '/'))) cur.push(d)
    }
    extraText.value = cur.join('\n')
    basedirMsg.value = '已加入: ' + drives.join(', ')
    basedirMsgErr.value = false
  } catch (e) {
    basedirMsg.value = '失败: ' + e
    basedirMsgErr.value = true
  }
}

async function saveBasedir(restart) {
  basedirSaving.value = true
  basedirMsg.value = ''
  try {
    const extra = extraText.value.split('\n').map(s => s.trim()).filter(Boolean)
    await api.SetOpenBasedir(basedirEnabled.value, extra)
    await loadBasedir()
    if (restart) {
      try {
        await api.RestartService('php')
        basedirMsg.value = '✓ 已保存并重启 PHP-CGI, 现在生效.'
      } catch (e) {
        basedirMsg.value = '✓ 已保存. PHP-CGI 重启失败: ' + e + ' (可去首页手动重启)'
      }
    } else {
      basedirMsg.value = '✓ 已保存. 重启 PHP-CGI 后生效.'
    }
    basedirMsgErr.value = false
  } catch (e) {
    basedirMsg.value = '失败: ' + e
    basedirMsgErr.value = true
  } finally {
    basedirSaving.value = false
  }
}

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
.tool-btn.basedir-btn {
  background: linear-gradient(135deg, #e6f7ec, #e6f0fa);
  border-color: rgba(95, 203, 111, 0.40);
}
.tool-btn.basedir-btn:hover {
  background: linear-gradient(135deg, #d6efdf, #d6e6f5);
}

.basedir-toggle {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 0; font-weight: 600; cursor: pointer;
}
.basedir-toggle input { transform: scale(1.15); }
.paths-block { margin-top: 12px; }
.paths-block .lbl { font-size: 12px; color: var(--text-secondary); margin-bottom: 4px; }
.lbl-row { display: flex; justify-content: space-between; align-items: center; }
.path-row {
  display: block; font-family: Consolas, monospace; font-size: 12px;
  background: rgba(95,203,111,0.08); color: var(--text);
  padding: 4px 8px; border-radius: 4px; margin: 2px 0; word-break: break-all;
}
.paths-block textarea {
  width: 100%; box-sizing: border-box; padding: 8px;
  font-family: Consolas, monospace; font-size: 12px;
  border: 1px solid var(--border); border-radius: 6px;
  background: #fff; resize: vertical;
}
.paths-block .tiny { font-size: 11px; color: var(--text-secondary); margin-top: 4px; }
.paths-block .tiny code { background: rgba(255,183,77,0.15); padding: 0 4px; border-radius: 3px; }
.mini-btn {
  font-size: 11px; padding: 3px 8px; border-radius: 4px;
  border: 1px solid var(--border); background: #fff; cursor: pointer;
  color: var(--text);
}
.mini-btn:hover:not(:disabled) { background: var(--primary-light); border-color: var(--primary); }
.mini-btn:disabled { opacity: 0.5; cursor: not-allowed; }
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
