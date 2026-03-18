# AI Prompt Engineering for ZetGrep

Since `ZetGrep` is a new and specialized tool, AI models (like Gemini, GPT, or Claude) may not have it in their training data yet. To get the best results, you should provide the AI with a "Knowledge Bootstrap" before asking for configurations.

---

## 🧠 Step 1: The Knowledge Bootstrap
**Copy and paste this "Context Block" into your AI chat first. This teaches the AI what ZetGrep is.**

> "I am working with a tool called `ZetGrep`. It is a high-performance regex orchestration engine written in Go. It has two main configuration formats: 
> 
> 1. **Tool Config (YAML)**: Defines how to process a regex match. Fields: `id`, `name`, `description`, `extract` (optional regex), `command` (bash command), and `field` (output label). It uses `{{match}}`, `{{file}}`, and `{{match[1]}}` as placeholders.
> 
> 2. **Input Config (YAML)**: Defines how to parse JSONL files. Fields: `format` (always 'jsonl'), `target` (field to scan), `id` (field to use as source name), and `decode` (boolean).
> 
> Do you understand this format? I will now ask you to create configurations for me."

---

## 🛠️ Step 2: The Action Prompts

### For a Custom Tool (Plugin)
> "Based on the `ZetGrep` format I provided, create a tool YAML that [DESCRIBE YOUR VISION]. 
> Example: 'Create a tool that takes an IP address match and runs a geo-lookup using the `geoiplookup` command.'"

### For a JSONL Input Configuration
> "I have a JSONL log file from a custom scanner. One line looks like this: [PASTE SAMPLE LINE]. 
> Create a `ZetGrep` input config YAML that scans the 'response_body' and uses 'request_id' as the identifier."

### For a Complex Extraction Tool
> "Create a `ZetGrep` tool that extracts only the domain from a URL match using a regex in the `extract` field, and then runs `host` on that domain."

---

## 💎 Pro-Tip: Multi-Tool Orchestration
You can even ask the AI to design a whole workflow:

> "I want to scan a 40GB JSONL dump for Base64 strings, decode them, and if the decoded string looks like a JSON object, I want to extract the 'admin' field. Give me the Pattern regex, the Input Config, and the Tool YAML needed for this ZetGrep workflow."

---

## 📝 Why this works
By providing the **Knowledge Bootstrap**, you turn the AI into a `ZetGrep` expert instantly. It will ensure that the `command` uses the correct `{{variable}}` syntax and that the YAML structure is exactly what the `zetgrep` binary expects.
