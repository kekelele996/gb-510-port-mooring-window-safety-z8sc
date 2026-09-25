<script setup lang="ts">
import { computed } from 'vue';
import type { DomainRecord, LineInspection } from '../../types/domain';
import { useAuth } from '../../hooks/useAuth';
import StatusBadge from './StatusBadge.vue';

const props = defineProps<{ records: DomainRecord[]; mode: 'window' | 'clearance'; blocks?: Record<string, LineInspection[]> }>();
const emit = defineEmits<{ confirm: [item: DomainRecord] }>();
const { session } = useAuth();
const roleRank: Record<string, number> = { viewer: 1, operator: 2, reviewer: 3, admin: 4 };
const canSubmit = computed(() => (roleRank[session.value?.role || ''] || 0) >= roleRank.operator);
const canReview = computed(() => (roleRank[session.value?.role || ''] || 0) >= roleRank.reviewer);
const displayedRecords = computed(() => {
  const records = [...props.records];
  if (props.mode === 'clearance') {
    records.sort((left, right) => Number(right.status === 'pending') - Number(left.status === 'pending'));
  }
  return records.slice(0, 3);
});

function blockingFor(item: DomainRecord): LineInspection[] {
  if (!item.planCode) return [];
  return props.blocks?.[item.planCode] || [];
}

function isBlocked(item: DomainRecord): boolean {
  return blockingFor(item).length > 0;
}

function blockSummary(item: DomainRecord): string {
  const first = blockingFor(item)[0];
  return first ? `${first.code} ${first.linePosition}：${first.blockedReason}` : '';
}

function canAct(item: DomainRecord): boolean {
  if (props.mode !== 'clearance' || item.status !== 'pending' || isBlocked(item)) return false;
  if (!item.submittedBy) return canSubmit.value;
  return canReview.value && item.submittedBy !== session.value?.username;
}

function actionLabel(item: DomainRecord): string {
  return item.submittedBy ? '复核并放行' : '提交安全确认';
}
</script>

<template>
  <section class="clearance-panel" aria-label="安全许可协同面板">
    <header>
      <div><span class="eyebrow">TWO-PERSON SAFETY</span><strong>{{ mode === 'window' ? '窗口许可依据' : '双人安全确认' }}</strong></div>
      <small>{{ mode === 'window' ? '窗口版本将随许可审计固化' : '提交人与复核人必须为不同账号' }}</small>
    </header>
    <div class="clearance-grid">
      <article v-for="item in displayedRecords" :key="item.id">
        <div class="clearance-title"><strong>{{ item.code }}</strong><StatusBadge :status="item.status"/></div>
        <p>{{ item.name }}</p>
        <dl>
          <dt>窗口版本</dt><dd>v{{ mode === 'window' ? item.version : (item.windowVersion || 1) }}</dd>
          <template v-if="mode === 'clearance'">
            <dt>系泊方案</dt><dd>{{ item.planCode || '未关联' }}</dd>
            <dt>首次提交</dt><dd>{{ item.submittedBy || '待提交' }}</dd>
            <dt>独立复核</dt><dd>{{ item.confirmedBy || '待复核' }}</dd>
          </template>
          <template v-else>
            <dt>风险等级</dt><dd>{{ item.riskLevel }}</dd>
            <dt>评估证据</dt><dd>{{ item.evidence || '待补充' }}</dd>
          </template>
        </dl>
        <el-alert v-if="mode === 'clearance' && item.status === 'pending' && isBlocked(item)" type="error" show-icon :closable="false"
          :title="`缆绳检查阻断：${blockSummary(item)}`" description="断股、超限磨损或待换绳解除前不能放行；待复核许可已退回"/>
        <el-alert v-else-if="mode === 'clearance' && item.status === 'pending' && !item.submittedBy && item.returnedReason" type="warning" show-icon :closable="false"
          :title="`许可已退回：${item.returnedReason}`" description="阻断检查已解除，可重新提交安全确认"/>
        <el-button v-if="mode === 'clearance' && canAct(item)" type="primary" @click="emit('confirm', item)">{{ actionLabel(item) }}</el-button>
        <small v-else-if="mode === 'clearance' && item.status === 'pending' && item.submittedBy && !isBlocked(item)">等待其他复核员确认</small>
      </article>
    </div>
  </section>
</template>
