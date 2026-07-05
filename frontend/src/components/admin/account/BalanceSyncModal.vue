<template>
  <BaseDialog :show="show" :title="t('admin.accounts.balanceSync.title')" width="extra-wide" @close="emit('close')">
    <div class="grid gap-4 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.25fr)]">
      <section class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <div>
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.accounts.balanceSync.sites') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.balanceSync.siteCount', { count: sites.length }) }}
            </div>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          </button>
        </div>

        <div class="max-h-[28rem] space-y-2 overflow-y-auto pr-1">
          <button
            v-for="site in sites"
            :key="site.id"
            type="button"
            class="w-full rounded-lg border px-3 py-2 text-left transition"
            :class="editingId === site.id ? 'border-primary-300 bg-primary-50 dark:border-primary-700 dark:bg-primary-950/20' : 'border-gray-200 bg-white hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:hover:bg-dark-700'"
            @click="editSite(site)"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ site.name }}</div>
                <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ site.platform }} · {{ site.base_url }}</div>
              </div>
              <span :class="site.enabled ? enabledClass : disabledClass">
                {{ site.enabled ? t('common.enabled') : t('common.disabled') }}
              </span>
            </div>
            <div class="mt-2 flex flex-wrap gap-2 text-xs text-gray-500 dark:text-gray-400">
              <span>{{ refreshStatusLabel(site.last_refresh_status) }}</span>
              <span v-if="site.last_refresh_at">{{ formatDateTime(site.last_refresh_at) }}</span>
              <span v-if="site.password_configured">{{ t('admin.accounts.balanceSync.passwordConfigured') }}</span>
            </div>
            <div v-if="site.last_refresh_error" class="mt-1 truncate text-xs text-rose-600 dark:text-rose-400">
              {{ site.last_refresh_error }}
            </div>
          </button>
          <div v-if="!loading && sites.length === 0" class="rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-400 dark:border-dark-600">
            {{ t('admin.accounts.balanceSync.emptySites') }}
          </div>
        </div>

        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="resetForm">
            <Icon name="plus" size="sm" class="mr-1" />
            {{ t('admin.accounts.balanceSync.newSite') }}
          </button>
          <button type="button" class="btn btn-primary btn-sm" :disabled="refreshingAll || sites.length === 0" @click="refreshAllSites">
            <Icon name="sync" size="sm" class="mr-1" :class="{ 'animate-spin': refreshingAll }" />
            {{ t('admin.accounts.balanceSync.refreshAll') }}
          </button>
        </div>
      </section>

      <section class="space-y-4">
        <form class="grid gap-3 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800" @submit.prevent="saveSite">
          <div class="grid gap-3 md:grid-cols-2">
            <label class="space-y-1">
              <span class="form-label">{{ t('admin.accounts.balanceSync.platform') }}</span>
              <select v-model="form.platform" class="form-input">
                <option value="newapi">NewAPI</option>
                <option value="sub2api">Sub2API</option>
              </select>
            </label>
            <label class="space-y-1">
              <span class="form-label">{{ t('common.name') }}</span>
              <input v-model.trim="form.name" class="form-input" required />
            </label>
          </div>
          <label class="space-y-1">
            <span class="form-label">{{ t('admin.accounts.balanceSync.baseUrl') }}</span>
            <input v-model.trim="form.base_url" class="form-input" required placeholder="https://example.com" />
          </label>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="space-y-1">
              <span class="form-label">{{ t('admin.accounts.balanceSync.username') }}</span>
              <input v-model.trim="form.username" class="form-input" autocomplete="username" />
            </label>
            <label class="space-y-1">
              <span class="form-label">{{ t('common.email') }}</span>
              <input v-model.trim="form.email" class="form-input" type="email" autocomplete="email" />
            </label>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="space-y-1">
              <span class="form-label">
                {{ t('common.password') }}
                <span v-if="editingId" class="text-xs font-normal text-gray-400">{{ t('admin.accounts.balanceSync.passwordKeepHint') }}</span>
              </span>
              <input v-model="form.password" class="form-input" type="password" :required="!editingId" autocomplete="new-password" />
            </label>
            <label class="space-y-1">
              <span class="form-label">{{ t('admin.accounts.balanceSync.refreshInterval') }}</span>
              <input v-model.number="form.refresh_interval_minutes" class="form-input" type="number" min="5" step="1" />
            </label>
          </div>
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('admin.accounts.balanceSync.enabled') }}</span>
          </label>
          <div class="flex flex-wrap justify-between gap-2">
            <button v-if="editingId" type="button" class="btn btn-secondary btn-sm text-rose-600" :disabled="saving" @click="deleteCurrentSite">
              <Icon name="trash" size="sm" class="mr-1" />
              {{ t('common.delete') }}
            </button>
            <span v-else></span>
            <div class="flex gap-2">
              <button v-if="editingId" type="button" class="btn btn-secondary btn-sm" :disabled="refreshingSite === editingId" @click="refreshCurrentSite">
                <Icon name="sync" size="sm" class="mr-1" :class="{ 'animate-spin': refreshingSite === editingId }" />
                {{ t('admin.accounts.balanceSync.refreshSite') }}
              </button>
              <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>
        </form>

        <div class="rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-600">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.balanceSync.snapshots') }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.balanceSync.snapshotCount', { count: snapshots.length }) }}</div>
            </div>
            <select v-model.number="snapshotSiteId" class="form-input max-w-52 text-sm" @change="loadSnapshots">
              <option :value="0">{{ t('admin.accounts.balanceSync.allSites') }}</option>
              <option v-for="site in sites" :key="site.id" :value="site.id">{{ site.name }}</option>
            </select>
          </div>
          <div class="max-h-[20rem] overflow-auto">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
              <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-700 dark:text-gray-400">
                <tr>
                  <th class="px-3 py-2">{{ t('admin.accounts.balanceSync.key') }}</th>
                  <th class="px-3 py-2">{{ t('admin.accounts.balanceSync.accountBalance') }}</th>
                  <th class="px-3 py-2">{{ t('admin.accounts.balanceSync.packageBalance') }}</th>
                  <th class="px-3 py-2">{{ t('admin.accounts.balanceSync.keyBalance') }}</th>
                  <th class="px-3 py-2">{{ t('admin.accounts.balanceSync.upstreamRate') }}</th>
                  <th class="px-3 py-2">{{ t('admin.accounts.balanceSync.match') }}</th>
                  <th class="px-3 py-2">{{ t('admin.accounts.balanceSync.accountId') }}</th>
                  <th class="px-3 py-2"></th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="snapshot in snapshots" :key="snapshot.id">
                  <td class="px-3 py-2">
                    <div class="font-medium text-gray-900 dark:text-white">{{ snapshot.key_name || snapshot.masked_key || snapshot.external_key_id }}</div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">{{ snapshot.site_name }} · ****{{ snapshot.key_last4 }}</div>
                  </td>
                  <td class="px-3 py-2 text-gray-700 dark:text-gray-300">{{ formatBalance(snapshot.account_balance) }}</td>
                  <td class="px-3 py-2 text-gray-700 dark:text-gray-300">{{ formatSubscriptionBalance(snapshot.subscription_balance) }}</td>
                  <td class="px-3 py-2 text-gray-700 dark:text-gray-300">{{ formatBalance(snapshot.balance) }}</td>
                  <td class="px-3 py-2 font-mono text-gray-700 dark:text-gray-300">{{ formatRate(snapshot.rate_multiplier) }}</td>
                  <td class="px-3 py-2">
                    <span :class="snapshotMatchClass(snapshot.match_status)">{{ snapshot.match_status }}</span>
                  </td>
                  <td class="px-3 py-2">
                    <input
                      class="form-input w-28 text-sm"
                      type="number"
                      min="1"
                      :value="accountIdBySnapshot[snapshot.id] ?? snapshot.account_id ?? ''"
                      @input="setSnapshotAccountId(snapshot.id, ($event.target as HTMLInputElement).value)"
                    />
                  </td>
                  <td class="px-3 py-2">
                    <div class="flex flex-wrap gap-2">
                      <button
                        type="button"
                        class="btn btn-primary btn-sm"
                        :disabled="bindingSnapshot === snapshot.id || !snapshotAccountId(snapshot)"
                        @click="bindSnapshot(snapshot)"
                      >
                        {{ t('admin.accounts.balanceSync.bind') }}
                      </button>
                      <button
                        v-if="snapshot.account_id"
                        type="button"
                        class="btn btn-secondary btn-sm"
                        :disabled="bindingSnapshot === snapshot.id"
                        @click="clearSnapshotBinding(snapshot)"
                      >
                        {{ t('admin.accounts.balanceSync.clearBinding') }}
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
            <div v-if="!snapshotsLoading && snapshots.length === 0" class="px-4 py-8 text-center text-sm text-gray-400">
              {{ t('admin.accounts.balanceSync.emptySnapshots') }}
            </div>
          </div>
        </div>
      </section>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { BalanceSite, BalanceSitePayload, BalanceSnapshot, ExternalBalanceAmount, ExternalSubscriptionBalance } from '@/types'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const enabledClass = 'rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
