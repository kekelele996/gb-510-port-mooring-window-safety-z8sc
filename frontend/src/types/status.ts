import type { EntityConfig } from './domain';

export type CallState = 'planned' | 'approach' | 'moored' | 'departed';
export const ALL_CALL_STATE: readonly CallState[] = ['planned', 'approach', 'moored', 'departed'];
export type ClearanceState = 'pending' | 'cleared' | 'restricted' | 'expired';
export const ALL_CLEARANCE_STATE: readonly ClearanceState[] = ['pending', 'cleared', 'restricted', 'expired'];
export type InspectionState = 'open' | 'blocking' | 'resolved' | 'superseded';
export const ALL_INSPECTION_STATE: readonly InspectionState[] = ['open', 'blocking', 'resolved', 'superseded'];
export type DefectLevel = 'none' | 'minor_wear' | 'wear_over_limit' | 'broken_strand';
export const ALL_DEFECT_LEVEL: readonly DefectLevel[] = ['none', 'minor_wear', 'wear_over_limit', 'broken_strand'];
export type InspectionConclusion = 'pass' | 'monitor' | 'replace_pending' | 'recheck_pass';
export const ALL_INSPECTION_CONCLUSION: readonly InspectionConclusion[] = ['pass', 'monitor', 'replace_pending', 'recheck_pass'];

export const DEFECT_LEVEL_LABELS: Record<DefectLevel, string> = {
  none: '无缺陷', minor_wear: '轻微磨损', wear_over_limit: '超限磨损', broken_strand: '断股',
};
export const INSPECTION_CONCLUSION_LABELS: Record<InspectionConclusion, string> = {
  pass: '检查通过', monitor: '观察使用', replace_pending: '待换绳', recheck_pass: '换绳复查通过',
};
export const INSPECTION_STATE_LABELS: Record<InspectionState, string> = {
  open: '已登记', blocking: '阻断中', resolved: '复查通过', superseded: '已被取代',
};

export const ENTITY_CONFIGS: readonly EntityConfig[] = [
  { key: 'vesselCall', path: 'vessels', label: '船舶靠泊', statuses: ['planned', 'approach', 'moored', 'departed'] as const },
  { key: 'mooringPlan', path: 'plans', label: '系泊方案', statuses: ['draft', 'review', 'approved', 'superseded'] as const },
  { key: 'weatherWindow', path: 'weather-windows', label: '风浪窗口', statuses: ['forecast', 'safe', 'restricted', 'expired'] as const },
  { key: 'safetyClearance', path: 'clearance', label: '安全许可', statuses: ['pending', 'cleared', 'restricted', 'expired'] as const },
  { key: 'lineInspection', path: 'inspections', label: '缆绳检查', statuses: ['open', 'blocking', 'resolved', 'superseded'] as const }
];
