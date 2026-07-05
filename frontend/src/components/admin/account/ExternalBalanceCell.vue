<template>
  <div class="min-w-[9rem] space-y-1">
    <div v-if="hasBalanceData" class="flex flex-col gap-1">
      <div v-if="accountBalance" class="flex flex-wrap items-baseline gap-1">
        <span class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.accounts.balanceSync.accountBalance') }}</span>
        <span class="font-mono text-sm font-medium text-gray-900 dark:text-white">
          {{ formatBalance(accountBalance) }}
        </span>
      </div>
      <div v-if="balance" class="flex flex-wrap items-baseline gap-1">
        <span class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.accounts.balanceSync.keyBalance') }}</span>
        <span class="font-mono text-sm font-medium text-gray-900 dark:text-white">
          {{ formatBalance(balance) }}
        </span>
      </div>
      <div v-if="subscriptionLines.length" class="flex flex-wrap gap-1">
        <span class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.accounts.balanceSync.packageBalance') }}</span>
        <span
          v-for="line in subscriptionLines"
          :key="line"
          class="rounded bg-slate-100 px-1.5 py-0.5 text-[11px] text-slate-600 dark:bg-slate-800 dark:text-slate-300"
        >
          {{ line }}
        </span>
      </div>
    </div>
    <div v-else class="text-sm text-gray-400 dark:text-dark-500">-</div>
    <div class="flex flex-wrap items-center gap-1 text-[11px] text-gray-500 dark:text-gray-400">
      <span v-if="matchStatus" :class="matchClass">{{ matchLabel }}</span>
      <span v-if="siteName">{{ siteName }}</span>
      <span v-if="keyName">/ {{ keyName }}</span>
      <span v-else-if="keyLast4">/ ****{{ keyLast4 }}</span>
    </div>
    <div v-if="fetchedAt" class="text-[11px] text-gray-400 dark:text-dark-500">
      {{ formatRelativeTime(fetchedAt) }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatRelativeTime } from '@/utils/format'
import type { ExternalBalanceAmount, ExternalSubscriptionBalance } from '@/types'

const props = defineProps<{
  balance?: ExternalBalanceAmount | null
  accountBalance?: ExternalBalanceAmount | null
  subscriptionBalance?: ExternalSubscriptionBalance | null
  matchStatus?: string
  siteName?: string
  keyName?: string
  keyLast4?: string
  fetchedAt?: string | null
}>()

const { t } = useI18n()

const hasBalanceData = computed(() => Boolean(props.accountBalance || props.balance || subscriptionLines.value.length))

const formatNumber = (value: number) => {
  if (Math.abs(value) >= 100) return value.toFixed(0)
  if (Math.abs(value) >= 1) return value.toFixed(2)
  return value.toFixed(4)
}

const formatBalance = (balance: ExternalBalanceAmount) => {
  if (balance.unlimited) return t('admin.accounts.balanceSync.unlimited')
  const value = typeof balance.remain === 'number'
    ? balance.remain
    : typeof balance.limit === 'number' && typeof balance.used === 'number'
      ? Math.max(balance.limit - balance.used, 0)
      : null
  if (value === null) return balance.unit || '-'
  return `${formatNumber(value)} ${balance.unit || ''}`.trim()
}

const formatWindow = (label: string, remain: number, unit: string) => `${label} ${formatNumber(remain)} ${unit}`.trim()

const subscriptionLines = computed(() => {
  const sub = props.subscriptionBalance
  if (!sub) return []
  const lines: string[] = []
  if (sub.daily) lines.push(formatWindow(t('admin.accounts.balanceSync.dailyShort'), sub.daily.remain, sub.daily.unit))
  if (sub.weekly) lines.push(formatWindow(t('admin.accounts.balanceSync.weeklyShort'), sub.weekly.remain, sub.weekly.unit))
  if (sub.monthly) lines.push(formatWindow(t('admin.accounts.balanceSync.monthlyShort'), sub.monthly.remain, sub.monthly.unit))
  return lines
})

const matchLabel = computed(() => {
  if (!props.matchStatus) return ''
  const key = `admin.accounts.balanceSync.matchStatus.${props.matchStatus}`
  const label = t(key)
  return label === key ? props.matchStatus : label
})

const matchClass = computed(() => {
  const base = 'rounded px-1.5 py-0.5 font-medium'
  if (props.matchStatus === 'matched' || props.matchStatus === 'manual') {
    return `${base} bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300`
  }
  if (props.matchStatus === 'ambiguous') {
    return `${base} bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300`
  }
  return `${base} bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300`
})
</script>