const disabledClass = 'rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300'

const sites = ref<BalanceSite[]>([])
const snapshots = ref<BalanceSnapshot[]>([])
const loading = ref(false)
const snapshotsLoading = ref(false)
const saving = ref(false)
const refreshingAll = ref(false)
const refreshingSite = ref<number | null>(null)
const bindingSnapshot = ref<number | null>(null)
const editingId = ref<number | null>(null)
const snapshotSiteId = ref(0)
const accountIdBySnapshot = reactive<Record<number, number | ''>>({})

const form = reactive<BalanceSitePayload>({
  platform: 'newapi',
  name: '',
  base_url: '',
  username: '',
  email: '',
  password: '',
  enabled: true,
  refresh_interval_minutes: 180
})

const resetForm = () => {
  editingId.value = null
  form.platform = 'newapi'
  form.name = ''
  form.base_url = ''
  form.username = ''
  form.email = ''
  form.password = ''
  form.enabled = true
  form.refresh_interval_minutes = 180
}

const editSite = (site: BalanceSite) => {
  editingId.value = site.id
  form.platform = site.platform
  form.name = site.name
  form.base_url = site.base_url
  form.username = site.username || ''
  form.email = site.email || ''
  form.password = ''
  form.enabled = site.enabled
  form.refresh_interval_minutes = site.refresh_interval_minutes || 180
  snapshotSiteId.value = site.id
  loadSnapshots()
}

