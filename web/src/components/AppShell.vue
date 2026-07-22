<script setup>
import { useRoute, useRouter } from 'vue-router'
import { HomeIcon, CircleStackIcon, QueueListIcon, ArrowRightStartOnRectangleIcon } from '@heroicons/vue/24/outline'
import { useAuthStore } from '../stores/auth'

const route = useRoute(); const router = useRouter(); const auth = useAuthStore()
const items = [
  { to:'/', label:'运营总览', icon:HomeIcon },
  { to:'/resources', label:'资源中心', icon:CircleStackIcon },
  { to:'/jobs', label:'任务中心', icon:QueueListIcon }
]
async function logout(){ await auth.logout(); router.push('/login') }
</script>

<template>
  <div class="min-h-screen bg-canvas lg:flex">
    <aside class="fixed inset-x-0 bottom-0 z-30 border-t border-slate-200 bg-white/95 backdrop-blur lg:inset-y-0 lg:left-0 lg:right-auto lg:w-64 lg:border-r lg:border-t-0">
      <div class="hidden h-24 items-center px-7 lg:flex"><div class="grid h-11 w-11 place-items-center rounded-2xl bg-ink text-lg font-black text-white">XP</div><div class="ml-3"><p class="m-0 font-bold">XPanel</p><p class="m-0 text-xs text-slate-400">Orchestration v2</p></div></div>
      <nav class="flex justify-around p-2 lg:block lg:space-y-2 lg:px-4">
        <router-link v-for="item in items" :key="item.to" :to="item.to" class="flex min-w-20 flex-col items-center gap-1 rounded-xl px-3 py-2 text-xs font-semibold text-slate-500 transition lg:min-w-0 lg:flex-row lg:gap-3 lg:px-4 lg:py-3 lg:text-sm" :class="route.path===item.to?'bg-slate-900 text-white':'hover:bg-slate-100 hover:text-slate-900'">
          <component :is="item.icon" class="h-5 w-5" />{{ item.label }}
        </router-link>
      </nav>
      <button @click="logout" class="absolute bottom-6 left-4 right-4 hidden items-center gap-3 rounded-xl px-4 py-3 text-sm font-semibold text-slate-500 hover:bg-slate-100 lg:flex"><ArrowRightStartOnRectangleIcon class="h-5 w-5"/>退出登录</button>
    </aside>
    <main class="w-full pb-24 lg:ml-64 lg:pb-10">
      <header class="flex h-20 items-center justify-between border-b border-slate-200 bg-white/80 px-5 backdrop-blur lg:px-10"><div><p class="m-0 text-xs font-medium uppercase tracking-[.16em] text-slate-400">NETWORK CONTROL</p><h1 class="m-0 mt-1 text-lg font-bold"><slot name="title">控制中心</slot></h1></div><div class="rounded-full bg-slate-100 px-4 py-2 text-sm font-semibold">{{ auth.user?.username }}</div></header>
      <section class="mx-auto max-w-7xl p-5 lg:p-10"><slot /></section>
    </main>
  </div>
</template>
