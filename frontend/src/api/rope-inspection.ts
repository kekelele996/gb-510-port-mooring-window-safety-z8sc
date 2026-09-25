
import { request } from './client';
import type { RopeInspection } from '../types/domain';

export async function listRopeInspections(page = 1, pageSize = 20, search = '', status = '', planCode = '') {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (search) params.set('search', search);
  if (status) params.set('status', status);
  if (planCode) params.set('planCode', planCode);
  return request<RopeInspection[]>(`/rope-inspections?${params.toString()}`);
}
export async function listCurrentRopeInspections(planCode = '') {
  const suffix = planCode ? `?planCode=${encodeURIComponent(planCode)}` : '';
  return request<RopeInspection[]>(`/rope-inspections/current${suffix}`);
}
export async function createRopeInspection(input: Partial<RopeInspection>) {
  return request<RopeInspection>('/rope-inspections', { method: 'POST', body: JSON.stringify(input) });
}
