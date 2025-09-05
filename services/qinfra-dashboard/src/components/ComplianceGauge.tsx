import { motion } from 'framer-motion'
import { CheckCircle, AlertCircle, XCircle } from 'lucide-react'

interface ComplianceFramework {
  name: string
  score: number
  status: 'compliant' | 'warning' | 'non-compliant'
  controls: {
    total: number
    passed: number
    failed: number
  }
}

export default function ComplianceGauge() {
  const frameworks: ComplianceFramework[] = [
    {
      name: 'SOC2',
      score: 96,
      status: 'compliant',
      controls: { total: 150, passed: 144, failed: 6 }
    },
    {
      name: 'HIPAA',
      score: 92,
      status: 'compliant',
      controls: { total: 120, passed: 110, failed: 10 }
    },
    {
      name: 'PCI-DSS',
      score: 88,
      status: 'warning',
      controls: { total: 200, passed: 176, failed: 24 }
    },
    {
      name: 'ISO 27001',
      score: 94,
      status: 'compliant',
      controls: { total: 114, passed: 107, failed: 7 }
    }
  ]
  
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'compliant':
        return <CheckCircle className="w-5 h-5 text-green-400" />
      case 'warning':
        return <AlertCircle className="w-5 h-5 text-yellow-400" />
      case 'non-compliant':
        return <XCircle className="w-5 h-5 text-red-400" />
      default:
        return null
    }
  }
  
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'compliant':
        return 'text-green-400'
      case 'warning':
        return 'text-yellow-400'
      case 'non-compliant':
        return 'text-red-400'
      default:
        return 'text-gray-400'
    }
  }
  
  const overallScore = Math.round(
    frameworks.reduce((acc, fw) => acc + fw.score, 0) / frameworks.length
  )
  
  return (
    <div className="space-y-4">
      <div className="relative h-48 flex items-center justify-center">
        <svg className="absolute inset-0 w-full h-full">
          <circle
            cx="50%"
            cy="50%"
            r="70"
            stroke="currentColor"
            strokeWidth="8"
            fill="none"
            className="text-gray-700"
          />
          <motion.circle
            cx="50%"
            cy="50%"
            r="70"
            stroke="url(#gradient)"
            strokeWidth="8"
            fill="none"
            strokeLinecap="round"
            strokeDasharray={440}
            strokeDashoffset={440 - (440 * overallScore) / 100}
            initial={{ strokeDashoffset: 440 }}
            animate={{ strokeDashoffset: 440 - (440 * overallScore) / 100 }}
            transition={{ duration: 1.5, ease: "easeInOut" }}
          />
          <defs>
            <linearGradient id="gradient" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#10b981" />
              <stop offset="100%" stopColor="#3b82f6" />
            </linearGradient>
          </defs>
        </svg>
        <div className="text-center">
          <div className="text-4xl font-bold text-white">{overallScore}%</div>
          <div className="text-sm text-gray-400 mt-1">Overall Compliance</div>
        </div>
      </div>
      
      <div className="space-y-3">
        {frameworks.map((framework, index) => (
          <motion.div
            key={framework.name}
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: index * 0.1 }}
            className="flex items-center justify-between p-3 bg-gray-700/30 rounded-lg"
          >
            <div className="flex items-center space-x-3">
              {getStatusIcon(framework.status)}
              <div>
                <div className="font-medium text-white">{framework.name}</div>
                <div className="text-xs text-gray-400">
                  {framework.controls.passed}/{framework.controls.total} controls passed
                </div>
              </div>
            </div>
            <div className={`text-lg font-semibold ${getStatusColor(framework.status)}`}>
              {framework.score}%
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  )
}