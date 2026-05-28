<template>
  <div class="modal-mask">
    <div class="modal" style="width: 680px">
      <div class="modal-header">🎨 切换主题</div>
      <div class="modal-body">
        <div class="theme-grid">
          <div v-for="t in themes" :key="t.id" class="theme-card"
               :class="{ active: t.id === currentId }" @click="apply(t.id)">
            <div class="preview" :style="{ background: t.previewBg || '#888', color: t.previewFg || '#fff' }">
              <span class="preview-name">{{ t.name }}</span>
              <span v-if="t.id === currentId" class="active-badge">✓ 当前</span>
            </div>
            <div class="info">
              <div class="meta">
                <span class="name">{{ t.name }}</span>
                <span v-if="t.builtin" class="tag tag-builtin">内置</span>
                <span v-else class="tag tag-custom">自定义</span>
              </div>
              <div v-if="t.author" class="author">作者: {{ t.author }}{{ t.version ? ' · v' + t.version : '' }}</div>
              <div v-if="t.description" class="desc">{{ t.description }}</div>
            </div>
            <div v-if="!t.builtin" class="actions">
              <button class="btn sm danger" @click.stop="removeOne(t)">删除</button>
            </div>
          </div>
        </div>

        <div v-if="error" class="error">{{ error }}</div>

        <div class="dev-tip">
          <strong>💡 开发自己的主题:</strong>
          编辑 <code>themes/example/</code> 改名复制一份, 修改 <code>theme.json</code> + <code>theme.css</code>
          (覆盖 CSS 变量) 即可. 点 "打开 themes 目录" 直接到那个文件夹.
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn" @click="openFolder">📁 打开 themes 目录</button>
        <button class="btn" @click="refresh">🔄 刷新列表</button>
        <button class="btn" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, onMounted } from 'vue'
const emit = defineEmits(['close'])
const api = inject('api')
const applyThemeFn = inject('applyTheme', () => {})
const currentIdRef = inject('currentThemeId', ref('default'))

const themes = ref([])
const currentId = ref(currentIdRef.value)
const error = ref('')

async function refresh() {
  try {
    themes.value = await api.ListThemes() || []
    currentId.value = currentIdRef.value
  } catch (e) {
    error.value = '加载主题失败: ' + e
  }
}

async function apply(id) {
  if (id === currentId.value) return
  error.value = ''
  try {
    const applied = await api.SetTheme(id)
    applyThemeFn(applied)
    currentId.value = id
  } catch (e) {
    error.value = '应用失败: ' + e
  }
}

async function removeOne(t) {
  if (!confirm('删除主题 "' + t.name + '"? (从 themes/ 目录里彻底删除)')) return
  try {
    await api.RemoveCustomTheme(t.id)
    // 如果删的是当前用的, 回退到 default
    if (t.id === currentId.value) {
      const applied = await api.SetTheme('default')
      applyThemeFn(applied)
      currentId.value = 'default'
    }
    await refresh()
  } catch (e) {
    error.value = '删除失败: ' + e
  }
}

async function openFolder() {
  try { await api.OpenThemesFolder() } catch (e) { /* ignore */ }
}

onMounted(refresh)
</script>

<style scoped>
.theme-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.theme-card {
  background: #fff; border: 2px solid var(--border);
  border-radius: 12px; overflow: hidden;
  cursor: pointer; transition: all 0.2s;
  display: flex; flex-direction: column;
}
.theme-card:hover { border-color: var(--primary); transform: translateY(-2px); box-shadow: var(--shadow-hover); }
.theme-card.active {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgba(255, 111, 158, 0.18);
}
.preview {
  height: 80px; padding: 14px; position: relative;
  display: flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 16px;
  text-shadow: 0 1px 4px rgba(0,0,0,0.18);
}
.preview-name { font-size: 18px; }
.active-badge {
  position: absolute; top: 8px; right: 10px;
  font-size: 11px; padding: 3px 8px; border-radius: 10px;
  background: rgba(255,255,255,0.9); color: #2d7a2d;
  text-shadow: none;
}
.info { padding: 12px 14px; flex: 1; }
.meta { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.meta .name { font-weight: 600; color: var(--text); }
.tag { font-size: 10px; padding: 2px 7px; border-radius: 8px; }
.tag-builtin { background: rgba(176,111,255,0.12); color: #6b3dbf; }
.tag-custom  { background: rgba(255,183,77,0.15); color: #b8762e; }
.author { font-size: 11px; color: var(--text-secondary); }
.desc { font-size: 12px; color: var(--text-secondary); margin-top: 4px; line-height: 1.5; }
.actions { padding: 0 14px 12px; display: flex; gap: 6px; }

.error {
  margin-top: 14px; padding: 8px 12px;
  background: #fff5f5; border-radius: 6px;
  color: var(--danger); font-size: 12px;
}

.dev-tip {
  margin-top: 16px; padding: 12px 14px;
  background: linear-gradient(135deg, #fffbeb, #fff0f8);
  border-left: 3px solid var(--primary);
  border-radius: 6px;
  font-size: 12px; color: var(--text-secondary); line-height: 1.65;
}
.dev-tip code {
  background: rgba(255,111,158,0.12); padding: 1px 6px; border-radius: 4px;
  font-family: Consolas, monospace; color: var(--primary-dark);
}
</style>
