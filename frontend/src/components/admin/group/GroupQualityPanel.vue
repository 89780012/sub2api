<template>
  <BaseDialog :show="show" :title="t('admin.groups.qualityPanel.title')" width="extra-extra-wide" @close="emit('close')">
    <div v-if="group" class="space-y-4">
      <div class="flex flex-wrap items-center gap-3 rounded-lg bg-gray-50 px-4 py-2.5 text-sm dark:bg-dark-700">
        <span class="inline-flex items-center gap-1.5" :class="platformColorClass">
          <PlatformIcon :platform="group.platform" size="sm" />
          {{ t('admin.groups.platforms.' + group.platform) }}
        </span>
        <span class="text-gray-400">|</span>
        <span class="font-medium text-gray-900 dark:text-white">{{ group.name }}</span>
        <span class="text-gray-400">|</span>
        <span class="text-gray-600 dark:text-gray-400">
          {{ t('admin.groups.qualityPanel.summary', { total: totalCount, known: knownCount, unknown: unknownCount }) }}
        </span>
      </div>

      <div class="flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-1">
          <div class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.groups.qualityPanel.description') }}
          </div>
          <div class="flex items-start gap-2 rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-700 dark:border-blue-900/50 dark:bg-blue-950/30 dark:text-blue-300">
            <Icon name="infoCircle" size="sm" class="mt-0.5 shrink-0" />
            <span>{{ t('admin.groups.qualityPanel.selectionHint') }}</span>
          </div>
        </div>
        <button type="button" class="btn btn-secondary btn-sm px-3 py-1.5" :disabled="loading" @click="loadQuality">
          <Icon v-if="loading" name="refresh" size="sm" class="mr-1 animate-spin" />
          <Icon v-else name="refresh" size="sm" class="mr-1" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <div v-if="loading" class="flex justify-center py-10">
        <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <div v-else-if="items.length === 0" class="rounded-lg border border-dashed border-gray-300 px-4 py-10 text-center text-sm text-gray-400 dark:border-dark-500 dark:text-gray-500">
        {{ t('admin.groups.qualityPanel.empty') }}
      </div>

      <template v-else>
        <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
          <div class="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/40 dark:bg-emerald-950/20">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-xs font-medium text-emerald-700 dark:text-emerald-300">{{ t('admin.groups.qualityPanel.summaryCards.preferred') }}</div>
                <div class="mt-1 text-2xl font-semibold text-emerald-900 dark:text-emerald-100">{{ preferredCount }}</div>
              </div>
              <div class="rounded-lg bg-emerald-100 p-2 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                <Icon name="checkCircle" size="sm" />
              </div>
            </div>
          </div>

          <div class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 dark:border-amber-900/40 dark:bg-amber-950/20">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-xs font-medium text-amber-700 dark:text-amber-300">{{ t('admin.groups.qualityPanel.summaryCards.watch') }}</div>
                <div class="mt-1 text-2xl font-semibold text-amber-900 dark:text-amber-100">{{ watchCount }}</div>
              </div>
              <div class="rounded-lg bg-amber-100 p-2 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">
                <Icon name="clock" size="sm" />
              </div>
            </div>
          </div>

          <div class="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 dark:border-rose-900/40 dark:bg-rose-950/20">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-xs font-medium text-rose-700 dark:text-rose-300">{{ t('admin.groups.qualityPanel.summaryCards.risk') }}</div>
                <div class="mt-1 text-2xl font-semibold text-rose-900 dark:text-rose-100">{{ riskCount }}</div>
              </div>
              <div class="rounded-lg bg-rose-100 p-2 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300">
                <Icon name="exclamationTriangle" size="sm" />
              </div>
            </div>
          </div>

          <div class="rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 dark:border-slate-700 dark:bg-slate-900/40">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-xs font-medium text-slate-700 dark:text-slate-300">{{ t('admin.groups.qualityPanel.summaryCards.lowConfidence') }}</div>
                <div class="mt-1 text-2xl font-semibold text-slate-900 dark:text-slate-100">{{ lowConfidenceCount }}</div>
              </div>
              <div class="rounded-lg bg-slate-200 p-2 text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                <Icon name="chartBar" size="sm" />
              </div>
            </div>
          </div>
        </div>

        <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
          <div class="max-h-[65vh] overflow-auto">
            <div class="divide-y divide-gray-100 dark:divide-dark-600">
              <section v-for="item in sortedItems" :key="item.account_id" class="px-4 py-4 hover:bg-gray-50 dark:hover:bg-dark-700/30">
                <div class="grid gap-4 xl:grid-cols-[minmax(0,2.2fr)_minmax(190px,0.95fr)_minmax(240px,1.1fr)_minmax(220px,1fr)_auto]">
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="truncate font-medium text-gray-900 dark:text-white">{{ item.account_name }}</span>
                      <span class="text-xs text-gray-400">#{{ item.account_id }}</span>
                      <span class="inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium" :class="statusClass(item.status)">
                        {{ item.status }}
                      </span>
                      <span
                        v-if="!item.schedulable"
                        class="inline-flex rounded-full bg-gray-200 px-2 py-0.5 text-[11px] font-medium text-gray-700 dark:bg-dark-600 dark:text-gray-300"
                      >
                        {{ t('admin.groups.qualityPanel.unschedulable') }}
                      </span>
                    </div>
                    <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                      <span>{{ item.platform }}</span>
                      <span>{{ item.account_type }}</span>
                      <span>{{ t('admin.groups.qualityPanel.scheduleRankLabel', { rank: item.schedule_rank }) }}</span>
                      <span>{{ t('admin.groups.qualityPanel.priorityLabel', { priority: item.priority }) }}</span>
                      <span>{{ t('admin.groups.qualityPanel.concurrencyLabel', { concurrency: item.concurrency }) }}</span>
                    </div>
                    <div class="mt-3 flex flex-wrap gap-2">
                      <span class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium" :class="mainStateClass(item)">
                        {{ mainStateLabel(item) }}
                      </span>
                      <span
                        v-for="chip in extraStateChips(item)"
                        :key="chip.label"
                        class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
                        :class="chip.className"
                      >
                        {{ chip.label }}
                      </span>
                    </div>
                    <div class="mt-3 flex items-start gap-2 text-xs text-gray-500 dark:text-gray-400">
                      <Icon name="infoCircle" size="xs" class="mt-0.5 shrink-0 text-gray-400 dark:text-gray-500" />
                      <div class="min-w-0">
                        <span class="font-medium text-gray-700 dark:text-gray-300">{{ t('admin.groups.qualityPanel.rankReasonTitle') }}:</span>
                        <span class="ml-1">{{ rankReason(item) }}</span>
                      </div>
                    </div>
                  </div>

                  <div class="space-y-1">
                    <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('admin.groups.qualityPanel.columns.score') }}
                    </div>
                    <div class="font-mono text-2xl font-semibold" :class="scoreToneClass(item)">
                      {{ displayScore(item) }}
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.groups.qualityPanel.baseQualityLabel', { value: formatScore(item.base_quality_score) }) }}
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.groups.qualityPanel.transientPenaltyLabel', { value: formatScore(item.transient_penalty) }) }}
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.groups.qualityPanel.appliedPenaltyLabel', { value: formatScore(item.applied_penalty) }) }}
                    </div>
                    <div v-if="item.recovery_credit > 0" class="text-xs text-sky-600 dark:text-sky-400">
                      {{ t('admin.groups.qualityPanel.recoveryCreditLabel', { value: formatScore(item.recovery_credit) }) }}
                    </div>
                  </div>

                  <div class="space-y-2">
                    <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('admin.groups.qualityPanel.columns.risk') }}
                    </div>
                    <div class="grid grid-cols-2 gap-2 text-sm">
                      <div class="rounded-md bg-gray-100 px-2.5 py-2 dark:bg-dark-700">
                        <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.slowStreakShort') }}</div>
                        <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ item.slow_streak }}</div>
                      </div>
                      <div class="rounded-md bg-gray-100 px-2.5 py-2 dark:bg-dark-700">
                        <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.errorStreakShort') }}</div>
                        <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ item.error_streak }}</div>
                      </div>
                    </div>
                    <div class="flex flex-wrap gap-2 text-xs">
                      <span class="inline-flex items-center rounded-full bg-emerald-100 px-2 py-0.5 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                        {{ t('admin.groups.qualityPanel.signalFast5s', { value: formatPercent(item.ttft_le_5s_rate) }) }}
                      </span>
                      <span class="inline-flex items-center rounded-full px-2 py-0.5" :class="slowRateClass(item.ttft_gt_10s_rate)">
                        {{ t('admin.groups.qualityPanel.signalSlow10s', { value: formatPercent(item.ttft_gt_10s_rate) }) }}
                      </span>
                      <span class="inline-flex items-center rounded-full px-2 py-0.5" :class="errorRateClass(item.error_rate)">
                        {{ t('admin.groups.qualityPanel.signalErrorRate', { value: formatPercent(item.error_rate) }) }}
                      </span>
                    </div>
                  </div>

                  <div class="space-y-2">
                    <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('admin.groups.qualityPanel.columns.samples') }}
                    </div>
                    <div class="grid grid-cols-2 gap-2 text-sm">
                      <div class="rounded-md bg-gray-100 px-2.5 py-2 dark:bg-dark-700">
                        <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.totalRequestsShort') }}</div>
                        <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ item.total_requests }}</div>
                      </div>
                      <div class="rounded-md bg-gray-100 px-2.5 py-2 dark:bg-dark-700">
                        <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.ttftSamplesShort') }}</div>
                        <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ item.ttft_sample_count }}</div>
                      </div>
                    </div>
                    <div class="space-y-1 text-xs text-gray-500 dark:text-gray-400">
                      <div>{{ t('admin.groups.qualityPanel.confidenceLabel', { value: formatPercent(item.sample_confidence) }) }}</div>
                      <div>{{ t('admin.groups.qualityPanel.auxWeightLabel', { value: formatPercent(item.auxiliary_weight) }) }}</div>
                    </div>
                  </div>

                  <div class="flex items-start justify-end xl:justify-center">
                    <button
                      type="button"
                      class="inline-flex items-center gap-1 rounded-md border border-gray-200 px-2.5 py-1.5 text-xs font-medium text-gray-600 transition hover:bg-gray-100 dark:border-dark-500 dark:text-gray-300 dark:hover:bg-dark-700"
                      @click="toggleExpanded(item.account_id)"
                    >
                      <Icon :name="isExpanded(item.account_id) ? 'chevronUp' : 'chevronDown'" size="xs" />
                      {{ isExpanded(item.account_id) ? t('common.collapse') : t('common.expand') }}
                    </button>
                  </div>
                </div>

                <div class="mt-3 flex flex-wrap gap-2">
                  <span
                    v-for="chip in signalChips(item)"
                    :key="chip.label"
                    class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
                    :class="chip.className"
                  >
                    {{ chip.label }}
                  </span>
                </div>

                <div v-if="isExpanded(item.account_id)" class="mt-4 border-t border-gray-200 pt-4 dark:border-dark-600">
                  <div class="grid gap-3 xl:grid-cols-3">
                    <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700/40">
                      <div class="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-200">
                        {{ t('admin.groups.qualityPanel.breakdownTitle') }}
                      </div>
                      <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                        <div>{{ t('admin.groups.qualityPanel.baseLabel', { value: formatScore(item.neutral_base) }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.successComponentLabel', { value: formatScore(item.success_component) }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.ttft5sLabel', { value: formatScore(item.ttft_5s_component) }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.ttft10sLabel', { value: formatScore(item.ttft_10s_component) }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.fastBonusLabel', { value: formatScore(item.fast_bonus) }) }}</div>
                        <div class="text-amber-600 dark:text-amber-400">{{ t('admin.groups.qualityPanel.slowPenaltyLabel', { value: formatScore(item.slow_penalty) }) }}</div>
                        <div class="text-rose-600 dark:text-rose-400">{{ t('admin.groups.qualityPanel.errorPenaltyLabel', { value: formatScore(item.error_penalty) }) }}</div>
                      </div>
                    </div>

                    <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700/40">
                      <div class="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-200">
                        {{ t('admin.groups.qualityPanel.recoveryTitle') }}
                      </div>
                      <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                        <div>{{ t('admin.groups.qualityPanel.totalRequestsLabel', { value: item.total_requests }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.failureRequestsLabel', { value: item.failure_requests }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.slowStreakLabel', { value: item.slow_streak }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.errorStreakLabel', { value: item.error_streak }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.recoverySuccessLabel', { value: item.recovery_success_streak }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.recoveryFastLabel', { value: item.recovery_fast_streak }) }}</div>
                        <div class="text-sky-600 dark:text-sky-400">{{ t('admin.groups.qualityPanel.recoveryCreditLabel', { value: formatScore(item.recovery_credit) }) }}</div>
                        <div class="text-rose-600 dark:text-rose-400">{{ t('admin.groups.qualityPanel.appliedPenaltyLabel', { value: formatScore(item.applied_penalty) }) }}</div>
                      </div>
                    </div>

                    <div class="rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700/40">
                      <div class="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-200">
                        {{ t('admin.groups.qualityPanel.auxiliaryTitle') }}
                      </div>
                      <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                        <div>{{ t('admin.groups.qualityPanel.auxRequestsLabel', { value: item.auxiliary_total_requests }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.auxFailuresLabel', { value: item.auxiliary_failure_requests }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.auxTTFTLabel', { value: item.auxiliary_ttft_sample_count }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.auxWeightLabel', { value: formatPercent(item.auxiliary_weight) }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.fast5sRateLabel', { value: formatPercent(item.ttft_le_5s_rate) }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.fast10sRateLabel', { value: formatPercent(item.ttft_le_10s_rate) }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.slow20sRateLabel', { value: formatPercent(item.ttft_gt_20s_rate) }) }}</div>
                        <div>{{ t('admin.groups.qualityPanel.slow40sRateLabel', { value: formatPercent(item.ttft_gt_40s_rate) }) }}</div>
                      </div>
                    </div>
                  </div>

                  <div class="mt-3 rounded-lg bg-gray-50 px-3 py-3 dark:bg-dark-700/40">
                    <div class="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-200">
                      {{ t('admin.groups.qualityPanel.formulaTitle') }}
                    </div>
                    <div class="mb-2 text-[11px] text-gray-500 dark:text-gray-400">
                      {{ item.schedule_sort_key }}
                    </div>
                    <code class="block whitespace-pre-wrap break-words rounded bg-gray-100 px-2 py-1.5 text-[11px] leading-5 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                      {{ item.score_breakdown }}
                    </code>
                  </div>
                </div>
              </section>
            </div>
          </div>
        </div>
      </template>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, GroupAccountQualityItem } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  show: boolean
  group: AdminGroup | null
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

type PanelChip = {
  label: string
  className: string
}

const loading = ref(false)
const items = ref<GroupAccountQualityItem[]>([])
const expandedAccountIds = ref<number[]>([])
const totalCount = ref(0)
const knownCount = ref(0)
const unknownCount = ref(0)

const platformColorClass = computed(() => {
  switch (props.group?.platform) {
    case 'anthropic': return 'text-orange-700 dark:text-orange-400'
    case 'openai': return 'text-emerald-700 dark:text-emerald-400'
    case 'antigravity': return 'text-purple-700 dark:text-purple-400'
    default: return 'text-blue-700 dark:text-blue-400'
  }
})

const getRecoveryTieBreak = (item: GroupAccountQualityItem) => {
  return item.recovery_credit + Math.min(item.recovery_success_streak * 0.01, 0.04) + Math.min(item.recovery_fast_streak * 0.02, 0.08)
}

const getMainState = (item: GroupAccountQualityItem) => {
  if (!item.quality_known) return 'unknown'
  if (item.error_streak > 0 || item.applied_penalty >= 0.35 || item.ttft_gt_40s_rate >= 0.15) return 'risk'
  if (item.slow_streak > 0 || item.applied_penalty > 0 || item.ttft_gt_20s_rate >= 0.2 || item.error_rate >= 0.12) return 'watch'
  if (item.sample_confidence < 0.4 || item.total_requests < 3) return 'lowConfidence'
  return 'stable'
}

const sortedItems = computed(() => {
  return [...items.value].sort((a, b) => {
    if (a.schedule_rank !== b.schedule_rank) return a.schedule_rank - b.schedule_rank
    return a.account_id - b.account_id
  })
})

const preferredCount = computed(() => sortedItems.value.filter(item => item.quality_known && getMainState(item) === 'stable').length)
const watchCount = computed(() => sortedItems.value.filter(item => {
  const state = getMainState(item)
  return state === 'watch'
}).length)
const riskCount = computed(() => sortedItems.value.filter(item => getMainState(item) === 'risk').length)
const lowConfidenceCount = computed(() => sortedItems.value.filter(item => !item.quality_known || item.sample_confidence < 0.4 || item.total_requests < 3).length)

const loadQuality = async () => {
  if (!props.group) return
  loading.value = true
  try {
    const data = await adminAPI.groups.getAccountQuality(props.group.id)
    items.value = data.items
    expandedAccountIds.value = []
    totalCount.value = data.account_count
    knownCount.value = data.known_account_count
    unknownCount.value = data.unknown_account_count
  } catch (error) {
    console.error('Failed to load group account quality:', error)
    appStore.showError(t('admin.groups.qualityPanel.failedToLoad'))
  } finally {
    loading.value = false
  }
}

watch(() => [props.show, props.group?.id], ([show, groupID]) => {
  if (show && groupID) {
    loadQuality()
  }
})

const toggleExpanded = (accountID: number) => {
  if (expandedAccountIds.value.includes(accountID)) {
    expandedAccountIds.value = expandedAccountIds.value.filter(id => id !== accountID)
    return
  }
  expandedAccountIds.value = [...expandedAccountIds.value, accountID]
}

const isExpanded = (accountID: number) => expandedAccountIds.value.includes(accountID)

const formatPercent = (value: number) => `${(value * 100).toFixed(1)}%`
const formatScore = (value: number) => value.toFixed(4)
const displayScore = (item: GroupAccountQualityItem) => item.quality_known ? formatScore(item.effective_quality_score) : '--'

const statusClass = (status: string) => {
  if (status === 'active') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
  if (status === 'inactive') return 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

const mainStateLabel = (item: GroupAccountQualityItem) => {
  const state = getMainState(item)
  switch (state) {
    case 'stable': return t('admin.groups.qualityPanel.states.stable')
    case 'watch': return t('admin.groups.qualityPanel.states.watch')
    case 'risk': return t('admin.groups.qualityPanel.states.risk')
    case 'lowConfidence': return t('admin.groups.qualityPanel.states.lowConfidence')
    default: return t('admin.groups.qualityPanel.states.unknown')
  }
}

const mainStateClass = (item: GroupAccountQualityItem) => {
  const state = getMainState(item)
  if (state === 'stable') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (state === 'watch') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (state === 'risk') return 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300'
  if (state === 'lowConfidence') return 'bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
  return 'bg-gray-200 text-gray-700 dark:bg-dark-600 dark:text-gray-300'
}

const rankReason = (item: GroupAccountQualityItem) => {
  if (!item.quality_known) return t('admin.groups.qualityPanel.rankReasons.unknown')

  if (item.priority > sortedItems.value[0]?.priority) return t('admin.groups.qualityPanel.rankReasons.lowerPriority')
  if (getMainState(item) === 'lowConfidence') return t('admin.groups.qualityPanel.rankReasons.lowConfidence')
  if (item.error_streak > 0) return t('admin.groups.qualityPanel.rankReasons.errorStreak')
  if (item.applied_penalty >= 0.35) {
    return item.recovery_credit > 0
      ? t('admin.groups.qualityPanel.rankReasons.recovering')
      : t('admin.groups.qualityPanel.rankReasons.highPenalty')
  }
  if (item.ttft_gt_40s_rate >= 0.15) return t('admin.groups.qualityPanel.rankReasons.severeSlow')
  if (item.slow_streak > 0) return t('admin.groups.qualityPanel.rankReasons.slowStreak')
  if (item.schedule_rank > 1 && item.last_used_at) return t('admin.groups.qualityPanel.rankReasons.recentlyUsed')
  if (item.applied_penalty > 0) {
    return item.recovery_credit > 0
      ? t('admin.groups.qualityPanel.rankReasons.recovering')
      : t('admin.groups.qualityPanel.rankReasons.penaltyActive')
  }
  if (item.ttft_gt_20s_rate >= 0.2 || item.error_rate >= 0.12) return t('admin.groups.qualityPanel.rankReasons.elevatedRisk')
  if (item.recovery_credit > 0 || getRecoveryTieBreak(item) > 0) return t('admin.groups.qualityPanel.rankReasons.stableRecoveryEdge')
  return t('admin.groups.qualityPanel.rankReasons.stableLeading')
}

const scoreToneClass = (item: GroupAccountQualityItem) => {
  const state = getMainState(item)
  if (!item.quality_known) return 'text-gray-500 dark:text-gray-400'
  if (state === 'risk') return 'text-rose-600 dark:text-rose-400'
  if (state === 'watch') return 'text-amber-600 dark:text-amber-400'
  return 'text-emerald-600 dark:text-emerald-400'
}

const extraStateChips = (item: GroupAccountQualityItem): PanelChip[] => {
  const chips: PanelChip[] = []
  if (item.recovery_credit > 0) {
    chips.push({
      label: t('admin.groups.qualityPanel.states.recovering'),
      className: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
    })
  }
  if (item.sample_confidence < 0.4 || item.total_requests < 3) {
    chips.push({
      label: t('admin.groups.qualityPanel.states.lowConfidence'),
      className: 'bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
    })
  }
  if (item.auxiliary_weight >= 0.1) {
    chips.push({
      label: t('admin.groups.qualityPanel.states.auxHeavy'),
      className: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
    })
  }
  return chips
}

const signalChips = (item: GroupAccountQualityItem): PanelChip[] => {
  const chips: PanelChip[] = []

  chips.push({
    label: t('admin.groups.qualityPanel.signalSuccessRate', { value: formatPercent(item.recent_success_rate) }),
    className: 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  })

  if (item.applied_penalty > 0) {
    chips.push({
      label: t('admin.groups.qualityPanel.signalPenalty', { value: formatScore(item.applied_penalty) }),
      className: 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300'
    })
  }

  if (item.recovery_credit > 0) {
    chips.push({
      label: t('admin.groups.qualityPanel.signalRecovery', { value: formatScore(item.recovery_credit) }),
      className: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
    })
  }

  if (item.auxiliary_weight > 0.05) {
    chips.push({
      label: t('admin.groups.qualityPanel.signalAuxWeight', { value: formatPercent(item.auxiliary_weight) }),
      className: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
    })
  }

  return chips
}

const slowRateClass = (value: number) => {
  if (value >= 0.3) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}

const errorRateClass = (value: number) => {
  if (value >= 0.12) return 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300'
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}
</script>
