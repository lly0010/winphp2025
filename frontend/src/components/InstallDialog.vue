<template>
  <div class="modal-mask">
    <div class="modal" style="width: 520px">
      <div class="modal-header">安装 / 切换 {{ kindLabel }} 版本</div>
      <div class="modal-body">
        <div class="form-row">
          <label>版本</label>
          <div class="input-group">
            <select v-model="selected">
              <option v-for="v in versions" :key="v.version" :value="v.version">
                {{ v.version }}{{ v.vs ? ' (' + v.vs + ')' : '' }}
              </option>
            </select>
          </div>
        </div>

        <div v-if="downloading" style="margin-top: 14px;">
          <div class="progress">
            <div class="bar" :style="{ width: percent + '%' }"></div>
          </div>
          <div class="prog-text">{{ progressText }}</div>
        </div>

        <div v-if="error" class="error">{{ error }}</div>
      </div>
      <div class="modal-footer">
        <button class="btn" @click="$emit('close')" :disabled="downloading">关闭</button>
        <button class="btn primary" @click="start" :disabled="downloading || !selected">
          {{ downloading ? '下载中...' : '开始安装' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, onMounted, onUnmounted } from 'vue'
const props = defineProps({ kind: String })
const emit = defineEmits(['close'])
const api = inject('api')
const runtime = inject('runtime')

const kindLabel = { nginx: 'Nginx', php: 'PHP', mysql: 'MySQL', postgresql: 'PostgreSQL' }[props.kind] || props.kind
const versions = ref([])
const selected = ref('')
const downloading = ref(false)
const loaded = ref(0)
const total = ref(0)
const error = ref('')
const percent = ref(0)
let off

onMounted(async () => {
  versions.value = await api.ListVersions(props.kind) || []
  if (versions.value.length) selected.value = versions.value[0].version
  if (runtime?.EventsOn) {
    off = runtime.EventsOn('install:progress', (p) => {
      if (p.kind !== props.kind) return
      loaded.value = p.loaded; total.value = p.total
      percent.value = total.value > 0 ? Math.round(loaded.value / total.value * 100) : 0
    })
  }
})
onUnmounted(() => { if (off) off() })

const progressText = () => {
  if (total.value > 0) {
    return (loaded.value / 1024 / 1024).toFixed(1) + ' / ' + (total.value / 1024 / 1024).toFixed(1) + ' MB  (' + percent.value + '%)'
  }
  return (loaded.value / 1024 / 1024).toFixed(1) + ' MB ...'
}

async function start() {
  downloading.value = true
  error.value = ''
  try {
    await api.InstallComponent(props.kind, selected.value)
    emit('close')
  } catch (e) {
    error.value = '' + e
  } finally {
    downloading.value = false
  }
}
</script>

<style scoped>
.progress { background: #e3e6ea; height: 10px; border-radius: 5px; overflow: hidden; }
.bar { background: var(--primary); height: 100%; transition: width 0.2s; }
.prog-text { font-size: 12px; color: var(--text-secondary); margin-top: 6px; }
.error { color: var(--danger); margin-top: 12px; padding: 8px 10px; background: #fff5f5; border-radius: 4px; font-size: 12px; max-height: 120px; overflow: auto; }
</style>
