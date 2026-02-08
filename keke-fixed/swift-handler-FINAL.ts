// Supabase Edge Function: swift-handler (FINAL FIX)
// ✅ Fixes Groq tool-calling error
// ✅ Handles general/code/research modes
// Deploy: supabase functions deploy swift-handler

import { serve } from "https://deno.land/std@0.168.0/http/server.ts"
import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Headers': 'authorization, x-client-info, apikey, content-type, x-pc-hash',
}

const AI_CONFIG = {
  provider: Deno.env.get('AI_PROVIDER') || 'groq',
  
  anthropic_key: Deno.env.get('ANTHROPIC_API_KEY'),
  openrouter_key: Deno.env.get('OPENROUTER_API_KEY'),
  openai_key: Deno.env.get('OPENAI_API_KEY'),
  groq_key: Deno.env.get('GROQ_API_KEYS'),
  
  models: {
    anthropic: {
      fast: 'claude-haiku-4-20250514',
      smart: 'claude-sonnet-4-20250514',
      deep: 'claude-opus-4-5-20251101'
    },
    openrouter: {
      fast: 'qwen/qwen3-32b',
      smart: 'meta-llama/llama-3.2-90b-vision-instruct:free',
      deep: 'qwen/qwen-2.5-72b-instruct:free'
    },
    openai: {
      fast: 'gpt-3.5-turbo',
      smart: 'gpt-4-turbo',
      deep: 'gpt-4'
    },
    groq: {
      fast: 'llama-3.3-70b-versatile',
      smart: 'llama-3.3-70b-versatile',
      deep: 'llama-3.3-70b-versatile'
    }
  },
  
  credit_rates: {
    anthropic: { fast: 1, smart: 3, deep: 8 },
    openrouter: { fast: 0, smart: 0, deep: 0 },
    openai: { fast: 1, smart: 4, deep: 6 },
    groq: { fast: 0, smart: 0, deep: 0 }
  },
  
  endpoints: {
    anthropic: 'https://api.anthropic.com/v1/messages',
    openrouter: 'https://openrouter.ai/api/v1/chat/completions',
    openai: 'https://api.openai.com/v1/chat/completions',
    groq: 'https://api.groq.com/openai/v1/chat/completions'
  }
}

serve(async (req) => {
  if (req.method === 'OPTIONS') {
    return new Response('ok', { headers: corsHeaders })
  }

  try {
    const authHeader = req.headers.get('Authorization')
    const pcHash = req.headers.get('X-PC-Hash')
    
    if (!authHeader || !pcHash) {
      return new Response(
        JSON.stringify({ error: 'Missing authorization' }),
        { status: 401, headers: { ...corsHeaders, 'Content-Type': 'application/json' }}
      )
    }

    const token = authHeader.replace('Bearer ', '')
    const { conversation, model, mode } = await req.json()

    const supabase = createClient(
      Deno.env.get('SUPABASE_URL')!,
      Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!
    )

    const { data: userData, error: userError } = await supabase.auth.getUser(token)
    if (userError || !userData.user) {
      return new Response(
        JSON.stringify({ error: 'Invalid token' }),
        { status: 401, headers: { ...corsHeaders, 'Content-Type': 'application/json' }}
      )
    }

    const userId = userData.user.id

    const { data: userRecord } = await supabase
      .from('users')
      .select('pc_hash')
      .eq('id', userId)
      .single()

    if (!userRecord || userRecord.pc_hash !== pcHash) {
      return new Response(
        JSON.stringify({ error: 'PC hash mismatch' }),
        { status: 403, headers: { ...corsHeaders, 'Content-Type': 'application/json' }}
      )
    }

    // Execute based on mode
    const result = await executeMode(conversation, model, mode || 'general')

    const { data: creditData } = await supabase
      .from('credits')
      .select('remaining')
      .eq('user_id', userId)
      .single()

    if (!creditData || creditData.remaining < result.credits_used) {
      return new Response(
        JSON.stringify({ error: 'Insufficient credits' }),
        { status: 402, headers: { ...corsHeaders, 'Content-Type': 'application/json' }}
      )
    }

    if (result.credits_used > 0) {
      await supabase.rpc('deduct_credits', {
        p_user_id: userId,
        p_action_type: mode === 'research' ? 'research' : mode === 'code' ? 'ask' : 'general',
        p_model_used: model,
        p_credits_used: result.credits_used,
        p_metadata: { 
          provider: AI_CONFIG.provider,
          tokens: result.total_tokens,
          mode: mode
        }
      })
    }

    return new Response(
      JSON.stringify({
        message: result.message,
        actions: result.actions,
        credits_used: result.credits_used,
        done: result.done,
        provider: AI_CONFIG.provider
      }),
      { headers: { ...corsHeaders, 'Content-Type': 'application/json' }}
    )

  } catch (error) {
    console.error('AI function error:', error)
    return new Response(
      JSON.stringify({ error: 'Internal server error', details: error.message }),
      { status: 500, headers: { ...corsHeaders, 'Content-Type': 'application/json' }}
    )
  }
})

