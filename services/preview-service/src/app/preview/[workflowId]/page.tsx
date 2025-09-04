'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Editor from '@monaco-editor/react'
import toast from 'react-hot-toast'
import { 
  Play, 
  Save, 
  Download, 
  FileText, 
  FolderOpen,
  Terminal,
  Package,
  RefreshCw,
  Copy,
  Check
} from 'lucide-react'

interface CapsuleFile {
  path: string
  content: string
  type: string
}

interface CapsuleData {
  id: string
  name: string
  language: string
  structure: { [key: string]: CapsuleFile }
  metadata: any
}

interface ExecutionResult {
  output: string
  errors: string
  success: boolean
}

export default function PreviewPage() {
  const params = useParams()
  const workflowId = params.workflowId as string
  
  const [capsuleData, setCapsuleData] = useState<CapsuleData | null>(null)
  const [selectedFile, setSelectedFile] = useState<string>('main.py')
  const [fileContent, setFileContent] = useState<string>('')
  const [isExecuting, setIsExecuting] = useState(false)
  const [executionResult, setExecutionResult] = useState<ExecutionResult | null>(null)
  const [showTerminal, setShowTerminal] = useState(false)
  const [copied, setCopied] = useState(false)

  // Load capsule data from workflow
  useEffect(() => {
    loadCapsuleData()
  }, [workflowId])

  const loadCapsuleData = async () => {
    try {
      // First get the QuantumDrops
      const dropsResponse = await fetch(`/api/capsules/${workflowId}/drops`)
      const dropsData = await dropsResponse.json()
      
      // Find the code drop
      const codeDrop = dropsData.drops?.find((d: any) => d.type === 'code')
      if (!codeDrop) {
        toast.error('No code found for this workflow')
        return
      }

      // Build a capsule from the code
      const capsuleResponse = await fetch('/api/capsule/v1/build', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          workflow_id: workflowId,
          name: 'preview-app',
          type: 'application',
          language: 'python', // TODO: Get from workflow
          code: codeDrop.artifact
        })
      })
      
      const capsule = await capsuleResponse.json()
      setCapsuleData(capsule)
      
      // Set initial file content
      if (capsule.structure['main.py']) {
        setFileContent(capsule.structure['main.py'].content)
      }
      
      toast.success('Project loaded successfully')
    } catch (error) {
      console.error('Error loading capsule:', error)
      toast.error('Failed to load project')
    }
  }

  const executeCode = async () => {
    if (!capsuleData) return
    
    setIsExecuting(true)
    setShowTerminal(true)
    setExecutionResult(null)
    
    try {
      const response = await fetch('/api/sandbox/v1/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: `preview-${workflowId}`,
          language: capsuleData.language,
          code: fileContent,
          timeout: 30
        })
      })
      
      const result = await response.json()
      
      // Poll for execution result
      setTimeout(async () => {
        try {
          const statusResponse = await fetch(`/api/sandbox/v1/executions/${result.id}`)
          const statusData = await statusResponse.json()
          
          setExecutionResult({
            output: statusData.output || 'Execution completed',
            errors: statusData.errors || '',
            success: statusData.success !== false
          })
        } catch (error) {
          setExecutionResult({
            output: '',
            errors: 'Failed to get execution result',
            success: false
          })
        }
      }, 3000)
      
      toast.success('Code execution started')
    } catch (error) {
      console.error('Execution error:', error)
      toast.error('Failed to execute code')
      setExecutionResult({
        output: '',
        errors: 'Execution failed',
        success: false
      })
    } finally {
      setIsExecuting(false)
    }
  }

  const saveFile = () => {
    if (!capsuleData) return
    
    // Update the file in the capsule structure
    capsuleData.structure[selectedFile].content = fileContent
    toast.success('File saved')
  }

  const downloadProject = () => {
    if (!capsuleData) return
    
    // Create a blob with all files
    const files = Object.entries(capsuleData.structure)
      .map(([path, file]) => `// File: ${path}\n${file.content}`)
      .join('\n\n' + '='.repeat(50) + '\n\n')
    
    const blob = new Blob([files], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${capsuleData.name}.txt`
    a.click()
    
    toast.success('Project downloaded')
  }

  const copyCode = () => {
    navigator.clipboard.writeText(fileContent)
    setCopied(true)
    toast.success('Code copied to clipboard')
    setTimeout(() => setCopied(false), 2000)
  }

  if (!capsuleData) {
    return (
      <div className="flex items-center justify-center h-screen bg-background">
        <div className="text-center">
          <RefreshCw className="w-8 h-8 animate-spin mx-auto mb-4" />
          <p>Loading project...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border px-4 py-2 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-lg font-semibold">QuantumLayer Preview</h1>
          <span className="text-sm text-muted-foreground">{capsuleData.name}</span>
        </div>
        
        <div className="flex items-center gap-2">
          <button
            onClick={executeCode}
            disabled={isExecuting}
            className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 disabled:opacity-50 rounded-md text-white text-sm"
          >
            <Play className="w-4 h-4" />
            {isExecuting ? 'Running...' : 'Run'}
          </button>
          
          <button
            onClick={saveFile}
            className="flex items-center gap-2 px-3 py-2 hover:bg-secondary rounded-md text-sm"
          >
            <Save className="w-4 h-4" />
            Save
          </button>
          
          <button
            onClick={copyCode}
            className="flex items-center gap-2 px-3 py-2 hover:bg-secondary rounded-md text-sm"
          >
            {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
            Copy
          </button>
          
          <button
            onClick={downloadProject}
            className="flex items-center gap-2 px-3 py-2 hover:bg-secondary rounded-md text-sm"
          >
            <Download className="w-4 h-4" />
            Download
          </button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* File Explorer */}
        <div className="w-64 file-explorer p-2">
          <div className="flex items-center gap-2 px-2 py-1 mb-2 text-sm font-semibold">
            <FolderOpen className="w-4 h-4" />
            Files
          </div>
          
          {Object.keys(capsuleData.structure).map(path => (
            <div
              key={path}
              onClick={() => {
                setSelectedFile(path)
                setFileContent(capsuleData.structure[path].content)
              }}
              className={`file-item ${selectedFile === path ? 'active' : ''}`}
            >
              <FileText className="w-4 h-4" />
              <span className="text-sm">{path}</span>
            </div>
          ))}
        </div>

        {/* Editor */}
        <div className="flex-1 flex flex-col">
          <div className="flex-1">
            <Editor
              height="100%"
              defaultLanguage={capsuleData.language}
              value={fileContent}
              onChange={(value) => setFileContent(value || '')}
              theme="vs-dark"
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                wordWrap: 'on',
                automaticLayout: true,
              }}
            />
          </div>

          {/* Terminal */}
          {showTerminal && (
            <div className="h-64 border-t border-border">
              <div className="flex items-center justify-between px-4 py-2 bg-secondary">
                <div className="flex items-center gap-2">
                  <Terminal className="w-4 h-4" />
                  <span className="text-sm font-semibold">Output</span>
                </div>
                <button
                  onClick={() => setShowTerminal(false)}
                  className="text-xs hover:text-foreground"
                >
                  Close
                </button>
              </div>
              
              <div className="terminal-output">
                {isExecuting && (
                  <div className="terminal-line">
                    <span className="text-yellow-500">Executing code...</span>
                  </div>
                )}
                
                {executionResult && (
                  <>
                    {executionResult.output && (
                      <div className="terminal-line">
                        <pre className="whitespace-pre-wrap">{executionResult.output}</pre>
                      </div>
                    )}
                    
                    {executionResult.errors && (
                      <div className="terminal-line terminal-error">
                        <pre className="whitespace-pre-wrap">{executionResult.errors}</pre>
                      </div>
                    )}
                    
                    {executionResult.success && !executionResult.output && !executionResult.errors && (
                      <div className="terminal-line terminal-success">
                        Execution completed successfully
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Status Bar */}
      <div className="border-t border-border px-4 py-1 flex items-center justify-between text-xs text-muted-foreground">
        <div className="flex items-center gap-4">
          <span>{capsuleData.language}</span>
          <span>{selectedFile}</span>
        </div>
        <div className="flex items-center gap-4">
          <span>Workflow: {workflowId.substring(0, 8)}...</span>
          <button
            onClick={() => setShowTerminal(!showTerminal)}
            className="flex items-center gap-1 hover:text-foreground"
          >
            <Terminal className="w-3 h-3" />
            Terminal
          </button>
        </div>
      </div>
    </div>
  )
}