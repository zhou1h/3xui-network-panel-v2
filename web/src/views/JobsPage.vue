<script setup>
import { onMounted, ref } from 'vue'
import AppShell from '../components/AppShell.vue'
import { api } from '../lib/api'
const items=ref([]);onMounted(async()=>{const {data}=await api.get('/jobs');items.value=data.items})
</script>

<template><AppShell><template #title>任务中心</template><div class="mb-6"><h2 class="text-2xl font-bold">任务流水</h2><p class="mt-2 text-sm text-slate-500">查看安装、检测、维护和编排任务。</p></div><div class="card overflow-hidden"><div v-if="!items.length" class="py-16 text-center text-sm text-slate-400">暂无任务</div><div v-for="item in items" :key="item.id" class="grid gap-2 border-b border-slate-100 p-5 sm:grid-cols-[1.2fr_.7fr_2fr]"><b>{{item.job_key}}</b><span class="text-sm">{{item.status}}</span><span class="text-sm text-slate-500">{{item.message||item.type}}</span></div></div></AppShell></template>
