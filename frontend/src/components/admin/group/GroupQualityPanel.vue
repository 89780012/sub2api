<template>
  <BaseDialog :show="show" :title="t('admin.groups.qualityPanel.title')" width="extra-wide" @close="emit('close')">
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

      <div class="flex items-center justify-between gap-3">
        <div class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.qualityPanel.description') }}
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

      <div v-else class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
        <div class="max-h-[65vh] overflow-auto">
          <table class="min-w-full text-sm">
            <thead class="sticky top-0 z-[1] bg-gray-50 dark:bg-dark-700">
              <tr class="border-b border-gray-200 dark:border-dark-600">
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.account') }}</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.score') }}</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.successRate') }}</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.ttftFast') }}</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.ttftSlow') }}</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.penalties') }}</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.samples') }}</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.auxiliary') }}</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.columns.formula') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-600">
              <tr v-for="item in items" :key="item.account_id" class="align-top hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-3 py-3">
                  <div class="flex flex-col gap-1">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="font-medium text-gray-900 dark:text-white">{{ item.account_name }}</span>
                      <span class="text-xs text-gray-400">#{{ item.account_id }}</span>
                      <span class="inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium" :class="statusClass(item.status)">
                        {{ item.status }}
                      </span>
                    </div>
                    <div class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                      <span>{{ item.platform }}</span>
                      <span>{{ item.account_type }}</span>
                      <span>{{ t('admin.groups.qualityPanel.priorityLabel', { priority: item.priority }) }}</span>
                      <span>{{ t('admin.groups.qualityPanel.concurrencyLabel', { concurrency: item.concurrency }) }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <div class="flex flex-col gap-1">
                    <span class="font-mono text-sm font-semibold" :class="item.quality_known ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
                      {{ formatScore(item.quality_score) }}
                    </span>
                    <span class="text-xs text-gray-500 dark:text-gray-400">
                      {{ item.quality_known ? t('admin.groups.qualityPanel.known') : t('admin.groups.qualityPanel.unknown') }}
                    </span>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <div class="space-y-1 text-xs">
                    <div>{{ formatPercent(item.recent_success_rate) }}</div>
                    <div class="text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.errorRateLabel', { value: formatPercent(item.error_rate) }) }}</div>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <div class="space-y-1 text-xs">
                    <div>&le;5s: {{ formatPercent(item.ttft_le_5s_rate) }} / +{{ formatScore(item.ttft_5s_component) }}</div>
                    <div>&le;10s: {{ formatPercent(item.ttft_le_10s_rate) }} / +{{ formatScore(item.ttft_10s_component) }}</div>
                    <div class="text-emerald-600 dark:text-emerald-400">{{ t('admin.groups.qualityPanel.fastBonusLabel', { value: formatScore(item.fast_bonus) }) }}</div>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <div class="space-y-1 text-xs">
                    <div>&gt;10s: {{ formatPercent(item.ttft_gt_10s_rate) }}</div>
                    <div>&gt;20s: {{ formatPercent(item.ttft_gt_20s_rate) }}</div>
                    <div>&gt;40s: {{ formatPercent(item.ttft_gt_40s_rate) }}</div>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <div class="space-y-1 text-xs">
                    <div class="text-amber-600 dark:text-amber-400">{{ t('admin.groups.qualityPanel.slowPenaltyLabel', { value: formatScore(item.slow_penalty) }) }}</div>
                    <div class="text-rose-600 dark:text-rose-400">{{ t('admin.groups.qualityPanel.errorPenaltyLabel', { value: formatScore(item.error_penalty) }) }}</div>
                    <div class="text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.baseLabel', { value: formatScore(item.neutral_base) }) }}</div>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <div class="space-y-1 text-xs">
                    <div>{{ t('admin.groups.qualityPanel.totalRequestsLabel', { value: item.total_requests }) }}</div>
                    <div>{{ t('admin.groups.qualityPanel.failureRequestsLabel', { value: item.failure_requests }) }}</div>
                    <div>{{ t('admin.groups.qualityPanel.ttftSamplesLabel', { value: item.ttft_sample_count }) }}</div>
                    <div class="text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.confidenceLabel', { value: formatPercent(item.sample_confidence) }) }}</div>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <div class="space-y-1 text-xs">
                    <div>{{ t('admin.groups.qualityPanel.auxRequestsLabel', { value: item.auxiliary_total_requests }) }}</div>
                    <div>{{ t('admin.groups.qualityPanel.auxFailuresLabel', { value: item.auxiliary_failure_requests }) }}</div>
                    <div>{{ t('admin.groups.qualityPanel.auxTTFTLabel', { value: item.auxiliary_ttft_sample_count }) }}</div>
                    <div class="text-gray-500 dark:text-gray-400">{{ t('admin.groups.qualityPanel.auxWeightLabel', { value: formatPercent(item.auxiliary_weight) }) }}</div>
                  </div>
                </td>
                <td class="px-3 py-3">
                  <div class="space-y-2">
                    <code class="block whitespace-pre-wrap break-words rounded bg-gray-100 px-2 py-1.5 text-[11px] leading-5 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                      {{ item.score_breakdown }}
                    </code>
                    <div class="grid grid-cols-2 gap-2 text-[11px] text-gray-500 dark:text-gray-400 xl:grid-cols-4">
                      <span>{{ t('admin.groups.qualityPanel.successComponentLabel', { value: formatScore(item.success_component) }) }}</span>
                      <span>{{ t('admin.groups.qualityPanel.fastBonusLabel', { value: formatScore(item.fast_bonus) }) }}</span>
                      <span>{{ t('admin.groups.qualityPanel.slowPenaltyLabel', { value: formatScore(item.slow_penalty) }) }}</span>
                      <span>{{ t('admin.groups.qualityPanel.errorPenaltyLabel', { value: formatScore(item.error_penalty) }) }}</span>
                    </div>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
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

const loading = ref(false)
const items = ref<GroupAccountQualityItem[]>([])
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

const loadQuality = async () => {
  if (!props.group) return
  loading.value = true
  try {
    const data = await adminAPI.groups.getAccountQuality(props.group.id)
    items.value = [...data.items].sort((a, b) => {
      if (b.quality_score !== a.quality_score) return b.quality_score - a.quality_score
      if (a.priority !== b.priority) return a.priority - b.priority
      return a.account_id - b.account_id
    })
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

watch(() => props.show, (show) => {
  if (show && props.group) {
    loadQuality()
  }
})

const formatPercent = (value: number) => `${(value * 100).toFixed(1)}%`
const formatScore = (value: number) => value.toFixed(4)

const statusClass = (status: string) => {
  if (status === 'active') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
  if (status === 'inactive') return 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}
</script>
