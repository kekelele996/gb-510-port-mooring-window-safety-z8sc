import { defineStore } from 'pinia';
import { createLineInspection, listLineInspections, transitionLineInspection } from '../api/line-inspection';
import type { LineInspection, PageMeta } from '../types/domain';

export const useLineInspectionStore = defineStore('lineInspection', {
  state: () => ({
    items: [] as LineInspection[],
    meta: { page: 1, pageSize: 20, total: 0 } as PageMeta,
    loading: false,
    error: '',
  }),
  actions: {
    async load(search = '', planCode = '') {
      this.loading = true;
      this.error = '';
      try {
        const result = await listLineInspections(1, 20, search, planCode);
        this.items = result.data;
        this.meta = result.meta || { page: 1, pageSize: 20, total: result.data.length };
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
      } finally {
        this.loading = false;
      }
    },
    async create(input: Partial<LineInspection>) {
      this.loading = true;
      this.error = '';
      try {
        await createLineInspection(input);
        await this.load();
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async transition(item: LineInspection, status: string, reason: string) {
      this.loading = true;
      this.error = '';
      try {
        await transitionLineInspection(item.id, status, item.version, reason);
        await this.load();
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
      } finally {
        this.loading = false;
      }
    },
  },
});
