<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import { X } from '@lucide/vue'
defineProps({ title: String, saving: Boolean })
const emit = defineEmits(['close', 'submit'])
const root = ref(null)
const titleId = useId()
let previousFocus
function trap(event) {
  if (event.key !== 'Tab') return
  const controls = [...root.value.querySelectorAll('button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href]')]
  const first = controls[0], last = controls.at(-1)
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus() }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus() }
}
onMounted(async () => { previousFocus = document.activeElement; await nextTick(); root.value?.querySelector('input, select, textarea, button')?.focus() })
onBeforeUnmount(() => previousFocus?.focus())
</script>

<template>
  <Teleport to="body">
    <div class="modal-backdrop maintenance-dialog-backdrop" @click.self="!saving && emit('close')">
      <form ref="root" class="modal watch-modal maintenance-dialog" role="dialog" aria-modal="true" :aria-labelledby="titleId" @submit.prevent="emit('submit')" @keydown="trap" @keydown.esc.stop.prevent="!saving && emit('close')">
        <button type="button" class="modal-close" aria-label="Tutup dialog" :disabled="saving" @click="emit('close')"><X :size="18" /></button>
        <h2 :id="titleId">{{ title }}</h2>
        <slot />
      </form>
    </div>
  </Teleport>
</template>