const payloadFromForm = (): BalanceSitePayload => {
  const payload: BalanceSitePayload = {
    platform: form.platform,
    name: form.name.trim(),
    base_url: form.base_url.trim(),
    username: form.username?.trim() || undefined,
    email: form.email?.trim() || undefined,
    enabled: form.enabled,
    refresh_interval_minutes: Number(form.refresh_interval_minutes) || 180
  }
  if (form.password && form.password.trim()) {
    payload.password = form.password
  }
  return payload
}

const loadSites = async () => {
  sites.value = await adminAPI.balanceSites.list()
  if (editingId.value && !sites.value.some(site => site.id === editingId.value)) {
    resetForm()
  }
}

const loadSnapshots = async () => {
  snapshotsLoading.value = true
  try {
    const result = await adminAPI.balanceSites.listSnapshots(snapshotSiteId.value || undefined)
    snapshots.value = result.items
    for (const key of Object.keys(accountIdBySnapshot)) {
      delete accountIdBySnapshot[Number(key)]
    }
  } finally {
    snapshotsLoading.value = false
  }
}

const loadAll = async () => {
  loading.value = true
  try {
    await loadSites()
    await loadSnapshots()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.balanceSync.loadFailed'))
  } finally {
    loading.value = false
  }
}

const saveSite = async () => {
  saving.value = true
  try {
    const payload = payloadFromForm()
    const site = editingId.value
      ? await adminAPI.balanceSites.update(editingId.value, payload)
      : await adminAPI.balanceSites.create(payload)
    editingId.value = site.id
    form.password = ''
    await loadSites()
    appStore.showSuccess(t('common.saved'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    saving.value = false
  }
}

const deleteCurrentSite = async () => {
  if (!editingId.value || !confirm(t('common.confirm'))) return
  saving.value = true
  try {
    await adminAPI.balanceSites.delete(editingId.value)
    resetForm()
    await loadAll()
    appStore.showSuccess(t('common.deleted'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    saving.value = false
  }
}

const refreshCurrentSite = async () => {
  if (!editingId.value) return
  refreshingSite.value = editingId.value
  try {
    await adminAPI.balanceSites.refresh(editingId.value)
    await loadAll()
    appStore.showSuccess(t('admin.accounts.balanceSync.refreshQueued'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.balanceSync.refreshFailed'))
  } finally {
    refreshingSite.value = null
  }
}

const refreshAllSites = async () => {
  refreshingAll.value = true
  try {
    await adminAPI.balanceSites.refreshAll()
    await loadAll()
    appStore.showSuccess(t('admin.accounts.balanceSync.refreshQueued'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.balanceSync.refreshFailed'))
  } finally {
    refreshingAll.value = false
  }
}

const setSnapshotAccountId = (snapshotId: number, value: string) => {
  const parsed = Number(value)
  accountIdBySnapshot[snapshotId] = Number.isFinite(parsed) && parsed > 0 ? parsed : ''
}

const snapshotAccountId = (snapshot: BalanceSnapshot) => {
  const value = accountIdBySnapshot[snapshot.id] ?? snapshot.account_id
  return typeof value === 'number' && value > 0 ? value : null
}

const bindSnapshot = async (snapshot: BalanceSnapshot) => {
  const accountId = snapshotAccountId(snapshot)
  if (!accountId) return
  bindingSnapshot.value = snapshot.id
  try {
    await adminAPI.balanceSites.upsertBinding(accountId, {
      site_id: snapshot.site_id,
      external_key_id: snapshot.external_key_id || null,
      key_last4: snapshot.key_last4
    })
    await loadSnapshots()
    appStore.showSuccess(t('admin.accounts.balanceSync.bindingUpdated'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    bindingSnapshot.value = null
  }
}

const clearSnapshotBinding = async (snapshot: BalanceSnapshot) => {
  if (!snapshot.account_id) return
  bindingSnapshot.value = snapshot.id
  try {
    await adminAPI.balanceSites.deleteBinding(snapshot.account_id)
    await loadSnapshots()
    appStore.showSuccess(t('admin.accounts.balanceSync.bindingCleared'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    bindingSnapshot.value = null
  }
}

const formatBalance = (balance?: ExternalBalanceAmount | null) => {
  if (!balance) return '-'
  if (balance.unlimited) return t('admin.accounts.balanceSync.unlimited')
  const value = typeof balance.remain === 'number'
    ? balance.remain
    : typeof balance.limit === 'number' && typeof balance.used === 'number'
      ? Math.max(balance.limit - balance.used, 0)
      : null
  if (value === null) return balance.unit || '-'
  return `${value.toFixed(value >= 1 ? 2 : 4)} ${balance.unit || ''}`.trim()
}

const formatSubscriptionBalance = (balance?: ExternalSubscriptionBalance | null) => {
  if (!balance) return '-'
  const parts: string[] = []
  if (balance.daily) parts.push(`${t('admin.accounts.balanceSync.dailyShort')} ${formatBalance(balance.daily)}`)
  if (balance.weekly) parts.push(`${t('admin.accounts.balanceSync.weeklyShort')} ${formatBalance(balance.weekly)}`)
  if (balance.monthly) parts.push(`${t('admin.accounts.balanceSync.monthlyShort')} ${formatBalance(balance.monthly)}`)
  return parts.length ? parts.join(' / ') : '-'
}

const formatRate = (rate?: number | null) => typeof rate === 'number' ? `${rate.toFixed(2)}x` : '-'

const refreshStatusLabel = (status: string) => {
  if (!status) return t('admin.accounts.balanceSync.notRefreshed')
  const key = `admin.accounts.balanceSync.refreshStatus.${status}`
  const label = t(key)
  return label === key ? status : label
}

const snapshotMatchClass = (status: string) => {
  const base = 'rounded px-1.5 py-0.5 text-xs font-medium'
  if (status === 'matched' || status === 'manual') return `${base} bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300`
  if (status === 'ambiguous') return `${base} bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300`
  return `${base} bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300`
}

watch(() => props.show, (show) => {
  if (show) {
    loadAll()
  }
})
</script>
