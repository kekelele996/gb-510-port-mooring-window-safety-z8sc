<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import MetricCard from '../components/common/MetricCard.vue';
import StatusBadge from '../components/common/StatusBadge.vue';
import { useAuth } from '../hooks/useAuth';
import { useRopeInspectionStore } from '../stores/rope-inspection';
import { useMooringPlanStore } from '../stores/mooring-plan';
import {
  ALL_ROPE_CONCLUSION, ALL_ROPE_DEFECT_LEVEL,
  ROPE_CONCLUSION_LABELS, ROPE_DEFECT_LABELS, ROPE_STATE_LABELS,
  isBlockingRopeInspection,
} from '../types/status';
import type { RopeConclusion, RopeDefectLevel } from '../types/status';
import { formatDate } from '../utils/format';

const store = useRopeInspectionStore();
const planStore = useMooringPlanStore();
const { session } = useAuth();
const search = ref('');
const statusFilter = ref('');
const showCreate = ref(false);
const formError = ref('');
const roleRank: Record<string, number> = { viewer: 1, operator: 2, reviewer: 3, admin: 4 };
const canWrite = computed(() => (roleRank[session.value?.role || ''] || 0) >= roleRank.operator);
const openCount = computed(() => store.current.filter((item) => item.status === 'open').length);
const affectedPlans = computed(() => new Set(store.current.filter((item) => item.status === 'open').map((item) => item.planCode)).size);

const emptyForm = () => ({
  code: '', planCode: '', linePosition: '', inspectedAt: '', inspector: '',
  defectLevel: 'none' as RopeDefectLevel, conclusion: 'passed' as RopeConclusion, notes: '',
});
const form = reactive(emptyForm());
const willBlock = computed(() => isBlockingRopeInspection(form.defectLevel, form.conclusion));

onMounted(() => {
  void store.load();
  void store.loadCurrent();
  void planStore.load('plans');
});

