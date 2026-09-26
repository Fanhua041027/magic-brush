<template>
  <Teleport to="body">
  <Transition name="auth-fade">
    <div class="auth-overlay" @click.self="$emit('close')">
      <form class="auth-card" @submit.prevent="submit">
        <button class="close-button" type="button" aria-label="关闭" @click="$emit('close')">×</button>
        <div class="brand-mark"><span>✦</span></div>
        <div class="eyebrow">MAGIC BRUSH</div>
        <h2>{{ setupMode ? '创建首个管理员' : (mustChange ? '请设置新密码' : '登录 AI Assistant') }}</h2>
        <p class="subtitle">{{ setupMode ? '首次使用，请设置本机管理员账号' : (mustChange ? '首次登录需要修改管理员初始密码' : '登录后即可使用 AI 面试与智能服务') }}</p>
        <template v-if="setupMode">
          <label>管理员账号<input v-model.trim="username" required minlength="3" maxlength="64" placeholder="3-64 位账号" autocomplete="username" /></label>
          <label>管理员密码<input v-model="password" required minlength="8" maxlength="72" type="password" placeholder="8-72 字节密码" autocomplete="new-password" /></label>
          <label>确认密码<input v-model="confirmPassword" required minlength="8" maxlength="72" type="password" placeholder="再次输入密码" autocomplete="new-password" /></label>
        </template>
        <template v-else-if="!mustChange">
          <label>账号<input v-model.trim="username" required minlength="3" maxlength="64" placeholder="输入账号" autocomplete="username" /></label>
          <label>密码<input v-model="password" required minlength="8" maxlength="72" type="password" placeholder="至少 8 位密码" autocomplete="current-password" /></label>
        </template>
        <template v-else>
          <label>新密码<input v-model="newPassword" required minlength="8" type="password" placeholder="至少 8 位密码" autocomplete="new-password" /></label>
          <label>确认新密码<input v-model="confirmPassword" required minlength="8" type="password" placeholder="再次输入新密码" autocomplete="new-password" /></label>
        </template>
        <button class="submit-button" :disabled="busy" type="submit">{{ busy ? '处理中…' : (setupMode ? '创建并登录' : (mustChange ? '保存新密码' : '登录')) }}<span>›</span></button>
        <button v-if="!mustChange && !setupMode" class="cancel-button" type="button" @click="$emit('close')">暂不登录</button>
        <div v-if="error" class="auth-error">{{ error }}</div>
      </form>
    </div>
  </Transition>
  </Teleport>
</template>
<script setup>
import { ref } from 'vue'
import { AuthBootstrapAdmin, AuthChangePassword, AuthHasAdmin, AuthLogin } from '../../wailsjs/go/app/App'
const emit = defineEmits(['authenticated', 'close'])
const username = ref(''); const password = ref(''); const newPassword = ref(''); const confirmPassword = ref('')
const error = ref(''); const busy = ref(false); const mustChange = ref(false); const setupMode = ref(false); const userId = ref('')
AuthHasAdmin().then(hasAdmin => { setupMode.value = !hasAdmin }).catch(() => {})
async function submit() {
  error.value = ''; busy.value = true
  try {
    if (setupMode.value) {
      if (password.value !== confirmPassword.value) { error.value = '两次输入的密码不一致'; return }
      const setupError = await AuthBootstrapAdmin(username.value, password.value)
      if (setupError) { error.value = setupError; return }
      setupMode.value = false
      const result = await AuthLogin(username.value, password.value)
      if (!result?.ok) { error.value = result?.error || '初始化后登录失败'; return }
      emit('authenticated', result.user)
      return
    }
    if (mustChange.value) {
      if (newPassword.value !== confirmPassword.value) { error.value = '两次输入的新密码不一致'; return }
      const result = await AuthChangePassword(password.value, newPassword.value)
      if (result) { error.value = result; return }
      emit('authenticated', { id: userId.value, username: username.value, role: 'admin', mustChangePassword: false })
      return
    }
    const result = await AuthLogin(username.value, password.value)
    if (!result?.ok) { error.value = result?.error || '登录失败'; return }
    if (result.user?.mustChangePassword) { userId.value = result.user.id; mustChange.value = true; return }
    emit('authenticated', result.user)
  } catch (e) { error.value = e?.message || '认证服务不可用' } finally { busy.value = false }
}
</script>
<style scoped>
.auth-overlay{position:fixed;inset:0;z-index:99999;display:grid;place-items:center;background:rgba(10,12,18,.48);backdrop-filter:blur(18px);-webkit-backdrop-filter:blur(18px);pointer-events:auto}.auth-card{position:relative;width:350px;padding:34px 36px 28px;border:1px solid rgba(255,255,255,.2);border-radius:24px;background:linear-gradient(145deg,rgba(48,51,65,.94),rgba(25,28,38,.96));box-shadow:0 24px 80px rgba(0,0,0,.48),inset 0 1px rgba(255,255,255,.12);color:#f6f7fb;display:flex;flex-direction:column;gap:12px}.close-button{position:absolute;right:16px;top:13px;border:0;background:transparent!important;color:#aeb4c4;font-size:25px;line-height:1;cursor:pointer}.brand-mark{width:48px;height:48px;margin:0 auto 2px;display:grid;place-items:center;border-radius:15px;background:linear-gradient(135deg,#9b8cff,#6558d8);box-shadow:0 8px 24px rgba(117,103,238,.35);font-size:25px}.eyebrow{text-align:center;color:#aaa1ff;font-size:10px;font-weight:700;letter-spacing:2px}.auth-card h2{margin:2px 0 0;text-align:center;font-size:22px;letter-spacing:-.3px}.subtitle{text-align:center;margin:0 0 9px;color:#aeb4c4;font-size:13px}.auth-card label{display:flex;flex-direction:column;gap:6px;color:#c8ccda;font-size:12px}.auth-card input{box-sizing:border-box;width:100%;padding:12px 13px;border:1px solid rgba(255,255,255,.12);border-radius:11px;background:rgba(8,10,16,.3);color:#fff;font:inherit;outline:0;transition:border-color .2s,box-shadow .2s}.auth-card input:focus{border-color:#9b8cff;box-shadow:0 0 0 3px rgba(155,140,255,.16)}.submit-button{display:flex;align-items:center;justify-content:center;gap:10px;margin-top:7px;padding:12px;border:0;border-radius:11px;background:linear-gradient(135deg,#9a89ff,#6f61e8);color:#fff;font-weight:600;cursor:pointer;box-shadow:0 7px 18px rgba(111,97,232,.28)}.submit-button span{font-size:20px;line-height:12px}.submit-button:disabled{opacity:.6;cursor:wait}.cancel-button{padding:7px;border:0;background:transparent;color:#9ea5b8;cursor:pointer}.cancel-button:hover{color:#fff}.auth-error{padding:9px;border-radius:9px;background:rgba(255,90,90,.1);color:#ff9a9a;text-align:center;font-size:12px}.auth-fade-enter-active,.auth-fade-leave-active{transition:opacity .2s ease}.auth-fade-enter-from,.auth-fade-leave-to{opacity:0}
</style>