// ═══════════════════════════════════════════════════════════════════════════
// MODE EXECUTION
// ═══════════════════════════════════════════════════════════════════════════

async function executeMode(conversation: any[], modelTier: string, mode: string) {
  const provider = AI_CONFIG.provider
  const models = AI_CONFIG.models[provider]
  const creditRates = AI_CONFIG.credit_rates[provider]
  
  const selectedModel = models[modelTier] || models.smart
  
  // ✅ Get tools based on mode (or NO tools for general)
  const tools = mode === 'general' ? null : 
                mode === 'research' ? getResearchTools() : 
                getCodeTools()

  // ✅ Get system prompt based on mode
  const systemPrompt = getSystemPrompt(mode)
  
  const messages = [
    { role: "system", content: systemPrompt },
    ...conversation
  ]

  let totalTokens = 0
  let maxRetries = 3
  let retryCount = 0
  
  while (retryCount < maxRetries) {
    try {
      const response = await callAI(messages, selectedModel, tools)
      
      totalTokens += (response.usage?.input_tokens || 0) + (response.usage?.output_tokens || 0)

      const actions = []
      let message = ""

      for (const block of response.content) {
        if (block.type === "text") {
          message += block.text
        } else if (block.type === "tool_use") {
          actions.push({
            type: block.name,
            ...block.input
          })
        }
      }

      const creditsUsed = Math.max(1, Math.ceil(totalTokens / 10000) * creditRates[modelTier])

      return {
        message: message,
        actions: actions,
        credits_used: creditsUsed,
        done: response.stop_reason === "end_turn" && actions.length === 0,
        total_tokens: totalTokens
      }
      
    } catch (error) {
      retryCount++
      
      // ✅ If Groq tool-calling fails, retry WITHOUT tools
      if (error.message.includes('tool_use_failed') && tools) {
        console.log(`Tool use failed, retrying without tools (attempt ${retryCount}/${maxRetries})`)
        
        // Retry with NO tools - just get text response
        const response = await callAI(messages, selectedModel, null)
        totalTokens += (response.usage?.input_tokens || 0) + (response.usage?.output_tokens || 0)
        
        let message = ""
        for (const block of response.content) {
          if (block.type === "text") {
            message += block.text
          }
        }
        
        const creditsUsed = Math.max(1, Math.ceil(totalTokens / 10000) * creditRates[modelTier])
        
        return {
          message: message,
          actions: [],
          credits_used: creditsUsed,
          done: true,
          total_tokens: totalTokens
        }
      }
      
      if (retryCount >= maxRetries) {
        throw error
      }
    }
  }
  
  throw new Error('Max retries exceeded')
}

// ═══════════════════════════════════════════════════════════════════════════
// SYSTEM PROMPTS
// ═══════════════════════════════════════════════════════════════════════════

function getSystemPrompt(mode: string): string {
  if (mode === 'general') {
    return `You are a helpful AI assistant for general questions and conversations.

Provide clear, accurate, and helpful responses to user questions.
Explain complex topics in an understandable way.
Be conversational and friendly.`
  }
  
  if (mode === 'research') {
    return `You are an ML research assistant with tools to help with experiments and analysis.

IMPORTANT: Use tools when appropriate:
- load_dataset: Load data files
- analyze_data: Run analysis
- train_model: Train models
- evaluate_model: Evaluate performance
- visualize: Create plots
- execute_command: Run commands

Be methodical and scientific in your approach.`
  }
  
  // mode === 'code'
  return `You are a coding assistant. When the user asks you to create or modify files, you MUST use the write_file tool.

CRITICAL RULES:
1. To CREATE a file: Use write_file with the full file content
2. To MODIFY a file: First use read_file, then write_file with updated content
3. NEVER just describe code - always create actual files
4. Include complete, working code in write_file calls

Available tools:
- list_files: See what files exist (use ONCE at start)
- read_file: Read existing files before modifying
- write_file: Create or update files (REQUIRED for coding tasks)
- execute_command: Run commands

Example:
User: "create a calculator app"
Correct: [Use write_file with complete calculator code]
Wrong: "Here's how to make a calculator..." [NO FILE CREATED]

ALWAYS CREATE FILES, NOT TEXT EXPLANATIONS.`
}

