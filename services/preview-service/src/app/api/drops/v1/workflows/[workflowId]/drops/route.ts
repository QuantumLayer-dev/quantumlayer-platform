import { NextRequest, NextResponse } from 'next/server';

// This mirrors the /api/capsules/[workflowId]/drops endpoint 
// but at the path the frontend expects
export async function GET(
  request: NextRequest,
  { params }: { params: { workflowId: string } }
) {
  const workflowId = params.workflowId;
  
  // Get QuantumDrops service URL from environment
  const quantumDropsUrl = process.env.QUANTUM_DROPS_URL || 'http://quantum-drops.quantumlayer.svc.cluster.local:8090';
  
  try {
    // Fetch drops from QuantumDrops service
    const response = await fetch(`${quantumDropsUrl}/api/v1/workflows/${workflowId}/drops`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      console.error(`Failed to fetch drops for workflow ${workflowId}: ${response.status}`);
      return NextResponse.json(
        { error: `Failed to fetch drops: ${response.statusText}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    
    // Transform drops data for frontend consumption
    const transformedData = {
      workflowId: data.workflow_id,
      drops: data.drops || [],
      totalDrops: data.total_drops || 0,
      files: transformDropsToFiles(data.drops || [])
    };
    
    return NextResponse.json(transformedData);
  } catch (error) {
    console.error('Error proxying to QuantumDrops:', error);
    return NextResponse.json(
      { error: 'Failed to fetch drops from backend service' },
      { status: 500 }
    );
  }
}

// Transform drops into file structure for Monaco editor
function transformDropsToFiles(drops: any[]) {
  const files: Record<string, { content: string; language: string }> = {};
  
  for (const drop of drops) {
    let fileName = '';
    let language = 'plaintext';
    
    // Map drop stage to file name and language
    switch (drop.stage) {
      case 'prompt_enhancement':
        fileName = 'prompt.txt';
        language = 'plaintext';
        break;
      case 'frd_generation':
        fileName = 'requirements.md';
        language = 'markdown';
        break;
      case 'code_generation':
        fileName = 'main.py';  // Default to Python, adjust based on actual language
        language = 'python';
        break;
      case 'test_plan_generation':
        fileName = 'test_plan.md';
        language = 'markdown';
        break;
      case 'test_generation':
        fileName = 'test_main.py';
        language = 'python';
        break;
      case 'documentation':
        fileName = 'README.md';
        language = 'markdown';
        break;
      case 'project_structure':
        fileName = 'structure.json';
        language = 'json';
        break;
      default:
        fileName = `${drop.stage}.txt`;
        language = 'plaintext';
    }
    
    // Handle structure type specifically
    if (drop.type === 'structure' && drop.artifact) {
      try {
        // Try to parse as JSON structure
        const structure = JSON.parse(drop.artifact);
        // Add each file from the structure
        Object.entries(structure).forEach(([path, content]) => {
          const ext = path.split('.').pop() || 'txt';
          files[path] = {
            content: content as string,
            language: getLanguageFromExtension(ext)
          };
        });
        continue;
      } catch (e) {
        // If not JSON, treat as plain text
      }
    }
    
    // Add the drop as a file
    if (drop.artifact) {
      files[fileName] = {
        content: drop.artifact,
        language: language
      };
    }
  }
  
  // If no files were created, add a default file
  if (Object.keys(files).length === 0) {
    files['README.md'] = {
      content: '# Generated Project\n\nNo code artifacts were generated yet.',
      language: 'markdown'
    };
  }
  
  return files;
}

// Helper function to get Monaco language from file extension
function getLanguageFromExtension(ext: string): string {
  const languageMap: Record<string, string> = {
    'py': 'python',
    'js': 'javascript',
    'ts': 'typescript',
    'jsx': 'javascript',
    'tsx': 'typescript',
    'java': 'java',
    'go': 'go',
    'rs': 'rust',
    'cpp': 'cpp',
    'c': 'c',
    'h': 'c',
    'hpp': 'cpp',
    'cs': 'csharp',
    'rb': 'ruby',
    'php': 'php',
    'swift': 'swift',
    'kt': 'kotlin',
    'scala': 'scala',
    'sh': 'shell',
    'bash': 'shell',
    'yml': 'yaml',
    'yaml': 'yaml',
    'json': 'json',
    'xml': 'xml',
    'html': 'html',
    'css': 'css',
    'scss': 'scss',
    'sql': 'sql',
    'md': 'markdown',
    'txt': 'plaintext',
    'dockerfile': 'dockerfile',
    'Dockerfile': 'dockerfile',
    'makefile': 'makefile',
    'Makefile': 'makefile',
  };
  
  return languageMap[ext.toLowerCase()] || 'plaintext';
}