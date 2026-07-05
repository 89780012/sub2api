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
          <div class="flex flex-wrap items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <span class="inline-flex rounded-full bg-slate-100 px-2 py-1 dark:bg-dark-700">
              {{ t('admin.groups.qualityPanel.primaryModeLabel', { mode: primaryModeLabel }) }}
            </span>
            <span v-if="activePrimaryItem" class="inline-flex rounded-full bg-emerald-100 px-2 py-1 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
              {{ activePrimaryBadgeLabel }} {{ activePrimaryItem.account_name }}
            </span>
            <span v-if="manualPrimaryItem" class="inline-flex rounded-full bg-blue-100 px-2 py-1 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
              {{ t('admin.groups.qualityPanel.primaryLabels.manualPrimary') }} {{ manualPrimaryItem.account_name }}
            </span>
          </div>
          <div v-if="activePrimaryMeta" class="text-xs text-gray-500 dark:text-gray-400">
            {{ activePrimaryMeta }}
          </div>
          <div
            v-if="recentScheduleLines.length"
            class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs text-gray-600 dark:border-dark-600 dark:bg-dark-800/60 dark:text-gray-300"
          >
            <div class="font-medium text-gray-800 dark:text-gray-100">
              {{ t('admin.groups.qualityPanel.recentScheduleTitle') }}
            </div>
            <div class="mt-1 space-y-1">
              <div
                v-for="(line, index) in recentScheduleLines"
                :key="`${index}-${line}`"
                class="leading-relaxed"
              >
                {{ line }}
              </div>
            </div>
          </div>
          <div
            v-if="failoverTakeoverHint"
            class="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-300"
          >
            <Icon name="sync" size="sm" class="mt-0.5 shrink-0" />
            <span>{{ failoverTakeoverHint }}</span>
          </div>
          <div class="flex items-start gap-2 rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-700 dark:border-blue-900/50 dark:bg-blue-950/30 dark:text-blue-300">
            <Icon name="infoCircle" size="sm" class="mt-0.5 shrink-0" />
            <span>{{ t('admin.groups.qualityPanel.selectionHint') }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="btn btn-secondary btn-sm px-3 py-1.5"
            :disabled="loading || actionLoading"
            @click="switchToAuto"
          >
            切回自动
          </button>
          <button type="button" class="btn btn-secondary btn-sm px-3 py-1.5" :disabled="loading || actionLoading" @click="loadQuality">
            <Icon v-if="loading" name="refresh" size="sm" class="mr-1 animate-spin" />
            <Icon v-else name="refresh" size="sm" class="mr-1" />
            {{ t('common.refresh') }}
          </button>
        </div>
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
                        v-if="item.is_active_primary"
                        class="inline-flex items-center rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                      >
                        {{ rowActivePrimaryLabel(item) }}
                      </span>
                      <span
                        v-if="item.is_manual_primary"
                        class="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-[11px] font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
                      >
                        {{ t('admin.groups.qualityPanel.primaryLabels.manualPrimary') }}
                      </span>
                      <span
                        v-if="item.is_pool_mode"
                        class="inline-flex items-center rounded-full bg-purple-100 px-2 py-0.5 text-[11px] font-medium text-purple-700 dark:bg-purple-900/30 dark:text-purple-300"
                      >
                        池模式
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
                    <div v-if="item.external_balance || item.external_rate_multiplier != null || item.external_balance_match_status" class="mt-3">
                      <div class="mb-1 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                        {{ t('admin.groups.qualityPanel.columns.externalBalance') }}
                      </div>
                      <ExternalBalanceCell
                        :balance="item.external_balance"
                        :subscription-balance="item.external_subscription_balance"
                        :match-status="item.external_balance_match_status"
                        :site-name="item.external_balance_site_name"
                        :key-name="item.external_balance_key_name"
                        :key-last4="item.external_balance_key_last4"
                        :fetched-at="item.external_balance_fetched_at"
                      />
                      <div v-if="typeof item.external_rate_multiplier === 'number'" class="mt-1 text-xs text-gray-600 dark:text-gray-300">
                        {{ t('admin.accounts.columns.upstreamRateMultiplier') }}:
                        <span class="font-mono">{{ item.external_rate_multiplier.toFixed(2) }}x</span>
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
                    <div class="flex flex-col items-end gap-2">
                      <button
                        type="button"
                        class="inline-flex items-center gap-1 rounded-md border border-blue-200 px-2.5 py-1.5 text-xs font-medium text-blue-700 transition hover:bg-blue-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-blue-900/50 dark:text-blue-300 dark:hover:bg-blue-950/30"
                        :disabled="actionLoading || item.is_manual_primary"
                        @click="setPrimary(item)"
                      >
                        设为主账号
                      </button>
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
import type { AdminGroup, GroupAccountQualityItem, UsageScheduleCandidateScore, UsageScheduleTrace } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import ExternalBalanceCell from '@/components/admin/account/ExternalBalanceCell.vue'

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
const actionLoading = ref(false)
const items = ref<GroupAccountQualityItem[]>([])
const expandedAccountIds = ref<number[]>([])
const totalCount = ref(0)
const knownCount = ref(0)
const unknownCount = ref(0)
const primaryMeta = ref<{
  primary_account_mode: 'off' | 'auto' | 'manual'
  manual_primary_account_id?: number | null
  active_primary_account_id?: number | null
  active_primary_source?: string
  active_primary_reason?: string
  active_primary_switched_at?: string | null
  primary_failover_cooldown_seconds?: number
  primary_allow_manual_auto_replace?: boolean
  recent_schedule_trace?: UsageScheduleTrace | null
  recent_schedule_request_id?: string
  recent_schedule_created_at?: string | null
} | null>(null)

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

const activePrimaryItem = computed(() => sortedItems.value.find(item => item.account_id === primaryMeta.value?.active_primary_account_id) || null)
const manualPrimaryItem = computed(() => sortedItems.value.find(item => item.account_id === primaryMeta.value?.manual_primary_account_id) || null)
const activePrimarySource = computed(() => primaryMeta.value?.active_primary_source || '')
const isFailoverCandidate = computed(() => activePrimarySource.value === 'failover_candidate')
const isFailoverPromoted = computed(() => activePrimarySource.value === 'failover_promoted')
const primaryModeLabel = computed(() => {
  switch (primaryMeta.value?.primary_account_mode) {
    case 'auto': return '自动'
    case 'manual': return '手动'
    default: return '关闭'
  }
})
const activePrimaryBadgeLabel = computed(() => {
  if (isFailoverCandidate.value) {
    return t('admin.groups.qualityPanel.primaryLabels.failoverCandidate') || '接管候选主账号'
  }
  return isFailoverPromoted.value
    ? t('admin.groups.qualityPanel.primaryLabels.failoverPrimary')
    : t('admin.groups.qualityPanel.primaryLabels.activePrimary')
})
const activePrimaryMeta = computed(() => {
  if (!primaryMeta.value) return ''

  const source = primaryMeta.value.active_primary_source
  const reason = primaryMeta.value.active_primary_reason
  const switchedAt = primaryMeta.value.active_primary_switched_at

  let label = ''
  if (source === 'manual_override' && reason === 'manual_set') {
    label = t('admin.groups.qualityPanel.primaryMeta.manualSet')
  } else if (source === 'failover_candidate' && reason === 'same_account_retry_exhausted') {
    const candidateLabel = t('admin.groups.qualityPanel.primaryMeta.failoverCandidate')
    label = candidateLabel === 'admin.groups.qualityPanel.primaryMeta.failoverCandidate'
      ? '故障转移接管中，仍在冷却确认期'
      : candidateLabel
  } else if (source === 'failover_promoted' && (reason === 'same_account_retry_exhausted' || reason === 'failover_stabilized')) {
    label = t('admin.groups.qualityPanel.primaryMeta.failoverPromoted')
  } else {
    label = [source, reason].filter(Boolean).join(' / ')
  }

  if (!label) return ''
  if (!switchedAt) return label

  const switchedText = new Date(switchedAt).toLocaleString()
  return t('admin.groups.qualityPanel.primaryMeta.withTime', {
    status: label,
    time: switchedText
  })
})
const failoverTakeoverHint = computed(() => {
  if ((!isFailoverPromoted.value && !isFailoverCandidate.value) || !activePrimaryItem.value) return ''
  if (isFailoverCandidate.value) {
    const message = t('admin.groups.qualityPanel.failoverCandidateHint', {
      account: activePrimaryItem.value.account_name
    })
    return message === 'admin.groups.qualityPanel.failoverCandidateHint'
      ? `${activePrimaryItem.value.account_name} 当前正在临时接管请求。如果在冷却窗口内持续稳定，它会自动转为正式主账号。`
      : message
  }
  return t('admin.groups.qualityPanel.failoverTakeoverHint', {
    account: activePrimaryItem.value.account_name
  })
})
const accountLabel = (accountID?: number | null, candidates: UsageScheduleCandidateScore[] = []) => {
  if (!accountID) return ''
  const itemName = sortedItems.value.find(item => item.account_id === accountID)?.account_name
  const candidateName = candidates.find(candidate => candidate.account_id === accountID)?.account_name
  return itemName || candidateName || `#${accountID}`
}

const translatePrimaryBypassReason = (reason?: string | null) => {
  if (!reason) return ''
  const key = `admin.groups.qualityPanel.primaryBypassReasons.${reason}`
  const label = t(key)
  return label === key ? reason : label
}

const recentScheduleLines = computed<string[]>(() => {
  const trace = primaryMeta.value?.recent_schedule_trace
  if (!trace) return []

  const candidates = trace.candidates || []
  const selectedLabel = accountLabel(trace.selected_account_id, candidates) || '-'
  const createdAt = primaryMeta.value?.recent_schedule_created_at
    ? new Date(primaryMeta.value.recent_schedule_created_at).toLocaleString()
    : ''
  const primaryCandidateID = trace.primary_candidate_id ||
    primaryMeta.value?.active_primary_account_id ||
    primaryMeta.value?.manual_primary_account_id ||
    null
  const primaryLabel = trace.primary_candidate_name || accountLabel(primaryCandidateID, candidates)
  const shouldShowPrimaryAttempt = Boolean(primaryLabel && (
    trace.primary_hit ||
    trace.primary_bypass_reason ||
    trace.layer === 'load_balance' ||
    candidates.length > 0
  ))
  const lines: string[] = []

  if (trace.layer === 'previous_response_id') {
    lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.previousResponse', {
      account: selectedLabel
    }))
  }

  if (shouldShowPrimaryAttempt) {
    lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.primaryTry', {
      account: primaryLabel
    }))
  }

  if (trace.primary_hit) {
    lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.primarySelected', {
      account: selectedLabel
    }))
  } else if (trace.primary_bypass_reason && primaryLabel) {
    lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.primarySkipped', {
      account: primaryLabel,
      reason: translatePrimaryBypassReason(trace.primary_bypass_reason)
    }))
  } else if (shouldShowPrimaryAttempt && trace.layer === 'load_balance') {
    lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.primarySkipped', {
      account: primaryLabel,
      reason: translatePrimaryBypassReason('primary_bypass_not_recorded')
    }))
  }

  if (trace.sticky_hit && trace.layer !== 'previous_response_id') {
    lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.stickySelected', {
      account: selectedLabel
    }))
  }

  if (trace.layer === 'load_balance' || candidates.length > 0) {
    const candidateLabels = candidates.map(candidate => {
      const label = accountLabel(candidate.account_id, candidates) || '-'
      if (!candidate.selected) return label
      return t('admin.groups.qualityPanel.recentScheduleSteps.candidateSelected', {
        account: label
      })
    })
    if (candidateLabels.length > 0) {
      lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.loadBalanceCandidates', {
        candidates: candidateLabels.join(' -> ')
      }))
    } else if (trace.layer === 'load_balance') {
      lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.loadBalanceNoCandidates'))
    }
  }

  if (trace.selected_account_id) {
    lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.finalSelected', {
      account: selectedLabel
    }))
  }

  if (createdAt) {
    lines.push(t('admin.groups.qualityPanel.recentScheduleSteps.time', {
      time: createdAt
    }))
  }

  if (lines.length === 0) {
    lines.push(trace.layer || t('admin.groups.qualityPanel.recentScheduleReasons.unknown'))
  }

  return lines
})