// ═══════════════════════════════════════════════════════════════════════════
// AI CALLERS
// ═══════════════════════════════════════════════════════════════════════════

async function callAI(messages: any[], model: string, tools?: any[]) {
  const provider = AI_CONFIG.provider
  
  switch (provider) {
    case 'anthropic':
      return await callAnthropic(messages, model, tools)
    case 'openrouter':
      return await callOpenRouter(messages, model, tools)
    case 'openai':
      return await callOpenAI(messages, model, tools)
    case 'groq':
      return await callGroq(messages, model, tools)
    default:
      throw new Error(`Unknown provider: ${provider}`)
  }
}

async function callAnthropic(messages: any[], model: string, tools?: any[]) {
  const systemMessage = messages.find(m => m.role === 'system')
  const conversationMessages = messages.filter(m => m.role !== 'system')
  
  const body: any = {
    model: model,
    max_tokens: 4096,
    messages: conversationMessages
  }
  
  if (systemMessage) {
    body.system = systemMessage.content
  }
  
  if (tools && tools.length > 0) {
    body.tools = tools
  }
  
  const response = await fetch(AI_CONFIG.endpoints.anthropic, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-api-key': AI_CONFIG.anthropic_key!,
      'anthropic-version': '2023-06-01'
    },
    body: JSON.stringify(body)
  })
  
  if (!response.ok) {
    const error = await response.text()
    throw new Error(`Anthropic API error: ${error}`)
  }
  
  const data = await response.json()
  
  return {
    content: data.content,
    usage: data.usage,
    stop_reason: data.stop_reason
  }
}

async function callGroq(messages: any[], model: string, tools?: any[]) {
  const openAIMessages = messages.map(msg => ({
    role: msg.role,
    content: msg.content
  }))
  
  const body: any = {
    model: model,
    messages: openAIMessages,
    max_tokens: 4096,
    temperature: 0.5
  }
  
  // ✅ FIX: Only add tools if provided AND model supports it
  if (tools && tools.length > 0) {
    body.tools = tools.map(tool => ({
      type: "function",
      function: {
        name: tool.name,
        description: tool.description,
        parameters: tool.input_schema
      }
    }))
    // ✅ DON'T force tool usage - let model decide
    // body.tool_choice = "auto" 
  }
  
  const response = await fetch(AI_CONFIG.endpoints.groq, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${AI_CONFIG.groq_key}`
    },
    body: JSON.stringify(body)
  })
  
  if (!response.ok) {
    const error = await response.text()
    throw new Error(`Groq API error: ${error}`)
  }
  
  const data = await response.json()
  const choice = data.choices[0]
  const content = []
  
  if (choice.message.content) {
    content.push({ type: "text", text: choice.message.content })
  }
  
  if (choice.message.tool_calls) {
    for (const toolCall of choice.message.tool_calls) {
      try {
        content.push({
          type: "tool_use",
          name: toolCall.function.name,
          input: JSON.parse(toolCall.function.arguments)
        })
      } catch (e) {
        console.error('Failed to parse tool arguments:', e)
      }
    }
  }
  
  return {
    content: content,
    usage: {
      input_tokens: data.usage?.prompt_tokens || 0,
      output_tokens: data.usage?.completion_tokens || 0
    },
    stop_reason: choice.finish_reason === 'stop' ? 'end_turn' : 'tool_calls'
  }
}

async function callOpenRouter(messages: any[], model: string, tools?: any[]) {
  const openAIMessages = messages.map(msg => ({
    role: msg.role,
    content: msg.content
  }))
  
  const body: any = {
    model: model,
    messages: openAIMessages,
    max_tokens: 4096
  }
  
  if (tools && tools.length > 0) {
    body.tools = tools.map(tool => ({
      type: "function",
      function: {
        name: tool.name,
        description: tool.description,
        parameters: tool.input_schema
      }
    }))
  }
  
  const response = await fetch(AI_CONFIG.endpoints.openrouter, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${AI_CONFIG.openrouter_key}`,
      'HTTP-Referer': 'https://keke.ai',
      'X-Title': 'Keke AI Terminal'
    },
    body: JSON.stringify(body)
  })
  
  if (!response.ok) {
    const error = await response.text()
    throw new Error(`OpenRouter API error: ${error}`)
  }
  
  const data = await response.json()
  const choice = data.choices[0]
  const content = []
  
  if (choice.message.content) {
    content.push({ type: "text", text: choice.message.content })
  }
  
  if (choice.message.tool_calls) {
    for (const toolCall of choice.message.tool_calls) {
      content.push({
        type: "tool_use",
        name: toolCall.function.name,
        input: JSON.parse(toolCall.function.arguments)
      })
    }
  }
  
  return {
    content: content,
    usage: {
      input_tokens: data.usage?.prompt_tokens || 0,
      output_tokens: data.usage?.completion_tokens || 0
    },
    stop_reason: choice.finish_reason === 'stop' ? 'end_turn' : 'tool_calls'
  }
}

