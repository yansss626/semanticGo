<script lang="ts" setup>
import { computed } from 'vue'
import { SvgIcon } from '@/components/common'
import { useAppStore} from '@/store'
import { NTooltip } from 'naive-ui'




const appStore = useAppStore()


const collapsed = computed(() => appStore.siderCollapsed)


function handleUpdateCollapsed() {
  appStore.setSiderCollapsed(!collapsed.value)
}




</script>

<template>
  <header>
  </header>
</template>
