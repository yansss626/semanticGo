<script lang="ts" setup>
import { computed, ref } from 'vue'
import MarkdownIt from 'markdown-it'
import mdKatex from '@traptitech/markdown-it-katex'
import mila from 'markdown-it-link-attributes'
import hljs from 'highlight.js'
import { useBasicLayout } from '@/hooks/useBasicLayout'
import { t } from '@/locales'

interface Props {
  inversion?: boolean
  error?: boolean
  text?: string
  loading?: boolean
  asRawText?: boolean
  answerSource?: 'public_model' | 'cache'
  tokenCount?: number
}

const props = defineProps<Props>()
const sourceLabel = computed(() => {
  if (props.answerSource === 'public_model')
    return `公有大模型 · 消耗 ${props.tokenCount ?? 0} tokens`

  if (props.answerSource === 'cache')
    return `缓存命中 · 节省 ${props.tokenCount ?? 0} tokens`

  return ''
})

const { isMobile } = useBasicLayout()

const textRef = ref<HTMLElement>()

const mdi = new MarkdownIt({
  linkify: true,
  highlight(code, language) {
    const validLang = !!(language && hljs.getLanguage(language))
    if (validLang) {
      const lang = language ?? ''
      return highlightBlock(hljs.highlight(code, { language: lang }).value, lang)
    }
    return highlightBlock(hljs.highlightAuto(code).value, '')
  },
})

mdi.use(mila, { attrs: { target: '_blank', rel: 'noopener' } })
mdi.use(mdKatex, { blockClass: 'katexmath-block rounded-md p-[10px]', errorColor: ' #cc0000' })

const wrapClass = computed(() => {
  return [
    'text-wrap',
    'min-w-[20px]',
    'rounded-xl',
    isMobile.value ? 'p-3' : 'px-4 py-3',
    props.inversion ? 'bg-[#DCFCE7] border border-[#BBF7D0]' : 'bg-white border border-[#E2E8F0] shadow-sm',
    props.inversion ? 'dark:bg-[#a1dc95]' : 'dark:bg-[#1e1e20]',
    props.inversion ? 'message-request' : 'message-reply',
    { 'text-red-500': props.error },
  ]
})

const text = computed(() => {
  const value = props.text ?? ''
  if (!props.asRawText)
    return mdi.render(value)
  return value
})

function highlightBlock(str: string, lang?: string) {
  return `<pre class="code-block-wrapper"><div class="code-block-header"><span class="code-block-header__lang">${lang}</span><span class="code-block-header__copy">${t('chat.copyCode')}</span></div><code class="hljs code-block-body ${lang}">${str}</code></pre>`
}

defineExpose({ textRef })
</script>

<template>
  <div class="text-[#334155]" :class="wrapClass">
    <template v-if="loading">
      <span class="dark:text-white w-[4px] h-[20px] block animate-blink" />
    </template>
    <template v-else>
      <div ref="textRef" class="leading-7 break-words text-[15px]">
        <div v-if="!inversion">         
          <div v-if="!asRawText" class="markdown-body" v-html="text" />
          <div v-else class="whitespace-pre-wrap" v-text="text" />
        </div>
        <div v-else class="whitespace-pre-wrap" v-text="text" />
      </div>
      <div
      v-if="!inversion && sourceLabel"
      class="mt-1.5 text-[12px] leading-5 text-[#94A3B8] dark:text-[#8B8B93]"
      >
      {{ sourceLabel }}
      </div>
    </template>
  </div>
</template>

<style lang="less">
@import url(./style.less);
</style>
