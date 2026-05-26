<template>
  <div class="modal-mask">
    <div class="modal" style="width: 620px">
      <div class="modal-header">添加自定义 {{ kindLabel }} 版本</div>
      <div class="modal-body">

        <div class="form-row">
          <label>来源</label>
          <div class="input-group">
            <label class="radio"><input type="radio" v-model="mode" value="url" /> 在线 URL</label>
            <label class="radio"><input type="radio" v-model="mode" value="file" /> 本地 zip 文件</label>
          </div>
        </div>

        <div class="form-row">
          <label>版本号</label>
          <div class="input-group">
            <input v-model.trim="version" :placeholder="versionHint" />
          </div>
        </div>

        <!-- URL 模式 -->
        <template v-if="mode === 'url'">
          <div class="form-row">
            <label>下载 URL</label>
            <div class="input-group">
              <textarea v-model="urlsText" rows="3"
                placeholder="每行一个 URL, 按顺序尝试. 例如:&#10;https://example.com/php-8.4.5-nts-Win32-vs17-x64.zip"
                style="font-family: Consolas, monospace; font-size: 12px;"></textarea>
            </div>
          </div>
        </template>

        <!-- 本地文件 -->
        <template v-else>
          <div class="form-row">
            <label>本地文件</label>
            <div class="input-group">
              <input v-model="filePath" readonly placeholder="点右侧浏览..." />
              <button class="btn" @click="pickFile" :disabled="busy">浏览...</button>
            </div>
          </div>
        </template>

        <div class="form-row">
          <label>子目录</label>
          <div class="input-group">
            <input v-model.trim="rootInZip" :placeholder="rootHint" />
          </div>
        </div>
        <div class="form-hint">
          zip 内的顶层子目录名. 例如 nginx 官方包内是 <code>nginx-1.27.3/</code>, MySQL 是
          <code>mysql-8.0.41-winx64/</code>. 留空会自动探测.
        </div>

        <div class="expect-card">
          <div class="expect-head">⚠ 安装后会校验以下文件必须存在 (不符合会报错)</div>
          <ul>
            <li v-for="f in expected" :key="f"><code>bin/{{ destSubdir }}/{{ f }}</code></li>
          </ul>
        </div>

        <div v-if="error" class="error">{{ error }}</div>
      </div>
      <div class="modal-footer">
        <button class="btn" @click="$emit('close')" :disabled="busy">取消</button>
        <button class="btn primary" @click="save" :disabled="busy || !canSave">
          {{ busy ? '保存中...' : '保存 (回到安装对话框选用)' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, computed, onMounted } from 'vue'
const props = defineProps({ kind: String })
const emit = defineEmits(['close', 'saved'])
const api = inject('api')

const kindLabel = { nginx: 'Nginx', php: 'PHP', mysql: 'MySQL', postgresql: 'PostgreSQL', postgres: 'PostgreSQL' }[props.kind] || props.kind
const destSubdir = props.kind === 'postgres' ? 'postgresql' : props.kind

const mode = ref('url')
const version = ref('')
const urlsText = ref('')
const filePath = ref('')
const rootInZip = ref('')
const expected = ref([])
const error = ref('')
const busy = ref(false)

const rootHints = {
  nginx: '例如 nginx-1.27.3',
  php: '一般留空',
  mysql: '例如 mysql-8.0.41-winx64',
  postgresql: '例如 pgsql',
  postgres: '例如 pgsql'
}
const rootHint = computed(() => rootHints[props.kind] || '可留空自动探测')

const versionHints = {
  nginx: '例如 1.27.4-custom',
  php: '例如 8.4.5',
  mysql: '例如 8.0.42',
  postgresql: '例如 17.3',
  postgres: '例如 17.3'
}
const versionHint = computed(() => versionHints[props.kind] || '例如 1.0.0')

const canSave = computed(() => {
  if (!version.value) return false
  if (mode.value === 'url') {
    return urlsText.value.trim().length > 0
  } else {
    return filePath.value.length > 0
  }
})

onMounted(async () => {
  expected.value = await api.ExpectedBinaries(props.kind) || []
})

async function pickFile() {
  busy.value = true
  try {
    const p = await api.PickLocalZip()
    if (p) filePath.value = p
  } catch (e) { error.value = '' + e }
  busy.value = false
}

async function save() {
  busy.value = true
  error.value = ''
  try {
    if (mode.value === 'url') {
      const urls = urlsText.value
        .split('\n')
        .map(l => l.trim())
        .filter(l => l.length > 0)
      if (urls.length === 0) {
        error.value = '至少需要一个 URL'
        return
      }
      await api.AddCustomVersion(props.kind, version.value, urls, rootInZip.value)
    } else {
      await api.AddCustomVersionLocal(props.kind, version.value, filePath.value, rootInZip.value)
    }
    emit('saved')
  } catch (e) {
    error.value = '' + e
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.radio { display: inline-flex; align-items: center; gap: 6px; margin-right: 18px; cursor: pointer; padding-top: 0; }
.radio input { margin: 0; }
textarea {
  width: 100%; min-height: 70px;
  border: 1px solid var(--border); border-radius: 6px;
  padding: 7px 10px; resize: vertical;
}
.form-hint code {
  background: #f0f2f5; padding: 1px 5px; border-radius: 3px;
  font-family: Consolas, monospace; font-size: 11px;
}
.expect-card {
  margin: 12px 0 4px;
  padding: 10px 14px;
  background: #fffbeb;
  border-left: 3px solid var(--warning);
  border-radius: 4px;
  font-size: 12px;
}
.expect-head { color: #856404; font-weight: 600; margin-bottom: 6px; }
.expect-card ul { list-style: none; padding: 0; margin: 0; }
.expect-card li { padding: 1px 0; }
.expect-card code {
  background: #fff3cd; padding: 1px 6px; border-radius: 3px;
  font-family: Consolas, monospace;
}
.error {
  color: var(--danger); margin-top: 12px;
  padding: 8px 10px; background: #fff5f5; border-radius: 4px;
  font-size: 12px; word-break: break-all;
}
</style>
