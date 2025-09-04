'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { ArrowRight, Code, Zap, Package, Globe } from 'lucide-react'

export default function HomePage() {
  const router = useRouter()
  const [workflowId, setWorkflowId] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const handleOpenPreview = () => {
    if (!workflowId.trim()) return
    setIsLoading(true)
    router.push(`/preview/${workflowId}`)
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Hero Section */}
      <div className="container mx-auto px-4 py-16">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold mb-4">
            QuantumLayer Preview Service
          </h1>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            Live code preview, editing, and execution for your AI-generated applications
          </p>
        </div>

        {/* Input Section */}
        <div className="max-w-md mx-auto mb-16">
          <div className="bg-card border border-border rounded-lg p-6">
            <label className="block text-sm font-medium mb-2">
              Enter Workflow ID
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                value={workflowId}
                onChange={(e) => setWorkflowId(e.target.value)}
                placeholder="extended-code-gen-xxxxx"
                className="flex-1 px-4 py-2 bg-background border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
                onKeyPress={(e) => e.key === 'Enter' && handleOpenPreview()}
              />
              <button
                onClick={handleOpenPreview}
                disabled={!workflowId.trim() || isLoading}
                className="px-6 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {isLoading ? 'Loading...' : 'Open Preview'}
                <ArrowRight className="w-4 h-4" />
              </button>
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              Get the workflow ID from your QuantumLayer generation
            </p>
          </div>
        </div>

        {/* Features */}
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6 max-w-4xl mx-auto">
          <div className="bg-card border border-border rounded-lg p-6">
            <Code className="w-8 h-8 mb-3 text-primary" />
            <h3 className="font-semibold mb-2">Monaco Editor</h3>
            <p className="text-sm text-muted-foreground">
              Professional code editing with syntax highlighting
            </p>
          </div>

          <div className="bg-card border border-border rounded-lg p-6">
            <Zap className="w-8 h-8 mb-3 text-primary" />
            <h3 className="font-semibold mb-2">Live Execution</h3>
            <p className="text-sm text-muted-foreground">
              Run your code instantly in a sandboxed environment
            </p>
          </div>

          <div className="bg-card border border-border rounded-lg p-6">
            <Package className="w-8 h-8 mb-3 text-primary" />
            <h3 className="font-semibold mb-2">Project Structure</h3>
            <p className="text-sm text-muted-foreground">
              Complete project with all necessary files
            </p>
          </div>

          <div className="bg-card border border-border rounded-lg p-6">
            <Globe className="w-8 h-8 mb-3 text-primary" />
            <h3 className="font-semibold mb-2">Shareable URLs</h3>
            <p className="text-sm text-muted-foreground">
              Share your preview with TTL-based URLs
            </p>
          </div>
        </div>

        {/* Recent Workflows (placeholder) */}
        <div className="mt-16 max-w-4xl mx-auto">
          <h2 className="text-2xl font-bold mb-6">Recent Workflows</h2>
          <div className="bg-card border border-border rounded-lg p-4">
            <p className="text-muted-foreground text-center py-8">
              No recent workflows. Enter a workflow ID above to get started.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}