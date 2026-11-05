<script setup lang="ts">
import { computed } from 'vue'
import { isValidTargetStep, isValidThreads, MAX_TARGET_STEP, MAX_THREADS } from '../utils/validation'

const props = defineProps<{ targetStep: number; numThreads: number }>()

const emit = defineEmits<{
  'update:targetStep': [value: number]
  'update:numThreads': [value: number]
}>()

const targetStepError = computed(() =>
  isValidTargetStep(props.targetStep) ? '' : `Enter a whole number from 1 to ${MAX_TARGET_STEP.toLocaleString('en-US')}.`,
)
const numThreadsError = computed(() =>
  isValidThreads(props.numThreads) ? '' : `Enter a whole number from 0 to ${MAX_THREADS.toLocaleString('en-US')}.`,
)

// An empty or half-typed field becomes NaN, which shows an error and disables
// Start. It used to become 0, and a step of 0 made the backend loop forever.
function numberFrom(event: Event): number {
  const input = event.target as HTMLInputElement
  return input.value.trim() === '' ? Number.NaN : input.valueAsNumber
}

function inputClass(invalid: boolean) {
  return invalid
    ? 'border-red-400 bg-red-50 focus:border-red-500 focus:ring-red-500'
    : 'border-gray-200 bg-gray-50 focus:border-blue-500 focus:ring-blue-500'
}
</script>

<template>
  <div class="flex flex-col gap-5">
    <div class="flex items-center justify-between">
      <div>
        <p class="text-xs font-semibold uppercase tracking-wide text-gray-500">Clip Length</p>
        <p class="mt-0.5 text-sm text-gray-700">Fixed at <span class="font-semibold">1 minute</span> per clip</p>
      </div>
      <span class="rounded-full bg-gray-100 px-3 py-1 text-xs font-semibold text-gray-500">Fixed</span>
    </div>

    <div class="border-t border-gray-100" />

    <div class="flex items-center justify-between gap-6">
      <div class="flex-1">
        <label for="target-step" class="text-xs font-semibold uppercase tracking-wide text-gray-500">Extract Every N Minutes</label>
        <p class="mt-0.5 text-xs text-gray-400">Time from the start of one clip to the start of the next</p>
        <p v-if="targetStepError" id="target-step-error" class="mt-1 text-xs font-medium text-red-600">{{ targetStepError }}</p>
      </div>
      <input
        id="target-step"
        :value="targetStep"
        @input="emit('update:targetStep', numberFrom($event))"
        type="number"
        min="1"
        :max="MAX_TARGET_STEP"
        step="1"
        inputmode="numeric"
        :aria-invalid="targetStepError ? 'true' : undefined"
        :aria-describedby="targetStepError ? 'target-step-error' : undefined"
        class="w-24 rounded-lg border px-3 py-2 text-center text-sm font-semibold text-gray-800 focus:bg-white focus:outline-none focus:ring-1 transition-colors"
        :class="inputClass(!!targetStepError)"
      />
    </div>

    <div class="border-t border-gray-100" />

    <div class="flex items-center justify-between gap-6">
      <div class="flex-1">
        <label for="num-threads" class="text-xs font-semibold uppercase tracking-wide text-gray-500">Parallel Processes</label>
        <p class="mt-0.5 text-xs text-gray-400">Concurrent ffmpeg processes · 0 = auto (all CPU cores)</p>
        <p v-if="numThreadsError" id="num-threads-error" class="mt-1 text-xs font-medium text-red-600">{{ numThreadsError }}</p>
      </div>
      <input
        id="num-threads"
        :value="numThreads"
        @input="emit('update:numThreads', numberFrom($event))"
        type="number"
        min="0"
        :max="MAX_THREADS"
        step="1"
        inputmode="numeric"
        :aria-invalid="numThreadsError ? 'true' : undefined"
        :aria-describedby="numThreadsError ? 'num-threads-error' : undefined"
        class="w-24 rounded-lg border px-3 py-2 text-center text-sm font-semibold text-gray-800 focus:bg-white focus:outline-none focus:ring-1 transition-colors"
        :class="inputClass(!!numThreadsError)"
      />
    </div>
  </div>
</template>
