<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'

const props = defineProps<{ content: string; streaming?: boolean }>()
marked.setOptions({ breaks: true, gfm: true })
const rendered = computed(() => DOMPurify.sanitize(marked.parse(props.content) as string))
</script>

<template>
  <div class="markdown-body" :class="{ streaming }" v-html="rendered" />
</template>