function localDateTimeValue(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0');
  const offsetMinutes = -date.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? '+' : '-';
  const offsetAbs = Math.abs(offsetMinutes);
  const offset = `${sign}${pad(Math.floor(offsetAbs / 60))}:${pad(offsetAbs % 60)}`;
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}${offset}`;
}

function openCreate() {
  Object.assign(form, emptyForm());
  form.code = `RI-${String(Date.now()).slice(-6)}`;
  form.planCode = planStore.items[0]?.code || '';
  form.inspectedAt = localDateTimeValue(new Date());
  form.inspector = session.value?.username || '';
  formError.value = '';
  showCreate.value = true;
}

async function submitCreate() {
  formError.value = '';
  if (!form.code || !form.planCode || !form.linePosition || !form.inspectedAt || !form.inspector) {
    formError.value = '请完整填写编码、系泊方案、缆位、检查时间和检查人';
    return;
  }
  const inspectedAt = new Date(form.inspectedAt).toISOString();
  const ok = await store.create({
    code: form.code, planCode: form.planCode, linePosition: form.linePosition, inspectedAt,
    inspector: form.inspector, defectLevel: form.defectLevel, conclusion: form.conclusion, notes: form.notes,
  });
  if (ok) showCreate.value = false;
}

function defectTagType(defectLevel: string): 'success' | 'info' | 'warning' | 'danger' {
  if (defectLevel === 'none') return 'success';
  if (defectLevel === 'wear_minor') return 'warning';
  return 'danger';
}
</script>

<template>
  <main class="workspace">
    <header class="page-header">
      <div>
        <p class="eyebrow">业务工作台</p>
        <h1>缆绳检查</h1>
        <p>按系泊方案和缆位记录检查时间、检查人、缺陷等级与处置结论；断股、超限磨损或待换绳将阻断关联许可放行。</p>
      </div>
      <el-button v-if="canWrite" type="primary" @click="openCreate">新增检查记录</el-button>
    </header>
    <section class="metrics">
      <MetricCard label="检查记录" :value="store.meta.total" detail="当前筛选范围"/>
      <MetricCard label="阻断中缆位" :value="openCount" detail="断股 / 超限磨损 / 待换绳"/>
      <MetricCard label="受影响方案" :value="affectedPlans" detail="关联许可不能放行"/>
    </section>
    <section class="clearance-panel" aria-label="当前结论">
      <header>
        <div><span class="eyebrow">CURRENT CONCLUSION</span><strong>当前结论</strong></div>
        <small>每个缆位保留最新一次检查；阻断中的检查会退回待复核许可，换绳复查通过后自动闭环</small>
      </header>
      <div class="clearance-grid">
        <article v-for="item in store.current" :key="item.id" :class="{ 'article-blocked': item.status === 'open' }">
          <div class="clearance-title">
            <strong>{{ item.planCode }} · {{ item.linePosition }}</strong>
            <StatusBadge :status="item.status" :label="ROPE_STATE_LABELS[item.status]"/>
          </div>
          <dl>
            <dt>缺陷等级</dt><dd>{{ ROPE_DEFECT_LABELS[item.defectLevel] }}</dd>
            <dt>处置结论</dt><dd>{{ ROPE_CONCLUSION_LABELS[item.conclusion] }}</dd>
            <dt>检查人</dt><dd>{{ item.inspector }}</dd>
            <dt>检查时间</dt><dd>{{ formatDate(item.inspectedAt) }}</dd>
          </dl>
          <el-alert v-if="item.status === 'open'" :title="`阻断原因：${item.blockedReason}`" type="error" :closable="false"/>
        </article>
        <p v-if="!store.current.length" class="muted">暂无检查记录</p>
      </div>
    </section>
    <section class="toolbar">
      <el-input v-model="search" placeholder="搜索检查编码、缆位或检查人" clearable/>
      <el-select v-model="statusFilter" placeholder="全部状态" clearable style="max-width: 160px">
        <el-option label="阻断中" value="open"/>
        <el-option label="已闭环" value="closed"/>
      </el-select>
      <el-button type="primary" @click="store.load(search, statusFilter)">查询</el-button>
      <el-button @click="search = ''; statusFilter = ''; store.load()">重置</el-button>
    </section>
    <el-alert v-if="store.error" :title="store.error" type="error" show-icon/>
    <section class="table-shell">
      <el-table v-loading="store.loading" :data="store.items">
        <el-table-column prop="code" label="编码" width="110"/>
        <el-table-column prop="planCode" label="系泊方案" width="100"/>
        <el-table-column prop="linePosition" label="缆位" width="100"/>
        <el-table-column label="缺陷等级" width="110">
          <template #default="{ row }">
            <el-tag :type="defectTagType(row.defectLevel)" size="small">{{ ROPE_DEFECT_LABELS[row.defectLevel] }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="处置结论" width="100">
          <template #default="{ row }">{{ ROPE_CONCLUSION_LABELS[row.conclusion] }}</template>
        </el-table-column>
        <el-table-column prop="inspector" label="检查人" width="100"/>
        <el-table-column label="检查时间" width="170">
          <template #default="{ row }">{{ formatDate(row.inspectedAt) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><StatusBadge :status="row.status" :label="ROPE_STATE_LABELS[row.status]"/></template>
        </el-table-column>
        <el-table-column label="阻断原因" min-width="220">
          <template #default="{ row }">{{ row.blockedReason || '—' }}</template>
        </el-table-column>
      </el-table>
    </section>
    <el-dialog v-model="showCreate" title="新增缆绳检查" width="560px">
      <el-form label-width="90px">
        <el-form-item label="检查编码" required>
          <el-input v-model="form.code" placeholder="如：RI-0102"/>
        </el-form-item>
        <el-form-item label="系泊方案" required>
          <el-select v-model="form.planCode" style="width: 100%" placeholder="选择归属系泊方案">
            <el-option v-for="plan in planStore.items" :key="plan.id" :label="`${plan.code} · ${plan.name}`" :value="plan.code"/>
          </el-select>
        </el-form-item>
        <el-form-item label="缆位" required>
          <el-input v-model="form.linePosition" placeholder="如：艏缆-1、艉缆-2、横缆-1"/>
        </el-form-item>
        <el-form-item label="检查时间" required>
          <el-date-picker v-model="form.inspectedAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width: 100%"/>
        </el-form-item>
        <el-form-item label="检查人" required>
          <el-input v-model="form.inspector"/>
        </el-form-item>
        <el-form-item label="缺陷等级" required>
          <el-select v-model="form.defectLevel" style="width: 100%">
            <el-option v-for="level in ALL_ROPE_DEFECT_LEVEL" :key="level" :label="ROPE_DEFECT_LABELS[level]" :value="level"/>
          </el-select>
        </el-form-item>
        <el-form-item label="处置结论" required>
          <el-select v-model="form.conclusion" style="width: 100%">
            <el-option v-for="item in ALL_ROPE_CONCLUSION" :key="item" :label="ROPE_CONCLUSION_LABELS[item]" :value="item"/>
          </el-select>
        </el-form-item>
        <el-form-item label="检查备注">
          <el-input v-model="form.notes" type="textarea" :rows="2"/>
        </el-form-item>
      </el-form>
      <el-alert v-if="willBlock" title="该结论将阻断关联许可放行，已提交待复核的许可会被退回并记录本次检查" type="warning" show-icon :closable="false"/>
      <el-alert v-else title="复查通过后，同一缆位未闭环的检查将自动闭环，关联许可可重新提交" type="success" show-icon :closable="false"/>
      <el-alert v-if="formError" :title="formError" type="error" show-icon :closable="false"/>
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" :loading="store.loading" @click="submitCreate">保存检查记录</el-button>
      </template>
    </el-dialog>
  </main>
</template>
