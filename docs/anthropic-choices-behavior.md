# Anthropic Choices and Message Serialization in go-langchain

## Overview
Anthropic's response structure and go-langchain's serialization behavior require special handling when building multi-turn conversations with tool use.

## Response Structure (Anthropic → go-langchain)

Anthropic API returns responses as an array of **content blocks**:
```
[text_block, tool_use_block, tool_use_block, ...]
```

go-langchain converts each content block into a **separate ContentChoice**:
- `type: "text"` → `ContentChoice{Content: "...", ToolCalls: []}`
- `type: "tool_use"` → `ContentChoice{Content: "", ToolCalls: [{...}]}`
- `type: "thinking"` → `ContentChoice{Content: "", GenerationInfo: {...}}`

**Key insight:** One Anthropic response can produce multiple Choices. For example:
- Response with text + 2 tool calls → 3 Choices
- Response with just text → 1 Choice

## Serialization Constraint (go-langchain → Anthropic)

The critical limitation is in `handleAIMessage()`:
```go
if toolCall, ok := msg.Parts[0].(llms.ToolCall); ok {
    // Only Parts[0] is serialized!
}
```

**This means:**
- Only `Parts[0]` of a MessageContent is serialized back to Anthropic
- If you create `MessageContent{Parts: [toolCall1, toolCall2]}`, only `toolCall1` is sent
- Multiple ToolCalls in one message **will lose data**

## Required Pattern: Interleaved Messages

To work around this limitation, tool calls must be **interleaved** as separate messages:

```
AI message: Parts[toolCall1]
Tool message: Parts[toolResult1]
AI message: Parts[toolCall2]
Tool message: Parts[toolResult2]
```

Not:
```
AI message: Parts[toolCall1, toolCall2]  // toolCall2 would be lost!
Tool message: Parts[toolResult1, toolResult2]
```

## Why Merging Choices is Necessary

When processing Anthropic's response:
1. Anthropic returns separate content blocks (potentially text + multiple tools)
2. go-langchain creates one Choice per block
3. We must merge these Choices to get the complete response
4. Then we must split them back into individual AI messages for serialization

The merge preserves all information for processing, but the split ensures proper serialization.

## Implementation Details in agentic_client.go

The choice-merging code performs this merge:
- Collects all content parts from separate Choices
- Collects all ToolCalls from separate Choices
- Creates one merged view for processing

Then later in the tool call processing, it **reverses** this by creating one AI message per ToolCall to avoid the serialization bug.
