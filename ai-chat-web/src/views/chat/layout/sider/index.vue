<script setup lang='ts'>
import type { CSSProperties } from 'vue'
import { computed, watch } from 'vue'
import { NButton, NLayoutSider, NTooltip } from 'naive-ui'
import List from './List.vue'
import { useAppStore, useChatStore } from '@/store'
import { useBasicLayout } from '@/hooks/useBasicLayout'


const appStore = useAppStore()
const chatStore = useChatStore()

const { isMobile } = useBasicLayout()


const collapsed = computed(() => appStore.siderCollapsed)

function handleAdd() {
  chatStore.addHistory({ title: 'New Chat', uuid: Date.now(), isEdit: false })
  if (isMobile.value)
    appStore.setSiderCollapsed(true)
}

function handleUpdateCollapsed() {
  appStore.setSiderCollapsed(!collapsed.value)
}

const getMobileClass = computed<CSSProperties>(() => {
  if (isMobile.value) {
    return {
      position: 'fixed',
      zIndex: 50,
    }
  }
  return {}
})

const mobileSafeArea = computed(() => {
  if (isMobile.value) {
    return {
      paddingBottom: 'env(safe-area-inset-bottom)',
    }
  }
  return {}
})

watch(
  isMobile,
  (val) => {
    appStore.setSiderCollapsed(val)
  },
  {
    immediate: true,
    flush: 'post',
  },
)
</script>

<template>
  <NLayoutSider
    :collapsed="collapsed"
    :collapsed-width="72"
    :width="260"
    collapse-mode="transform"
    position="absolute"
  >
    <div class="flex flex-col h-full" :style="mobileSafeArea">
      <div class="relative flex items-center justify-center h-14 border-b">
        <NTooltip>
          <template #trigger>
            <button
              class="absolute left-4 flex items-center justify-center w-8 h-8 rounded-lg hover:bg-gray-100"
              @click="handleUpdateCollapsed"
            >
              <SvgIcon
                class="text-2xl"
                icon="ri:menu-line"
              />
            </button>
          </template>
          {{ collapsed ? '打开边栏' : '关闭边栏' }}
        </NTooltip>


      </div>
      <main class="flex flex-col flex-1 min-h-0">
        <div v-if="!collapsed" class="px-4 pt-3">
          <NButton dashed block @click="handleAdd">
            新建聊天
          </NButton>
        </div>
        <div v-if="!collapsed" class="flex-1 min-h-0 pb-4 overflow-hidden">
          <List />
        </div>
      </main>
    </div>
  </NLayoutSider>
  <template v-if="isMobile">
    <div v-show="!collapsed" class="fixed inset-0 z-40 bg-black/40" @click="handleUpdateCollapsed" />
  </template>
</template>
