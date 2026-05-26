<template>
  <div class="modal-mask">
    <div class="modal" style="width: 600px">
      <div class="modal-header">安装 / 切换 {{ kindLabel }} 版本</div>
      <div class="modal-body">
        <div class="form-row">
          <label>版本</label>
          <div class="input-group">
            <select v-model="selected" :disabled="downloading">
              <option v-for="v in versions" :key="v.version" :value="v.version">
                {{ v.version }}{{ v.vs ? ' (' + v.vs + ')' : '' }}
              </option>
            </select>
          </div>
        </div>

        <!-- 下载前预览 URL 列表 -->
        <div v-if="!downloading" class="url-preview">
          <div class="url-preview-head">
            候选下载源 (按顺序尝试, 一个失败自动换下一个)
            <a class="toggle" @click="urlOpen = !urlOpen">{{ urlOpen ? '收起 ▴' : '展开 ▾' }}</a>
          </div>
          <ul v-show="urlOpen">
            <li v-for="(u, i) in previewUrls" :key="i">
              <span class="idx">{{ i + 1 }}.</span>
              <span class="u">{{ u }}</span>
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
        <button class="btn primary" @click="start" :disabled="downloading || !selected || previewUrls.length === 0">
          {{ downloading ? '下载中...' : '开始安装' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, computed, watch, onMounted, onUnmounted } from 'vue'
const props = defineProps({ kind: String })
const emit = defineEmits(['close'])
const api = inject('api')
const runtime = inject('runtime')

const kindLabel = { nginx: 'Nginx', php: 'PHP', mysql: 'MySQL', postgresql: 'PostgreSQL' }[props.kind] || props.kind
const versions = ref([])
const selected = ref('')
const previewUrls = ref([])
const urlOpen = ref(false)

const downloading = ref(false)
const loaded = ref(0)
const total = ref(0)
const error = ref('')
const percent = ref(0)
const currentUrl = ref('')
let offProgress, offLog

onMounted(async () => {
  versions.value = await api.ListVersions(props.kind) || []
  if (versions.value.length) selected.value = versions.value[0].version
  await refreshPreview()

  if (runtime?.EventsOn) {
    offProgress = runtime.EventsOn('install:progress', (p) => {
      if (p.kind !== props.kind) return
      loaded.value = p.loaded; total.value = p.total
      percent.value = total.value > 0 ? Math.round(loaded.value / total.value * 100) : 0
    })
    // 后端日志带"下载 (第 x/x 次): URL", 截取来显示当前 URL
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
  if (!selected.value) { previewUrls.value = []; return }
  try { previewUrls.value = await api.PreviewUrls(props.kind, selected.value) || [] }
  catch { previewUrls.value = [] }
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
</script>

<style scoped>
.progress { background: #e3e6ea; height: 10px; border-radius: 5px; overflow: hidden; }
.bar { background: var(--primary); height: 100%; transition: width 0.2s; }
.prog-text { font-size: 12px; color: var(--text-secondary); margin-top: 6px; }
.cur-url { font-size: 11px; color: var(--text-secondary); margin-top: 4px; word-break: break-all; }

.error { color: var(--danger); margin-top: 12px; padding: 8px 10px; background: #fff5f5; border-radius: 4px; font-size: 12px; max-height: 120px; overflow: auto; word-break: break-all; }

.url-preview {
  margin-top: 8px;
  background: #f7f8fa;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 12px;
}
.url-preview-head {
  color: var(--text-secondary); display: flex; justify-content: space-between; align-items: center;
}
.url-preview .toggle { color: var(--primary); cursor: pointer; font-size: 11px; }
.url-preview ul { list-style: none; padding: 6px 0 0; margin: 0; }
.url-preview li { display: flex; gap: 6px; padding: 3px 0; line-height: 1.5; word-break: break-all; }
.url-preview .idx { color: var(--text-disabled); flex-shrink: 0; }
.url-preview .u { color: var(--text-secondary); }
.url-preview .empty { color: var(--danger); }
</style>
