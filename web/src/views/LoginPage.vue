<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth=useAuthStore(); const router=useRouter(); const username=ref(''); const password=ref(''); const busy=ref(false); const error=ref('')
async function submit(){ busy.value=true; error.value=''; try{await auth.login(username.value,password.value); router.push('/')}catch{error.value='用户名或密码不正确'}finally{busy.value=false} }
</script>

<template>
  <div class="grid min-h-screen place-items-center bg-[#edf1f7] p-5">
    <div class="card w-full max-w-md overflow-hidden">
      <div class="bg-slate-950 px-8 pb-8 pt-10 text-white"><div class="mb-8 grid h-12 w-12 place-items-center rounded-2xl bg-blue-600 text-lg font-black">XP</div><h1 class="text-3xl font-bold tracking-tight">欢迎回来</h1><p class="mt-2 text-sm text-slate-400">登录网络资源编排控制台</p></div>
      <form @submit.prevent="submit" class="space-y-5 p-8">
        <label class="block"><span class="mb-2 block text-sm font-semibold">管理员账号</span><input v-model.trim="username" autocomplete="username" class="field" required /></label>
        <label class="block"><span class="mb-2 block text-sm font-semibold">密码</span><input v-model="password" type="password" autocomplete="current-password" class="field" required /></label>
        <p v-if="error" class="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600">{{ error }}</p>
        <button class="btn-primary w-full py-3" :disabled="busy">{{ busy?'正在登录…':'登录' }}</button>
      </form>
    </div>
  </div>
</template>
