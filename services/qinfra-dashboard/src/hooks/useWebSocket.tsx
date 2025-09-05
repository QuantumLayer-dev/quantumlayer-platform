import { useEffect, useState, useCallback } from 'react'
import io, { Socket } from 'socket.io-client'
import toast from 'react-hot-toast'
import { useDashboardStore } from '@/lib/store'

export function useWebSocket() {
  const [socket, setSocket] = useState<Socket | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const { updateMetrics } = useDashboardStore()
  
  useEffect(() => {
    const socketInstance = io(process.env.NEXT_PUBLIC_WS_URL || 'http://localhost:8099', {
      transports: ['websocket'],
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 1000,
    })
    
    socketInstance.on('connect', () => {
      setIsConnected(true)
      console.log('Connected to QInfra WebSocket')
    })
    
    socketInstance.on('disconnect', () => {
      setIsConnected(false)
      console.log('Disconnected from QInfra WebSocket')
    })
    
    socketInstance.on('drift-alert', (data) => {
      toast.error(`Drift Alert: ${data.nodeId} - ${data.driftLevel}% drift detected`, {
        duration: 6000,
      })
      updateMetrics({ lastDriftAlert: data })
    })
    
    socketInstance.on('patch-update', (data) => {
      toast.success(`Patch ${data.patchId} ${data.status} on ${data.nodeCount} nodes`, {
        duration: 4000,
      })
      updateMetrics({ lastPatchUpdate: data })
    })
    
    socketInstance.on('compliance-change', (data) => {
      const message = data.improved 
        ? `Compliance improved: ${data.framework} now at ${data.score}%`
        : `Compliance degraded: ${data.framework} dropped to ${data.score}%`
      
      if (data.improved) {
        toast.success(message)
      } else {
        toast.error(message)
      }
      updateMetrics({ complianceScore: data.overallScore })
    })
    
    socketInstance.on('metrics-update', (data) => {
      updateMetrics(data)
    })
    
    socketInstance.on('anomaly-detected', (data) => {
      toast.error(`Anomaly Detected: ${data.type} on ${data.affectedNodes} nodes`, {
        duration: 8000,
      })
    })
    
    socketInstance.on('error', (error) => {
      console.error('WebSocket error:', error)
      toast.error('Connection error. Some real-time features may be unavailable.')
    })
    
    setSocket(socketInstance)
    
    return () => {
      socketInstance.disconnect()
    }
  }, [updateMetrics])
  
  const sendMessage = useCallback((event: string, data: any) => {
    if (socket && isConnected) {
      socket.emit(event, data)
    } else {
      console.warn('Socket not connected, unable to send message')
    }
  }, [socket, isConnected])
  
  const subscribe = useCallback((channel: string) => {
    if (socket && isConnected) {
      socket.emit('subscribe', { channel })
    }
  }, [socket, isConnected])
  
  const unsubscribe = useCallback((channel: string) => {
    if (socket && isConnected) {
      socket.emit('unsubscribe', { channel })
    }
  }, [socket, isConnected])
  
  return {
    socket,
    isConnected,
    sendMessage,
    subscribe,
    unsubscribe,
  }
}