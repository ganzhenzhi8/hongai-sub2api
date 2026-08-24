<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-5 p-4 md:p-6">
      <header>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('admin.integration.accountImportTitle') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.integration.accountImportDescription') }}</p>
      </header>

      <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.integration.targets') }}</h2>
        <div class="mt-4 grid gap-3 sm:grid-cols-2">
          <label class="flex min-h-12 cursor-pointer items-center gap-3 rounded-md border border-gray-200 px-4 text-sm text-gray-700 dark:border-dark-600 dark:text-gray-200">
            <input v-model="targets" type="checkbox" value="local" class="h-4 w-4" :disabled="showCreate" />
            <span class="flex-1">hongai.co</span>
          </label>
          <label class="flex min-h-12 cursor-pointer items-center gap-3 rounded-md border border-gray-200 px-4 text-sm text-gray-700 dark:border-dark-600 dark:text-gray-200">
            <input v-model="targets" type="checkbox" value="eamon88" class="h-4 w-4" :disabled="showCreate" />
            <span class="flex-1">eamon88.com</span>
          </label>
        </div>

        <div class="mt-5 flex flex-wrap items-center gap-3">
          <button type="button" class="btn btn-primary" :disabled="loading || targets.length === 0" @click="showCreate = true">
            {{ t('admin.integration.openOriginalCreate') }}
          </button>
          <span v-if="loading" class="text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</span>
          <span v-else-if="loadError" class="text-sm text-red-600 dark:text-red-400">{{ loadError }}</span>
        </div>
      </section>

      <details class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <summary class="cursor-pointer px-5 py-4 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.integration.jsonMode') }}
        </summary>
        <form class="space-y-4 border-t border-gray-200 p-5 dark:border-dark-700" @submit.prevent="submitJSON">
          <div class="grid gap-4 lg:grid-cols-2">
            <GroupSelector
              v-if="targets.includes('local')"
              v-model="jsonLocalGroupIDs"
              :groups="localGroups"
              :label="t('admin.integration.localGroups')"
            />
            <GroupSelector
              v-if="targets.includes('eamon88')"
              v-model="jsonRemoteGroupIDs"
              :groups="remoteGroups"
              :label="t('admin.integration.remoteGroups')"
            />
          </div>
          <textarea v-model="jsonText" class="input min-h-64 w-full font-mono text-xs" :placeholder="jsonPlaceholder" spellcheck="false"></textarea>
          <button type="submit" class="btn btn-primary" :disabled="submittingJSON || targets.length === 0">
            {{ submittingJSON ? t('common.loading') : t('admin.integration.startImport') }}
          </button>
          <p v-if="jsonMessage" :class="jsonMessageType === 'error' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'" class="text-sm">
            {{ jsonMessage }}
          </p>
        </form>
      </details>
    </div>

    <CreateAccountModal
      :show="showCreate"
      :proxies="modalProxies"
      :groups="modalGroups"
      :group-targets="modalGroupTargets"
      :create-override="modalCreateOverride"
      @close="showCreate = false"
      @created="handleCreated"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import CreateAccountModal from '@/components/account/CreateAccountModal.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import { adminAPI } from '@/api/admin'
import { listEamonGroups, pushAccount, type IntegrationAccountInput, type IntegrationGroup } from '@/api/admin/integration'
import type { AdminGroup, CreateAccountRequest, GroupPlatform, Proxy } from '@/types'

type MessageType = 'success' | 'error'

const { t } = useI18n()
const targets = ref<string[]>(['local'])
const localGroups = ref<AdminGroup[]>([])
const remoteGroups = ref<AdminGroup[]>([])
const localProxies = ref<Proxy[]>([])
const showCreate = ref(false)
const loading = ref(true)
const loadError = ref('')
const jsonText = ref('')
const jsonLocalGroupIDs = ref<number[]>([])
const jsonRemoteGroupIDs = ref<number[]>([])
const submittingJSON = ref(false)
const jsonMessage = ref('')
const jsonMessageType = ref<MessageType>('success')
const jsonPlaceholder = '[{"name":"账号","platform":"openai","type":"apikey","credentials":{"base_url":"https://api.openai.com","api_key":"sk-..."}}]'

