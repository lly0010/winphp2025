<template>
  <div class="svc-card" :class="'kind-' + kind">
    <div class="head">
      <div class="name">
        <span class="emoji">{{ kindIcon }}</span>
        {{ label }}
      </div>
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
import { inject, reactive, computed } from 'vue'
const props = defineProps({ kind: String, label: String, status: Object })
const emit = defineEmits(['install', 'uninstall', 'config', 'autostart', 'custom'])
const api = inject('api')

const iconMap = {
  nginx: '🌐',
  php: '🐘',
  mysql: '🐬',
  postgres: '🌳',
  postgresql: '🌳',
  redis: '📦'
}
const kindIcon = computed(() => iconMap[props.kind] || '✿')

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
  border: 1px solid var(--border-soft);
  backdrop-filter: blur(8px);
  transition: all 0.25s cubic-bezier(.4,.2,.2,1);
  position: relative; overflow: hidden;
}
.svc-card::before {
  content: ''; position: absolute;
  top: 0; left: 0; right: 0; height: 3px;
  background: var(--header-grad);
  opacity: 0.65;
  transition: opacity 0.25s;
}
.svc-card:hover {
  transform: translateY(-3px);
  box-shadow: var(--shadow-hover);
  border-color: rgba(255, 111, 158, 0.30);
}
.svc-card:hover::before { opacity: 1; }
/* 每个 kind 自己的顶部色带 */
.svc-card.kind-nginx::before    { background: linear-gradient(90deg, #5fcb6f, #b6e2bb); }
.svc-card.kind-php::before      { background: linear-gradient(90deg, #8993be, #b06fff); }
.svc-card.kind-mysql::before    { background: linear-gradient(90deg, #4a8fd6, #67c2f5); }
.svc-card.kind-postgres::before { background: linear-gradient(90deg, #5680b9, #336791); }
.svc-card.kind-redis::before    { background: linear-gradient(90deg, #d82c20, #ff6b5e); }

.head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.name {
  font-size: 17px; font-weight: 700;
  background: var(--header-grad);
  -webkit-background-clip: text; background-clip: text;
  color: transparent;
  display: flex; align-items: center; gap: 6px;
}
.emoji {
  font-size: 20px;
  filter: drop-shadow(0 2px 4px rgba(0,0,0,0.08));
  display: inline-block;
}
.svc-card:hover .emoji {
  animation: emoji-bounce 0.6s ease-out;
}
@keyframes emoji-bounce {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  30%      { transform: translateY(-4px) rotate(-8deg); }
  60%      { transform: translateY(-2px) rotate(6deg); }
}

.status {
  font-size: 12px; color: var(--text-secondary);
  padding: 3px 10px; border-radius: 12px;
  background: rgba(0,0,0,0.04);
  white-space: nowrap;
}
.status.on { color: #2d7a2d; background: rgba(95, 203, 111, 0.14); }

.info {
  font-size: 12px; line-height: 1.85; color: var(--text-secondary);
  border-top: 1px dashed var(--border); padding-top: 8px;
}
.info .lbl {
  display: inline-block; width: 38px; color: #b3a8c0;
  font-size: 11px;
}
.muted { color: #b3a8c0; }

.row { display: flex; gap: 6px; }
.row .btn { flex: 1; padding: 6px 0; font-size: 13px; }

.custom-btn {
  background: linear-gradient(135deg, #fff4e0, #ffe9d6);
  border-color: #f5d27a; color: #b8762e;
  font-size: 12px; padding: 8px 0;
}
.custom-btn:hover {
  background: linear-gradient(135deg, #ffe9c8, #ffd9b0);
  border-color: #e0b84a; color: #8a5a1a;
}

.auto-btn { margin-top: 4px; padding: 9px 0; font-size: 12px; }
.auto-btn.active {
  color: #2d7a2d; border-color: rgba(95, 203, 111, 0.35);
  background: rgba(95, 203, 111, 0.06);
}
</style>
