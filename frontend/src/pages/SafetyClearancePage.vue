<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import EntityPage from '../components/EntityPage.vue';
import ClearancePanel from '../components/common/ClearancePanel.vue';
import type { DomainRecord, LineInspection } from '../types/domain';
import { ENTITY_CONFIGS } from '../types/status';
import { useSafetyClearanceStore } from '../stores/safety-clearance';
import { listBlockingInspections } from '../api/line-inspection';

const store = useSafetyClearanceStore();
const blocking = ref<LineInspection[]>([]);
const blocksByPlan = computed(() => {
  const grouped: Record<string, LineInspection[]> = {};
  for (const item of blocking.value) {
    (grouped[item.planCode] ||= []).push(item);
  }
  return grouped;
});

async function refreshBlocks() {
  try {
    blocking.value = (await listBlockingInspections()).data;
  } catch {
    blocking.value = [];
  }
}

onMounted(() => void refreshBlocks());

const confirmClearance = async (item: DomainRecord) => {
  await store.confirmClearance('clearance', item);
  await refreshBlocks();
};
</script>

<template>
  <EntityPage :config="ENTITY_CONFIGS[3]" :store="store" hide-transitions>
    <template #insight><ClearancePanel :records="store.items" mode="clearance" :blocks="blocksByPlan" @confirm="confirmClearance"/></template>
  </EntityPage>
</template>
