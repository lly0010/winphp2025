<template>
  <div class="home">
    <h1 class="page-title">控制台</h1>

    <div class="svc-grid">
      <ServiceCard kind="nginx"     label="Nginx"       :status="status.nginx"    @install="openInstall('nginx')"
                   @uninstall="confirmUninstall('nginx')" @config="editConfig('nginx', 'nginx.conf')"
                   @autostart="toggleAuto('nginx')" />
      <ServiceCard kind="php"       label="PHP-CGI"     :status="status.php"      @install="openInstall('php')"
                   @uninstall="confirmUninstall('php')" @config="editConfig('php', 'php.ini')"
                   @autostart="toggleAuto('php')" />
      <ServiceCard kind="mysql"     label="MySQL"       :status="status.mysql"    @install="openInstall('mysql')"
                   @uninstall="confirmUninstall('mysql')" @config="editConfig('mysql', 'my.ini')"
                   @autostart="toggleAuto('mysql')" />
      <ServiceCard kind="postgres"  label="PostgreSQL"  :status="status.postgres" @install="openInstall('postgresql')"
                   @uninstall="confirmUninstall('postgres')" @config="editConfig('postgres', 'postgresql.conf')"
                   @autostart="toggleAuto('postgres')" />
    </div>

    <div class="home-grid">
      <div class="card">
        <div class="card-title">快速操作</div>
        <div class="quick-grid">
          <button class="btn primary lg col-2" @click="addSite('php')">+ 新建 PHP 网站</button>
          <button class="btn lg col-2" @click="addSite('wordpress')">WordPress 站点</button>
          <button class="btn" @click="api.OpenInBrowser('http://localhost')">浏览 localhost</button>
          <button class="btn" @click="api.OpenInBrowser('http://localhost/phpinfo.php')">phpinfo</button>
          <button class="btn" @click="openFolder('www')">www 目录</button>
          <button class="btn" @click="openFolder('logs')">日志目录</button>
          <button class="btn" @click="openFolder('root')">面板根目录</button>
          <button class="btn" @click="editConfig('hosts', 'hosts')">编辑 hosts</button>
          <button class="btn" @click="api.NginxReload()">Nginx reload</button>
        </div>
      </div>

      <div class="card">
        <div class="card-title">我的网站
          <span class="muted">({{ sites.length }})</span>
          <button class="btn sm" style="float:right" @click="$emit('goto', 'sites')">管理 →</button>
        </div>
        <div v-if="sites.length === 0" class="empty">
          暂无站点, 点 "+ 新建 PHP 网站" 创建你的第一个网站
        </div>
        <table v-else class="table">
          <thead><tr><th>名称</th><th>域名</th><th>端口</th><th></th></tr></thead>
          <tbody>
            <tr v-for="s in sites" :key="s.name">
              <td>{{ s.name }}</td>
              <td>{{ s.serverName }}</td>
              <td>{{ s.port }}</td>
              <td style="text-align:right">
                <button class="btn sm" @click="visitSite(s)">浏览</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 安装版本对话框 -->
    <InstallDialog v-if="installKind" :kind="installKind" @close="installKind = null" />

    <!-- 新建站点对话框 -->
    <AddSiteDialog v-if="addSiteOpen" :preset-template="addSiteTpl"
                   :php-version="status.php.version" :mysql-running="status.mysql.running"
                   @close="addSiteOpen = false" @added="onSiteAdded" />

    <!-- 配置编辑器 -->
    <ConfigEditor v-if="cfgKey" :ckey="cfgKey" :title="cfgTitle" @close="cfgKey = null" />
  </div>
</template>

<script setup>
import { inject, ref, onMounted } from 'vue'
import ServiceCard from '../components/ServiceCard.vue'
import InstallDialog from '../components/InstallDialog.vue'
import AddSiteDialog from '../components/AddSiteDialog.vue'
import ConfigEditor from '../components/ConfigEditor.vue'

const props = defineProps({ status: Object })
const api = inject('api')

const sites = ref([])
const installKind = ref(null)
const addSiteOpen = ref(false)
const addSiteTpl = ref('php')
const cfgKey = ref(null)
const cfgTitle = ref('')

async function refresh() {
  if (api.ListSites) sites.value = await api.ListSites() || []
}
onMounted(refresh)

function openInstall(kind) { installKind.value = kind }

async function confirmUninstall(kind) {
  let keep = false
  if (kind === 'mysql' || kind === 'postgres') {
    const c = window.confirm(`卸载 ${kind} 数据库?\n按"确定"=保留 data 目录 (备份到 tmp/), 按"取消"=放弃`)
    if (!c) return
    keep = true
  } else {
    if (!window.confirm(`确认卸载 ${kind}? 对应 bin 目录将被删除`)) return
  }
  try { await api.UninstallComponent(kind, keep) }
  catch (e) { alert('卸载失败: ' + e) }
}

function editConfig(key, title) {
  cfgKey.value = key
  cfgTitle.value = title
}

async function toggleAuto(kind) {
  // 调用 autostart 列表来判断当前是否已启用, 这里简化: 直接尝试 enable, 失败提示
  try {
    const list = await api.AutoStartList()
    const item = list.find(i => i.key === kind)
    if (item && item.Installed) {
      await api.DisableAutoStart(kind)
    } else {
      await api.EnableAutoStart(kind)
    }
  } catch (e) {
    alert('操作失败: ' + e)
  }
}

function addSite(tpl) {
  addSiteTpl.value = tpl
  addSiteOpen.value = true
}

async function onSiteAdded() {
  addSiteOpen.value = false
  await refresh()
}

function visitSite(s) {
  let url = 'http://' + s.serverName
  if (s.port && s.port !== 80) url += ':' + s.port
  api.OpenInBrowser(url)
}

async function openFolder(key) {
  const p = await api.GetPaths()
  const map = { www: p.wwwDir, logs: p.logsDir, root: p.root, bin: p.binDir }
  if (map[key]) await api.OpenFolder(map[key])
}
</script>

<style scoped>
.page-title { font-size: 20px; font-weight: 600; margin: 0 0 16px; color: var(--text); }

.svc-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 16px;
}
@media (max-width: 1100px) { .svc-grid { grid-template-columns: repeat(2, 1fr); } }

.home-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: 14px;
}
.card-title { font-weight: 600; font-size: 15px; margin-bottom: 12px; }
.muted { color: var(--text-secondary); font-weight: normal; font-size: 12px; }

.quick-grid {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px;
}
.quick-grid .col-2 { grid-column: span 3; }
@media (min-width: 700px) {
  .quick-grid .col-2 { grid-column: span 1; }
  .quick-grid .col-2:first-child { grid-column: span 2; }
}

.empty { color: var(--text-secondary); text-align: center; padding: 30px 0; font-size: 13px; }
</style>
