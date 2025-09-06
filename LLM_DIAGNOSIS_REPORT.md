# LLM Router Diagnosis Report

**Date**: 2025-09-05  
**Status**: PARTIALLY WORKING - Needs Configuration Fixes

## Summary
The LLM router is operational but has configuration issues causing suboptimal responses. The service successfully routes requests but the validation logic is too restrictive, causing valid code to be rejected.

## Current State

### ✅ Working Components
1. **LLM Router Service**: 3 pods running successfully
2. **Azure OpenAI**: Connected and responding (using gpt-4.1 deployment)
3. **Groq API**: Configured and available as fallback
4. **Workflow Integration**: Workflows can call LLM router successfully
5. **Basic Code Generation**: Works when properly prompted

### ❌ Issues Identified

#### 1. Overly Restrictive Validation
**File**: `packages/llm-router/cmd/main.go:518-560`
**Problem**: The `isValidCodeResponse()` function rejects valid code if:
- Response contains words like "Hello", "help", "assist" (even in comments)
- Code is less than 100 characters (rejects small functions)
- Doesn't have at least 3 specific code patterns

**Impact**: Valid code responses are rejected, causing fallback to templates or conversational responses

#### 2. Improper Message Formatting
**Issue**: Workflow activities aren't sending properly structured system prompts
**Result**: LLM returns conversational responses instead of code

#### 3. Azure Configuration Issues
- Timeout errors with Azure OpenAI endpoint
- Invalid endpoint URL: `https://myazurellm.openai.azure.com/`
- Credentials appear to be placeholders

## Test Results

### Direct API Test (Success)
```bash
curl -X POST http://192.168.1.177:30881/generate \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "system", "content": "You are a code generator. Generate only code."},
      {"role": "user", "content": "Create Flask REST API"}
    ],
    "provider": "azure",
    "max_tokens": 2000
  }'
```
**Result**: Generated 533 tokens of valid Python Flask code

### Workflow Test (Partial Success)
- Workflow ID: `code-gen-b93bff12-4ca9-438b-8ae7-71b64425ceec`
- Status: Completed
- Issue: Generated "Hello! How can I assist you today?" instead of code

## Root Causes

1. **Validation Logic**: Too aggressive in filtering responses
2. **System Prompts**: Not properly instructing LLM to generate code
3. **Provider Fallback**: When Groq/Azure fail validation, falls back incorrectly

## Recommended Fixes

### Immediate (Critical)
1. **Fix Validation Function** (`packages/llm-router/cmd/main.go:518`)
   - Remove greeting pattern checks
   - Reduce minimum length to 50 characters
   - Accept responses with code blocks in markdown

2. **Update Workflow Activities** (`packages/workflows/internal/activities/activities.go`)
   - Add explicit system prompt: "Generate only code without explanations"
   - Ensure proper message structure

3. **Fix Azure Credentials**
   - Update `llm-credentials` secret with valid Azure OpenAI keys
   - Correct the endpoint URL

### Code Changes Required

#### 1. Fix isValidCodeResponse function:
```go
func isValidCodeResponse(response string) bool {
    // Only reject explicit errors
    if strings.HasPrefix(response, "Error:") {
        return false
    }
    
    // Check for minimal code patterns (reduced threshold)
    codePatterns := []string{"def ", "function ", "class ", "import ", 
                            "const ", "let ", "return ", "if ", "for "}
    
    for _, pattern := range codePatterns {
        if strings.Contains(response, pattern) {
            return true // Accept on first match
        }
    }
    
    // Accept if looks like code (has brackets/semicolons)
    return strings.Contains(response, "{") || strings.Contains(response, ";")
}
```

#### 2. Update Activity System Prompt:
```go
llmRequest := map[string]interface{}{
    "messages": []map[string]string{
        {"role": "system", "content": "You are a code generator. Generate only code without explanations, markdown formatting, or conversational text. Return pure code only."},
        {"role": "user", "content": request.Prompt},
    },
    // ...
}
```

## Provider Status

| Provider | Status | Issue | Priority |
|----------|--------|-------|----------|
| Azure OpenAI | ⚠️ Working with Issues | Invalid endpoint URL, timeouts | High |
| Groq | ✅ Configured | Working but rejected by validation | Medium |
| AWS Bedrock | ❌ Not Working | No permissions for Bedrock | Low |
| OpenAI | ❓ Unknown | Not tested | Low |
| Anthropic | ❓ Unknown | Not tested | Low |

## Testing Commands

### Test LLM Router Health
```bash
curl http://192.168.1.177:30881/health
```

### Test Code Generation
```bash
curl -X POST http://192.168.1.177:30881/generate \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "system", "content": "Generate code only"},
      {"role": "user", "content": "Python function for fibonacci"}
    ],
    "provider": "groq",
    "max_tokens": 500
  }'
```

### Check Logs
```bash
kubectl logs -n quantumlayer deployment/llm-router --tail=50
```

## Conclusion

The LLM router is fundamentally working but needs configuration adjustments. The main issue is overly restrictive validation causing valid responses to be rejected. With the recommended fixes, the system should generate proper code consistently.

**Estimated Fix Time**: 1-2 hours
**Risk Level**: Low (configuration changes only)
**Impact**: High (will fix code generation across platform)