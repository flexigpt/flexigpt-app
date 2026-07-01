# Observations

- openai tool loop. openai tool calls does a deep dive a lot, for flexigpt, it goes from backend to frontend to others.
  - [ ] i think there needs to be a way to control its spread.
  - [ ] most probably need to have a way to say that do step by step, and identify domains and then do things etc.

- [ ] in a tooling session, there are no parallel tool calls being made for some reason.
  - [ ] better force the caller to say that do a bfs kind of tool calls.
- [ ] running linters tests etc is shell driven, the agents.md or similar file may help to say what are commands.
  - [ ] for task file it may be simpler, but ned a way of discovery and exec.
- [ ] asking for things that are missing should be encouraged in some ways.
  - [ ] most probably llms are now more in "agentic" mode so they dont really ask questions, but we can shine a lot with human in loops rather than direct.

- [ ] once nice thing i noticed that may or may not be available in others, in flexigpt, i can remove/add tools if there are errors etc. and llm may work with base tools i.e read/write/delete files are unambiguous

- [ ] in skills world, especially with artifact driven dev, we can almost get away with without sending a lot of things again and again, latest context and last 1/2 chats. attachments may pose a problem here, but can be thought through properly so that these things are persisted properly.

- [ ] lots of persona like templates can be added for sys prompt templates.

## Model analysis

- [ ] model providers with dedicated platforms:
  - [ ] openai -> best use responses api -> done.
  - [ ] anthropic -> use messages api -> done.
  - [ ] google gemini -> use generate content api -> done.
  - [ ] xai -> responses api, 4.x doesnt support thinking as such, "inbuilt" thinking is what they say, so no control
  - [ ] mistral -> chat completions compatible but looks a bit different. check cleanly again.

  - [ ] cohere -> non sota inference
  - [ ] meta api -> chat completions, but getting api keys is in closed beta as of now.

- [ ] Aggregate API endpoint providers:
  - [ ] HF -> responses beta, completions stable
  - [ ] Openrouter -> responses beta, completions stable

- [ ] Aggregate Cloud services with AI deployments
  - [ ] Amazon -> (non-sota models exist) -> needs aws specific converse/conversestream sdk intergation.
  - [ ] Azure -> (non-sota models exist) -> more chat completions like. needs a adapter.
  - [ ] nvidia nim/dgx cloud -> (non-sota models exist) - ??
  - [ ] Oracle -> ??

- [ ] Chinese model providers with dedicated platforms: GLM (z ai), minimax, qwen (alibaba), kimi (moonshot ai), deepseek, yi models (01.ai), StepFun
  - [ ] some specialize dedicated platforms for these have very slow access/login.
  - [ ] data mostly sent to chinese infra and can be accessed by chinese govt.
  - [ ] most privacy policy dictates that data will be used for training.
  - [ ] better access models via openrouter/hf/other aggregators as needed

- [ ] Local providers:
  - [ ] llama.cpp -> chat completions
  - [ ] LMStudio, Ollama -> anthropic messages, responses, chat completions, all are supported.
  - [ ] vLLM -> responses, chat completions
  - [ ] GPT4All -> chat completions

## MCP analysis

brave search
figma

| Atlassian | Software Development | `https://mcp.atlassian.com/v1/sse` | OAuth2.1 🔐 | [Atlassian](https://atlassian.com) |
| AWS Knowledge | Software Development | `https://knowledge-mcp.global.api.aws` | Open | [AWS](https://aws.github.io/) |
| Box | Document Management | `https://mcp.box.com` | OAuth2.1 🔐| [Box](https://box.com) |
| Canva | Design | `https://mcp.canva.com/mcp` | OAuth2.1 | [Canva](https://canva.com) |
| Cloudflare Workers | Software Development | `https://bindings.mcp.cloudflare.com/sse` | OAuth2.1 | [Cloudflare](https://cloudflare.com) |
| Cloudflare Observability | Observability | `https://observability.mcp.cloudflare.com/sse` | OAuth2.1 | [Cloudflare](https://cloudflare.com) |
| GitHub | Software Development | `https://api.githubcopilot.com/mcp` | OAuth2.1 🔐 | [GitHub](https://github.com) |
| Indeed | Job Board | `https://mcp.indeed.com/claude/mcp` | OAuth2.1 | [Indeed](https://indeed.com) |
| Netlify | Software Development | `https://netlify-mcp.netlify.app/mcp` | OAuth2.1 | [Netlify](https://netlify.com) |
| Notion | Project Management | `https://mcp.notion.com/sse` | OAuth2.1 | [Notion](https://notion.so) |
| Stack Overflow | Software Development | `https://mcp.stackoverflow.com` | OAuth2.1 | [StackOverflow](https://stackoverflow.com) |
| Vercel | Software Development | `https://mcp.vercel.com/` | OAuth2.1 | [Vercel](https://vercel.com) |
| Cloudflare Docs | Documentation | `https://docs.mcp.cloudflare.com/sse` | Open | [Cloudflare](https://cloudflare.com) |
| Exa Search | Search | `https://mcp.exa.ai/mcp` | Open | [Exa](https://exa.ai) |
| Hugging Face | Software Development | `https://hf.co/mcp` | Open | [Hugging Face](https://huggingface.co) |
| Google Big Query | Data Analysis | `https://bigquery.googleapis.com/mcp` | API Key | [Google](https://docs.cloud.google.com/bigquery/docs/reference/mcp) |
| Google Compute Engine | Developer Tools | `https://compute.googleapis.com/mcp` | API Key | [Google](https://docs.cloud.google.com/compute/docs/reference/mcp) |
| Google GKE | Developer Tools | `https://container.googleapis.com/mcp` | API Key | [Google](https://docs.cloud.google.com/kubernetes-engine/docs/reference/mcp) |
| Google Maps | Mapping | `https://mapstools.googleapis.com/mcp` | API Key | [Google](https://developers.google.com/maps/ai/grounding-lite/reference/mcp) |
[Other google mcps](https://github.com/google/mcp)

maybe:
| BGPT | Scientific Research | `https://bgpt.pro/mcp/sse` | Open / API Key | [BGPT](https://bgpt.pro/mcp) |
| Wix | CMS | `https://mcp.wix.com/sse` | OAuth2.1 | [Wix](https://wix.com) |
