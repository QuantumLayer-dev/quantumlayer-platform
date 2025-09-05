import { motion } from 'framer-motion'
import { TrendingUp, TrendingDown } from 'lucide-react'

interface MetricCardProps {
  title: string
  value: string | number
  trend?: number
  icon: React.ReactNode
  color: 'green' | 'red' | 'blue' | 'purple' | 'yellow'
}

const colorClasses = {
  green: 'from-green-500 to-emerald-600',
  red: 'from-red-500 to-pink-600',
  blue: 'from-blue-500 to-cyan-600',
  purple: 'from-purple-500 to-violet-600',
  yellow: 'from-yellow-500 to-orange-600',
}

const bgClasses = {
  green: 'bg-green-900/20',
  red: 'bg-red-900/20',
  blue: 'bg-blue-900/20',
  purple: 'bg-purple-900/20',
  yellow: 'bg-yellow-900/20',
}

export default function MetricCard({ title, value, trend, icon, color }: MetricCardProps) {
  return (
    <motion.div
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      className="relative overflow-hidden rounded-xl bg-gray-800/50 backdrop-blur-sm border border-gray-700 p-6"
    >
      <div className="absolute top-0 right-0 w-32 h-32 -mr-8 -mt-8">
        <div className={`w-full h-full bg-gradient-to-br ${colorClasses[color]} opacity-20 rounded-full blur-2xl`} />
      </div>
      
      <div className="relative">
        <div className="flex items-center justify-between mb-4">
          <div className={`p-3 rounded-lg ${bgClasses[color]}`}>
            {icon}
          </div>
          {trend !== undefined && (
            <div className={`flex items-center space-x-1 text-sm ${
              trend > 0 ? 'text-green-400' : 'text-red-400'
            }`}>
              {trend > 0 ? (
                <TrendingUp className="w-4 h-4" />
              ) : (
                <TrendingDown className="w-4 h-4" />
              )}
              <span>{Math.abs(trend)}%</span>
            </div>
          )}
        </div>
        
        <h3 className="text-gray-400 text-sm font-medium mb-1">{title}</h3>
        <p className="text-2xl font-bold text-white">{value}</p>
      </div>
    </motion.div>
  )
}