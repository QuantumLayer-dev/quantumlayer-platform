import { NextRequest, NextResponse } from 'next/server'
import { v4 as uuidv4 } from 'uuid'

// In production, use Redis or database
const previewStore = new Map<string, any>()

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { workflowId, capsuleId, ttlMinutes = 60 } = body
    
    if (!workflowId) {
      return NextResponse.json(
        { error: 'workflowId is required' },
        { status: 400 }
      )
    }
    
    // Generate unique preview ID
    const previewId = `preview-${uuidv4().substring(0, 8)}`
    const expiresAt = new Date(Date.now() + (ttlMinutes * 60 * 1000))
    
    // Store preview metadata
    const previewData = {
      id: previewId,
      workflowId,
      capsuleId,
      createdAt: new Date().toISOString(),
      expiresAt: expiresAt.toISOString(),
      ttlMinutes,
      accessCount: 0,
    }
    
    previewStore.set(previewId, previewData)
    
    // Schedule cleanup
    setTimeout(() => {
      previewStore.delete(previewId)
    }, ttlMinutes * 60 * 1000)
    
    // Generate URLs
    const baseUrl = process.env.NEXT_PUBLIC_BASE_URL || `http://192.168.1.217:30900`
    const previewUrl = `${baseUrl}/preview/${workflowId}`
    const shareableUrl = `${baseUrl}/p/${previewId}`
    
    return NextResponse.json({
      success: true,
      previewId,
      previewUrl,
      shareableUrl,
      expiresAt: expiresAt.toISOString(),
      ttlMinutes,
      message: `Preview will be available for ${ttlMinutes} minutes`
    })
  } catch (error) {
    console.error('Error creating preview:', error)
    return NextResponse.json(
      { error: 'Failed to create preview' },
      { status: 500 }
    )
  }
}

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const previewId = searchParams.get('id')
  
  if (!previewId) {
    return NextResponse.json(
      { error: 'Preview ID is required' },
      { status: 400 }
    )
  }
  
  const previewData = previewStore.get(previewId)
  
  if (!previewData) {
    return NextResponse.json(
      { error: 'Preview not found or expired' },
      { status: 404 }
    )
  }
  
  // Check if expired
  if (new Date(previewData.expiresAt) < new Date()) {
    previewStore.delete(previewId)
    return NextResponse.json(
      { error: 'Preview has expired' },
      { status: 410 }
    )
  }
  
  // Increment access count
  previewData.accessCount++
  
  return NextResponse.json({
    success: true,
    preview: previewData
  })
}