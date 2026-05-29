<template>
  <div>
    <h1 class="page-title">数据库</h1>
    <div class="db-grid">
      <div class="card">
        <div class="card-title">MySQL <span class="muted">{{ status.mysql.version || '未安装' }}</span></div>
        <div class="info">
          <div>状态: <span :class="status.mysql.running ? 'on' : 'off'">{{ status.mysql.running ? '运行中' : '未运行' }}</span></div>
          <div>端口: 3306 (绑定 127.0.0.1)</div>
          <div>默认账号: <code>root</code> / (空密码)</div>
          <div>数据目录: <code>bin/mysql/data</code></div>
        </div>
        <div class="actions">
          <button class="btn primary" @click="pwdDlg = true">修改 root 密码</button>
          <button class="btn" @click="openUserDlg">创建用户/库</button>
          <button class="btn" @click="$emit('cli', 'mysql')">命令行</button>
        </div>
      </div>

      <div class="card">
        <div class="card-title">PostgreSQL <span class="muted">{{ status.postgres.version || '未安装' }}</span></div>
        <div class="info">
          <div>状态: <span :class="status.postgres.running ? 'on' : 'off'">{{ status.postgres.running ? '运行中' : '未运行' }}</span></div>
          <div>端口: 5432</div>
          <div>默认账号: <code>postgres</code> / (空密码, trust 认证)</div>
          <div>数据目录: <code>bin/postgresql/data</code></div>
        </div>
        <div class="actions">
          <button class="btn" @click="$emit('cli', 'psql')">命令行 (psql)</button>
        </div>
        <div class="tip">
          修改密码: 在 psql 里执行<br>
          <code>ALTER USER postgres WITH PASSWORD '新密码';</code><br>
          然后把 pg_hba.conf 的 trust 改成 scram-sha-256 重启服务.
        </div>
      </div>

      <div class="card">
        <div class="card-title">Redis <span class="muted">{{ status.redis.version || '未安装' }}</span></div>
        <div class="info">
          <div>状态: <span :class="status.redis.running ? 'on' : 'off'">{{ status.redis.running ? '运行中' : '未运行' }}</span></div>
          <div>端口: 6379 (绑定 127.0.0.1)</div>
          <div>密码: <span :class="hasRedisPwd ? 'on' : 'off'">{{ hasRedisPwd ? '已设置 (' + redisPwdMask + ')' : '未设置' }}</span></div>
          <div>配置文件: <code>bin/redis/redis.windows.conf</code></div>
        </div>
        <div class="actions">
          <button class="btn primary" @click="openRedisDlg">{{ hasRedisPwd ? '修改密码' : '设置密码' }}</button>
          <button v-if="hasRedisPwd" class="btn danger" @click="clearRedisPwd" :disabled="redisBusy">移除密码</button>
        </div>
      </div>
    </div>

    <!-- MySQL root 密码 -->
    <div v-if="pwdDlg" class="modal-mask" @click.self="pwdDlg = false">
      <div class="modal">
        <div class="modal-header">修改 MySQL root 密码</div>
        <div class="modal-body">
          <div class="form-row">
            <label>新密码</label>
            <div class="input-group"><input v-model="newPwd" type="password" placeholder="至少 4 位" /></div>
          </div>
          <div v-if="pwdErr" class="error">{{ pwdErr }}</div>
          <div class="hint">只能用于从空密码改起 (首次安装的状态). 已设过密码请用 mysql 命令行修改.</div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="pwdDlg = false">取消</button>
          <button class="btn primary" @click="savePwd" :disabled="busy">{{ busy ? '...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <!-- MySQL 创建用户/库 -->
    <div v-if="userDlg" class="modal-mask" @click.self="userDlg = false">
      <div class="modal" style="width: 520px">
        <div class="modal-header">创建 MySQL 用户和库</div>
        <div class="modal-body">
          <div class="form-row">
            <label>用户名</label>
            <div class="input-group"><input v-model="newUser.user" placeholder="myapp (仅字母数字下划线)" /></div>
          </div>
          <div class="form-row">
            <label>用户密码</label>
            <div class="input-group"><input v-model="newUser.userPwd" type="password" placeholder="给这个用户设的密码" /></div>
          </div>
          <div class="form-row">
            <label>关联数据库</label>
            <div class="input-group"><input v-model="newUser.dbName" placeholder="留空则只建用户不建库" /></div>
          </div>
          <div class="form-hint">填了库名会同时 <code>CREATE DATABASE</code> 并把整个库 GRANT 给这个用户.</div>
          <div class="form-row">
            <label>连接 host</label>
            <div class="input-group">
              <select v-model="newUser.host" style="flex: 1">
                <option value="localhost">localhost (仅本机)</option>
                <option value="%">% (任意来源)</option>
                <option value="127.0.0.1">127.0.0.1</option>
              </select>
            </div>
          </div>
          <div class="form-row">
            <label>root 密码</label>
            <div class="input-group"><input v-model="newUser.rootPwd" type="password" placeholder="MySQL root 密码 (没设过就留空)" /></div>
          </div>
          <div v-if="userErr" class="error">{{ userErr }}</div>
          <div v-if="userOk" class="ok-msg">{{ userOk }}</div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="userDlg = false" :disabled="userBusy">关闭</button>
          <button class="btn primary" @click="saveUser" :disabled="userBusy">{{ userBusy ? '创建中...' : '创建' }}</button>
        </div>
      </div>
    </div>

    <!-- Redis 密码 -->
    <div v-if="redisDlg" class="modal-mask" @click.self="redisDlg = false">
      <div class="modal">
        <div class="modal-header">{{ hasRedisPwd ? '修改 Redis 密码' : '设置 Redis 密码' }}</div>
        <div class="modal-body">
          <div class="form-row">
            <label>新密码</label>
            <div class="input-group"><input v-model="newRedisPwd" type="password" placeholder="留空 = 移除密码" /></div>
          </div>
          <div class="hint">
            写入 <code>redis.windows.conf</code> 的 <code>requirepass</code>.
            <strong>Redis 在运行的话立即生效</strong> (面板会用 CONFIG SET 同步), 不需要重启.
            <br>之后用 <code>redis-cli -a "密码" ...</code> 连接.
          </div>
          <div v-if="redisErr" class="error">{{ redisErr }}</div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="redisDlg = false" :disabled="redisBusy">取消</button>
          <button class="btn primary" @click="saveRedisPwd" :disabled="redisBusy">{{ redisBusy ? '...' : '保存' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, computed, onMounted, watch } from 'vue'
const props = defineProps({ status: Object })
const api = inject('api')

const pwdDlg = ref(false)
const newPwd = ref('')
const pwdErr = ref('')
const busy = ref(false)

const userDlg = ref(false)
const userBusy = ref(false)
const userErr = ref('')
const userOk = ref('')
const newUser = ref({ user: '', userPwd: '', dbName: '', host: 'localhost', rootPwd: '' })

const redisDlg = ref(false)
const redisBusy = ref(false)
const redisErr = ref('')
const newRedisPwd = ref('')
const redisPwd = ref('')

const hasRedisPwd = computed(() => !!redisPwd.value)
const redisPwdMask = computed(() => {
  if (!redisPwd.value) return ''
  const n = redisPwd.value.length
  if (n <= 2) return '*'.repeat(n)
  return redisPwd.value[0] + '*'.repeat(Math.max(1, n - 2)) + redisPwd.value.slice(-1)
})

async function loadRedis() {
  try { redisPwd.value = await api.RedisGetPassword() || '' }
  catch { redisPwd.value = '' }
}
onMounted(loadRedis)
watch(() => props.status?.redis?.running, loadRedis)

async function savePwd() {
  if (newPwd.value.length < 4) { pwdErr.value = '至少 4 位'; return }
  busy.value = true
  try { await api.MysqlSetPassword(newPwd.value); pwdDlg.value = false; pwdErr.value = '' }
  catch (e) { pwdErr.value = '' + e }
  finally { busy.value = false }
}

function openUserDlg() {
  userErr.value = ''
  userOk.value = ''
  userDlg.value = true
}

async function saveUser() {
  userErr.value = ''
  userOk.value = ''
  if (!newUser.value.user) { userErr.value = '请填用户名'; return }
  if (!/^[a-zA-Z0-9_]+$/.test(newUser.value.user)) { userErr.value = '用户名只能字母数字下划线'; return }
  if (newUser.value.dbName && !/^[a-zA-Z0-9_]+$/.test(newUser.value.dbName)) { userErr.value = '库名只能字母数字下划线'; return }
  userBusy.value = true
  try {
    await api.MysqlCreateUser({
      rootPwd: newUser.value.rootPwd,
      dbName: newUser.value.dbName,
      user: newUser.value.user,
      userPwd: newUser.value.userPwd,
      host: newUser.value.host
    })
    userOk.value = '✓ 已创建 ' + newUser.value.user + '@' + newUser.value.host +
      (newUser.value.dbName ? ' 并授权库 ' + newUser.value.dbName : '')
  } catch (e) {
    userErr.value = '' + e
  } finally {
    userBusy.value = false
  }
}

function openRedisDlg() {
  redisErr.value = ''
  newRedisPwd.value = redisPwd.value || ''
  redisDlg.value = true
}

async function saveRedisPwd() {
  redisErr.value = ''
  redisBusy.value = true
  try {
    await api.RedisSetPassword(newRedisPwd.value)
    await loadRedis()
    redisDlg.value = false
  } catch (e) {
    redisErr.value = '' + e
  } finally {
    redisBusy.value = false
  }
}

async function clearRedisPwd() {
  if (!confirm('确定移除 Redis 密码? 移除后无需密码即可访问.')) return
  redisBusy.value = true
  try {
    await api.RedisSetPassword('')
    await loadRedis()
  } catch (e) {
    alert('移除失败: ' + e)
  } finally {
    redisBusy.value = false
  }
}
</script>

<style scoped>
.page-title {
  font-size: 22px; font-weight: 700; margin: 0 0 18px;
  background: var(--header-grad);
  -webkit-background-clip: text; background-clip: text; color: transparent;
}
.db-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 14px; }
.card-title { font-size: 16px; font-weight: 600; margin-bottom: 12px; }
.muted { color: var(--text-secondary); font-weight: normal; font-size: 13px; margin-left: 8px; }
.info { line-height: 1.9; color: var(--text-secondary); font-size: 13px; margin-bottom: 14px; }
.info .on { color: var(--success); }
.info .off { color: #c0c5cc; }
.info code { background: #f0f2f5; padding: 1px 6px; border-radius: 3px; font-family: Consolas, monospace; }
.actions { display: flex; gap: 8px; flex-wrap: wrap; }
.tip { margin-top: 12px; padding: 10px; background: #fffbeb; border-radius: 4px; font-size: 12px; color: #856404; line-height: 1.6; }
.tip code { background: #fff3cd; padding: 1px 4px; border-radius: 3px; }
.hint { margin-top: 10px; padding: 10px 12px; font-size: 12px; color: var(--text-secondary); background: #fffbeb; border-left: 3px solid var(--warning); border-radius: 4px; line-height: 1.7; }
.hint code { background: #fff3cd; padding: 1px 5px; border-radius: 3px; font-family: Consolas, monospace; }
.hint strong { color: var(--danger); }
.error { color: var(--danger); margin-top: 10px; padding: 8px 10px; background: #fff5f5; border-radius: 4px; font-size: 12px; word-break: break-all; white-space: pre-wrap; }
.ok-msg { color: #2d7a2d; margin-top: 10px; padding: 8px 10px; background: rgba(95,203,111,0.10); border-radius: 4px; font-size: 12px; }
.form-hint { font-size: 11px; color: var(--text-secondary); margin: -4px 0 8px 110px; }
.form-hint code { background: #f0f2f5; padding: 1px 4px; border-radius: 3px; font-family: Consolas, monospace; }
</style>
