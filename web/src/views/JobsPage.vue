<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import AppShell from '../components/AppShell.vue'
import { api } from '../lib/api'
const items=ref([]);let timer
async function load(){const {data}=await api.get('/jobs');items.value=data.items}
onMounted(async()=>{await load();timer=setInterval(load,3000)});onUnmounted(()=>clearInterval(timer))
</script>

<template><AppShell><template #title>任务中心</template><div class="mb-6 flex items-end justify-between"><div><h2 class="text-2xl font-bold">任务流水</h2><p class="mt-2 text-sm text-slate-500">查看检测、Reality 优选、维护和编排任务。</p></div><button class="btn-secondary" @click="load">刷新</button></div><div class="card overflow-hidden"><div v-if="!items.length" class="py-16 text-center text-sm text-slate-400">暂无任务</div><div v-for="item in items" :key="item.id" class="grid gap-3 border-b border-slate-100 p-5 sm:grid-cols-[1.2fr_.7fr_2fr]"><div><b>{{item.job_key}}</b><p class="mt-1 text-xs text-slate-400">{{item.type}}</p></div><span class="w-fit rounded-full px-2.5 py-1 text-xs font-semibold" :class="item.status==='completed'?'bg-emerald-50 text-emerald-600':item.status==='failed'?'bg-red-50 text-red-600':'bg-blue-50 text-blue-600'">{{item.status}}</span><div><span class="text-sm text-slate-600">{{item.message||item.type}}</span><div v-if="item.payload?.targets?.length" class="mt-3 space-y-1 rounded-xl bg-slate-50 p-3 text-xs"><p v-for="target in item.payload.targets.slice(0,5)" :key="target.target"><b>{{target.sni}}</b> · {{target.target}} · {{target.latencyMs}} ms</p></div></div></div></div></AppShell></template>