function normalizeRemoteGroup(group: IntegrationGroup): AdminGroup {
  // The remote API intentionally exposes only fields needed by the selector.
  // Defaults below form a UI adapter; newly added AdminGroup admin-only fields
  // must not make cross-site group selection fail to build.
  return {
    id: group.id,
    name: group.name,
    description: null,
    platform: group.platform as GroupPlatform,
    rate_multiplier: 1,
    is_exclusive: false,
    status: group.status === 'inactive' ? 'inactive' : 'active',
    subscription_type: 'standard',
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    allow_image_generation: false,
    allow_batch_image_generation: false,
    image_rate_independent: false,
    image_rate_multiplier: 1,
    batch_image_discount_multiplier: 1,
    batch_image_hold_multiplier: 1,
    image_price_1k: null,
    image_price_2k: null,
    image_price_4k: null,
    video_rate_independent: false,
    video_rate_multiplier: 1,
    video_price_480p: null,
    video_price_720p: null,
    video_price_1080p: null,
    web_search_price_per_call: null,
    peak_rate_enabled: false,
    peak_start: '',
    peak_end: '',
    peak_rate_multiplier: 1,
    claude_code_only: false,
    fallback_group_id: null,
    fallback_group_id_on_invalid_request: null,
    allow_live: false,
    require_oauth_only: false,
    require_privacy_set: false,
    created_at: '',
    updated_at: '',
    model_routing: null,
    model_routing_enabled: false,
    mcp_xml_inject: false,
    sort_order: 0,
    account_count: 0
  } as AdminGroup
}

const groupTargets = computed(() => {
  const result: Array<{ key: string; label: string; groups: AdminGroup[] }> = []
  if (targets.value.includes('local')) result.push({ key: 'local', label: t('admin.integration.localGroups'), groups: localGroups.value })
  if (targets.value.includes('eamon88')) result.push({ key: 'eamon88', label: t('admin.integration.remoteGroups'), groups: remoteGroups.value })
  return result
})

// A local-only import is the stock account form without any interception.
// Cross-site imports reuse the same form UI, but route the completed payload
// through the integration endpoint so each site receives its own group IDs.
const isLocalOnly = computed(() => targets.value.length === 1 && targets.value[0] === 'local')
const modalGroups = computed(() => isLocalOnly.value ? localGroups.value : [])
const modalGroupTargets = computed(() => isLocalOnly.value ? undefined : groupTargets.value)
const modalCreateOverride = computed(() => isLocalOnly.value ? undefined : createAcrossSites)

// Proxy IDs belong to one database. Expose them only when creating on hongai alone.
const modalProxies = computed(() => isLocalOnly.value ? localProxies.value : [])

onMounted(async () => {
  loading.value = true
  try {
    const [local, remote, proxies] = await Promise.all([
      adminAPI.groups.getAll(),
      listEamonGroups(),
      adminAPI.proxies.getAll()
    ])
    localGroups.value = local
    remoteGroups.value = remote.map(normalizeRemoteGroup)
    localProxies.value = proxies
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t('admin.integration.loadFailed')
  } finally {
    loading.value = false
  }
})

function resultFailures(result: Awaited<ReturnType<typeof pushAccount>>): string[] {
  return Object.entries(result.results)
    .filter(([, value]) => !!value && typeof value === 'object' && 'ok' in value && (value as { ok?: boolean }).ok === false)
    .map(([target]) => target)
}

async function createAcrossSites(payload: CreateAccountRequest, groupIDsByTarget: Record<string, number[]>) {
  if (targets.value.length === 0) throw new Error(t('admin.integration.selectTarget'))
  const account: IntegrationAccountInput = {
    ...payload,
    request_id: `original-form-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    item_id: '0'
  }
  const result = await pushAccount(targets.value, account, groupIDsByTarget.local || [], groupIDsByTarget.eamon88 || [])
  const failures = resultFailures(result)
  if (failures.length > 0) throw new Error(t('admin.integration.targetFailure', { targets: failures.join(', ') }))
}

function handleCreated() {
  showCreate.value = false
}

function normalizeAccounts(value: unknown): IntegrationAccountInput[] {
  const records = Array.isArray(value) ? value : [value]
  if (!records.every((item) => !!item && typeof item === 'object')) throw new Error(t('admin.integration.invalidJSON'))
  const requestID = `ui-json-${Date.now()}-${Math.random().toString(36).slice(2)}`
  return records.map((item, index) => ({
    ...(item as IntegrationAccountInput),
    request_id: requestID,
    item_id: String(index),
    credentials: (item as IntegrationAccountInput).credentials || {}
  }))
}

async function submitJSON() {
  submittingJSON.value = true
  jsonMessage.value = ''
  try {
    const accounts = normalizeAccounts(JSON.parse(jsonText.value))
    const failed: string[] = []
    for (const account of accounts) {
      const result = await pushAccount(targets.value, account, jsonLocalGroupIDs.value, jsonRemoteGroupIDs.value)
      if (resultFailures(result).length > 0) failed.push(account.name)
    }
    if (failed.length > 0) throw new Error(t('admin.integration.partialFailure', { names: failed.join(', ') }))
    jsonMessageType.value = 'success'
    jsonMessage.value = t('admin.integration.importSuccess', { count: accounts.length })
  } catch (error) {
    jsonMessageType.value = 'error'
    jsonMessage.value = error instanceof Error ? error.message : t('admin.integration.importFailed')
  } finally {
    submittingJSON.value = false
  }
}
</script>
