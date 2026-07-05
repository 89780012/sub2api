import { apiClient } from '../client'
import type {
  BalanceBindingPayload,
  BalanceRefreshResult,
  BalanceSite,
  BalanceSitePayload,
  BalanceSnapshot
} from '@/types'

export async function list(): Promise<BalanceSite[]> {
  const { data } = await apiClient.get<BalanceSite[]>('/admin/balance-sites')
  return data
}

export async function create(payload: BalanceSitePayload): Promise<BalanceSite> {
  const { data } = await apiClient.post<BalanceSite>('/admin/balance-sites', payload)
  return data
}

export async function update(id: number, payload: BalanceSitePayload): Promise<BalanceSite> {
  const { data } = await apiClient.put<BalanceSite>(`/admin/balance-sites/${id}`, payload)
  return data
}

export async function deleteSite(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/balance-sites/${id}`)
  return data
}

export async function refresh(id: number): Promise<BalanceRefreshResult> {
  const { data } = await apiClient.post<BalanceRefreshResult>(`/admin/balance-sites/${id}/refresh`)
  return data
}

export async function refreshAll(): Promise<{ results: BalanceRefreshResult[] }> {
  const { data } = await apiClient.post<{ results: BalanceRefreshResult[] }>('/admin/balance-sites/refresh')
  return data
}

export async function listSnapshots(siteId?: number): Promise<{ items: BalanceSnapshot[] }> {
  const { data } = await apiClient.get<{ items: BalanceSnapshot[] }>('/admin/balance-sites/snapshots', {
    params: siteId ? { site_id: siteId } : undefined
  })
  return data
}

export async function upsertBinding(accountId: number, payload: BalanceBindingPayload): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(`/admin/accounts/${accountId}/balance-binding`, payload)
  return data
}

export async function deleteBinding(accountId: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/accounts/${accountId}/balance-binding`)
  return data
}

export const balanceSitesAPI = {
  list,
  create,
  update,
  delete: deleteSite,
  refresh,
  refreshAll,
  listSnapshots,
  upsertBinding,
  deleteBinding
}

export default balanceSitesAPI
