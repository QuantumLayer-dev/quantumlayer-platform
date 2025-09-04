import { NextResponse } from 'next/server'

export async function GET() {
  return NextResponse.json({ 
    status: 'healthy',
    service: 'preview-service',
    timestamp: new Date().toISOString()
  })
}