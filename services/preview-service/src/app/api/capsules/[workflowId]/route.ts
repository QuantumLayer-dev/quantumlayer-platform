import { NextRequest, NextResponse } from 'next/server';

// API route to get capsule information
export async function GET(
  request: NextRequest,
  { params }: { params: { workflowId: string } }
) {
  const workflowId = params.workflowId;
  
  // Get service URLs from environment
  const capsuleBuilderUrl = process.env.CAPSULE_BUILDER_URL || 'http://capsule-builder.quantumlayer.svc.cluster.local:8092';
  const quantumDropsUrl = process.env.QUANTUM_DROPS_URL || 'http://quantum-drops.quantumlayer.svc.cluster.local:8090';
  
  try {
    // First, try to get capsule from Capsule Builder
    // Note: The capsule builder might not have persisted data, so we'll fall back to QuantumDrops
    
    // For now, we'll primarily use QuantumDrops as source of truth
    const dropsResponse = await fetch(`${quantumDropsUrl}/api/v1/workflows/${workflowId}/drops`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!dropsResponse.ok) {
      return NextResponse.json(
        { error: `Workflow not found: ${workflowId}` },
        { status: 404 }
      );
    }

    const dropsData = await dropsResponse.json();
    
    // Create a capsule-like response from drops data
    const capsule = {
      id: `capsule-${workflowId}`,
      workflowId: workflowId,
      name: 'Generated Project',
      description: 'AI-generated code project',
      createdAt: new Date().toISOString(),
      drops: dropsData.drops || [],
      metadata: {
        totalDrops: dropsData.total_drops || 0,
        stages: extractStages(dropsData.drops || []),
        language: detectLanguage(dropsData.drops || []),
      }
    };
    
    return NextResponse.json(capsule);
  } catch (error) {
    console.error('Error fetching capsule data:', error);
    return NextResponse.json(
      { error: 'Failed to fetch capsule information' },
      { status: 500 }
    );
  }
}

// Extract unique stages from drops
function extractStages(drops: any[]): string[] {
  const stages = new Set<string>();
  drops.forEach(drop => {
    if (drop.stage) {
      stages.add(drop.stage);
    }
  });
  return Array.from(stages);
}

// Detect primary language from drops
function detectLanguage(drops: any[]): string {
  // Look for code generation drop
  const codeDrop = drops.find(d => d.stage === 'code_generation');
  if (codeDrop && codeDrop.artifact) {
    // Simple heuristic based on code patterns
    const code = codeDrop.artifact;
    if (code.includes('def ') || code.includes('import ')) return 'python';
    if (code.includes('function ') || code.includes('const ')) return 'javascript';
    if (code.includes('package ') && code.includes('func ')) return 'go';
    if (code.includes('public class ')) return 'java';
    if (code.includes('fn ') && code.includes('let ')) return 'rust';
  }
  
  return 'unknown';
}