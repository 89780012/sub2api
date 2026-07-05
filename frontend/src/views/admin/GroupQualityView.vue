<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="inline-flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                <Icon name="brain" size="md" />
              </span>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.groups.qualityPanel.title') }}
                </h1>
                <p class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.qualityPanel.pageDescription') }}
                </p>
              </div>
            </div>
          </div>

          <div class="flex w-full flex-col gap-2 sm:flex-row xl:w-[560px]">
            <Select
              v-model="selectedGroupId"
              :options="groupOptions"
              :placeholder="t('admin.groups.qualityPanel.selectGroup')"
              :search-placeholder="t('admin.groups.searchGroups')"
              :disabled="groupsLoading"
              class="min-w-0 flex-1"
            />
            <button
              type="button"
              class="btn btn-secondary shrink-0"
              :disabled="groupsLoading"
              :title="t('common.refresh')"
              @click="loadGroups"
            >
              <Icon name="refresh" size="md" :class="groupsLoading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </div>

      <div v-if="groupsLoading" class="rounded-lg border border-gray-200 bg-white px-4 py-12 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400">
        {{ t('admin.groups.qualityPanel.loadingGroups') }}
      </div>

      <GroupQualityPanel
        v-else-if="selectedGroup"
        :show="true"
        :group="selectedGroup"
        embedded
      />

      <div v-else class="rounded-lg border border-dashed border-gray-300 bg-white px-4 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-400">
        {{ t('admin.groups.qualityPanel.noGroupSelected') }}
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, GroupPlatform, SelectOption } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupQualityPanel from '@/components/admin/group/GroupQualityPanel.vue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()

const groups = ref<AdminGroup[]>([])
const groupsLoading = ref(false)
const selectedGroupId = ref<number | null>(null)

const platformLabel = (platform: GroupPlatform) => t(`admin.groups.platforms.${platform}`)

const groupOptions = computed<SelectOption[]>(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: `${platformLabel(group.platform)} · ${group.name}${group.status === 'inactive' ? ` (${t('common.inactive')})` : ''}`
  }))
)

const selectedGroup = computed(() =>
  groups.value.find((group) => group.id === selectedGroupId.value) || null
)

const routeGroupId = computed(() => {
  const raw = route.query.group_id
  const value = Array.isArray(raw) ? raw[0] : raw
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
})

const syncSelectionFromRoute = () => {
  const nextId = routeGroupId.value
  if (nextId && groups.value.some((group) => group.id === nextId)) {
    selectedGroupId.value = nextId
    return
  }
  if (!selectedGroupId.value && groups.value.length > 0) {
    selectedGroupId.value = groups.value[0].id
  }
}

const loadGroups = async () => {
  groupsLoading.value = true
  try {
    groups.value = await adminAPI.groups.getAllIncludingInactive()
    syncSelectionFromRoute()
  } catch (error) {
    console.error('Failed to load groups for quality page:', error)
    appStore.showError(t('admin.groups.failedToLoad'))
  } finally {
    groupsLoading.value = false
  }
}

watch(routeGroupId, () => {
  syncSelectionFromRoute()
})

watch(selectedGroupId, (groupId) => {
  const current = routeGroupId.value
  if (groupId && groupId !== current) {
    router.replace({
      query: {
        ...route.query,
        group_id: String(groupId)
      }
    })
  }
})

onMounted(() => {
  loadGroups()
})
</script>
