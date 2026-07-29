<script setup>
import { Inbox } from '@lucide/vue'
defineProps({ title: String, text: String })
</script>

<template>
  <div class="empty-state">
    <span><Inbox :size="26" /></span>
    <strong>{{ title }}</strong>
    <p>{{ text }}</p>
  </div>
</template>
