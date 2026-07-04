import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountPoolRateAnalysisView from '../AccountPoolRateAnalysisView.vue'

const { getPoolRateAnalysis, showError } = vi.hoisted(() => ({
  getPoolRateAnalysis: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin/accounts', () => ({
  accountsAPI: {
    getPoolRateAnalysis
  },
  default: {
    getPoolRateAnalysis
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const responseWithRows = {
  items: [
    {
      group_id: 1,
      group_name: 'Pool A',
      platform: 'openai',
      configured_rate_multiplier: 1,
      requests: 2,
      valid_requests: 2,
      uncovered_requests: 0,
      input_tokens: 100,
      output_tokens: 200,
      valid_input_tokens: 100,
      valid_output_tokens: 200,
      theoretical_cost: 10,
      account_cost: 1.8,
      inferred_multiplier: 0.18,
      coverage_rate: 1
    },
    {
      group_id: 2,
      group_name: 'Pool B',
      platform: 'anthropic',
      configured_rate_multiplier: 1,
      requests: 1,
      valid_requests: 0,
      uncovered_requests: 1,
      input_tokens: 10,
      output_tokens: 20,
      valid_input_tokens: 0,
      valid_output_tokens: 0,
      theoretical_cost: 0,
      account_cost: 0,
      inferred_multiplier: null,
      coverage_rate: 0
    }
  ],
  summary: {
    groups: 2,
    valid_groups: 1,
    requests: 3,
    valid_requests: 2,
    uncovered_requests: 1,
    theoretical_cost: 10,
    account_cost: 1.8
  }
}

const mountView = () => mount(AccountPoolRateAnalysisView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      DateRangePicker: {
        props: ['startDate', 'endDate'],
        emits: ['update:startDate', 'update:endDate', 'change'],
        template: `
          <button
            data-test="date-change"
            @click="$emit('update:startDate', '2026-07-01'); $emit('update:endDate', '2026-07-03'); $emit('change', { startDate: '2026-07-01', endDate: '2026-07-03', preset: null })"
          >
            date
          </button>
        `
      },
      DataTable: {
        props: ['columns', 'data', 'loading', 'serverSideSort', 'defaultSortKey', 'defaultSortOrder', 'rowKey'],
        emits: ['sort'],
        template: `
          <div data-test="rate-table">
            <button data-test="sort-account-cost" @click="$emit('sort', 'account_cost', 'desc')">sort</button>
            <div v-for="row in data" :key="row.group_id" data-test="rate-row">
              <slot name="cell-inferred_multiplier" :row="row" :value="row.inferred_multiplier" />
            </div>
            <slot v-if="!data || data.length === 0" name="empty" />
          </div>
        `
      },
      Icon: true,
      RouterLink: {
        props: ['to'],
        template: '<a><slot /></a>'
      }
    }
  }
})

describe('AccountPoolRateAnalysisView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-04T10:00:00+08:00'))
    getPoolRateAnalysis.mockReset()
    showError.mockReset()
    getPoolRateAnalysis.mockResolvedValue(responseWithRows)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('loads the last 7 local days by default', async () => {
    mountView()
    await flushPromises()

    const today = new Date()
    const start = new Date(today)
    start.setDate(today.getDate() - 6)

    expect(getPoolRateAnalysis).toHaveBeenCalledWith(expect.objectContaining({
      start_date: formatLocalDate(start),
      end_date: formatLocalDate(today),
      sort_by: 'inferred_multiplier',
      sort_order: 'asc'
    }), expect.objectContaining({
      signal: expect.any(AbortSignal)
    }))
  })

  it('marks rows without a valid multiplier as missing valid samples', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.rateAnalysis.noValidSamples')
  })

  it('reloads with the selected server-side sort', async () => {
    const wrapper = mountView()
    await flushPromises()
    getPoolRateAnalysis.mockClear()

    await wrapper.find('[data-test="sort-account-cost"]').trigger('click')
    await flushPromises()

    expect(getPoolRateAnalysis).toHaveBeenCalledWith(expect.objectContaining({
      sort_by: 'account_cost',
      sort_order: 'desc'
    }), expect.any(Object))
  })

  it('reloads after date range changes', async () => {
    const wrapper = mountView()
    await flushPromises()
    getPoolRateAnalysis.mockClear()

    await wrapper.find('[data-test="date-change"]').trigger('click')
    await flushPromises()

    expect(getPoolRateAnalysis).toHaveBeenCalledWith(expect.objectContaining({
      start_date: '2026-07-01',
      end_date: '2026-07-03'
    }), expect.any(Object))
  })
})
