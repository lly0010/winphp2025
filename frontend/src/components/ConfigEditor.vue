<template>
  <div class="modal-mask">
    <div class="modal" style="width: 880px; height: 80vh;">
      <div class="modal-header">编辑: {{ title }}</div>
      <div class="modal-body" style="display: flex; flex-direction: column; padding: 0; height: calc(100% - 110px);">
        <textarea v-model="text" class="editor" spellcheck="false"></textarea>
      </div>
      <div class="modal-footer">
        <span class="muted" style="margin-right: auto; font-size: 12px;">{{ message }}</span>
        <button class="btn" @click="$emit('close')">取消</button>
        <button class="btn primary" @click="save" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, onMounted } from 'vue'
const props = defineProps({ ckey: String, title: String })
const emit = defineEmits(['close'])
const api = inject('api')

const text = ref('')
const saving = ref(false)
const message = ref('')

onMounted(async () => {
  try {
    if (props.ckey === 'hosts') text.value = await api.ReadHosts()
    else text.value = await api.ReadConfig(props.ckey)
  } catch (e) {
    text.value = '读取失败: ' + e
  }
})

async function save() {
  saving.value = true
  message.value = ''
  try {
    if (props.ckey === 'hosts') await api.WriteHosts(text.value)
    else await api.WriteConfig(props.ckey, text.value)
    message.value = '已保存, 重启对应服务生效'
    setTimeout(() => emit('close'), 600)
  } catch (e) {
    message.value = '保存失败: ' + e
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.editor {
  flex: 1; width: 100%; height: 100%;
  border: 0; outline: 0;
  font-family: Consolas, Monaco, "Courier New", monospace;
  font-size: 13px; padding: 16px; resize: none;
}
.muted { color: var(--text-secondary); }
</style>
