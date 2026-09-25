import { request } from './client';
import type { LineInspection } from '../types/domain';

export async function listLineInspections(page = 1, pageSize = 20, search = '', planCode = '', status = '') {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (search) params.set('search', search);
  if (planCode) params.set('planCode', planCode);
  if (status) params.set('status', status);
  return request<LineInspection[]>(`/inspections?${params.toString()}`);
}
export async function listBlockingInspections() {
  return request<LineInspection[]>('/inspections?page=1&pageSize=100&status=blocking');
}
export async function createLineInspection(input: Partial<LineInspection>) {
  return request<LineInspection>('/inspections', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionLineInspection(id: number, status: string, expectedVersion: number, reason: string) {
  return request<LineInspection>(`/inspections/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
