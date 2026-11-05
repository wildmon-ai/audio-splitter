// Limits shared with the backend (backend/config.go).
export const MAX_TARGET_STEP = 100_000
export const MAX_RECORD_STEP = 1_000
export const MAX_THREADS = 1_024

function isWholeNumberIn(value: unknown, min: number, max: number): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= min && value <= max
}

export const isValidTargetStep = (value: unknown): value is number => isWholeNumberIn(value, 1, MAX_TARGET_STEP)

export const isValidRecordStep = (value: unknown): value is number => isWholeNumberIn(value, 1, MAX_RECORD_STEP)

export const isValidThreads = (value: unknown): value is number => isWholeNumberIn(value, 0, MAX_THREADS)

export const isString = (value: unknown): value is string => typeof value === 'string'
