<template>
  <div>
    <h1 class="page-title">网站</h1>
    <div class="actions">
      <button class="btn primary" @click="dlgOpen = true">+ 新建网站</button>
      <button class="btn" @click="refresh">刷新</button>
      <button class="btn" @click="api.NginxReload()">Nginx reload</button>
    </div>
    <div class="card" style="margin-top: 14px;">
      <table class="table">
        <thead><tr><th>名称</th><th>域名</th><th>端口</th><th>根目录</th><th>类型</th><th>创建</th><th></th></tr></thead>
        <tbody>
          <tr v-for="s in sites" :key="s.name">
            <td>{{ s.name }}</td>
            <td>{{ s.serverName }}</td>
            <td>{{ s.port }}</td>
            <td><code>{{ s.root }}</code></td>
            <td>{{ tplLabel(s.template) }}</td>
            <td><span class="muted">{{ s.createdAt }}</span></td>
            <td style="text-align: right; white-space: nowrap;">
              <button class="btn sm" @click="visit(s)">浏览</button>
              <button class="btn sm" @click="openDir(s)">目录</button>
              <button class="btn sm" @click="editVhost(s)">vhost</button>
              <button class="btn sm danger" @click="del(s)">删除</button>
            </td>
          </tr>
          <tr v-if="sites.length === 0">
            <td colspan="7" style="text-align: center; color: var(--text-secondary); padding: 30px;">
              暂无站点
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <AddSiteDialog v-if="dlgOpen" :php-version="status.php.version" :mysql-running="status.mysql.running" @close="dlgOpen = false" @added="onAdded" />
    <ConfigEditor v-if="vhostName" :ckey="'vhost:' + vhostName" :title="'vhost: ' + vhostName" @close="vhostName = null" />
  </div>
</template>

<script setup>
import { inject, ref, onMounted } from 'vue'
import AddSiteDialog from '../components/AddSiteDialog.vue'
import ConfigEditor from '../components/ConfigEditor.vue'
const api = inject('api')
const status = inject('status')
const sites = ref([])
const dlgOpen = ref(false)
const vhostName = ref(null)

const tplMap = { php: '普通 PHP', laravel: 'Laravel', wordpress: 'WordPress', static: '纯静态' }
const tplLabel = (t) => tplMap[t] || t || '普通 PHP'

async function refresh() { sites.value = await api.ListSites() || [] }
onMounted(refresh)

function visit(s) {
  let url = 'http://' + s.serverName
  if (s.port !== 80) url += ':' + s.port
  api.OpenInBrowser(url)
}
async function openDir(s) { await api.OpenFolder(s.root) }
function editVhost(s) { vhostName.value = s.name }
async function del(s) {
  if (!confirm('删除站点 ' + s.name + '? (根目录文件不会被删除)')) return
  await api.RemoveSite(s.name, true)
  refresh()
}
async function onAdded() { dlgOpen.value = false; await refresh() }
</script>

<style scoped>
.actions { display: flex; gap: 8px; }
.muted { color: var(--text-secondary); font-size: 12px; }
code { font-family: Consolas, monospace; font-size: 12px; }
.page-title {
  font-size: 22px; font-weight: 700; margin: 0 0 18px;
  background: var(--header-grad);
  -webkit-background-clip: text; background-clip: text; color: transparent;
}
</style>
