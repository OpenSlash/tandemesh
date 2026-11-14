package agent

// Runtime describes an agent CLI that an execution host can provide.
type Runtime struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Install string `json:"install"`
	Docs    string `json:"docs"`
	ACP     bool   `json:"acp"`
}

// RuntimeCatalog lists the agent CLIs Orca knows how to detect.
var RuntimeCatalog = []Runtime{
	{
		ID: "claude", Name: "Claude Code",
		Install: "npm install -g @anthropic-ai/claude-code",
		Docs:    "https://docs.anthropic.com/en/docs/claude-code", ACP: true,
	},
	{
		ID: "codex", Name: "Codex CLI",
		Install: "npm install -g @openai/codex",
		Docs:    "https://github.com/openai/codex", ACP: true,
	},
	{
		ID: "opencode", Name: "OpenCode",
		Install: "npm install -g opencode-ai",
		Docs:    "https://opencode.ai/docs/", ACP: true,
	},
	{
		ID: "goose", Name: "Goose",
		Install: "brew install block-goose-cli",
		Docs:    "https://block.github.io/goose/", ACP: false,
	},
}
