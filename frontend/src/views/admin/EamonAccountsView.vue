<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6 p-4 md:p-6">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">eamon88 账号管理</h1>
          <p class="mt-1 text-sm text-gray-500">当前管理目标：eamon88.com</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="reload">刷新</button>
      </div>
      <div class="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <table class="min-w-full text-left text-sm">
          <thead class="border-b border-gray-200 text-xs uppercase text-gray-500 dark:border-dark-700">
            <tr><th class="px-4 py-3">名称</th><th class="px-4 py-3">平台/类型</th><th class="px-4 py-3">并发</th><th class="px-4 py-3">优先级</th><th class="px-4 py-3">调度</th><th class="px-4 py-3">操作</th></tr>
          </thead>
          <tbody>
            <tr v-for="account in accounts" :key="account.id" class="border-b border-gray-100 dark:border-dark-700">
              <td class="px-4 py-3"><input v-model="account.name" class="input min-w-40" /></td>
              <td class="px-4 py-3">{{ account.platform }} / {{ account.type }}</td>
              <td class="px-4 py-3"><input v-model.number="account.concurrency" type="number" min="0" class="input w-20" /></td>
              <td class="px-4 py-3"><input v-model.number="account.priority" type="number" class="input w-20" /></td>
              <td class="px-4 py-3"><label class="flex items-center gap-2"><input v-model="account.schedulable" type="checkbox" /><span>{{ account.status }}</span></label></td>
              <td class="px-4 py-3 whitespace-nowrap">
                <button class="mr-3 text-primary-600 hover:underline" @click="save(account)">保存</button>
                <button class="text-red-600 hover:underline" @click="remove(account.id)">删除</button>
              </td>
            </tr>
            <tr v-if="!loading && accounts.length === 0"><td colspan="6" class="px-4 py-8 text-center text-gray-500">暂无账号或远程功能未开启</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { deleteEamonAccount, listEamonAccounts, updateEamonAccount, type IntegrationAccount } from '@/api/admin/integration'

const accounts = ref<IntegrationAccount[]>([])
const loading = ref(false)

async function reload() {
  loading.value = true
  try {
    const result = await listEamonAccounts({ page: 1, page_size: 100 })
    accounts.value = result.accounts ?? []
  } finally {
    loading.value = false
  }
}

async function remove(id: number) {
  if (!window.confirm('确认删除 eamon88 上的这个账号吗？')) return
  await deleteEamonAccount(id)
  await reload()
}

async function save(account: IntegrationAccount) {
  const updated = await updateEamonAccount(account.id, {
    name: account.name,
    concurrency: account.concurrency,
    priority: account.priority,
    schedulable: account.schedulable
  })
  Object.assign(account, updated)
}

onMounted(reload)
</script>
