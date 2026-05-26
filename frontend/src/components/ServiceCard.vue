<template>
  <div class="svc-card">
    <div class="head">
      <div class="name">{{ label }}</div>
      <span class="status" :class="{ on: status?.running }">
        <i class="dot" :class="status?.running ? 'on' : 'off'"></i>
        {{ status?.running ? '运行中' : '未运行' }}
      </span>
    </div>
    <div class="info">
      <div><span class="lbl">版本</span>{{ status?.version || '未安装' }}</div>
      <div><span class="lbl">端口</span>{{ status?.port }}</div>
      <div><span class="lbl">服务</span>
        <template v-if="status?.serviceInstalled">
          <span style="color: var(--success)">已注册 ({{ status.serviceStatus }})</span>
        </template>
        <template v-else><span class="muted">未注册</span></template>
      </div>
    </div>
    <div class="row">
      <button class="btn" :disabled="!status?.binInstalled || busy.start"
              @click="run('Start')">{{ busy.start ? '...' : '启动' }}</button>
      <button class="btn" :disabled="!status?.binInstalled || busy.stop"
              @click="run('Stop')">{{ busy.stop ? '...' : '停止' }}</button>
      <button class="btn" :disabled="!status?.binInstalled || busy.restart"
              @click="run('Restart')">{{ busy.restart ? '...' : '重启' }}</button>
    </div>
    <div class="row">
      <button class="btn" style="flex: 2" @click="$emit('install')">安装 / 切换版本</button>
      <button class="btn danger" :disabled="!status?.binInstalled" @click="$emit('uninstall')">卸载</button>
      <button class="btn" :disabled="!status?.binInstalled" @click="$emit('config')">配置</button>
    </div>
    <div class="row">
      <button class="btn custom-btn" @click="$emit('custom')" title="添加自定义版本: 自己的下载 URL 或本地 zip 文件">
        🛠 自定义版本
      </button>
    </div>
    <button class="btn auto-btn" :class="{ active: status?.serviceInstalled }"
            :disabled="!status?.binInstalled" @click="$emit('autostart')">
      {{ status?.serviceInstalled ? '✓ 已设为开机自启 (点击取消)' : '注册为开机自启服务' }}
    </button>
  </div>
</template>

<script setup>
import { inject, reactive } from 'vue'
const props = defineProps({ kind: String, label: String, status: Object })
const emit = defineEmits(['install', 'uninstall', 'config', 'autostart', 'custom'])
const api = inject('api')

const busy = reactive({ start: false, stop: false, restart: false })
async function run(action) {
  const key = action.toLowerCase()
  busy[key] = true
  try {
    if (action === 'Start') await api.StartService(props.kind)
    else if (action === 'Stop') await api.StopService(props.kind)
    else if (action === 'Restart') await api.RestartService(props.kind)
  } catch (e) { alert(action + ' 失败: ' + e) }
  finally { busy[key] = false }
}
</script>

<style scoped>
.svc-card {
  background: var(--bg-card); border-radius: var(--radius);
  box-shadow: var(--shadow); padding: 16px;
  display: flex; flex-direction: column; gap: 10px;
}
.head { display: flex; align-items: center; justify-content: space-between; }
.name { font-size: 17px; font-weight: 600; color: var(--primary); }
.status {
  font-size: 12px; color: var(--text-secondary);
  padding: 2px 8px; border-radius: 10px;
  background: #f0f2f5;
}
.status.on { color: var(--success); background: rgba(60,170,60,0.08); }
.info {
  font-size: 12px; line-height: 1.7; color: var(--text-secondary);
  border-top: 1px solid var(--border); padding-top: 8px;
}
.info .lbl { display: inline-block; width: 38px; color: #a8aeb5; }
.muted { color: #a8aeb5; }

.row { display: flex; gap: 6px; }
.row .btn { flex: 1; padding: 6px 0; font-size: 13px; }

.custom-btn {
  background: #fff8e6; border-color: #f5d27a; color: #8a6611;
  font-size: 12px; padding: 7px 0;
}
.custom-btn:hover { background: #fff0c2; border-color: #e0b84a; color: #6b4f0a; }

.auto-btn {
  margin-top: 4px; padding: 8px 0; font-size: 12px;
}
.auto-btn.active { color: var(--success); border-color: rgba(60,170,60,0.3); }
</style>
