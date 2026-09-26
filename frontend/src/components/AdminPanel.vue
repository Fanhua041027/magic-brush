<template>
  <Teleport to="body">
    <section v-if="user?.role === 'admin'" class="admin-panel" style="--wails-draggable: no-drag" @mousedown.stop @click.stop>
    <header><h2>管理员后台</h2><button type="button" @click.stop="$emit('close')">关闭</button></header>
    <form @submit.prevent="createUser" class="create-form">
      <input v-model.trim="newUsername" minlength="3" placeholder="新用户账号" required />
      <input v-model="newPassword" minlength="8" type="password" placeholder="初始密码（至少8位）" required />
      <button :disabled="busy" type="submit" @click.stop>创建用户</button>
    </form>
    <p v-if="message" class="message">{{ message }}</p>
    <table><thead><tr><th>账号</th><th>角色</th><th>状态</th><th>累计使用</th><th>最近登录</th><th></th></tr></thead>
      <tbody><tr v-for="item in users" :key="item.id"><td>{{ item.username }}</td><td>{{ item.role }}</td><td>{{ item.enabled ? '启用' : '停用' }}</td><td>{{ formatDuration(item.totalSeconds) }}</td><td>{{ formatDate(item.lastLoginAt) }}</td><td><button v-if="item.role !== 'admin'" type="button" @mousedown.stop @click.stop="toggle(item)">{{ item.enabled ? '停用' : '启用' }}</button></td></tr></tbody>
    </table>
    </section>
  </Teleport>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import { AuthCreateUser, AuthSetUserEnabled, AuthUsers } from '../../wailsjs/go/app/App'
defineProps({ user: Object })
defineEmits(['close'])
const users = ref([]); const newUsername = ref(''); const newPassword = ref(''); const message = ref(''); const busy = ref(false)
async function load(){ try { users.value = await AuthUsers() || [] } catch (e) { message.value = e?.message || '加载用户失败' } }
async function createUser(){ busy.value=true; message.value=''; try { const e=await AuthCreateUser(newUsername.value,newPassword.value); if(e){message.value=e}else{newUsername.value='';newPassword.value='';message.value='用户已创建';await load()} } catch (e) { message.value = e?.message || '创建用户失败' } finally {busy.value=false} }
async function toggle(item){ message.value=''; try { const e=await AuthSetUserEnabled(item.id, !item.enabled); message.value=e || (item.enabled ? '用户已停用' : '用户已启用'); await load() } catch (e) { message.value=e?.message || '更新用户状态失败' } }
function formatDuration(s){const n=Number(s)||0;return `${Math.floor(n/3600)}小时 ${Math.floor(n%3600/60)}分`}
function formatDate(v){return v ? new Date(v).toLocaleString('zh-CN') : '—'}
onMounted(load)
</script>
<style scoped>
.admin-panel{position:fixed;inset:8%;z-index:100000;pointer-events:auto;--wails-draggable:no-drag;padding:24px;background:#191c26;color:#f5f7ff;border:1px solid #394052;border-radius:14px;box-shadow:0 20px 70px #000b;overflow:auto}.admin-panel header{display:flex;justify-content:space-between;align-items:center}.admin-panel button{padding:7px 12px;border:1px solid #4a5265;border-radius:6px;background:#292e3b;color:inherit;cursor:pointer}.create-form{display:flex;gap:8px;margin:20px 0}.create-form input{padding:8px;background:#242936;color:inherit;border:1px solid #41495b;border-radius:6px}table{width:100%;border-collapse:collapse;font-size:13px}th,td{padding:10px;text-align:left;border-bottom:1px solid #303644}.message{color:#a7e6bd}
</style>
