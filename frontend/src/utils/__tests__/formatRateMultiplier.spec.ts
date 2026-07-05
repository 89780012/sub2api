import { describe, expect, it } from 'vitest'
import { formatRateMultiplier } from '../format'

describe('formatRateMultiplier', () => {
  it('formats multipliers with the requested fixed precision', () => {
    expect(formatRateMultiplier(0.07, { fractionDigits: 3 })).toBe('0.070x')
    expect(formatRateMultiplier(1, { fractionDigits: 3 })).toBe('1.000x')
  })

  it('keeps two decimals by default and supports nullish fallbacks', () => {
    expect(formatRateMultiplier(1.234)).toBe('1.23x')
    expect(formatRateMultiplier(null)).toBe('-')
    expect(formatRateMultiplier(Number.NaN, { fallback: 'n/a' })).toBe('n/a')
  })
})
