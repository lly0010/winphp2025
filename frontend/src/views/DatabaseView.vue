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

    <!-- MySQL 数据库 + 账号 管理表 (只在 MySQL 运行时显示) -->
    <div v-if="status.mysql.running" class="manage-section">
      <div class="sec-head">
        <h2 class="sec-title">MySQL 数据库 / 账号</h2>
        <div class="root-pwd-row">
          <label>root 密码</label>
          <input v-model="rootPwd" type="password" placeholder="root 密码 (没设过就留空)" @keyup.enter="refreshAll" />
          <button class="btn" @click="refreshAll" :disabled="loading">{{ loading ? '加载中...' : '刷新' }}</button>
        </div>
      </div>
      <div v-if="loadErr" class="error">{{ loadErr }}</div>

      <!-- 数据库表 -->
      <div class="card list-card">
        <div class="list-head">
          <div class="list-title">数据库 <span class="cnt">{{ userDbCount }}</span></div>
          <button class="btn sm primary" @click="openDbDlg">+ 新建数据库</button>
        </div>
        <table class="table">
          <thead><tr><th style="width: 40%">名称</th><th>字符集</th><th></th></tr></thead>
          <tbody>
            <tr v-for="d in databases" :key="d.name" :class="{ sysrow: d.system }">
              <td><code>{{ d.name }}</code><span v-if="d.system" class="sys-tag">系统</span></td>
              <td>{{ d.charset || '-' }}</td>
              <td style="text-align: right">
                <button class="btn sm danger" :disabled="d.system" @click="dropDatabase(d)">删除</button>
              </td>
            </tr>
            <tr v-if="databases.length === 0">
              <td colspan="3" class="empty">{{ loaded ? '暂无数据库' : '点 "刷新" 加载列表' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 账号表 -->
      <div class="card list-card">
        <div class="list-head">
          <div class="list-title">账号 <span class="cnt">{{ userAccountCount }}</span></div>
          <button class="btn sm primary" @click="openUserDlg">+ 新建账号 (可同时建库)</button>
        </div>
        <table class="table">
          <thead><tr><th style="width: 35%">用户名</th><th>Host</th><th></th></tr></thead>
          <tbody>
            <tr v-for="u in users" :key="u.user + '@' + u.host" :class="{ sysrow: u.system || u.user === 'root' }">
              <td>
                <code>{{ u.user }}</code>
                <span v-if="u.user === 'root'" class="sys-tag root-tag">root</span>
                <span v-else-if="u.system" class="sys-tag">系统</span>
              </td>
              <td><code>{{ u.host }}</code></td>
              <td style="text-align: right">
                <button class="btn sm danger" :disabled="u.system || u.user === 'root'" @click="dropUser(u)">删除</button>
              </td>
            </tr>
            <tr v-if="users.length === 0">
              <td colspan="3" class="empty">{{ loaded ? '暂无账号' : '点 "刷新" 加载列表' }}</td>
            </tr>
          </tbody>
        </table>
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

    <!-- 只建库 -->
    <div v-if="dbDlg" class="modal-mask" @click.self="dbDlg = false">
      <div class="modal">
        <div class="modal-header">新建数据库</div>
        <div class="modal-body">
          <div class="form-row">
            <label>库名</label>
            <div class="input-group"><input v-model="newDbName" placeholder="myapp (仅字母数字下划线)" @keyup.enter="saveDb" /></div>
          </div>
          <div class="form-hint">字符集固定 utf8mb4 / utf8mb4_unicode_ci.</div>
          <div v-if="dbErr" class="error">{{ dbErr }}</div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="dbDlg = false" :disabled="dbBusy">取消</button>
          <button class="btn primary" @click="saveDb" :disabled="dbBusy">{{ dbBusy ? '创建中...' : '创建' }}</button>
        </div>
      </div>
    </div>

    <!-- 创建用户/库 -->
    <div v-if="userDlg" class="modal-mask" @click.self="userDlg = false">
      <div class="modal" style="width: 520px">
        <div class="modal-header">创建 MySQL 账号 / 库</div>
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

const rootPwd = ref('')
const loading = ref(false)
const loaded = ref(false)
const loadErr = ref('')
const databases = ref([])
const users = ref([])

const dbDlg = ref(false)
const newDbName = ref('')
const dbErr = ref('')
const dbBusy = ref(false)

const userDlg = ref(false)
const userBusy = ref(false)
const userErr = ref('')
const userOk = ref('')
const newUser = ref({ user: '', userPwd: '', dbName: '', host: 'localhost' })

const redisDlg = ref(false)
const redisBusy = ref(false)
const redisErr = ref('')
const newRedisPwd = ref('')
const redisPwd = ref('')

const userDbCount = computed(() => databases.value.filter(d => !d.system).length)
const userAccountCount = computed(() => users.value.filter(u => !u.system && u.user !== 'root').length)

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

async function refreshAll() {
  loadErr.value = ''
  loading.value = true
  try {
    databases.value = await api.MysqlListDatabases(rootPwd.value) || []
    users.value = await api.MysqlListUsers(rootPwd.value) || []
    loaded.value = true
  } catch (e) {
    loadErr.value = '加载失败: ' + e + '\n常见原因: root 密码不对.'
    databases.value = []
    users.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadRedis()
  if (props.status?.mysql?.running) refreshAll()
})
watch(() => props.status?.redis?.running, loadRedis)
watch(() => props.status?.mysql?.running, (v) => { if (v) refreshAll() })

async function savePwd() {
  if (newPwd.value.length < 4) { pwdErr.value = '至少 4 位'; return }
  busy.value = true
  try {
    await api.MysqlSetPassword(newPwd.value)
    rootPwd.value = newPwd.value
    pwdDlg.value = false
    pwdErr.value = ''
    await refreshAll()
  }
  catch (e) { pwdErr.value = '' + e }
  finally { busy.value = false }
}

function openDbDlg() {
  newDbName.value = ''
  dbErr.value = ''
  dbDlg.value = true
}

async function saveDb() {
  dbErr.value = ''
  if (!/^[a-zA-Z0-9_]+$/.test(newDbName.value)) { dbErr.value = '库名只能字母数字下划线'; return }
  dbBusy.value = true
  try {
    await api.MysqlCreateDb(newDbName.value, rootPwd.value)
    dbDlg.value = false
    await refreshAll()
  } catch (e) {
    dbErr.value = '' + e
  } finally {
    dbBusy.value = false
  }
}

async function dropDatabase(d) {
  if (d.system) return
  if (!confirm('确定删除数据库 ' + d.name + '? 此操作不可恢复, 库里所有数据都会丢.')) return
  try {
    await api.MysqlDropDatabase(d.name, rootPwd.value)
    await refreshAll()
  } catch (e) {
    alert('删除失败: ' + e)
  }
}

function openUserDlg() {
  userErr.value = ''
  userOk.value = ''
  newUser.value = { user: '', userPwd: '', dbName: '', host: 'localhost' }
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
      rootPwd: rootPwd.value,
      dbName: newUser.value.dbName,
      user: newUser.value.user,
      userPwd: newUser.value.userPwd,
      host: newUser.value.host
    })
    userOk.value = '✓ 已创建 ' + newUser.value.user + '@' + newUser.value.host +
      (newUser.value.dbName ? ' 并授权库 ' + newUser.value.dbName : '')
    await refreshAll()
  } catch (e) {
    userErr.value = '' + e
  } finally {
    userBusy.value = false
  }
}