async function callOpenAI(messages: any[], model: string, tools?: any[]) {
  const openAIMessages = messages.map(msg => ({
    role: msg.role,
    content: msg.content
  }))
  
  const body: any = {
    model: model,
    messages: openAIMessages,
    max_tokens: 4096
  }
  
  if (tools && tools.length > 0) {
    body.tools = tools.map(tool => ({
      type: "function",
      function: {
        name: tool.name,
        description: tool.description,
        parameters: tool.input_schema
      }
    }))
  }
  
  const response = await fetch(AI_CONFIG.endpoints.openai, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${AI_CONFIG.openai_key}`
    },
    body: JSON.stringify(body)
  })
  
  if (!response.ok) {
    const error = await response.text()
    throw new Error(`OpenAI API error: ${error}`)
  }
  
  const data = await response.json()
  const choice = data.choices[0]
  const content = []
  
  if (choice.message.content) {
    content.push({ type: "text", text: choice.message.content })
  }
  
  if (choice.message.tool_calls) {
    for (const toolCall of choice.message.tool_calls) {
      content.push({
        type: "tool_use",
        name: toolCall.function.name,
        input: JSON.parse(toolCall.function.arguments)
      })
    }
  }
  
  return {
    content: content,
    usage: {
      input_tokens: data.usage?.prompt_tokens || 0,
      output_tokens: data.usage?.completion_tokens || 0
    },
    stop_reason: choice.finish_reason === 'stop' ? 'end_turn' : 'tool_calls'
  }
}

// ═══════════════════════════════════════════════════════════════════════════
// TOOL DEFINITIONS
// ═══════════════════════════════════════════════════════════════════════════

function getCodeTools() {
  return [
    {
      name: "list_files",
      description: "List all files in the project. Use ONCE at the start to see what exists.",
      input_schema: {
        type: "object",
        properties: { 
          path: { type: "string", description: "Directory path (use '.' for current)" } 
        },
        required: ["path"]
      }
    },
    {
      name: "read_file",
      description: "Read contents of an existing file before modifying it.",
      input_schema: {
        type: "object",
        properties: { 
          path: { type: "string", description: "File path" } 
        },
        required: ["path"]
      }
    },
    {
      name: "write_file",
      description: "Create or replace a file with COMPLETE content. Must include ALL code.",
      input_schema: {
        type: "object",
        properties: {
          path: { type: "string", description: "File path (e.g. 'app.py')" },
          content: { type: "string", description: "FULL file content" }
        },
        required: ["path", "content"]
      }
    },
    {
      name: "execute_command",
      description: "Run a shell command",
      input_schema: {
        type: "object",
        properties: { 
          command: { type: "string", description: "Command to run" } 
        },
        required: ["command"]
      }
    }
  ]
}

function getResearchTools() {
  return [
    {
      name: "load_dataset",
      description: "Load a dataset file",
      input_schema: {
        type: "object",
        properties: {
          path: { type: "string" },
          format: { type: "string", enum: ["csv", "parquet", "numpy"] }
        },
        required: ["path"]
      }
    },
    {
      name: "analyze_data",
      description: "Run statistical analysis",
      input_schema: {
        type: "object",
        properties: {
          analysis_type: { type: "string" },
          parameters: { type: "object" }
        },
        required: ["analysis_type"]
      }
    },
    {
      name: "train_model",
      description: "Train ML model",
      input_schema: {
        type: "object",
        properties: {
          model_type: { type: "string" },
          config: { type: "object" }
        },
        required: ["model_type"]
      }
    },
    {
      name: "evaluate_model",
      description: "Evaluate model",
      input_schema: {
        type: "object",
        properties: {
          model_path: { type: "string" },
          metrics: { type: "array", items: { type: "string" } }
        },
        required: ["model_path"]
      }
    },
    {
      name: "visualize",
      description: "Create visualization",
      input_schema: {
        type: "object",
        properties: {
          viz_type: { type: "string" },
          data: { type: "object" }
        },
        required: ["viz_type"]
      }
    },
    {
      name: "execute_command",
      description: "Run command",
      input_schema: {
        type: "object",
        properties: { command: { type: "string" } },
        required: ["command"]
      }
    }
  ]
}
