package llmprovider

import "github.com/tmc/langchaingo/llms"

// This file provides conversion between llmprovider types and langchain
// llms types. It exists only for the transition period while we migrate
// providers one at a time. Once all providers use llmprovider types
// directly, this file can be deleted.

// --- Role conversion ---

func RoleFromLangchain(r llms.ChatMessageType) Role {
	switch r {
	case llms.ChatMessageTypeSystem:
		return RoleSystem
	case llms.ChatMessageTypeHuman:
		return RoleHuman
	case llms.ChatMessageTypeAI:
		return RoleAI
	case llms.ChatMessageTypeTool:
		return RoleTool
	default:
		return RoleHuman
	}
}

func (r Role) ToLangchain() llms.ChatMessageType {
	switch r {
	case RoleSystem:
		return llms.ChatMessageTypeSystem
	case RoleHuman:
		return llms.ChatMessageTypeHuman
	case RoleAI:
		return llms.ChatMessageTypeAI
	case RoleTool:
		return llms.ChatMessageTypeTool
	default:
		return llms.ChatMessageTypeHuman
	}
}

// --- Message conversion ---

func MessageFromLangchain(lc llms.MessageContent) Message {
	msg := Message{
		Role:  RoleFromLangchain(lc.Role),
		Parts: make([]Part, 0, len(lc.Parts)),
	}
	for _, p := range lc.Parts {
		msg.Parts = append(msg.Parts, PartFromLangchain(p))
	}
	return msg
}

func MessagesFromLangchain(lc []llms.MessageContent) []Message {
	msgs := make([]Message, len(lc))
	for i, m := range lc {
		msgs[i] = MessageFromLangchain(m)
	}
	return msgs
}

func (m Message) ToLangchain() llms.MessageContent {
	lc := llms.MessageContent{
		Role:  m.Role.ToLangchain(),
		Parts: make([]llms.ContentPart, 0, len(m.Parts)),
	}
	for _, p := range m.Parts {
		lc.Parts = append(lc.Parts, PartToLangchain(p))
	}
	return lc
}

func MessagesToLangchain(msgs []Message) []llms.MessageContent {
	lc := make([]llms.MessageContent, len(msgs))
	for i, m := range msgs {
		lc[i] = m.ToLangchain()
	}
	return lc
}

// --- Part conversion ---

func PartFromLangchain(p llms.ContentPart) Part {
	switch v := p.(type) {
	case llms.TextContent:
		return TextPart{Text: v.Text}
	case llms.ToolCall:
		tc := ToolCallPart{
			ID:   v.ID,
			Name: v.FunctionCall.Name,
		}
		if v.FunctionCall != nil {
			tc.Arguments = v.FunctionCall.Arguments
		}
		return tc
	case llms.ToolCallResponse:
		return ToolResultPart{
			ToolCallID: v.ToolCallID,
			Name:       v.Name,
			Content:    v.Content,
		}
	default:
		// Unsupported part types become empty text
		return TextPart{}
	}
}

func PartToLangchain(p Part) llms.ContentPart {
	switch v := p.(type) {
	case TextPart:
		return llms.TextContent{Text: v.Text}
	case ToolCallPart:
		return llms.ToolCall{
			ID:   v.ID,
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      v.Name,
				Arguments: v.Arguments,
			},
		}
	case ToolResultPart:
		return llms.ToolCallResponse{
			ToolCallID: v.ToolCallID,
			Name:       v.Name,
			Content:    v.Content,
		}
	case ThinkingPart:
		// Langchain has no equivalent; drop thinking parts
		return llms.TextContent{}
	default:
		return llms.TextContent{}
	}
}

// --- Response conversion ---

func ResponseFromLangchain(lc *llms.ContentResponse) *Response {
	if lc == nil {
		return nil
	}
	resp := &Response{
		Choices: make([]*Choice, len(lc.Choices)),
	}
	for i, c := range lc.Choices {
		resp.Choices[i] = ChoiceFromLangchain(c)
	}
	return resp
}

func ChoiceFromLangchain(lc *llms.ContentChoice) *Choice {
	if lc == nil {
		return nil
	}
	ch := &Choice{
		Content:        lc.Content,
		StopReason:     lc.StopReason,
		GenerationInfo: lc.GenerationInfo,
	}
	for _, tc := range lc.ToolCalls {
		ch.ToolCalls = append(ch.ToolCalls, ToolCallPart{
			ID:   tc.ID,
			Name: tc.FunctionCall.Name,
			Arguments: func() string {
				if tc.FunctionCall != nil {
					return tc.FunctionCall.Arguments
				}
				return ""
			}(),
		})
	}
	return ch
}

func (r *Response) ToLangchain() *llms.ContentResponse {
	if r == nil {
		return nil
	}
	lc := &llms.ContentResponse{
		Choices: make([]*llms.ContentChoice, len(r.Choices)),
	}
	for i, c := range r.Choices {
		lc.Choices[i] = c.ToLangchain()
	}
	return lc
}

func (c *Choice) ToLangchain() *llms.ContentChoice {
	if c == nil {
		return nil
	}
	lc := &llms.ContentChoice{
		Content:        c.Content,
		StopReason:     c.StopReason,
		GenerationInfo: c.GenerationInfo,
	}
	for _, tc := range c.ToolCalls {
		lc.ToolCalls = append(lc.ToolCalls, llms.ToolCall{
			ID:   tc.ID,
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      tc.Name,
				Arguments: tc.Arguments,
			},
		})
	}
	return lc
}

// --- Tool conversion ---

func ToolFromLangchain(t llms.Tool) Tool {
	tool := Tool{Type: t.Type}
	if t.Function != nil {
		tool.Function = &FunctionDef{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  t.Function.Parameters,
		}
	}
	return tool
}

func ToolsFromLangchain(tools []llms.Tool) []Tool {
	out := make([]Tool, len(tools))
	for i, t := range tools {
		out[i] = ToolFromLangchain(t)
	}
	return out
}

func (t Tool) ToLangchain() llms.Tool {
	lc := llms.Tool{Type: t.Type}
	if t.Function != nil {
		lc.Function = &llms.FunctionDefinition{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  t.Function.Parameters,
		}
	}
	return lc
}

func ToolsToLangchain(tools []Tool) []llms.Tool {
	out := make([]llms.Tool, len(tools))
	for i, t := range tools {
		out[i] = t.ToLangchain()
	}
	return out
}
