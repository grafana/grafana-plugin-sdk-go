package mcp

// Prompt is a registerable MCP prompt template.
type Prompt struct {
	Name        string
	Description string
	// Template is the literal prompt text. Argument substitution is the
	// caller's responsibility for v1.
	Template string
	// Arguments declares any prompt arguments (name, description, required).
	Arguments []PromptArgument
}

type PromptArgument struct {
	Name        string
	Description string
	Required    bool
}

func (s *Server) RegisterPrompt(p Prompt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.prompts {
		if existing.Name == p.Name {
			s.prompts[i] = p
			return
		}
	}
	s.prompts = append(s.prompts, p)
}
