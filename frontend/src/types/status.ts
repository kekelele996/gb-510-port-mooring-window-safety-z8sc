import type { EntityConfig } from './domain';

export type CallState = 'planned' | 'approach' | 'moored' | 'departed';
export const ALL_CALL_STATE: readonly CallState[] = ['planned', 'approach', 'moored', 'departed'];
export type ClearanceState = 'pending' | 'cleared' | 'restricted' | 'expired';
export const ALL_CLEARANCE_STATE: readonly ClearanceState[] = ['pending', 'cleared', 'restricted', 'expired'];

export const ENTITY_CONFIGS: readonly EntityConfig[] = [
  { key: 'vesselCall', path: 'vessels', label: '船舶靠泊', statuses: ['planned', 'approach', 'moored', 'departed'] as const },
  { key: 'mooringPlan', path: 'plans', label: '系泊方案', statuses: ['draft', 'review', 'approved', 'superseded'] as const },
  { key: 'weatherWindow', path: 'weather-windows', label: '风浪窗口', statuses: ['forecast', 'safe', 'restricted', 'expired'] as const },
  { key: 'safetyClearance', path: 'clearance', label: '安全许可', statuses: ['pending', 'cleared', 'restricted', 'expired'] as const }
];

// Rope inspection 缆绳检查 enumerations mirror backend/internal/constants/status.go.
export type RopeInspectionState = 'open' | 'closed';
export const ALL_ROPE_INSPECTION_STATE: readonly RopeInspectionState[] = ['open', 'closed'];
export type RopeDefectLevel = 'none' | 'wear_minor' | 'wear_overlimit' | 'broken_strand';
export const ALL_ROPE_DEFECT_LEVEL: readonly RopeDefectLevel[] = ['none', 'wear_minor', 'wear_overlimit', 'broken_strand'];
export type RopeConclusion = 'passed' | 'monitor' | 'replace_pending';
export const ALL_ROPE_CONCLUSION: readonly RopeConclusion[] = ['passed', 'monitor', 'replace_pending'];

export const ROPE_STATE_LABELS: Record<RopeInspectionState, string> = { open: '阻断中', closed: '已闭环' };
export const ROPE_DEFECT_LABELS: Record<RopeDefectLevel, string> = { none: '无缺陷', wear_minor: '轻微磨损', wear_overlimit: '超限磨损', broken_strand: '断股' };
export const ROPE_CONCLUSION_LABELS: Record<RopeConclusion, string> = { passed: '通过', monitor: '观察使用', replace_pending: '待换绳' };

// 断股、超限磨损或待换绳时关联许可不能放行。
export function isBlockingRopeInspection(defectLevel: string, conclusion: string): boolean {
  return defectLevel === 'wear_overlimit' || defectLevel === 'broken_strand' || conclusion === 'replace_pending';
}
