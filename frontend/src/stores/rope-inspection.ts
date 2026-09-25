import { defineStore } from 'pinia';
import { createRopeInspection, listCurrentRopeInspections, listRopeInspections } from '../api/rope-inspection';
import type { PageMeta, RopeInspection } from '../types/domain';

// Rope inspections need plan/status filters and a current-conclusion view, so
// this store is explicit instead of reusing the generic entity factory.
export const useRopeInspectionStore = defineStore('ropeInspection', {
  state: () => ({
    items: [] as RopeInspection[],
    current: [] as RopeInspection[],
    meta: { page: 1, pageSize: 20, total: 0 } as PageMeta,
    loading: false,
    error: '',
  }),
  actions: {
    async load(search = '', status = '', planCode = '') {
      this.loading = true;
      this.error = '';
      try {
        const result = await listRopeInspections(1, 20, search, status, planCode);
        this.items = result.data;
        this.meta = result.meta || { page: 1, pageSize: 20, total: result.data.length };
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
      } finally {
        this.loading = false;
      }
    },
    async loadCurrent(planCode = '') {
      try {
        const result = await listCurrentRopeInspections(planCode);
        this.current = result.data;
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
      }
    },
    async create(input: Partial<RopeInspection>): Promise<boolean> {
      this.loading = true;
      this.error = '';
      try {
        await createRopeInspection(input);
        await Promise.all([this.load(), this.loadCurrent()]);
        return true;
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
        return false;
      } finally {
        this.loading = false;
      }
    },
  },
});
