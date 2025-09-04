import { NextRequest, NextResponse } from 'next/server';

// Proxy to Sandbox Executor service
export async function POST(request: NextRequest) {
  const sandboxUrl = process.env.SANDBOX_EXECUTOR_URL || 'http://sandbox-executor.quantumlayer.svc.cluster.local:8085';
  
  try {
    const body = await request.json();
    
    // Forward request to sandbox executor
    const response = await fetch(`${sandboxUrl}/api/v1/execute`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body)
    });

    if (!response.ok) {
      const error = await response.text();
      return NextResponse.json(
        { error: `Sandbox execution failed: ${error}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error) {
    console.error('Error proxying to sandbox:', error);
    return NextResponse.json(
      { error: 'Failed to execute code in sandbox' },
      { status: 500 }
    );
  }
}