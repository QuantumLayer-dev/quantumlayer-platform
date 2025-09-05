import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'

interface DriftNode {
  id: string
  name: string
  drift: number
  region: string
  type: string
}

export default function DriftHeatmap() {
  const [nodes, setNodes] = useState<DriftNode[]>([])
  
  useEffect(() => {
    // Generate sample nodes for demonstration
    const regions = ['us-east-1', 'us-west-2', 'eu-west-1', 'ap-south-1']
    const types = ['web', 'api', 'db', 'cache', 'worker']
    
    const generatedNodes = []
    for (let i = 0; i < 48; i++) {
      generatedNodes.push({
        id: `node-${i}`,
        name: `node-${i}`,
        drift: Math.random(),
        region: regions[Math.floor(Math.random() * regions.length)],
        type: types[Math.floor(Math.random() * types.length)],
      })
    }
    setNodes(generatedNodes)
  }, [])
  
  const getDriftColor = (drift: number) => {
    if (drift < 0.3) return 'bg-green-600'
    if (drift < 0.5) return 'bg-yellow-600'
    if (drift < 0.75) return 'bg-orange-600'
    return 'bg-red-600'
  }
  
  const getDriftLabel = (drift: number) => {
    if (drift < 0.3) return 'Low'
    if (drift < 0.5) return 'Medium'
    if (drift < 0.75) return 'High'
    return 'Critical'
  }
  
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-12 gap-1">
        {nodes.map((node, index) => (
          <motion.div
            key={node.id}
            initial={{ scale: 0, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ delay: index * 0.01 }}
            className="relative group"
          >
            <div
              className={`aspect-square rounded ${getDriftColor(node.drift)} hover:scale-110 transition-transform cursor-pointer`}
              title={`${node.name} (${node.region}): ${(node.drift * 100).toFixed(1)}% drift`}
            />
            <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-2 py-1 bg-gray-900 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-10">
              <div className="font-semibold">{node.name}</div>
              <div className="text-gray-400">{node.region} • {node.type}</div>
              <div className={`font-medium ${
                node.drift < 0.5 ? 'text-green-400' : 'text-red-400'
              }`}>
                {(node.drift * 100).toFixed(1)}% drift
              </div>
            </div>
          </motion.div>
        ))}
      </div>
      
      <div className="flex items-center justify-between pt-4 border-t border-gray-700">
        <div className="flex items-center space-x-4">
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 bg-green-600 rounded" />
            <span className="text-xs text-gray-400">Low (&lt;30%)</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 bg-yellow-600 rounded" />
            <span className="text-xs text-gray-400">Medium (30-50%)</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 bg-orange-600 rounded" />
            <span className="text-xs text-gray-400">High (50-75%)</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 bg-red-600 rounded" />
            <span className="text-xs text-gray-400">Critical (&gt;75%)</span>
          </div>
        </div>
        
        <div className="text-sm text-gray-400">
          {nodes.filter(n => n.drift > 0.75).length} nodes need immediate attention
        </div>
      </div>
    </div>
  )
}