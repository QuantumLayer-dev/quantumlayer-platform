import { motion } from 'framer-motion'
import { Clock, CheckCircle, AlertCircle, Package, Zap } from 'lucide-react'

interface PatchEvent {
  id: string
  type: 'scheduled' | 'deployed' | 'failed' | 'rollback'
  title: string
  description: string
  time: string
  severity: 'low' | 'medium' | 'high' | 'critical'
}

export default function PatchTimeline() {
  const events: PatchEvent[] = [
    {
      id: '1',
      type: 'deployed',
      title: 'Security Update KB5032190',
      description: 'Windows Server critical security patch',
      time: '2 hours ago',
      severity: 'critical'
    },
    {
      id: '2',
      type: 'scheduled',
      title: 'Linux Kernel 6.5.0-15',
      description: 'Kernel update for Ubuntu servers',
      time: 'in 4 hours',
      severity: 'high'
    },
    {
      id: '3',
      type: 'deployed',
      title: 'OpenSSL 3.0.13',
      description: 'TLS vulnerability patch',
      time: '6 hours ago',
      severity: 'critical'
    },
    {
      id: '4',
      type: 'rollback',
      title: 'Docker Engine 24.0.7',
      description: 'Rolled back due to compatibility issues',
      time: '12 hours ago',
      severity: 'medium'
    },
    {
      id: '5',
      type: 'scheduled',
      title: 'PostgreSQL 15.5',
      description: 'Database security and performance update',
      time: 'Tomorrow 2:00 AM',
      severity: 'high'
    }
  ]
  
  const getEventIcon = (type: string) => {
    switch (type) {
      case 'deployed':
        return <CheckCircle className="w-5 h-5 text-green-400" />
      case 'scheduled':
        return <Clock className="w-5 h-5 text-blue-400" />
      case 'failed':
        return <AlertCircle className="w-5 h-5 text-red-400" />
      case 'rollback':
        return <AlertCircle className="w-5 h-5 text-yellow-400" />
      default:
        return <Package className="w-5 h-5 text-gray-400" />
    }
  }
  
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-900/50 border-red-700'
      case 'high':
        return 'bg-orange-900/50 border-orange-700'
      case 'medium':
        return 'bg-yellow-900/50 border-yellow-700'
      case 'low':
        return 'bg-green-900/50 border-green-700'
      default:
        return 'bg-gray-900/50 border-gray-700'
    }
  }
  
  return (
    <div className="relative">
      <div className="absolute left-6 top-0 bottom-0 w-px bg-gray-700" />
      
      <div className="space-y-4">
        {events.map((event, index) => (
          <motion.div
            key={event.id}
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: index * 0.1 }}
            className="relative flex items-start space-x-4"
          >
            <div className="relative z-10 bg-gray-800 p-1.5 rounded-full">
              {getEventIcon(event.type)}
            </div>
            
            <div className={`flex-1 p-4 rounded-lg border ${getSeverityColor(event.severity)}`}>
              <div className="flex items-start justify-between">
                <div>
                  <h4 className="font-medium text-white">{event.title}</h4>
                  <p className="text-sm text-gray-400 mt-1">{event.description}</p>
                </div>
                <span className="text-xs text-gray-500">{event.time}</span>
              </div>
              
              <div className="flex items-center space-x-2 mt-2">
                <span className={`text-xs px-2 py-1 rounded-full ${
                  event.severity === 'critical' ? 'bg-red-900/50 text-red-400' :
                  event.severity === 'high' ? 'bg-orange-900/50 text-orange-400' :
                  event.severity === 'medium' ? 'bg-yellow-900/50 text-yellow-400' :
                  'bg-green-900/50 text-green-400'
                }`}>
                  {event.severity}
                </span>
                
                {event.type === 'deployed' && (
                  <span className="text-xs px-2 py-1 rounded-full bg-green-900/50 text-green-400">
                    <Zap className="inline w-3 h-3 mr-1" />
                    Successful
                  </span>
                )}
              </div>
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  )
}