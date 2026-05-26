<template>
  <div class="modal-mask">
    <div class="modal" style="width: 560px">
      <div class="modal-header">新建 PHP 网站</div>
      <div class="modal-body">
        <div class="form-row">
          <label>站点名称</label>
          <div class="input-group"><input v-model="name" placeholder="myblog" @input="onNameInput" /></div>
        </div>
        <div class="form-hint">(用作 vhost 文件名 + 默认目录名)</div>

        <div class="form-row">
          <label>域名</label>
          <div class="input-group"><input v-model="serverName" @input="serverManual = true" placeholder="myblog.local" /></div>
        </div>

        <div class="form-row">
          <label>端口</label>
          <div class="input-group" style="max-width: 120px"><input v-model.number="port" type="number" /></div>
        </div>

        <div class="form-row">
          <label>根目录</label>
          <div class="input-group">
            <input v-model="root" @input="rootManual = true" placeholder="www/myblog" />
          </div>
        </div>

        <div class="form-row">
          <label>网站类型</label>
          <div class="input-group">
            <select v-model="template">
              <option value="php">普通 PHP 项目 (index.php)</option>
              <option value="laravel">Laravel / ThinkPHP / Symfony (root 指向 public/)</option>
              <option value="wordpress">WordPress (空目录, 自行解压)</option>
              <option value="static">纯静态 HTML (index.html)</option>
            </select>
          </div>
        </div>

        <div class="form-row">
          <label>PHP 版本</label>
          <div class="input-group">
            <span :class="phpVersion ? 'ok' : 'warn'">
              {{ phpVersion ? phpVersion + ' (127.0.0.1:9000)' : '未安装 - 请先到首页安装 PHP' }}
            </span>
          </div>
        </div>

        <div class="form-row">
          <label>数据库</label>
          <div class="input-group" style="flex-direction: column; align-items: stretch; gap: 6px;">
            <label class="cb"><input type="checkbox" v-model="createDb" /> 同时创建 MySQL 数据库 (名为 <code>{{ dbName }}</code>)</label>
            <input v-model="dbPwd" type="password" placeholder="MySQL root 密码 (留空表示无)" v-show="createDb" />
          </div>
        </div>

        <div class="form-row">
          <label></label>
          <div class="input-group" style="flex-direction: column; align-items: flex-start; gap: 6px;">
            <label class="cb"><input type="checkbox" v-model="addHosts" /> 自动写入 hosts 文件 (需管理员)</label>
            <label class="cb"><input type="checkbox" v-model="reload" /> 创建后立即 nginx reload</label>
          </div>
        </div>

        <div v-if="error" class="error">{{ error }}</div>
      </div>
      <div class="modal-footer">
        <button class="btn" @click="$emit('close')" :disabled="submitting">取消</button>
        <button class="btn primary" @click="submit" :disabled="submitting">
          {{ submitting ? '创建中...' : '创建站点' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, computed, onMounted } from 'vue'
const props = defineProps({ presetTemplate: { type: String, default: 'php' }, phpVersion: String, mysqlRunning: Boolean })
const emit = defineEmits(['close', 'added'])
const api = inject('api')

const name = ref('')
const serverName = ref('')
const port = ref(80)
const root = ref('')
const template = ref(props.presetTemplate)
const createDb = ref(props.presetTemplate === 'wordpress')
const dbPwd = ref('')
const addHosts = ref(true)
const reload = ref(true)
const submitting = ref(false)
const error = ref('')
const rootManual = ref(false)
const serverManual = ref(false)

let wwwDir = ''
onMounted(async () => {
  const p = await api.GetPaths()
  wwwDir = p.wwwDir
})

const dbName = computed(() => (name.value || '').replace(/[^a-zA-Z0-9_]/g, '_'))

function onNameInput() {
  if (!serverManual.value) serverName.value = name.value ? name.value + '.local' : ''
  if (!rootManual.value && wwwDir) root.value = wwwDir + '\\' + name.value
}

async function submit() {
  if (!name.value) { error.value = '请填写站点名'; return }
  if (!serverName.value) { error.value = '请填写域名'; return }
  if (createDb.value && !props.mysqlRunning) {
    error.value = 'MySQL 未运行, 无法自动创建数据库. 请先启动 MySQL 或取消勾选.'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    await api.AddSite({
      name: name.value,
      serverName: serverName.value,
      root: root.value,
      port: Number(port.value) || 80,
      template: template.value,
      addHosts: addHosts.value
    })
    if (createDb.value) {
      try { await api.MysqlCreateDb(dbName.value, dbPwd.value) }
      catch (e) { console.warn('createDb failed:', e); error.value = '数据库创建失败 (站点已创建): ' + e; return }
    }
    if (reload.value) await api.NginxReload()
    emit('added')
  } catch (e) {
    error.value = '' + e
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.cb { display: flex; align-items: center; gap: 6px; cursor: pointer; }
.ok { color: var(--success); }
.warn { color: var(--danger); }
.error { color: var(--danger); margin-top: 12px; padding: 8px 10px; background: #fff5f5; border-radius: 4px; font-size: 12px; }
code { background: #f0f2f5; padding: 1px 6px; border-radius: 3px; }
</style>
