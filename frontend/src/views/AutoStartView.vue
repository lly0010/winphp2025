<template>
  <div>
    <h1 class="page-title">开机自启</h1>

    <div class="card">
      <div class="bigops">
        <button class="btn primary lg" @click="enableAll" :disabled="busy">✓ 一键启用全部</button>
        <button class="btn lg" @click="disableAll" :disabled="busy">✗ 一键禁用全部</button>
        <button class="btn lg" @click="pickNssm">手动指定 nssm.exe...</button>
      </div>
      <p class="hint">
        首次启用会自动下载 NSSM (~200KB) 到 bin/. 如果网络访问 nssm.cc 失败 (国内可能 503),
        可点"手动指定" 选择已下载的 nssm.exe.
      </p>
    </div>

    <div class="card" style="margin-top: 14px;">
      <table class="table">
        <thead><tr><th>项目</th><th>状态</th><th></th></tr></thead>
        <tbody>
          <tr v-for="i in items" :key="i.key">
            <td>
              <strong>{{ i.label }}</strong>
              <span v-if="!i.binReady && i.key !== 'panel'" class="warn">(组件未安装)</span>
            </td>
            <td>
              <span v-if="i.installed" class="ok">✓ 已启用{{ i.running ? ' (运行中)' : '' }}</span>
              <span v-else class="muted">✗ 未启用</span>
            </td>
            <td style="text-align: right;">
              <button class="btn sm" :disabled="!i.binReady && i.key !== 'panel'" @click="toggle(i)">
                {{ i.installed ? '禁用' : '启用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, onMounted, onUnmounted } from 'vue'
const api = inject('api')
const items = ref([])
const busy = ref(false)
let timer

async function refresh() {
  items.value = await api.AutoStartList() || []
}

async function toggle(i) {
  busy.value = true
  try {
    if (i.installed) await api.DisableAutoStart(i.key)
    else await api.EnableAutoStart(i.key)
    await refresh()
  } catch (e) { alert('操作失败: ' + e) }
  finally { busy.value = false }
}

async function enableAll() {
  busy.value = true
  try { await api.EnableAllAutoStart(); await refresh() }
  catch (e) { alert(e) }
  finally { busy.value = false }
}

async function disableAll() {
  if (!confirm('确认禁用全部开机自启? 会卸载已注册的 Windows 服务.')) return
  busy.value = true
  try { await api.DisableAllAutoStart(); await refresh() }
  catch (e) { alert(e) }
  finally { busy.value = false }
}

async function pickNssm() {
  try { await api.PickAndInstallNssm() }
  catch (e) { alert(e) }
}

onMounted(() => { refresh(); timer = setInterval(refresh, 3000) })
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.page-title { font-size: 20px; font-weight: 600; margin: 0 0 16px; }
.bigops { display: flex; gap: 10px; margin-bottom: 14px; }
.bigops .btn { padding: 12px 22px; }
.hint { margin: 0; font-size: 12px; color: var(--text-secondary); }
.ok { color: var(--success); }
.muted { color: var(--text-secondary); }
.warn { color: var(--danger); font-size: 12px; margin-left: 6px; }
</style>
