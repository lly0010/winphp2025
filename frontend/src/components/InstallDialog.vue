<template>
  <div class="modal-mask">
    <div class="modal" style="width: 620px">
      <div class="modal-header">安装 / 切换 {{ kindLabel }} 版本</div>
      <div class="modal-body">

        <div class="form-row">
          <label>版本</label>
          <div class="input-group">
            <select v-model="selected" :disabled="downloading">
              <option v-for="v in versions" :key="v.version" :value="v.version">
                {{ v.version }}{{ v.vs ? ' (' + v.vs + ')' : '' }}{{ v.custom ? '  • 自定义' : '' }}{{ v.localZip ? '  [本地 zip]' : '' }}
              </option>
            </select>
            <button class="btn" @click="customOpen = true" :disabled="downloading" title="添加自定义版本">+ 自定义</button>
            <button v-if="selectedEntry?.custom" class="btn sm danger" @click="removeCustom" :disabled="downloading"
                    title="删除该自定义版本 (不影响内置)">删除</button>
          </div>
        </div>

        <!-- 下载前预览 URL 列表 -->
        <div v-if="!downloading" class="url-preview">
          <div class="url-preview-head">
            <span v-if="isLocal">本地 zip 文件 (无需下载)</span>
            <span v-else>候选下载源 (按顺序尝试)</span>
            <span class="head-right">
              <a v-if="!isLocal && previewUrls.length > 0" class="toggle test"
                 @click="testConnectivity" :class="{ busy: testing }">
                {{ testing ? '测试中...' : '🌐 测试连通性' }}
              </a>
              <a v-if="!isLocal" class="toggle" @click="urlOpen = !urlOpen">{{ urlOpen ? '收起 ▴' : '展开 ▾' }}</a>
            </span>
          </div>
          <ul v-if="isLocal" class="single">
            <li><code>{{ selectedEntry.localZip }}</code></li>
          </ul>
          <ul v-else v-show="urlOpen">
            <li v-for="(u, i) in previewUrls" :key="i">
              <span class="idx">{{ i + 1 }}.</span>
              <span class="u">{{ u }}</span>
              <span v-if="testResults[u]" class="test-result" :class="{ ok: testResults[u].ok, fail: !testResults[u].ok }">
                <template v-if="testResults[u].ok">✓ {{ testResults[u].status }} · {{ testResults[u].elapsedMs }}ms</template>
                <template v-else>✗ {{ testResults[u].status || '-' }} {{ testResults[u].error || 'unreachable' }}</template>
              </span>
            </li>
            <li v-if="previewUrls.length === 0" class="empty">无可用 URL (检查 sources.json)</li>
          </ul>
        </div>

        <!-- 下载中 -->
        <div v-if="downloading" style="margin-top: 14px;">
          <div class="progress">
            <div class="bar" :style="{ width: percent + '%' }"></div>
          </div>
          <div class="prog-text">{{ progressText }}</div>
          <div class="cur-url" v-if="currentUrl">正在: {{ currentUrl }}</div>
        </div>

        <div v-if="error" class="error">{{ error }}</div>
      </div>
      <div class="modal-footer">
        <button v-if="downloading" class="btn danger" @click="cancel">取消下载</button>
        <button v-else class="btn" @click="$emit('close')">关闭</button>
        <button class="btn primary" @click="start"
                :disabled="downloading || !selected || (!isLocal && previewUrls.length === 0)">
          {{ downloading ? '安装中...' : '开始安装' }}
        </button>
      </div>
    </div>

    <CustomVersionDialog v-if="customOpen" :kind="kind" @close="customOpen = false" @saved="onCustomSaved" />
  </div>
</template>

<script setup>
import { inject, ref, computed, watch, onMounted, onUnmounted } from 'vue'
import CustomVersionDialog from './CustomVersionDialog.vue'
const props = defineProps({ kind: String })
const emit = defineEmits(['close'])
const api = inject('api')
const runtime = inject('runtime')

const kindLabel = { nginx: 'Nginx', php: 'PHP', mysql: 'MySQL', postgresql: 'PostgreSQL' }[props.kind] || props.kind
const versions = ref([])
const selected = ref('')
const previewUrls = ref([])
const urlOpen = ref(false)
const customOpen = ref(false)

const downloading = ref(false)
const loaded = ref(0)
const total = ref(0)
const error = ref('')
const percent = ref(0)
const currentUrl = ref('')
const testing = ref(false)
const testResults = ref({}) // {url: {ok, status, elapsedMs, error}}
let offProgress, offLog

const selectedEntry = computed(() => versions.value.find(v => v.version === selected.value))
const isLocal = computed(() => !!(selectedEntry.value && selectedEntry.value.localZip))

async function reloadVersions(keep) {
  versions.value = await api.ListVersions(props.kind) || []
  if (keep && versions.value.some(v => v.version === keep)) {
    selected.value = keep
  } else if (versions.value.length) {
    selected.value = versions.value[0].version
  }
}

