import { create } from 'zustand'

interface DashboardMetrics {
  healthScore: number
  criticalVulnerabilities: number
  complianceScore: number
  patchSuccessRate: number
  activeNodes: number
  totalImages: number
  mttr: string
  securityScore: number
  lastDriftAlert?: any
  lastPatchUpdate?: any
  driftNodes?: Array<{
    id: string
    name: string
    drift: number
    region: string
    type: string
  }>
}

interface DashboardStore {
  metrics: DashboardMetrics
  updateMetrics: (updates: Partial<DashboardMetrics>) => void
  resetMetrics: () => void
}

const defaultMetrics: DashboardMetrics = {
  healthScore: 98,
  criticalVulnerabilities: 3,
  complianceScore: 94,
  patchSuccessRate: 99.2,
  activeNodes: 847,
  totalImages: 42,
  mttr: '4.2',
  securityScore: 96,
}

export const useDashboardStore = create<DashboardStore>((set) => ({
  metrics: defaultMetrics,
  updateMetrics: (updates) =>
    set((state) => ({
      metrics: { ...state.metrics, ...updates },
    })),
  resetMetrics: () => set({ metrics: defaultMetrics }),
}))