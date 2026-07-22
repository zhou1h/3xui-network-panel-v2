<script setup>
import { onMounted, ref } from 'vue'
import AppShell from '../components/AppShell.vue'
import { api } from '../lib/api'

const stats=ref({resources:0,online:0,activeJobs:0,clients:0,nodes:0,socks5:0})
onMounted(async()=>{const {data}=await api.get('/dashboard'); stats.value=data})
const cards=[['资源服务器','resources'],['在线资源','online'],['节点','nodes'],['客户端','clients'],['SOCKS5','socks5'],['执行中任务','activeJobs']]
</script>

<template><AppShell><template #title>运营总览</template>
  <div class="mb-8"><h2 class="text-2xl font-bold tracking-tight">基础设施概况</h2><p class="mt-2 text-sm text-slate-500">统一查看资源、节点、客户端与任务状态。</p></div>
  <div class="grid grid-cols-2 gap-4 lg:grid-cols-3"><div v-for="card in cards" :key="card[1]" class="card p-5 lg:p-7"><p class="text-sm font-medium text-slate-500">{{card[0]}}</p><p class="mt-3 text-3xl font-bold tabular-nums">{{stats[card[1]]}}</p></div></div>
  <div class="card mt-6 p-6"><div class="flex items-center gap-3"><span class="h-2.5 w-2.5 rounded-full bg-emerald-500"></span><div><p class="font-semibold">控制服务运行正常</p><p class="mt-1 text-sm text-slate-500">PostgreSQL、Redis 与 API 已连接。</p></div></div></div>
</AppShell></template>
