<template>
  <AppLayout>
    <div class="space-y-5">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('admin.accounts.rateAnalysis.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.rateAnalysis.description') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <DateRangePicker
            v-model:start-date="startDate"
            v-model:end-date="endDate"
            @change="reload"
          />
          <button class="btn btn-secondary" :disabled="loading" @click="reload">
            <Icon name="refresh" size="sm" class="mr-1.5" />
            {{ t('common.refresh') }}
          </button>
          <router-link class="btn btn-primary" to="/admin/channels/pricing">
            <Icon name="cog" size="sm" class="mr-1.5" />
            {{ t('admin.accounts.rateAnalysis.pricingRules') }}
          </router-link>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3 lg:grid-cols-6">
        <div v-for="card in summaryCards" :key="card.key" class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ card.label }}</div>
          <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ card.value }}</div>
        </div>
      </div>

      <div class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200">
        {{ t('admin.accounts.rateAnalysis.coverageHint') }}
        <router-link class="font-medium underline underline-offset-2" to="/admin/channels/pricing">
          {{ t('admin.accounts.rateAnalysis.pricingRules') }}
        </router-link>
      </div>

      <DataTable
        :columns="columns"
        :data="items"
        :loading="loading"
        :server-side-sort="true"
        :default-sort-key="sortBy"
        :default-sort-order="sortOrder"
        row-key="group_id"
        @sort="handleSort"
      >
        <template #cell-group_name="{ row }">
          <div class="min-w-0">
            <div class="truncate font-medium text-gray-900 dark:text-white">
              {{ displayGroupName(row) }}
            </div>
            <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
              {{ row.group_id > 0 ? `#${row.group_id}` : t('admin.accounts.ungroupedGroup') }}
            </div>
          </div>
        </template>

        <template #cell-coverage_rate="{ row }">
          <div class="space-y-1">
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ formatPercent(row.coverage_rate) }}
            </div>
            <div class="h-1.5 w-24 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
              <div class="h-full rounded-full bg-primary-500" :style="{ width: `${Math.max(0, Math.min(100, row.coverage_rate * 100))}%` }"></div>
            </div>
          </div>
        </template>

        <template #cell-inferred_multiplier="{ row }">
          <div v-if="row.inferred_multiplier !== null" class="font-semibold text-gray-900 dark:text-white">
            {{ formatMultiplier(row.inferred_multiplier) }}
          </div>
          <span v-else class="inline-flex rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
            {{ t('admin.accounts.rateAnalysis.noValidSamples') }}
          </span>
        </template>

        <template #empty>
          <div class="py-10 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.rateAnalysis.empty') }}
          </div>
        </template>
      </DataTable>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import { accountsAPI } from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'
import type { AccountPoolRateAnalysisItem, AccountPoolRateAnalysisResponse, AccountPoolRateAnalysisSortBy } from '@/types'
import type { Column } from '@/components/common/types'

const { t } = useI18n()
const appStore = useAppStore()

const formatDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = new Date()
const start = new Date(today)
start.setDate(today.getDate() - 6)

const startDate = ref(formatDate(start))
const endDate = ref(formatDate(today))
const sortBy = ref<AccountPoolRateAnalysisSortBy>('inferred_multiplier')
const sortOrder = ref<'asc' | 'desc'>('asc')
const loading = ref(false)
const response = ref<AccountPoolRateAnalysisResponse | null>(null)
let abortController: AbortController | null = null

const isAbortError = (error: unknown): boolean => {
  const maybeError = error as { name?: unknown; code?: unknown } | null
  return maybeError?.name === 'AbortError' || maybeError?.name === 'CanceledError' || maybeError?.code === 'ERR_CANCELED'
}

const items = computed(() => response.value?.items ?? [])
const summary = computed(() => response.value?.summary ?? {
  groups: 0,
  valid_groups: 0,
  requests: 0,
  valid_requests: 0,
  uncovered_requests: 0,
  theoretical_cost: 0,
  account_cost: 0
})

