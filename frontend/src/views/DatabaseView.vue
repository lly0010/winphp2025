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
    </div>

    <!-- 修改密码对话框 -->
    <div v-if="pwdDlg" class="modal-mask">
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
  </div>
</template>

<script setup>
import { inject, ref } from 'vue'
const props = defineProps({ status: Object })
const api = inject('api')

const pwdDlg = ref(false)
const newPwd = ref('')
const pwdErr = ref('')
const busy = ref(false)

async function savePwd() {
  if (newPwd.value.length < 4) { pwdErr.value = '至少 4 位'; return }
  busy.value = true
  try { await api.MysqlSetPassword(newPwd.value); pwdDlg.value = false; pwdErr.value = '' }
  catch (e) { pwdErr.value = '' + e }
  finally { busy.value = false }
}
</script>

<style scoped>
.page-title {
  font-size: 22px; font-weight: 700; margin: 0 0 18px;
  background: var(--header-grad);
  -webkit-background-clip: text; background-clip: text; color: transparent;
}
.db-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.card-title { font-size: 16px; font-weight: 600; margin-bottom: 12px; }
.muted { color: var(--text-secondary); font-weight: normal; font-size: 13px; margin-left: 8px; }
.info { line-height: 1.9; color: var(--text-secondary); font-size: 13px; margin-bottom: 14px; }
.info .on { color: var(--success); }
.info .off { color: #c0c5cc; }
.info code { background: #f0f2f5; padding: 1px 6px; border-radius: 3px; font-family: Consolas, monospace; }
.actions { display: flex; gap: 8px; }
.tip { margin-top: 12px; padding: 10px; background: #fffbeb; border-radius: 4px; font-size: 12px; color: #856404; line-height: 1.6; }
.tip code { background: #fff3cd; padding: 1px 4px; border-radius: 3px; }
.hint { margin-top: 10px; font-size: 12px; color: var(--text-secondary); }
.error { color: var(--danger); margin-top: 10px; }
</style>
