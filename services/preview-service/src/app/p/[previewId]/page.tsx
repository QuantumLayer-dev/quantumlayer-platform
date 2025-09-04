'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { AlertCircle, Clock, ExternalLink } from 'lucide-react'

interface PreviewData {
  id: string
  workflowId: string
  capsuleId?: string
  createdAt: string
  expiresAt: string
  ttlMinutes: number
  accessCount: number
}

export default function ShareablePreviewPage() {
  const params = useParams()
  const router = useRouter()
  const previewId = params.previewId as string
  
  const [previewData, setPreviewData] = useState<PreviewData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [timeRemaining, setTimeRemaining] = useState<string>('')

  useEffect(() => {
    fetchPreviewData()
  }, [previewId])

  useEffect(() => {
    if (!previewData) return
    
    const timer = setInterval(() => {
      const remaining = calculateTimeRemaining(previewData.expiresAt)
      if (remaining === 'Expired') {
        setError('This preview has expired')
        clearInterval(timer)
      } else {
        setTimeRemaining(remaining)
      }
    }, 1000)
    
    return () => clearInterval(timer)
  }, [previewData])

  const fetchPreviewData = async () => {
    try {
      const response = await fetch(`/api/preview?id=${previewId}`)
      const data = await response.json()
      
      if (!response.ok) {
        setError(data.error || 'Failed to load preview')
        return
      }
      
      setPreviewData(data.preview)
      
      // Redirect to actual preview page
      setTimeout(() => {
        router.push(`/preview/${data.preview.workflowId}`)
      }, 2000)
    } catch (err) {
      setError('Failed to load preview')
    } finally {
      setLoading(false)
    }
  }

  const calculateTimeRemaining = (expiresAt: string): string => {
    const now = new Date().getTime()
    const expiry = new Date(expiresAt).getTime()
    const diff = expiry - now
    
    if (diff <= 0) return 'Expired'
    
    const minutes = Math.floor(diff / (1000 * 60))
    const seconds = Math.floor((diff % (1000 * 60)) / 1000)
    
    if (minutes > 60) {
      const hours = Math.floor(minutes / 60)
      return `${hours}h ${minutes % 60}m`
    }
    
    return `${minutes}m ${seconds}s`
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-lg">Loading preview...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="bg-card border border-destructive rounded-lg p-8 max-w-md">
          <AlertCircle className="w-12 h-12 text-destructive mx-auto mb-4" />
          <h2 className="text-xl font-bold text-center mb-2">Preview Unavailable</h2>
          <p className="text-center text-muted-foreground">{error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="bg-card border border-border rounded-lg p-8 max-w-md">
        <h1 className="text-2xl font-bold mb-4">QuantumLayer Preview</h1>
        
        {previewData && (
          <>
            <div className="space-y-3 mb-6">
              <div className="flex items-center gap-2 text-sm">
                <Clock className="w-4 h-4 text-muted-foreground" />
                <span className="text-muted-foreground">Time remaining:</span>
                <span className="font-semibold text-primary">{timeRemaining}</span>
              </div>
              
              <div className="text-sm">
                <span className="text-muted-foreground">Preview ID:</span>
                <code className="ml-2 px-2 py-1 bg-secondary rounded">{previewId}</code>
              </div>
              
              <div className="text-sm">
                <span className="text-muted-foreground">Access count:</span>
                <span className="ml-2">{previewData.accessCount} views</span>
              </div>
            </div>
            
            <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
              <ExternalLink className="w-4 h-4" />
              <span>Redirecting to preview...</span>
            </div>
          </>
        )}
      </div>
    </div>
  )
}