onMounted(async () => {
  await reloadVersions()
  await refreshPreview()

  if (runtime?.EventsOn) {
    offProgress = runtime.EventsOn('install:progress', (p) => {
      if (p.kind !== props.kind) return
      loaded.value = p.loaded; total.value = p.total
      percent.value = total.value > 0 ? Math.round(loaded.value / total.value * 100) : 0
    })
    offLog = runtime.EventsOn('log', (e) => {
      if (!downloading.value) return
      const m = e.msg.match(/下载 \(第.*次\): (https?:\/\/\S+)/)
      if (m) currentUrl.value = m[1]
    })
  }
})
onUnmounted(() => {
  if (offProgress) offProgress()
  if (offLog) offLog()
})

watch(selected, refreshPreview)
async function refreshPreview() {
  testResults.value = {} // 换版本就清掉旧测试结果
  if (!selected.value) { previewUrls.value = []; return }
  if (isLocal.value) { previewUrls.value = []; return }
  try { previewUrls.value = await api.PreviewUrls(props.kind, selected.value) || [] }
  catch { previewUrls.value = [] }
}

async function testConnectivity() {
  if (testing.value || previewUrls.value.length === 0) return
  testing.value = true
  urlOpen.value = true
  // 先全部置 "测试中" 占位 (空对象表示 pending, 不显示)
  testResults.value = {}
  try {
    const results = await api.TestUrls(previewUrls.value)
    const map = {}
    for (const r of (results || [])) map[r.url] = r
    testResults.value = map
  } catch (e) {
    error.value = '测试失败: ' + e
  } finally {
    testing.value = false
  }
}

const progressText = computed(() => {
  if (total.value > 0) {
    return (loaded.value / 1024 / 1024).toFixed(1) + ' / ' + (total.value / 1024 / 1024).toFixed(1) + ' MB  (' + percent.value + '%)'
  }
  return (loaded.value / 1024 / 1024).toFixed(1) + ' MB ...'
})

async function start() {
  downloading.value = true
  error.value = ''
  currentUrl.value = ''
  loaded.value = 0; total.value = 0; percent.value = 0
  try {
    await api.InstallComponent(props.kind, selected.value)
    emit('close')
  } catch (e) {
    error.value = '' + e
  } finally {
    downloading.value = false
  }
}

async function cancel() {
  try { await api.CancelInstall(props.kind) } catch (e) { /* ignore */ }
}

async function onCustomSaved() {
  customOpen.value = false
  await reloadVersions()
  // 切到刚加的版本 (最后一个 custom)
  const last = [...versions.value].reverse().find(v => v.custom)
  if (last) selected.value = last.version
  await refreshPreview()
}

async function removeCustom() {
  if (!selectedEntry.value || !selectedEntry.value.custom) return
  if (!confirm('确定删除自定义版本 "' + selected.value + '"? (只删除 config/custom_sources.json 里的记录, 不影响已安装文件)')) return
  try {
    await api.RemoveCustomVersion(props.kind, selected.value)
    await reloadVersions()
    await refreshPreview()
  } catch (e) {
    error.value = '' + e
  }
}
</script>

<style scoped>
.progress { background: #e3e6ea; height: 10px; border-radius: 5px; overflow: hidden; }
.bar { background: var(--primary); height: 100%; transition: width 0.2s; }
.prog-text { font-size: 12px; color: var(--text-secondary); margin-top: 6px; }
.cur-url { font-size: 11px; color: var(--text-secondary); margin-top: 4px; word-break: break-all; }

.error {
  color: var(--danger); margin-top: 12px;
  padding: 8px 10px; background: #fff5f5; border-radius: 4px;
  font-size: 12px; max-height: 160px; overflow: auto; word-break: break-all;
  white-space: pre-wrap;
}

.url-preview {
  margin-top: 8px; background: #f7f8fa; border-radius: 6px; padding: 8px 12px; font-size: 12px;
}
.url-preview-head {
  color: var(--text-secondary); display: flex; justify-content: space-between; align-items: center;
}
.url-preview .toggle { color: var(--primary); cursor: pointer; font-size: 11px; }
.url-preview ul { list-style: none; padding: 6px 0 0; margin: 0; }
.url-preview ul.single { padding: 6px 0 2px; }
.url-preview li { display: flex; gap: 6px; padding: 3px 0; line-height: 1.5; word-break: break-all; align-items: baseline; }
.url-preview .idx { color: var(--text-disabled); flex-shrink: 0; }
.url-preview .u { color: var(--text-secondary); flex: 1; }
.url-preview code { color: var(--primary); font-family: Consolas, monospace; }
.url-preview .empty { color: var(--danger); }
.url-preview .head-right { display: flex; gap: 12px; align-items: center; }
.url-preview .toggle.test { color: var(--success); }
.url-preview .toggle.test.busy { color: var(--text-disabled); pointer-events: none; }
.url-preview .test-result {
  flex-shrink: 0; font-size: 11px; padding: 1px 6px; border-radius: 3px;
  font-family: Consolas, monospace;
}
.url-preview .test-result.ok   { color: #2d7a2d; background: rgba(60,170,60,0.08); }
.url-preview .test-result.fail { color: #c23030; background: rgba(217,74,74,0.08); }

.btn.sm { padding: 4px 10px; font-size: 12px; }
</style>
