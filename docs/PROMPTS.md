# AI Prompt Templates for Configuration

Since `zetgrep` uses standardized YAML formats, you can use AI to quickly generate complex scanning and orchestration logic. Copy and paste these prompts into your favorite LLM.

---

## 🛠️ Prompt for Custom Tools
**Use this when you want to build a new post-processing plugin.**

> "I am using the `zetgrep` pattern matching tool. I need a tool YAML configuration that does the following: [DESCRIBE YOUR VISION, e.g., 'Takes a URL match, runs a curl request to check for a specific header, and extracts the Server version']. 
>
> Please provide the output in this format:
> id: [unique_id]
> name: [Friendly Name]
> description: [Purpose]
> command: [The Linux command, use {{match}} for the input]
> field: [The label for the result]"

---

## 📦 Prompt for JSONL Input Configs
**Use this when you have a unique data format from a custom scanner.**

> "I have a JSONL file where each line looks like this: [PASTE ONE LINE OF YOUR DATA]. 
>
> I need a `zetgrep` input configuration YAML that scans the [SPECIFY FIELD, e.g., 'raw_response'] field using the [SPECIFY ID, e.g., 'domain_name'] field as the identifier. 
>
> Please provide the output in this format:
> format: jsonl
> target: [target_field]
> id: [id_field]
> decode: [true/false]"

---

## 💎 Prompt for Advanced Output Templates
**Use this to design a professional report or dashboard view.**

> "I am running a `zetgrep` scan with the following tools active: [LIST TOOLS]. I want a custom output template for the `-o` flag that creates a [DESCRIBE STYLE, e.g., 'Clean, pipe-delimited security log'].
>
> Available variables are {{pattern}}, {{file}}, {{match}}, and {{tool:ID}}. Please provide the string for the -o flag."

---

## 🧠 Tips for "Max Power" Configurations
- **Chaining**: Tell the AI: "Create a command that pipes the output of `base64 -d` into `grep` and then into `awk`."
- **Regex Extraction**: Tell the AI: "Include an `extract` regex in the tool config that specifically pulls out the UUID from a long string."
