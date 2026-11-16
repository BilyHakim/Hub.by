<script setup>
import { computed } from 'vue'
import { renderAIMarkdown } from '../utils/aiMarkdown'

const props = defineProps({ content: { type: String, required: true } })
const html = computed(() => renderAIMarkdown(props.content))
</script>

<template>
  <div class="ai-answer ai-markdown" v-html="html" />
</template>
