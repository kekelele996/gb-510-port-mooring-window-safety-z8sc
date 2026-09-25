<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import type { DomainRecord, LineInspection } from '../types/domain';
import {
  ALL_DEFECT_LEVEL, ALL_INSPECTION_CONCLUSION,
  DEFECT_LEVEL_LABELS, INSPECTION_CONCLUSION_LABELS, INSPECTION_STATE_LABELS,
} from '../types/status';
import { formatDate } from '../utils/format';
import { useAuth } from '../hooks/useAuth';
import { useLineInspectionStore } from '../stores/line-inspection';
import { listMooringPlan } from '../api/mooring-plan';
import StatusBadge from '../components/common/StatusBadge.vue';
import MetricCard from '../components/common/MetricCard.vue';
import ConfirmDialog from '../components/common/ConfirmDialog.vue';

const store = useLineInspectionStore();
const { session } = useAuth();
const roleRank: Record<string, number> = { viewer: 1, operator: 2, reviewer: 3, admin: 4 };
const canWrite = computed(() => (roleRank[session.value?.role || ''] || 0) >= roleRank.operator);

const search = ref('');
const plans = ref<DomainRecord[]>([]);
const showCreate = ref(false);
const createError = ref('');
const pending = ref<{ item: LineInspection; status: string; reason: string; label: string } | null>(null);

const form = reactive({
  planCode: '', linePosition: '', inspectedAt: new Date(), inspector: '',
  defectLevel: 'none', conclusion: 'pass', description: '',
});

const blockingCount = computed(() => store.items.filter((item) => item.status === 'blocking').length);
const positionCount = computed(() => new Set(store.items.map((item) => `${item.planCode}@${item.linePosition}`)).size);

// 每个缆位的当前结论：列表按检查时间倒序，首次出现即最新记录。
const currentConclusions = computed(() => {
  const latest = new Map<string, LineInspection>();
  for (const item of store.items) {
    const key = `${item.planCode}@${item.linePosition}`;
    if (!latest.has(key)) latest.set(key, item);
  }
  return [...latest.values()];
});

const wouldBlock = computed(() =>
  form.defectLevel === 'broken_strand' || form.defectLevel === 'wear_over_limit' || form.conclusion === 'replace_pending');

onMounted(async () => {
  void store.load();
  try {
    plans.value = (await listMooringPlan(1, 100)).data;
  } catch {
    plans.value = [];
  }
});

function defectLabel(level: string): string {
  return DEFECT_LEVEL_LABELS[level as keyof typeof DEFECT_LEVEL_LABELS] || level;
}
function conclusionLabel(conclusion: string): string {
  return INSPECTION_CONCLUSION_LABELS[conclusion as keyof INSPECTION_CONCLUSION_LABELS] || conclusion;
}
function stateLabel(status: string): string {
  return INSPECTION_STATE_LABELS[status as keyof typeof INSPECTION_STATE_LABELS] || status;
}

function openCreate() {
  if (!canWrite.value) return;
  form.planCode = plans.value[0]?.code || '';
  form.linePosition = '';
  form.inspectedAt = new Date();
  form.inspector = session.value?.username || '';
  form.defectLevel = 'none';
  form.conclusion = 'pass';
  form.description = '';
  createError.value = '';
  showCreate.value = true;
}

async function submitCreate() {
  if (!form.planCode || !form.linePosition.trim() || !form.inspector.trim() || !form.inspectedAt) {
    createError.value = '系泊方案、缆位、检查时间和检查人均为必填';
    return;
  }
  const now = Date.now();
  try {
    await store.create({
      code: `LI-${String(now).slice(-6)}`,
      name: `${form.planCode} ${form.linePosition.trim()}检查`,
      description: form.description.trim() || '带缆前缆绳状态核对',
      planCode: form.planCode,
      linePosition: form.linePosition.trim(),
      inspectedAt: new Date(form.inspectedAt).toISOString(),
      inspector: form.inspector.trim(),
      defectLevel: form.defectLevel,
      conclusion: form.conclusion,
    });
    showCreate.value = false;
  } catch (error) {
    createError.value = error instanceof Error ? error.message : String(error);
  }
}

function askResolve(item: LineInspection) {
  pending.value = { item, status: 'resolved', reason: '现场换绳完成，复查通过', label: '换绳复查通过' };
}
function askBlock(item: LineInspection) {
  pending.value = { item, status: 'blocking', reason: '复核发现缆绳缺陷，升级为阻断', label: '标记阻断' };
}
async function confirmTransition() {
  if (!pending.value) return;
  await store.transition(pending.value.item, pending.value.status, pending.value.reason);
  pending.value = null;
}
</script>

