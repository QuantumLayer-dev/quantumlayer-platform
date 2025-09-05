import { motion } from 'framer-motion'

interface RiskItem {
  id: string
  name: string
  probability: number
  impact: number
  category: string
}

export default function RiskMatrix() {
  const risks: RiskItem[] = [
    { id: '1', name: 'Unpatched CVE-2024-1234', probability: 4, impact: 5, category: 'Security' },
    { id: '2', name: 'Config drift in prod', probability: 3, impact: 4, category: 'Configuration' },
    { id: '3', name: 'Expired SSL certificates', probability: 2, impact: 5, category: 'Compliance' },
    { id: '4', name: 'Resource exhaustion', probability: 3, impact: 3, category: 'Performance' },
    { id: '5', name: 'Backup failure', probability: 1, impact: 5, category: 'DR' },
    { id: '6', name: 'Network latency spike', probability: 4, impact: 2, category: 'Network' },
    { id: '7', name: 'Memory leak in API', probability: 2, impact: 3, category: 'Application' },
    { id: '8', name: 'DDoS vulnerability', probability: 2, impact: 4, category: 'Security' },
  ]
  
  const getRiskColor = (probability: number, impact: number) => {
    const score = probability * impact
    if (score <= 5) return 'bg-green-600'
    if (score <= 10) return 'bg-yellow-600'
    if (score <= 15) return 'bg-orange-600'
    return 'bg-red-600'
  }
  
  const getRiskLevel = (probability: number, impact: number) => {
    const score = probability * impact
    if (score <= 5) return 'Low'
    if (score <= 10) return 'Medium'
    if (score <= 15) return 'High'
    return 'Critical'
  }
  
  const getCategoryColor = (category: string) => {
    const colors: { [key: string]: string } = {
      'Security': 'text-red-400',
      'Configuration': 'text-blue-400',
      'Compliance': 'text-purple-400',
      'Performance': 'text-yellow-400',
      'DR': 'text-green-400',
      'Network': 'text-cyan-400',
      'Application': 'text-orange-400'
    }
    return colors[category] || 'text-gray-400'
  }
  
  const matrix = Array(5).fill(null).map(() => Array(5).fill(null))
  
  risks.forEach(risk => {
    const x = risk.impact - 1
    const y = 5 - risk.probability
    if (!matrix[y][x]) matrix[y][x] = []
    matrix[y][x].push(risk)
  })
  
  return (
    <div className="space-y-4">
      <div className="relative">
        <div className="grid grid-cols-6 gap-1">
          <div className="col-span-1 row-span-1"></div>
          {[1, 2, 3, 4, 5].map(i => (
            <div key={i} className="text-center text-xs text-gray-400">
              {i}
            </div>
          ))}
          
          {[5, 4, 3, 2, 1].map((probability, y) => (
            <>
              <div key={`label-${probability}`} className="text-center text-xs text-gray-400 flex items-center justify-center">
                {probability}
              </div>
              {[1, 2, 3, 4, 5].map(impact => (
                <motion.div
                  key={`${probability}-${impact}`}
                  initial={{ scale: 0, opacity: 0 }}
                  animate={{ scale: 1, opacity: 1 }}
                  transition={{ delay: (y * 5 + (impact - 1)) * 0.02 }}
                  className={`aspect-square rounded flex items-center justify-center ${
                    getRiskColor(probability, impact)
                  } bg-opacity-20 border ${
                    getRiskColor(probability, impact).replace('bg-', 'border-')
                  } relative group`}
                >
                  {matrix[5 - probability][impact - 1] && matrix[5 - probability][impact - 1].length > 0 && (
                    <>
                      <span className="text-white font-bold text-sm">
                        {matrix[5 - probability][impact - 1].length}
                      </span>
                      <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 p-2 bg-gray-900 rounded shadow-lg opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-20 min-w-[200px]">
                        {matrix[5 - probability][impact - 1].map(risk => (
                          <div key={risk.id} className="text-xs whitespace-nowrap mb-1">
                            <span className={getCategoryColor(risk.category)}>
                              [{risk.category}]
                            </span>
                            <span className="text-white ml-1">{risk.name}</span>
                          </div>
                        ))}
                      </div>
                    </>
                  )}
                </motion.div>
              ))}
            </>
          ))}
        </div>
        
        <div className="absolute -bottom-8 left-1/2 transform -translate-x-1/2 text-xs text-gray-400">
          Impact →
        </div>
        <div className="absolute top-1/2 -left-14 transform -translate-y-1/2 -rotate-90 text-xs text-gray-400">
          Probability →
        </div>
      </div>
      
      <div className="mt-12 pt-4 border-t border-gray-700">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <div className="w-3 h-3 bg-green-600 rounded" />
              <span className="text-xs text-gray-400">Low Risk</span>
            </div>
            <div className="flex items-center space-x-2">
              <div className="w-3 h-3 bg-yellow-600 rounded" />
              <span className="text-xs text-gray-400">Medium Risk</span>
            </div>
            <div className="flex items-center space-x-2">
              <div className="w-3 h-3 bg-orange-600 rounded" />
              <span className="text-xs text-gray-400">High Risk</span>
            </div>
            <div className="flex items-center space-x-2">
              <div className="w-3 h-3 bg-red-600 rounded" />
              <span className="text-xs text-gray-400">Critical Risk</span>
            </div>
          </div>
          <div className="text-sm text-gray-400">
            {risks.filter(r => r.probability * r.impact > 15).length} critical risks identified
          </div>
        </div>
      </div>
    </div>
  )
}