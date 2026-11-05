import { describe, it, expect, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { usePersisted } from './usePersisted'
import { isValidTargetStep } from '../utils/validation'

describe('usePersisted', () => {
  beforeEach(() => localStorage.clear())

  it('starts from the default when nothing is stored', () => {
    expect(usePersisted('step', 10).value).toBe(10)
  })

  it('restores and saves the value', async () => {
    localStorage.setItem('step', '5')
    const step = usePersisted('step', 10)
    expect(step.value).toBe(5)

    step.value = 7
    await nextTick()
    expect(localStorage.getItem('step')).toBe('7')
  })

  it('falls back to the default when the stored text is not JSON', () => {
    localStorage.setItem('step', '{oops')
    expect(usePersisted('step', 10).value).toBe(10)
  })

  it('falls back to the default when the stored value is invalid', () => {
    // A 0 saved after the field was emptied used to make every run loop forever.
    localStorage.setItem('step', '0')
    expect(usePersisted('step', 10, isValidTargetStep).value).toBe(10)
  })
})
