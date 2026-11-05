import { ref, watch, type Ref } from 'vue'

/**
 * A setting kept in localStorage. A stored value that cannot be parsed, or
 * that fails isValid (for example a 0 saved by an older version), falls back
 * to the default instead of breaking the app.
 */
export function usePersisted<T>(
  key: string,
  defaultValue: T,
  isValid: (value: unknown) => boolean = () => true,
): Ref<T> {
  const value = ref(read(key, defaultValue, isValid)) as Ref<T>
  watch(value, (v) => {
    try {
      localStorage.setItem(key, JSON.stringify(v))
    } catch {
      // Storage full or unavailable: the setting is just not remembered.
    }
  })
  return value
}

function read<T>(key: string, defaultValue: T, isValid: (value: unknown) => boolean): T {
  try {
    const stored = localStorage.getItem(key)
    if (stored === null) return defaultValue
    const parsed: unknown = JSON.parse(stored)
    return isValid(parsed) ? (parsed as T) : defaultValue
  } catch {
    return defaultValue
  }
}
