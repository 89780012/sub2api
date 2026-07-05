import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ExternalBalanceCell from '../ExternalBalanceCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const labels: Record<string, string> = {
    'admin.accounts.balanceSync.accountBalance': 'Account Balance',
    'admin.accounts.balanceSync.keyBalance': 'Key Balance',
    'admin.accounts.balanceSync.packageBalance': 'Package Balance',
    'admin.accounts.balanceSync.dailyShort': 'D',
    'admin.accounts.balanceSync.weeklyShort': 'W',
    'admin.accounts.balanceSync.monthlyShort': 'M',
    'admin.accounts.balanceSync.matchStatus.matched': 'Matched',
    'admin.accounts.balanceSync.unlimited': 'Unlimited',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => labels[key] ?? key,
    }),
  }
})

describe('ExternalBalanceCell', () => {
  it('renders account, key, and package balances as independent values', () => {
    const wrapper = mount(ExternalBalanceCell, {
      props: {
        accountBalance: { remain: 12.345, unit: 'USD' },
        balance: { remain: 3, unit: 'USD' },
        subscriptionBalance: {
          daily: { remain: 5, unit: 'USD' },
          monthly: { remain: 30, unit: 'USD' },
        },
        matchStatus: 'matched',
      },
    })

    const text = wrapper.text()
    expect(text).toContain('Account Balance')
    expect(text).toContain('12.35 USD')
    expect(text).toContain('Key Balance')
    expect(text).toContain('3.00 USD')
    expect(text).toContain('Package Balance')
    expect(text).toContain('D 5.00 USD')
    expect(text).toContain('M 30.00 USD')
  })

  it('does not synthesize missing key or package balances from account balance', () => {
    const wrapper = mount(ExternalBalanceCell, {
      props: {
        accountBalance: { remain: 7, unit: 'USD' },
      },
    })

    const text = wrapper.text()
    expect(text).toContain('Account Balance')
    expect(text).toContain('7.00 USD')
    expect(text).not.toContain('Key Balance')
    expect(text).not.toContain('Package Balance')
  })
})
