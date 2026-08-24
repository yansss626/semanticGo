<script setup lang='ts'>
import { computed } from 'vue'
import { NLayout, NLayoutContent } from 'naive-ui'
import { useRouter } from 'vue-router'
import Sider from './sider/index.vue'
// import Permission from './Permission.vue'
import { useBasicLayout } from '@/hooks/useBasicLayout'
import { useAppStore, useChatStore } from '@/store'
import { SvgIcon } from '@/components/common'

const appStore = useAppStore()

const collapsed = computed(
  () => appStore.siderCollapsed,
)


function handleUpdateCollapsed() {
  appStore.setSiderCollapsed(!collapsed.value)
}
const router = useRouter()

const chatStore = useChatStore()
// const authStore = useAuthStore()

router.replace({ name: 'Chat', params: { uuid: chatStore.active } })

const { isMobile } = useBasicLayout()


/*
onBeforeMount(() => {
  const access_token = getCookieValue('sso_0voice_access_token')
  if (!access_token)
    window.location.href = 'https://user.0voice.com?sys=ai'
})

*/
// const needPermission = computed(() => !!authStore.session?.auth && !authStore.token)
// const needPermission = !localStorage.access_token
const getMobileClass = computed(() => {
  if (isMobile.value)
    return ['rounded-none', 'shadow-none']
  return [
    'border',
    'rounded-xl',
    'shadow-sm',
    'border-gray-200',
    'dark:border-neutral-800',
  ]
})

const getContainerClass = computed(() => {
  return [
    'h-full',
    // { 'pl-[260px]': !isMobile.value && !collapsed.value },
    // { 'right-[0]': !isMobile.value && !collapsed.value },
  ]
})
</script>

<template>
  <div
  class="h-full bg-[#f5f7fa] dark:bg-[#16181d] transition-all"
  :class="[isMobile ? 'p-0' : 'p-0']"
  >
    <div class="h-full overflow-hidden" :class="getMobileClass">
      <NLayout class="relative h-full">
          <button
            class="
            fixed
            top-4
            left-4
            z-[100]
            flex
            items-center
            justify-center
            w-10
            h-10
            rounded-lg
            hover:bg-gray-100
            "
            @click="handleUpdateCollapsed"
          >

            <SvgIcon
              class="text-2xl"
              icon="ri:menu-line"
            />

          </button>
          <div class="absolute inset-y-0 left-0 z-50">
              <Sider />
          </div>
          <NLayoutContent class="h-full">
          <RouterView v-slot="{ Component, route }">
            <component :is="Component" :key="route.fullPath" />
          </RouterView>
        </NLayoutContent>
      </NLayout>
    </div>
    <!-- <Permission :visible="needPermission" /> -->
  </div>
</template>
