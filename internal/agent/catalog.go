package agent

// Runtime describes an agent CLI that an execution host can provide.
type Runtime struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Install string `json:"install"`
	Docs    string `json:"docs"`
	Command string `json:"command,omitempty"`
	ACP     bool   `json:"acp"`
}

type ACPBackendSpec struct {
	Command string
	Args    []string
	Env     []string
}

// DefaultACPBackends maps agent backends to their ACP launcher.
var DefaultACPBackends = map[string]ACPBackendSpec{
	"claude":   {Command: "claude-agent-acp"},
	"codex":    {Command: "codex-acp"},
	"deepseek": {Command: "dsh", Args: []string{"acp"}},
	"gemini":   {Command: "gemini", Args: []string{"--experimental-acp"}},
	"cursor":   {Command: "cursor-agent", Args: []string{"acp"}},
	"qwen":     {Command: "qwen", Args: []string{"--acp", "--experimental-skills"}},
	"opencode": {Command: "opencode", Args: []string{"acp"}},
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
		ID: "deepseek", Name: "DeepSeek Harness",
		Command: "dsh", Install: "npm install -g @deepseek-ai/dsh",
		Docs: "https://github.com/deepseek-ai/deepseek-harness", ACP: true,
	},
	{
		ID: "gemini", Name: "Gemini CLI",
		Install: "npm install -g @google/gemini-cli",
		Docs:    "https://github.com/google-gemini/gemini-cli", ACP: true,
	},
	{
		ID: "cursor", Name: "Cursor Agent",
		Command: "cursor-agent", Install: "curl https://cursor.com/install -fsS | bash",
		Docs: "https://cursor.com/docs/cli/acp", ACP: true,
	},
	{
		ID: "qwen", Name: "Qwen Code",
		Install: "npm install -g @qwen-code/qwen-code",
		Docs:    "https://github.com/QwenLM/qwen-code", ACP: true,
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
