import type { MonitorStatusCounts, MonitorStatusSample, UsageSelectionReason } from '@/types'

type Translate = (key: string, params?: Record<string, unknown>) => string

export interface SelectionReasonDetail {
  label: string
  value: string
}

export interface SelectionCandidateComparisonStep {
  title: string
  lines: string[]
  selected: boolean
  routed: boolean
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

const formatAccountLabel = (name: unknown, id: unknown): string => {
  const accountName = isPresent(name) ? String(name) : '-'
  return `${accountName}${isPresent(id) ? ` #${id}` : ''}`
}

const isSameAccount = (
  candidate: NonNullable<UsageSelectionReason['candidates']>[number],
  accountID: unknown,
  accountName: unknown,
): boolean => {
  if (isPresent(candidate.account_id) && isPresent(accountID)) {
    return Number(candidate.account_id) === Number(accountID)
  }
  if (isPresent(candidate.account_name) && isPresent(accountName)) {
    return String(candidate.account_name) === String(accountName)
  }
  return false
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
  const monitor = formatMatchedMonitor(quality)
  const counts = quality.counts_3 ? `3=${formatMonitorCounts(quality.counts_3, t)}` : ''
  return [monitor, counts].filter(Boolean).join(' ')
}

const formatMatchedMonitor = (quality: NonNullable<UsageSelectionReason['monitor_quality']>): string => {
  const name = isPresent(quality.monitor_name) ? String(quality.monitor_name) : ''
  const id = isPresent(quality.monitor_id) ? `#${quality.monitor_id}` : ''
  return [name, id].filter(Boolean).join(' ')
}

const formatMonitorStatusLabel = (status: string, t: Translate): string => {
  switch (status) {
    case 'operational':
      return t('usage.monitorGreen')
    case 'degraded':
      return t('usage.monitorOrange')
    case 'failed':
    case 'error':
      return t('usage.monitorRed')
    default:
      return t('usage.monitorUnknown')
  }
}

const formatMonitorStatusSample = (sample: MonitorStatusSample, index: number, t: Translate): string => {
  const status = String(sample.status || 'unknown')
  const checkedAt = sample.checked_at ? `@${sample.checked_at}` : ''
  return `${index + 1}.${formatMonitorStatusLabel(status, t)}${checkedAt}`
}

export const formatMonitorRecentSamples = (
  samples: MonitorStatusSample[] | null | undefined,
  t: Translate,
): string => {
  if (!samples || samples.length === 0) return '-'
  return samples.map((sample, index) => formatMonitorStatusSample(sample, index, t)).join(' ')
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

export const formatSelectionCompareSummary = (
  reason: UsageSelectionReason | null | undefined,
  t: Translate,
  fallback = '-',
): string => {
  if (!reason) return fallback
  const candidateCount = Array.isArray(reason.candidates) ? reason.candidates.length : undefined
  const selected = formatAccountLabel(reason.account_name, reason.account_id)
  const routed = formatAccountLabel(reason.final_account_name ?? reason.account_name, reason.final_account_id ?? reason.account_id)
  const parts = [
    `${t('usage.selectionCandidatesCompared')}:${candidateCount ?? reason.candidate_count ?? '-'}`,
    `${t('usage.selectionAccount')}:${selected}`,
    `${t('usage.selectionFinalAccount')}:${routed}`,
  ]
  return parts.join(' | ')
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
    details.push({
      label: t('usage.selectionAccount'),
      value: formatAccountLabel(reason.account_name, reason.account_id),
    })
  }
  if (
    isPresent(reason.final_account_name) ||
    isPresent(reason.final_account_id) ||
    isPresent(reason.account_name) ||
    isPresent(reason.account_id)
  ) {
    details.push({
      label: t('usage.selectionFinalAccount'),
      value: formatAccountLabel(reason.final_account_name ?? reason.account_name, reason.final_account_id ?? reason.account_id),
    })
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
    const matchedMonitor = formatMatchedMonitor(quality)
    if (matchedMonitor) {
      details.push({ label: t('usage.selectionMonitor'), value: matchedMonitor })
    }
    if (isPresent(quality.primary_model)) {
      details.push({ label: t('usage.selectionMonitorPrimaryModel'), value: String(quality.primary_model) })
    }
    if (isPresent(quality.snapshot_at)) {
      details.push({ label: t('usage.selectionSnapshotAt'), value: String(quality.snapshot_at) })
    }
    details.push({ label: t('usage.selectionMonitor3'), value: formatMonitorCounts(quality.counts_3, t) })
    details.push({ label: t('usage.selectionMonitor5'), value: formatMonitorCounts(quality.counts_5, t) })
    details.push({ label: t('usage.selectionMonitor7'), value: formatMonitorCounts(quality.counts_7, t) })
    details.push({
      label: t('usage.selectionMonitorRecent7'),
      value: formatMonitorRecentSamples(quality.recent_7, t),
    })
    details.push({
      label: t('usage.selectionMonitorOrder'),
      value: quality.latest_first === true ? t('usage.selectionLatestFirst') : (quality.order || '-'),
    })
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

export const formatSelectionCandidateLines = (
  reason: UsageSelectionReason | null | undefined,
  t: Translate,
): string[] => {
  if (!reason?.candidates?.length) return []
  return reason.candidates.map((candidate, index) => {
    const monitorName = isPresent(candidate.monitor_name)
      ? ` ${String(candidate.monitor_name)}`
      : (isPresent(candidate.monitor_id) ? ` #${candidate.monitor_id}` : '')
    const rank = candidate.rank ?? index + 1
    return [
      `${rank}. ${formatAccountLabel(candidate.account_name, candidate.account_id)}`,
      monitorName.trim(),
      `3=${formatMonitorCounts(candidate.monitor_3, t)}`,
      `5=${formatMonitorCounts(candidate.monitor_5, t)}`,
      `7=${formatMonitorCounts(candidate.monitor_7, t)}`,
    ].filter(Boolean).join(' | ')
  })
}

export const formatSelectionCandidateComparisonSteps = (
  reason: UsageSelectionReason | null | undefined,
  t: Translate,
): SelectionCandidateComparisonStep[] => {
  if (!reason?.candidates?.length) return []
  return reason.candidates.map((candidate, index) => {
    const rank = candidate.rank ?? index + 1
    const monitorLabel = isPresent(candidate.monitor_name)
      ? String(candidate.monitor_name)
      : (isPresent(candidate.monitor_id) ? `#${candidate.monitor_id}` : '-')
    const selected = isSameAccount(candidate, reason.account_id, reason.account_name)
    const routed = isSameAccount(candidate, reason.final_account_id ?? reason.account_id, reason.final_account_name ?? reason.account_name)
    return {
      title: `${rank}. ${formatAccountLabel(candidate.account_name, candidate.account_id)}`,
      selected,
      routed,
      lines: [
        `${t('usage.selectionMonitor')}: ${monitorLabel}`,
        `${t('usage.selectionMonitor3')}: ${formatMonitorCounts(candidate.monitor_3, t)}`,
        `${t('usage.selectionMonitor5')}: ${formatMonitorCounts(candidate.monitor_5, t)}`,
        `${t('usage.selectionMonitor7')}: ${formatMonitorCounts(candidate.monitor_7, t)}`,
      ],
    }
  })
}

export const formatSelectionCompareExport = (
  reason: UsageSelectionReason | null | undefined,
  t: Translate,
): string => {
  if (!reason) return ''
  const candidates = formatSelectionCandidateLines(reason, t).join(' || ')
  return [
    `${t('usage.selectionAccount')}: ${formatAccountLabel(reason.account_name, reason.account_id)}`,
    `${t('usage.selectionFinalAccount')}: ${formatAccountLabel(reason.final_account_name ?? reason.account_name, reason.final_account_id ?? reason.account_id)}`,
    `${t('usage.selectionCandidatesCompared')}: ${candidates || '-'}`,
  ].join('; ')
}

export const formatSelectionReasonExport = (
  reason: UsageSelectionReason | null | undefined,
  t: Translate,
): string => {
  if (!reason) return ''
  return [
    ...formatSelectionReasonDetails(reason, t).map(({ label, value }) => `${label}: ${value}`),
    `${t('usage.selectionCompare')}: ${formatSelectionCompareExport(reason, t)}`,
  ].join('; ')
}
