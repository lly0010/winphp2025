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
        <button class="btn install-btn" @click="installOpen = true">+ 在线安装 (redis / xdebug ...)</button>
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

    <!-- 在线安装对话框 -->
    <div v-if="installOpen" class="modal-mask" @click.self="installOpen = false">
      <div class="modal" style="width: 640px">
        <div class="modal-header">在线安装 PHP 扩展 (PECL)</div>
        <div class="modal-body">
          <div class="form-row">
            <label>扩展</label>
            <div class="input-group">
              <select v-model="installName" :disabled="installing">
                <option v-for="e in installable" :key="e.name" :value="e.name">
                  {{ e.display }}{{ e.type === 'zend_extension' ? ' (zend)' : '' }}
                </option>
              </select>
            </div>
          </div>
          <div class="form-row">
            <label>版本</label>
            <div class="input-group">
              <select v-model="installVer" :disabled="installing">
                <option v-for="v in availableVersions" :key="v" :value="v">{{ v }}</option>
              </select>
            </div>
          </div>
          <div v-if="selectedExt" class="ext-desc">
            <strong>{{ selectedExt.display }}</strong>
            <p v-if="selectedExt.note">{{ selectedExt.note }}</p>
            <p v-if="selectedExt.deps && selectedExt.deps.length">
              依赖: <code>{{ selectedExt.deps.join(', ') }}</code>
              <span class="auto-tag">✓ 会自动联装</span>
            </p>
          </div>
          <div class="hint">
            从 <code>windows.php.net/downloads/pecl/releases/</code> 拉对应你的 PHP 版本的预编译
            DLL, 自动放进 <code>bin/php/ext/</code> 并在 php.ini 加 <code>extension={{ installName }}</code>.
            装完点 "应用 (重启 PHP-CGI)" 生效.
          </div>

          <div v-if="installing" style="margin-top: 14px">
            <div class="progress"><div class="bar" :style="{ width: percent + '%' }"></div></div>
            <div class="prog-text">{{ progText }}</div>
          </div>
          <div v-if="installMsg" :class="['msg', installErr ? 'err' : 'ok']">{{ installMsg }}</div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="installOpen = false" :disabled="installing">关闭</button>
          <button class="btn primary" @click="doInstall" :disabled="installing || !installName || !installVer">
            {{ installing ? '安装中...' : '开始安装' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, onMounted, onUnmounted, computed, watch } from 'vue'
const status = inject('status')
const api = inject('api')
const runtime = inject('runtime')

const exts = ref([])
const filter = ref('')

const installOpen = ref(false)
const installable = ref([])
const installName = ref('')
const installVer = ref('')
const installing = ref(false)
const installMsg = ref('')
const installErr = ref(false)
const loaded = ref(0)
const total = ref(0)
let offProg

const selectedExt = computed(() => installable.value.find(e => e.name === installName.value))
const availableVersions = computed(() => selectedExt.value?.versions || [])
const percent = computed(() => total.value > 0 ? Math.round(loaded.value / total.value * 100) : 0)
const progText = computed(() => {
  if (total.value > 0) {
    return (loaded.value / 1024 / 1024).toFixed(1) + ' / ' + (total.value / 1024 / 1024).toFixed(1) + ' MB (' + percent.value + '%)'
  }
  return (loaded.value / 1024 / 1024).toFixed(1) + ' MB ...'
})

async function refresh() {
  exts.value = await api.PhpExtensions() || []
}
async function loadInstallable() {
  if (api.PhpInstallableExts) {
    installable.value = await api.PhpInstallableExts() || []
    if (installable.value.length && !installName.value) {
      installName.value = installable.value[0].name
    }
  }
}
onMounted(() => {
  refresh()
  loadInstallable()
  if (runtime?.EventsOn) {
    offProg = runtime.EventsOn('phpext:progress', (p) => {
      if (!installing.value) return
      loaded.value = p.loaded; total.value = p.total
    })
  }
})
onUnmounted(() => { if (offProg) offProg() })

watch(installName, () => {
  installVer.value = selectedExt.value?.default || availableVersions.value[0] || ''
  installMsg.value = ''
})

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

async function doInstall() {
  installing.value = true
  installMsg.value = ''
  installErr.value = false
  loaded.value = 0; total.value = 0
  try {
    await api.PhpInstallExtension(installName.value, installVer.value)
    installMsg.value = '✓ 已装好. 点 "应用 (重启 PHP-CGI)" 让它生效.'
    installErr.value = false
    await refresh()
  } catch (e) {
    installMsg.value = '失败: ' + e
    installErr.value = true
  } finally {
    installing.value = false
  }
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

.install-btn {
  background: linear-gradient(135deg, #ffe5ef, #f3e8ff);
  border-color: rgba(255, 111, 158, 0.35); color: var(--primary-dark);
}
.install-btn:hover { filter: brightness(1.05); }

.ext-desc {
  margin-top: 4px; padding: 10px 12px;
  background: rgba(176,111,255,0.06); border-radius: 8px;
  font-size: 12px; color: var(--text-secondary); line-height: 1.6;
}
.ext-desc strong { color: var(--text); }
.ext-desc p { margin: 4px 0 0; }
.ext-desc code { background: rgba(255,111,158,0.10); padding: 1px 5px; border-radius: 3px; font-family: Consolas, monospace; }
.ext-desc .auto-tag {
  display: inline-block; margin-left: 6px;
  font-size: 10px; padding: 2px 7px; border-radius: 8px;
  background: rgba(95,203,111,0.14); color: #2d7a2d;
}

.hint {
  margin-top: 12px; padding: 10px 12px;
  background: #fffbeb; border-left: 3px solid var(--warning); border-radius: 4px;
  font-size: 12px; color: var(--text-secondary); line-height: 1.65;
}
.hint code { background: #fff3cd; padding: 1px 5px; border-radius: 3px; font-family: Consolas, monospace; }

.progress { background: #e3e6ea; height: 10px; border-radius: 5px; overflow: hidden; margin-top: 8px; }
.bar { background: var(--header-grad); height: 100%; transition: width 0.2s; }
.prog-text { font-size: 12px; color: var(--text-secondary); margin-top: 6px; }

.msg { margin-top: 12px; padding: 8px 12px; border-radius: 6px; font-size: 12px; word-break: break-all; white-space: pre-wrap; }
.msg.ok { color: #2d7a2d; background: rgba(95,203,111,0.10); }
.msg.err { color: var(--danger); background: #fff5f5; }
</style>
