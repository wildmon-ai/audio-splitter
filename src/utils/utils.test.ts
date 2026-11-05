import { describe, it, expect } from 'vitest'
import { errorMessage, formatDuration } from './format'
import { isValidRecordStep, isValidTargetStep, isValidThreads, MAX_TARGET_STEP, MAX_THREADS } from './validation'

describe('validation', () => {
  it('accepts whole target steps from 1 to the maximum', () => {
    for (const ok of [1, 10, MAX_TARGET_STEP]) expect(isValidTargetStep(ok)).toBe(true)
    for (const bad of [0, -1, 2.5, NaN, Infinity, MAX_TARGET_STEP + 1, '10', null, undefined]) {
      expect(isValidTargetStep(bad)).toBe(false)
    }
  })

  it('accepts 0 (automatic) and up for parallel processes', () => {
    for (const ok of [0, 1, MAX_THREADS]) expect(isValidThreads(ok)).toBe(true)
    for (const bad of [-1, 1.5, NaN, MAX_THREADS + 1]) expect(isValidThreads(bad)).toBe(false)
  })

  it('requires a record step of at least 1', () => {
    expect(isValidRecordStep(1)).toBe(true)
    expect(isValidRecordStep(0)).toBe(false)
  })
})

describe('formatDuration', () => {
  it('shows minutes only when there are some', () => {
    expect(formatDuration(0)).toBe('0s')
    expect(formatDuration(9)).toBe('9s')
    expect(formatDuration(75)).toBe('1m 15s')
  })
})

describe('errorMessage', () => {
  it('removes the wrapper ipcRenderer.invoke adds', () => {
    const err = new Error("Error invoking remote method 'start-processing': Error: A job is already running.")
    expect(errorMessage(err)).toBe('A job is already running.')
  })

  it('handles plain errors and non-errors', () => {
    expect(errorMessage(new Error('boom'))).toBe('boom')
    expect(errorMessage('text')).toBe('text')
  })
})
