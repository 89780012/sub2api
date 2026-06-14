import type { MonitorStatusCounts, UsageSelectionReason } from '@/types'

type Translate = (key: string, params?: Record<string, unknown>) => string

export interface SelectionReasonDetail {
  label: string
  value: string
}

const ruleLabelKeys: Record<string, string> = {
  monitor_quality_priority_load_lru: 'usage.selectionRules.monitorQualityPriorityLoadLru',
  monitor_quality_priority_lru: 'usage.selectionRules.monitorQualityPriorityLru',
  sticky: 'usage.selectionRules.sticky',
  openai_advanced_scheduler: 'usage.selectionRules.openaiAdvancedScheduler',
  legacy_order: 'usage.selectionRules.legacyOrder',
}

const tieBreakerLabelKeys: Record<string, string> = {
  monitor_quality_3_5_7: 'usage.selectionTieBreakers.monitorQuality357',
  priority: 'usage.priority',
  load: 'usage.selectionLoad',
  lru: 'usage.selectionTieBreakers.lru',
}

const isPresent = (value: unknown): value is string | number | boolean =>
  value !== null && value !== undefined && value !== ''

const formatMaybeNumber = (value: unknown): string => {
  if (typeof value === 'number') {
    return Number.isInteger(value) ? value.toString() : value.toFixed(4).replace(/0+$/, '').replace(/\.$/, '')
  }
  return isPresent(value) ? String(value) : '-'
}

export const formatSelectionRule = (rule: unknown, t: Translate): string => {
  const key = typeof rule === 'string' ? rule : ''
  return ruleLabelKeys[key] ? t(ruleLabelKeys[key]) : (key || '-')
}

export const formatMonitorCounts = (counts: MonitorStatusCounts | null | undefined, t: Translate): string => {
  if (!counts) return '-'
  return [
    `${t('usage.monitorGreen')}:${counts.green ?? 0}`,
    `${t('usage.monitorOrange')}:${counts.orange ?? 0}`,
    `${t('usage.monitorRed')}:${counts.red ?? 0}`,
    `${t('usage.monitorUnknown')}:${counts.unknown ?? 0}`,
  ].join(' ')
}

const formatMonitorSummary = (reason: UsageSelectionReason, t: Translate): string => {
  const quality = reason.monitor_quality
  if (!quality) return ''
  if (quality.known === false) return t('usage.selectionMonitorNotMatched')
  return quality.counts_3 ? `3=${formatMonitorCounts(quality.counts_3, t)}` : ''
}

export const formatSelectionReasonSummary = (
  reason: UsageSelectionReason | null | undefined,
  t: Translate,
  fallback = '-',
): string => {
  if (!reason) return fallback
  const parts = [formatSelectionRule(reason.rule, t)]
  const monitor = formatMonitorSummary(reason, t)
  if (monitor) parts.push(monitor)
  if (isPresent(reason.priority)) parts.push(`${t('usage.priority')} ${reason.priority}`)
  return parts.filter(Boolean).join(' | ')
}

export const formatSelectionTieBreakers = (
  tieBreakers: unknown,
  t: Translate,
): string => {
  if (!Array.isArray(tieBreakers) || tieBreakers.length === 0) return '-'
  return tieBreakers
    .map((item) => {
      const key = String(item)
      return tieBreakerLabelKeys[key] ? t(tieBreakerLabelKeys[key]) : key
    })
    .join(' > ')
}

export const formatSelectionReasonDetails = (
  reason: UsageSelectionReason | null | undefined,
  t: Translate,
): SelectionReasonDetail[] => {
  if (!reason) return []

  const details: SelectionReasonDetail[] = [
    { label: t('usage.selectionSummary'), value: formatSelectionReasonSummary(reason, t) },
    { label: t('usage.selectionRule'), value: formatSelectionRule(reason.rule, t) },
  ]

  if (isPresent(reason.account_name) || isPresent(reason.account_id)) {
    const name = isPresent(reason.account_name) ? String(reason.account_name) : '-'
    details.push({ label: t('usage.selectionAccount'), value: `${name}${isPresent(reason.account_id) ? ` #${reason.account_id}` : ''}` })
  }
  if (isPresent(reason.endpoint)) {
    details.push({ label: t('usage.selectionEndpoint'), value: String(reason.endpoint) })
  }
  if (isPresent(reason.priority)) {
    details.push({ label: t('usage.priority'), value: String(reason.priority) })
  }
  if (isPresent(reason.candidate_count)) {
    details.push({ label: t('usage.selectionCandidates'), value: String(reason.candidate_count) })
  }
  if (isPresent(reason.top_k)) {
    details.push({ label: t('usage.selectionTopK'), value: String(reason.top_k) })
  }
  if (isPresent(reason.load_skew)) {
    details.push({ label: t('usage.selectionLoadSkew'), value: formatMaybeNumber(reason.load_skew) })
  }

  const quality = reason.monitor_quality
  if (quality) {
    details.push({
      label: t('usage.selectionMonitorMatched'),
      value: quality.known === false ? t('usage.no') : t('usage.yes'),
    })
    details.push({ label: t('usage.selectionMonitor3'), value: formatMonitorCounts(quality.counts_3, t) })
    details.push({ label: t('usage.selectionMonitor5'), value: formatMonitorCounts(quality.counts_5, t) })
    details.push({ label: t('usage.selectionMonitor7'), value: formatMonitorCounts(quality.counts_7, t) })
  }

  if (reason.load) {
    details.push({
      label: t('usage.selectionLoad'),
      value: [
        `${t('usage.selectionLoadRate')}:${formatMaybeNumber(reason.load.load_rate)}`,
        `${t('usage.selectionConcurrency')}:${formatMaybeNumber(reason.load.current_concurrency)}`,
        `${t('usage.selectionWaiting')}:${formatMaybeNumber(reason.load.waiting_count)}`,
      ].join(' '),
    })
  }

  if (reason.wait_plan) {
    details.push({
      label: t('usage.selectionWaitPlan'),
      value: [
        `${t('usage.selectionTimeoutMs')}:${formatMaybeNumber(reason.wait_plan.timeout_ms)}`,
        `${t('usage.selectionMaxWaiting')}:${formatMaybeNumber(reason.wait_plan.max_waiting)}`,
      ].join(' '),
    })
  }

  details.push({
    label: t('usage.selectionTieBreakerLabel'),
    value: formatSelectionTieBreakers(reason.tie_breakers, t),
  })

  return details
}

export const formatSelectionReasonExport = (
  reason: UsageSelectionReason | null | undefined,
  t: Translate,
): string => {
  if (!reason) return ''
  return formatSelectionReasonDetails(reason, t)
    .map(({ label, value }) => `${label}: ${value}`)
    .join('; ')
}
