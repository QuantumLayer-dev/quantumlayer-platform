'use client'

import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { 
  Shield, 
  AlertTriangle, 
  CheckCircle, 
  TrendingUp,
  Server,
  Activity,
  Package,
  Clock,
  BarChart3,
  Globe,
  Lock,
  Zap
} from 'lucide-react'
import MetricCard from '@/components/MetricCard'
import DriftHeatmap from '@/components/DriftHeatmap'
import PatchTimeline from '@/components/PatchTimeline'
import ComplianceGauge from '@/components/ComplianceGauge'
import RiskMatrix from '@/components/RiskMatrix'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useDashboardStore } from '@/lib/store'

export default function ExecutiveDashboard() {
  const { metrics, updateMetrics } = useDashboardStore()
  const { isConnected } = useWebSocket()

  useEffect(() => {
    fetchDashboardMetrics()
    const interval = setInterval(fetchDashboardMetrics, 30000)
    return () => clearInterval(interval)
  }, [])

  const fetchDashboardMetrics = async () => {
    try {
      const [aiResponse, registryResponse] = await Promise.all([
        fetch('/api/ai/risk-dashboard'),
        fetch('/api/qinfra/images')
      ])
      
      if (aiResponse.ok && registryResponse.ok) {
        const aiData = await aiResponse.json()
        const registryData = await registryResponse.json()
        
        updateMetrics({
          ...aiData,
          totalImages: registryData.images?.length || 0,
          activeNodes: aiData.active_nodes || 847,
        })
      }
    } catch (error) {
      console.error('Failed to fetch metrics:', error)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900">
      <div className="p-8">
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-8"
        >
          <div className="flex items-center justify-between mb-4">
            <div>
              <h1 className="text-4xl font-bold text-white mb-2">
                QInfra Command Center
              </h1>
              <p className="text-gray-400">
                Enterprise Infrastructure Resilience Platform
              </p>
            </div>
            <div className="flex items-center space-x-4">
              <div className={`flex items-center space-x-2 px-4 py-2 rounded-lg ${
                isConnected ? 'bg-green-900/30 text-green-400' : 'bg-red-900/30 text-red-400'
              }`}>
                <Activity className="w-4 h-4" />
                <span className="text-sm">{isConnected ? 'Live' : 'Offline'}</span>
              </div>
              <div className="text-gray-400 text-sm">
                {new Date().toLocaleString()}
              </div>
            </div>
          </div>
        </motion.div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <MetricCard
            title="Infrastructure Health"
            value={`${metrics.healthScore || 98}%`}
            trend={+2.3}
            icon={<Shield className="w-6 h-6" />}
            color="green"
          />
          <MetricCard
            title="Critical Vulnerabilities"
            value={metrics.criticalVulnerabilities || 3}
            trend={-25}
            icon={<AlertTriangle className="w-6 h-6" />}
            color="red"
          />
          <MetricCard
            title="Compliance Score"
            value={`${metrics.complianceScore || 94}%`}
            trend={+5.1}
            icon={<CheckCircle className="w-6 h-6" />}
            color="blue"
          />
          <MetricCard
            title="Patch Success Rate"
            value={`${metrics.patchSuccessRate || 99.2}%`}
            trend={+1.2}
            icon={<TrendingUp className="w-6 h-6" />}
            color="purple"
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.1 }}
            className="lg:col-span-2"
          >
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-xl p-6 border border-gray-700">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xl font-semibold text-white">Infrastructure Drift Analysis</h2>
                <div className="flex items-center space-x-2">
                  <Server className="w-5 h-5 text-gray-400" />
                  <span className="text-sm text-gray-400">
                    {metrics.activeNodes || 847} nodes monitored
                  </span>
                </div>
              </div>
              <DriftHeatmap />
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.2 }}
          >
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-xl p-6 border border-gray-700">
              <h2 className="text-xl font-semibold text-white mb-4">Compliance Status</h2>
              <ComplianceGauge />
            </div>
          </motion.div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.3 }}
          >
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-xl p-6 border border-gray-700">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xl font-semibold text-white">Patch Deployment Timeline</h2>
                <Package className="w-5 h-5 text-gray-400" />
              </div>
              <PatchTimeline />
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.4 }}
          >
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-xl p-6 border border-gray-700">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xl font-semibold text-white">Risk Assessment Matrix</h2>
                <AlertTriangle className="w-5 h-5 text-gray-400" />
              </div>
              <RiskMatrix />
            </div>
          </motion.div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.5 }}
            className="bg-gradient-to-br from-blue-900/30 to-blue-800/30 backdrop-blur-sm rounded-xl p-6 border border-blue-700/50"
          >
            <div className="flex items-center justify-between mb-4">
              <Globe className="w-8 h-8 text-blue-400" />
              <span className="text-2xl font-bold text-white">
                {metrics.totalImages || 42}
              </span>
            </div>
            <h3 className="text-white font-medium">Golden Images</h3>
            <p className="text-gray-400 text-sm mt-1">CIS/STIG Hardened</p>
            <div className="mt-4 pt-4 border-t border-blue-700/50">
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">AWS</span>
                <span className="text-blue-400">12</span>
              </div>
              <div className="flex justify-between text-sm mt-1">
                <span className="text-gray-400">Azure</span>
                <span className="text-blue-400">15</span>
              </div>
              <div className="flex justify-between text-sm mt-1">
                <span className="text-gray-400">GCP</span>
                <span className="text-blue-400">15</span>
              </div>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6 }}
            className="bg-gradient-to-br from-green-900/30 to-green-800/30 backdrop-blur-sm rounded-xl p-6 border border-green-700/50"
          >
            <div className="flex items-center justify-between mb-4">
              <Zap className="w-8 h-8 text-green-400" />
              <span className="text-2xl font-bold text-white">
                {metrics.mttr || '4.2'}m
              </span>
            </div>
            <h3 className="text-white font-medium">Mean Time to Recovery</h3>
            <p className="text-gray-400 text-sm mt-1">RTO Achievement</p>
            <div className="mt-4 pt-4 border-t border-green-700/50">
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">This Month</span>
                <span className="text-green-400">99.8%</span>
              </div>
              <div className="flex justify-between text-sm mt-1">
                <span className="text-gray-400">DR Drills</span>
                <span className="text-green-400">12/12</span>
              </div>
              <div className="flex justify-between text-sm mt-1">
                <span className="text-gray-400">Failovers</span>
                <span className="text-green-400">100%</span>
              </div>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.7 }}
            className="bg-gradient-to-br from-purple-900/30 to-purple-800/30 backdrop-blur-sm rounded-xl p-6 border border-purple-700/50"
          >
            <div className="flex items-center justify-between mb-4">
              <Lock className="w-8 h-8 text-purple-400" />
              <span className="text-2xl font-bold text-white">
                {metrics.securityScore || 96}%
              </span>
            </div>
            <h3 className="text-white font-medium">Security Posture</h3>
            <p className="text-gray-400 text-sm mt-1">Zero Trust Compliance</p>
            <div className="mt-4 pt-4 border-t border-purple-700/50">
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">CVE Patched</span>
                <span className="text-purple-400">247/250</span>
              </div>
              <div className="flex justify-between text-sm mt-1">
                <span className="text-gray-400">RBAC</span>
                <span className="text-purple-400">Active</span>
              </div>
              <div className="flex justify-between text-sm mt-1">
                <span className="text-gray-400">Encryption</span>
                <span className="text-purple-400">100%</span>
              </div>
            </div>
          </motion.div>
        </div>
      </div>
    </div>
  )
}