<template>
  <main class="workspace">
    <header class="page-header">
      <div>
        <p class="eyebrow">缆绳安全</p>
        <h1>缆绳检查</h1>
        <p>带缆前核对缆绳状态；断股、超限磨损或待换绳时阻断关联安全许可放行。</p>
      </div>
      <el-button v-if="canWrite" type="primary" @click="openCreate">新增检查记录</el-button>
    </header>
    <section class="metrics">
      <MetricCard label="检查记录" :value="store.meta.total" detail="当前筛选范围"/>
      <MetricCard label="阻断中" :value="blockingCount" detail="关联许可禁止放行"/>
      <MetricCard label="涉及缆位" :value="positionCount" detail="方案 + 缆位"/>
    </section>
    <section class="clearance-panel" aria-label="缆位当前结论">
      <header>
        <div><span class="eyebrow">CURRENT CONCLUSION</span><strong>缆位当前结论</strong></div>
        <small>每个缆位取最新一次检查；历史记录保留在下方列表</small>
      </header>
      <div v-if="currentConclusions.length" class="clearance-grid">
        <article v-for="item in currentConclusions" :key="item.id">
          <div class="clearance-title"><strong>{{ item.planCode }} · {{ item.linePosition }}</strong><StatusBadge :status="item.status"/></div>
          <p>{{ defectLabel(item.defectLevel) }} · {{ conclusionLabel(item.conclusion) }}</p>
          <dl>
            <dt>检查时间</dt><dd>{{ formatDate(item.inspectedAt) }}</dd>
            <dt>检查人</dt><dd>{{ item.inspector }}</dd>
            <dt v-if="item.status === 'blocking'">阻断原因</dt><dd v-if="item.status === 'blocking'">{{ item.blockedReason }}</dd>
            <dt v-if="item.resolvedBy">复查人</dt><dd v-if="item.resolvedBy">{{ item.resolvedBy }}</dd>
          </dl>
        </article>
      </div>
      <div v-else class="empty">暂无缆绳检查记录</div>
    </section>
    <section class="toolbar">
      <el-input v-model="search" placeholder="搜索检查编码、名称或缆位" clearable/>
      <el-button type="primary" @click="store.load(search)">查询</el-button>
      <el-button @click="search = ''; store.load()">重置</el-button>
    </section>
    <el-alert v-if="store.error" :title="store.error" type="error" show-icon/>
    <section class="table-shell">
      <el-table v-loading="store.loading" :data="store.items">
        <el-table-column prop="code" label="编码" width="110"/>
        <el-table-column label="系泊方案 / 缆位" min-width="180">
          <template #default="{ row }"><strong>{{ row.planCode }}</strong><small>{{ row.linePosition }}</small></template>
        </el-table-column>
        <el-table-column label="检查时间" width="170">
          <template #default="{ row }">{{ formatDate(row.inspectedAt) }}</template>
        </el-table-column>
        <el-table-column prop="inspector" label="检查人" width="100"/>
        <el-table-column label="缺陷等级" width="110">
          <template #default="{ row }">{{ defectLabel(row.defectLevel) }}</template>
        </el-table-column>
        <el-table-column label="处置结论" width="130">
          <template #default="{ row }">{{ conclusionLabel(row.conclusion) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }"><StatusBadge :status="row.status"/><small>{{ stateLabel(row.status) }}</small></template>
        </el-table-column>
        <el-table-column label="阻断原因" min-width="150">
          <template #default="{ row }"><span :class="{ muted: !row.blockedReason }">{{ row.blockedReason || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button v-if="canWrite && row.status === 'blocking'" link type="primary" @click="askResolve(row)">换绳复查通过</el-button>
            <el-button v-else-if="canWrite && row.status === 'open'" link type="warning" @click="askBlock(row)">标记阻断</el-button>
            <span v-else-if="!canWrite" class="muted">只读权限</span>
            <span v-else class="muted">流程结束</span>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="showCreate" title="新增缆绳检查" width="520px">
      <el-form label-width="90px">
        <el-form-item label="系泊方案" required>
          <el-select v-model="form.planCode" placeholder="选择系泊方案" style="width: 100%">
            <el-option v-for="plan in plans" :key="plan.id" :label="`${plan.code} · ${plan.name}`" :value="plan.code"/>
          </el-select>
        </el-form-item>
        <el-form-item label="缆位" required>
          <el-input v-model="form.linePosition" placeholder="如：船首左舷1#缆"/>
        </el-form-item>
        <el-form-item label="检查时间" required>
          <el-date-picker v-model="form.inspectedAt" type="datetime" style="width: 100%"/>
        </el-form-item>
        <el-form-item label="检查人" required>
          <el-input v-model="form.inspector"/>
        </el-form-item>
        <el-form-item label="缺陷等级" required>
          <el-select v-model="form.defectLevel" style="width: 100%">
            <el-option v-for="level in ALL_DEFECT_LEVEL" :key="level" :label="defectLabel(level)" :value="level"/>
          </el-select>
        </el-form-item>
        <el-form-item label="处置结论" required>
          <el-select v-model="form.conclusion" style="width: 100%">
            <el-option v-for="item in ALL_INSPECTION_CONCLUSION" :key="item" :label="conclusionLabel(item)" :value="item"/>
          </el-select>
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="现场检查情况、换绳安排等"/>
        </el-form-item>
      </el-form>
      <el-alert v-if="wouldBlock" title="该缺陷或结论将阻断相关系泊方案的安全许可放行，待复核许可会被退回" type="error" show-icon :closable="false"/>
      <el-alert v-if="createError" :title="createError" type="error" show-icon style="margin-top: 10px"/>
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" :loading="store.loading" @click="submitCreate">保存检查</el-button>
      </template>
    </el-dialog>

    <ConfirmDialog :model-value="Boolean(pending)" :title="pending?.label || '确认操作'" @update:model-value="pending = null" @confirm="confirmTransition">
      <template v-if="pending">
        <p>{{ pending.item.code }} · {{ pending.item.planCode }} {{ pending.item.linePosition }}</p>
        <p v-if="pending.status === 'resolved'">复查通过后，相关系泊方案的安全许可可重新提交放行。</p>
        <p v-else>标记阻断后，相关系泊方案的待复核许可将被退回，且不能放行。</p>
        <strong>{{ stateLabel(pending.item.status) }} → {{ stateLabel(pending.status) }}</strong>
      </template>
    </ConfirmDialog>
  </main>
</template>
