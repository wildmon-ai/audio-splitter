<script setup lang="ts">
defineProps<{
  label: string
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

async function pickFolder() {
  const selected = await window.electronAPI.selectFolder()
  if (selected) {
    emit('update:modelValue', selected)
  }
}
</script>

<template>
  <div class="flex flex-col gap-1.5">
    <label class="text-xs font-semibold uppercase tracking-wide text-gray-500">{{ label }}</label>
    <div class="flex gap-2">
      <input
        :value="modelValue"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        type="text"
        class="flex-1 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm font-mono text-gray-800 placeholder-gray-400 focus:border-blue-500 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500 transition-colors"
        placeholder="Select a folder…"
      />
      <button
        @click="pickFolder"
        class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100 hover:border-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1 transition-colors"
      >
        Browse
      </button>
    </div>
  </div>
</template>