async function dropUser(u) {
  if (u.system || u.user === 'root') return
  if (!confirm('确定删除账号 ' + u.user + '@' + u.host + '?')) return
  try {
    await api.MysqlDropUser(u.user, u.host, rootPwd.value)
    await refreshAll()
  } catch (e) {
    alert('删除失败: ' + e)
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

.manage-section { margin-top: 24px; }
.sec-head { display: flex; justify-content: space-between; align-items: flex-end; margin-bottom: 12px; gap: 12px; flex-wrap: wrap; }
.sec-title { font-size: 18px; font-weight: 600; margin: 0; color: var(--text); }
.root-pwd-row { display: flex; align-items: center; gap: 8px; }
.root-pwd-row label { font-size: 12px; color: var(--text-secondary); }
.root-pwd-row input { padding: 6px 10px; border: 1px solid var(--border); border-radius: 6px; font-size: 13px; width: 220px; }

.list-card { margin-bottom: 14px; }
.list-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.list-title { font-size: 14px; font-weight: 600; color: var(--text); }
.list-title .cnt {
  display: inline-block; margin-left: 6px;
  font-size: 11px; padding: 1px 8px; border-radius: 10px;
  background: var(--primary-light); color: var(--primary-dark); font-weight: normal;
}
.table { width: 100%; border-collapse: collapse; }
.table th, .table td { padding: 8px 10px; border-bottom: 1px solid var(--border-soft); font-size: 13px; text-align: left; }
.table th { color: var(--text-secondary); font-weight: 500; font-size: 12px; }
.table tr.sysrow { color: var(--text-secondary); background: rgba(0,0,0,0.02); }
.table .empty { text-align: center; color: var(--text-secondary); padding: 24px; }
.table code { background: #f0f2f5; padding: 1px 6px; border-radius: 3px; font-family: Consolas, monospace; font-size: 12px; }
.sys-tag {
  display: inline-block; margin-left: 6px;
  font-size: 10px; padding: 1px 6px; border-radius: 8px;
  background: rgba(180,180,180,0.18); color: #777;
}
.sys-tag.root-tag { background: rgba(255,111,158,0.18); color: var(--primary-dark); }
</style>