const loadQuality = async () => {
  if (!props.group) return
  loading.value = true
  try {
    const data = await adminAPI.groups.getAccountQuality(props.group.id)
    items.value = data.items
    primaryMeta.value = {
      primary_account_mode: data.primary_account_mode,
      manual_primary_account_id: data.manual_primary_account_id,
      active_primary_account_id: data.active_primary_account_id,
      active_primary_source: data.active_primary_source,
      active_primary_reason: data.active_primary_reason,
      active_primary_switched_at: data.active_primary_switched_at,
      primary_failover_cooldown_seconds: data.primary_failover_cooldown_seconds,
      primary_allow_manual_auto_replace: data.primary_allow_manual_auto_replace,
      recent_schedule_trace: data.recent_schedule_trace,
      recent_schedule_request_id: data.recent_schedule_request_id,
      recent_schedule_created_at: data.recent_schedule_created_at
    }
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

const setPrimary = async (item: GroupAccountQualityItem) => {
  if (!props.group) return
  actionLoading.value = true
  try {
    await adminAPI.groups.updatePrimaryAccount(props.group.id, {
      primary_account_mode: 'manual',
      manual_primary_account_id: item.account_id,
      primary_failover_cooldown_seconds: primaryMeta.value?.primary_failover_cooldown_seconds ?? 30,
      primary_allow_manual_auto_replace: primaryMeta.value?.primary_allow_manual_auto_replace ?? false
    })
    await loadQuality()
  } catch (error) {
    console.error('Failed to update primary account:', error)
    appStore.showError('设置主账号失败')
  } finally {
    actionLoading.value = false
  }
}

const switchToAuto = async () => {
  if (!props.group) return
  actionLoading.value = true
  try {
    await adminAPI.groups.updatePrimaryAccount(props.group.id, {
      primary_account_mode: 'auto',
      manual_primary_account_id: null,
      primary_failover_cooldown_seconds: primaryMeta.value?.primary_failover_cooldown_seconds ?? 30,
      primary_allow_manual_auto_replace: primaryMeta.value?.primary_allow_manual_auto_replace ?? false
    })
    await loadQuality()
  } catch (error) {
    console.error('Failed to switch primary account mode:', error)
    appStore.showError('切回自动失败')
  } finally {
    actionLoading.value = false
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

const rowActivePrimaryLabel = (item: GroupAccountQualityItem) => {
  if (item.account_id === primaryMeta.value?.active_primary_account_id && isFailoverCandidate.value) {
    const label = t('admin.groups.qualityPanel.primaryLabels.failoverCandidate')
    return label === 'admin.groups.qualityPanel.primaryLabels.failoverCandidate' ? '接管候选主账号' : label
  }
  if (item.account_id === primaryMeta.value?.active_primary_account_id && isFailoverPromoted.value) {
    return t('admin.groups.qualityPanel.primaryLabels.failoverPrimary')
  }
  return t('admin.groups.qualityPanel.primaryLabels.activePrimary')
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
  if (item.is_pool_mode) {
    chips.push({
      label: `同号重试 ${item.pool_mode_retry_count} 次`,
      className: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300'
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
