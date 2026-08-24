import { apiClient } from '../client'

export interface IntegrationGroup {
  id: number
  name: string
  platform: string
  status: string
}

export interface IntegrationAccount {
  id: number
  name: string
  platform: string
  type: string
  extra?: Record<string, unknown>
  proxy_id?: number | null
  concurrency?: number
  priority?: number
  rate_multiplier?: number
  load_factor?: number | null
  status?: string
  schedulable?: boolean
  group_ids?: number[]
  expires_at?: string | null
  auto_pause_on_expired?: boolean
}

export interface IntegrationAccountInput {
  request_id: string
  item_id: string
  name: string
  notes?: string | null
  platform: string
  type: string
  credentials: Record<string, unknown>
  extra?: Record<string, unknown>
  proxy_id?: number | null
  concurrency?: number
  priority?: number
  group_ids?: number[]
  rate_multiplier?: number
  load_factor?: number | null
  schedulable?: boolean
  expires_at?: number | null
  auto_pause_on_expired?: boolean
  upstream_billing_probe_enabled?: boolean
}

export async function listLocalGroups(): Promise<IntegrationGroup[]> {
  const { data } = await apiClient.get<IntegrationGroup[]>('/admin/groups/all')
  return data
}

export async function listEamonGroups(): Promise<IntegrationGroup[]> {
  const { data } = await apiClient.get<{ groups: IntegrationGroup[] }>('/admin/integration/eamon/groups')
  return data.groups ?? []
}

export async function pushAccount(
  targets: string[],
  account: IntegrationAccountInput,
  localGroupIDs: number[] = [],
  remoteGroupIDs: number[] = []
) {
  const { data } = await apiClient.post<{ results: Record<string, unknown> }>('/admin/integration/eamon/accounts', {
    targets,
    account,
    local_group_ids: localGroupIDs,
    remote_group_ids: remoteGroupIDs
  })
  return data
}

export async function listEamonAccounts(params?: Record<string, string | number>) {
  const { data } = await apiClient.get<{ accounts: IntegrationAccount[]; total: number }>('/admin/integration/eamon/accounts', {
    params
  })
  return data
}

export async function updateEamonAccount(id: number, payload: Partial<IntegrationAccountInput>) {
  const { data } = await apiClient.put<{ account: IntegrationAccount }>(`/admin/integration/eamon/accounts/${id}`, payload)
  return data.account
}

export async function deleteEamonAccount(id: number) {
  await apiClient.delete(`/admin/integration/eamon/accounts/${id}`)
}