const columns = computed<Column[]>(() => [
  { key: 'group_name', label: t('admin.accounts.rateAnalysis.columns.pool'), sortable: false },
  { key: 'platform', label: t('admin.accounts.columns.platform'), formatter: (value) => formatPlatform(String(value || '')) },
  { key: 'configured_rate_multiplier', label: t('admin.accounts.rateAnalysis.columns.configuredRate'), formatter: (value) => formatMultiplier(Number(value)) },
  { key: 'requests', label: t('admin.accounts.rateAnalysis.columns.requests'), sortable: true, formatter: (value) => formatNumber(Number(value)) },
  { key: 'coverage_rate', label: t('admin.accounts.rateAnalysis.columns.coverage'), sortable: true },
  { key: 'input_tokens', label: t('admin.accounts.rateAnalysis.columns.inputTokens'), formatter: (value) => formatNumber(Number(value)) },
  { key: 'output_tokens', label: t('admin.accounts.rateAnalysis.columns.outputTokens'), formatter: (value) => formatNumber(Number(value)) },
  { key: 'theoretical_cost', label: t('admin.accounts.rateAnalysis.columns.theoreticalCost'), sortable: true, formatter: (value) => formatCurrency(Number(value)) },
  { key: 'account_cost', label: t('admin.accounts.rateAnalysis.columns.accountCost'), sortable: true, formatter: (value) => formatCurrency(Number(value)) },
  { key: 'inferred_multiplier', label: t('admin.accounts.rateAnalysis.columns.inferredMultiplier'), sortable: true }
])

const summaryCards = computed(() => [
  { key: 'groups', label: t('admin.accounts.rateAnalysis.summary.groups'), value: formatNumber(summary.value.groups) },
  { key: 'valid_groups', label: t('admin.accounts.rateAnalysis.summary.validGroups'), value: formatNumber(summary.value.valid_groups) },
  { key: 'requests', label: t('admin.accounts.rateAnalysis.summary.requests'), value: formatNumber(summary.value.requests) },
  { key: 'uncovered', label: t('admin.accounts.rateAnalysis.summary.uncoveredRequests'), value: formatNumber(summary.value.uncovered_requests) },
  { key: 'theoretical', label: t('admin.accounts.rateAnalysis.summary.theoreticalCost'), value: formatCurrency(summary.value.theoretical_cost) },
  { key: 'account', label: t('admin.accounts.rateAnalysis.summary.accountCost'), value: formatCurrency(summary.value.account_cost) }
])

const displayGroupName = (row: AccountPoolRateAnalysisItem): string => {
  if (row.group_name) return row.group_name
  return t('admin.accounts.ungroupedGroup')
}

const formatNumber = (value: number): string => new Intl.NumberFormat().format(value || 0)

const formatCurrency = (value: number): string => {
  return `$${new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 4,
    maximumFractionDigits: 8
  }).format(value || 0)}`
}

const formatMultiplier = (value: number): string => {
  if (!Number.isFinite(value)) return '-'
  return `${new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4
  }).format(value)}x`
}

const formatPercent = (value: number): string => {
  return new Intl.NumberFormat(undefined, {
    style: 'percent',
    minimumFractionDigits: 1,
    maximumFractionDigits: 1
  }).format(value || 0)
}

const formatPlatform = (platform: string): string => {
  if (!platform) return '-'
  const key = `admin.accounts.platforms.${platform}`
  const label = t(key)
  return label === key ? platform : label
}

const timezoneName = (): string | undefined => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return undefined
  }
}

const reload = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const nextResponse = await accountsAPI.getPoolRateAnalysis({
      start_date: startDate.value,
      end_date: endDate.value,
      sort_by: sortBy.value,
      sort_order: sortOrder.value,
      timezone: timezoneName()
    }, { signal: controller.signal })
    if (abortController === controller) {
      response.value = nextResponse
    }
  } catch (error: unknown) {
    if (isAbortError(error)) return
    appStore.showError(t('admin.accounts.rateAnalysis.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  if (key === 'inferred_multiplier' || key === 'account_cost' || key === 'theoretical_cost' || key === 'requests' || key === 'coverage_rate') {
    sortBy.value = key
    sortOrder.value = order
    reload()
  }
}

onMounted(() => {
  reload()
})

onUnmounted(() => {
  abortController?.abort()
})
</script>
