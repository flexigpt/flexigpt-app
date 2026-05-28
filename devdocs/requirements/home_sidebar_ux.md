# Home/Sidebar

- Need better pins in home
- Examples and thoughts

```text
──────────────────── (Top static)
🏠 Home                -> Landing page, Recent activity, Dashboards
💬 Chat                -> Chat UI, Conversation lists

──────────────────── (Mid dynamic, Min 8px spacer above)

🟦 Apps                -> Grid & marketplace of installable apps
🗒️ AI-Notepad          -> Example pinned app
🖼️ Image-Gen           -> Example pinned app
// max 5 pinned-app icons, drag to rearrange list

──────────────────── (Mid dynamic, Min 8px spacer below)

──────────────────── (Bottom static)
// May be we can have "Assistants" in place of skills too and all the below are ways to create assistants
🧩 Skills             -> Build & edit: (Below tabs in a expanded drawer).
                        1. Prompts
                        2. Tools
                        3. Model presets
                        4. Data/Doc Sources
                        5. Assistants is a preset of things from above 4 things.

📊 Insights           -> Usage, cost, performance dashboards
❓ Help               -> Docs, tutorials, support
⚙️👤 Account           -> Manage: (Below tabs in a expanded drawer)
                        1. Profile/Workspace
                        2. Billing
                        // May combine 3 and 4 if required, depends on density of info in each
                        3. App preferences: Themes, shortcuts, etc.
                        4. Security & Keys.
```

```mermaid
graph TD
%% ───────────────────────────────
%% 1. MAIN SIDEBAR NAVIGATION
%% ───────────────────────────────
home[🏠 Home]
chat[💬 Chat]
apps[🟦 Apps]
insights[📊 Insights]
help[❓ Help]
account[⚙️👤 Account]

%% sidebar order (dashed to show UI order, not data-flow)
home -.-> chat
chat -.-> apps
apps -.-> skillsSection
skillsSection -.-> insights
insights -.-> help
help -.-> account


%% ───────────────────────────────
%% 2. PINNED / MARKETPLACE APPS
%% ───────────────────────────────
aiNotepad["🗒️ AI-Notepad"]
imageGen["🖼️ Image-Gen"]

apps --> aiNotepad
apps --> imageGen


%% ───────────────────────────────
%% 3. SKILLS / ASSISTANTS AREA
%% ───────────────────────────────
subgraph skillsSection["🧩 Skills / Assistants"]
  prompts["Prompt Templates"]
  tools["Tools"]
  modelPresets["Model Presets"]
  dataSources["Data / Doc Sources"]
  assistants["Assistants<br/>(Agent Presets)"]
end

prompts --> assistants
tools --> assistants
modelPresets --> assistants
dataSources --> assistants


%% ───────────────────────────────
%% 4. CHAT SESSION RELATION
%% ───────────────────────────────
chatSession["ChatSession<br/>(loads Persona)"]
assistants -->|persona loader| chatSession


%% ───────────────────────────────
%% 5. ACCOUNT DRAWER
%% ───────────────────────────────
subgraph accountDetails["Account Sections"]
  profile["Profile / Workspace"]
  billing["Billing"]
  prefs["App Preferences"]
  security["Security & Keys"]
end

account --> profile
account --> billing
account --> prefs
account --> security
```
