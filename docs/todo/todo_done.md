# Project Done Items

- Design and implement AppLayout
  - [x] Manages overall layout structure

  - [x] Sidebar on the left
    - [x] Handles click events
    - [x] Settings icons at the bottom
    - [x] Should be hidden behind a hamburger on small screen. Always visible on large.
    - [x] Clicking icons should redirect to corresponding pages

- Theming
  - [x] Define and apply dark and light themes
  - [x] Applies consistent theming across all components

- Chat page
  - [x] a navbar with new chat button on left, export button on right, search in center
    - [x] Search should list conversations that match the search input. Also should behave like a autocomplete

  - [x] send message input field with a "send" button
    - [x] It should have "type message..." as hint
    - [x] send button should be enabled only when something is typed
    - [x] styled as a box with top as circular
    - [x] fixed at bottom of the parent container
    - [ ] Input box styling issues. should be centered on medium screens and small screens too.

  - [x] chat message list
    - [x] These are in the body, below navbar and above input
    - [x] is a card layout scrollable vertically and can have user and then assistant

  - [x] Each chat bubble should be a card
    - [x] It should have avatar/icon on left/right depending on the user
    - [x] Card body should support code with highlighting.
      - [x] Code block should have a header that has language on right, copy button on left.
      - [x] It should have scroll horizontal support.
    - [x] Footer:
      - [x] It should support copy, edit, button (depends on type of user) on left side.
      - [x] after it, there should be a "feedback" button. that should open a input text field with a send button.
        - [x] It should have a like dislike button on left side
        - [x] people can add free text feedback in input field
      - [x] Footer also should have a dropdown on right, which opens a code block on click, that is json sent of the request/response.

  - [ ] Advanced:
    - [ ] ~~Support add file, add image~~
    - [ ] ~~Support streaming response~~

- Electron support
  - [x] Support electron based bundling for desktop
  - [x] Test with prod build and run appimage.
  - [ ] ~~Add icon in title bar and ass appIcon for tray~~
  - [x] Add a desktop entry so that appimage can be registered with the system

- Settings screen
  - [x] A settings page is a multi card page.
  - [x] The settings page has a multiple cards and a "export settings" functionality
  - [x] Support all settings from vscode extension
  - [ ] ~~"import settings" functionality too~~
  - Card widget:
    - [x] Each card widget has a header i.e the settings name
    - [x] The body is a table of key value pairs
    - [x] The first row is a fixed frozen row with colum names i.e column1: key, column 2: value
    - [ ] ~~There are then rows of "predefined keys", these rows are not deleteable. Also the keys are not editable, only the values are.~~
    - [ ] ~~There should be capability to add and remove other rows.~~
    - [x] There is should be a capability to display a value as "hidden" i.e "\*\*\*", based on some input type (e.g: password or secret or apikeys)
    - [ ] ~~There should be a explicit "save settings" button too. That can have a dummy implementation for now. Ideally this should be in card footer.~~

- Settings State DB
  - [x] Create a state db that can store and retrieve settings
  - [x] Connect all above widgets to a background settings state lib (ideally json file based)
  - [ ] ~~The lib should provide CRUD functionality with some keys being immutable (for above "predefined keys")~~
  - [ ] ~~It should also provide and import export functionality. The export should be as a json file, same with import.~~

- LLM Provider
  - [x] Implement llm providers
  - [x] connect this with chat page

- Conversation State DB:
  - [x] Add a "conversations state db".
  - [x] It should provide a functionality of add a conversation, remove a conversation, list conversation titles, remove all conversations
  - [x] A conversation has a title, id (uuidv7), created time, modified time, list of messages
  - [x] For each conversation it should have a add message, list all messages interface.
  - [x] Each conversation should be saved as a "json file" in some predefined folder.
  - [x] Connect UI to this state db

- Conversation list
  - [x] Fetch conversation titles and add it to the drawer
  - [x] provide a new chat, delete one conversation, delete all conversations actions

- [x] code inline line overflow handle
- [x] feedback integration removal
- [x] sort order of drop down, during load
- [x] sort order of drop down, after new chat
- [x] title population
- [x] Full copy for assistant
- [x] Streaming
  - [x] aiprovider support
  - [x] page support
  - [x] add other providers
    - [x] Openai
    - [x] anthropic
    - [x] google
  - [x] someway to show things are loading
- [x] Content:
  - [x] Emoji
  - [ ] ~~Subscript~~
  - [ ] ~~custom headings~~
  - [ ] ~~navigate in same doc~~
  - [ ] ~~Math handling~~
- [x] packaged app debug
- [x] Move ai provider to electron main
- [x] screen blanks if lot of data is put in inputfield
- [x] selection doesnt reset on new conversation in search bar
- [x] today vs yesterday in group
- [x] logger is imported in securejsondb before it is set as ipc logger
- [x] error handling review and fix
- [x] Blanket error handle in api fetch chat helper
- [x] Add edit functionality
- [x] ability to discard previous message history in next call
- [x] ability to choose provider and model and temperature in the chat itself.
- [x] populate initial choices in input options
- [x] Take default model and temp from settings without caching
- [x] focus on edit box if someone presses edit. as of now focus is at end of page
- [x] In chat the last used model should be stored ideally so that next time you load the chat it loads that model by default
- [x] url in output and input should be clickable and copiable
- [x] Enable right click context menu
- [x] word wrap for single word if it is too long.
- [x] scroll handling for edit text area
- [x] Icon of user and assistant should be near bottom of card rather than at top
- [x] Mac install shows no title bar, better add a plain titlebar.
- [x] Mac: Add how to install non signed pkg in readme. click > ok > settings > privacy and security > blocked > open anyway
- [x] Mac icns file not present. Need proper icns file added
- [x] chat load is not happening before a request fire
- [x] if the app is on for a long time, looks like the conversation list is removed
- [x] there is some jerkiness in chat load for today

- Wails migrate
  - [x] Translate securejsondb to go
  - [x] Translate settings store and conversation store to go
  - [x] Move next frontend to appropriate folder and change config and build system as seen in wails template

  - [x] Transform next app to work with multiple backend provider like electron or wails.
    - [x] Involves segregating models and creating a facade to redirect to window apis that are injected by backend
    - [x] Ideally interface declaration should be present in frontend and all calls should be made to interface by components, the exact implementation should decide which backend is present and redirect there.
    - [x] wails main.go app integrations
    - [x] Logger integration

  - [x] Translate aiprovider to go. Try and use langchaingo directly
  - [x] Implement streaming.
  - [x] Integrate other AI providers: anthropic, google, hf

- [x] Add support for first getting available ai providers and then configuring them accordingly. UI needs to be modified as collapsed cards and disabled cards
- [x] Only enabled chat options should be available
- [x] Get most consts through API
- [x] AI provider: get masked http details from langchaingo whenever available and then send in details
- [ ] ~~Error wrapper and unwrapper on backend main and base apis~~
- [x] Whether to stream or not should be decided by model in conjunction with provider in frontend.
- [x] integrate file logger
- [x] Math formulas in content needs to render properly
- [x] Mermaid diagrams and sequence diagrams should render properly
- [x] Adapt download button to wails
- [x] ~~Need a resend button too~~
- [x] Cursor should move to text area after loading a chat or other chat actions
- [x] Scroll bug in input text area
- [x] load chats properly at new chat etc. handle caching as needed
  - [x] Wails app test doesnt show any delay. Need to test electron.
- [x] Flatpak installer for Linux
- [x] PKG installer for Mac
- [x] NSIS installer for Windows
- [x] Wails save file dialog opens and saves but file not seen on system
- [x] Remove electron
- [x] Remove nextjs and use react router v7
  - [x] migrate to react 19
  - [x] use vite
  - [x] migrate to eslint 9
  - [x] use spa mode
  - [x] remove postcss
  - [x] remove eslint, prettier, knip from top

- [x] hydrationfallback addition
- [x] use loaders to do client loading of pre dom loading content like settings
- [x] Buttons in nav bar changed styling during migration, rectify
- [x] font google local source
- [x] Need an about section someplace with version of software, dev name etc
  - [x] Added simple version in title bar for now.
- [x] For user message, do not render as markdown, render as is.
- [x] Support developer message for openai. Add Formatting re-enabled on first line of developer message in >o1 models for md output.

- [x] Add models as a configuration in providers where each model can be enabled disabled as required. Add a custom model entry support too.
  - [x] UI with card table and actions
  - [x] Modal for edit/add
  - [x] Backend integration
  - [x] Do not delete inbuilt models, just disable them.
  - [x] Input options temperature should come from either modelparams or modelinfo
  - [x] Link add model
  - [x] Add a custom provider and model name setting too so that an OAI compatible provider and model can be added
  - [ ] Add a new provider flow in providerSet
    - [x] engine delete end to end
    - [x] chat completion prefix end to end
    - [x] need promptlength and output as max in setting and param need to be similar in names, info need to be about "global defaults"
    - [x] setting provider attr for custom providers
      - [x] At load time need to init default + settings specific custom providers and models
      - [x] Better do it inside go rather than from frontend. may solve for api key thing too
    - [x] Need a dynamic time key encoder decoder for secrets
    - [x] Dont allow disabling default model
    - [x] atleast one model should be enabled
    - [x] default should have only enabled models
    - [x] reasoning models dont have tick as of now
    - [x] add buttons should be disabled until add reqs are met.
  - [x] Chat options is showing the new default model as default + the actual default model as tick mark
  - [x] delete of custom new providers need to be there similar to models
  - [x] add a note in add provider that only OAI compatible api can be served

- [x] Overflow issue in parent i.e whole page sometimes moves up
- [x] Param input content should preserve tabs/spaces etc

- [x] allow dot and spaces in providername and model name
- [x] reasoning models should have a "reasoning" effort param too.
  - [x] Add reasoning support in modelparams
  - [x] reasoning effort comes in multiple flavors: medium high less etc. number from x to y. find a way to represent them.
  - [x] See if temperature is almost always optional. Anthropic supports reasoning + normal mode too. Need to see how to represent that too without real overwhelming of the user. Represent appropriately in UI
    - [x] For anthropic: temp and thinking are not compatible. streaming is present. thinking is controlled as "budget tokens "
    - [x] For openai: temp and thinking are not compatible. can pass temp as 1 to some models after dec 24. streaming is present in newer api. thinking is controlled as effort, low medium high
    - [x] google doesnt mention any control on thinking model

  - [x] created a fork and try to merge pr for anthropic and openai in until we move away from langchaingo.

- [x] Support additional params from UI to backend
- [x] Represent reasoning process in UI
- [x] Conversations sometime dont load at init from homepage
- [x] deprecate 4.5preview
- [x] Home page overflows some times
- [x] Chat input fields scroll height mismatches with textarea
- [x] move packaging to build dir and add apple plist etc. also use apple developerid
- [x] Inter font files need to be cached in repo
  - [x] Did this by using font source npm
- [x] Signing for Mac pkg and Win
- [x] Mac pkg install is add the .app in ~/build/bin. It should be in application. Check the settings, conversation and log location
- [ ] ~~Ideally should detect any code or math blocks even in input, annotate them and then display. i.e format a normal text into proper md.~~
- [x] Math support broke in between, do math rendering before gfm
- [x] see how to set version increment seamlessly at one place only
  - [x] Version string to be passed via ldconfig to code and use from there
  - [x] render templates for flatpak and mac

- [x] JSONRPC
  - [x] jsonrpc message protocol implementation
  - [x] Supprot batch for jsonrpc
  - [x] stdio conn
  - [x] humaadapter for jsonrpc
    - [x] openapi docs for jsonrpc
  - [x] http transport for jsonrpc
  - [x] stdio transport

- [x] New conversation is created again and again even if empty
- [x] Input param validation: If formatting reenabled is there in runtime input params prompt it is still sent as double
- [x] Icon resolutions: revalidate icon and its resolutions available vs used

- [x] Initial inbuilt models should be available by default without a entry in settings.
- [x] Settings should be overrides and new/custom models/providers

- [x] Performance: There is large cpu consumption sometime, debug
  - [x] It is mainly seen if both wails dev and current app is open and being used wails://wails consumes 100%
  - [x] Also check when one request was fired but not allowed to complete for some reason
  - [x] Also, if the message thread gets very very big there seems to be a issue.
  - [x] it is seen that post data streaming the cpu percentage shoots up. May be stream callback leaks or looped somewhere?
  - [x] check again after installing new version for cpu issues

- [x] Conversation collection month provider has hardcoded partitions that returns current month only. Need to derive month from filename
- [x] katex and math symbols are on top of the search bar
- [ ] ~~If user has scrolled up, dont keep scrolling at bottom. this seems to introduce some unwanted cpy~~
- [x] Use shiki instead of react syntax highlighter
- [x] Nvidia issue debug and Ubuntu mate env vars fix
- [x] Move syntax highlighting to a worker

- [x] Better title deduce
  - [x] Use compromise and stopword for heuristic based titles for now

- [x] support events in simplemapdb
- [x] add fts using sqlite for simplemapdb
- [x] test and integrate fts with conversations
- [x] mtime should be added to cols in conversations and it should be used to do a incremental walk along with path used as externalID
- [x] the engine somehow has path etc as inbuilt. It looks very tightly coupled to files or something called as path. Ideally it should have opaque and its own semantics that the consumer will adapt to
- [x] Title change in chat causes new file to be created. ideally we want to only one file.
  - [x] separate out put conversation and add messages to conversation so that things are idempotent and id based comparisons can be made in put
  - [x] need listfiles with filtering in dirstore

- Entry Point (Focus / Empty Query)
  - [x] On focus, call `listConversations(pageSize=20)`
    - [x] Render a "Recent conversations" list inside the dropdown.
  - [x] Each row: title + last-modified date.
  - [x] Up/Down / mouse-hover highlight; ⏎ or click navigates to the convo.

- Local Autocomplete (Query Length 1-2)
  - [x] Filter the already-loaded titles in memory; update instantly.
  - [x] Footer item: "Press Enter to search all messages" (inactive until length ≥ 3).
  - [x] No backend round-trip.

- Debounced Full-Text Search (Query Length ≥ 3)
  - [x] After 300 ms of inactivity fire `searchConversations(q, undefined, 20)`.
  - [x] If another keystroke occurs, cancel the in-flight promise.
  - [x] Present results in two sub-groups
    - [x] 1. Title matches
    - [x] 2. Message matches (with 1-line highlighted snippet).
  - [x] Show spinner overlay while waiting; keep previous list dimmed for stability.

- Scroll-to-Load "More Results"
  - [x] Dropdown body has its own scroll container (max-height ~60 vh).
  - [x] When the user scrolls to 80 % of the container and `nextToken` exists
    - [x] -> automatically call `searchConversations(q, nextToken, 20)` and append rows.
  - [x] Optional footer: "End of results" or "Loading more".

- Interaction Rules
  - [x] up/down moves through visible items; focus wraps at top/bottom.
  - [x] enter or click on an item -> open convo.
  - [ ] ~~If item originated from a message match, auto-scroll & briefly highlight that message. (not sure how?)~~
  - [x] escape clears search & closes dropdown.
  - [ ] ~~If the user presses enter with no item highlighted, do nothing (they must choose).~~

- Performance & Networking
  - [x] Debounce: 300 ms; per-query request count rarely > 4.
  - [x] Cache last 5 queries (query string -> result set) for instant recall.
  - [x] Re-use a single `AbortController` per search bar instance to cancel stale fetches.

- Edge & Empty States
  - [x] No recent conversations -> "No conversations yet" illustration.
  - [x] No matches -> "No results for ‘XYZ’".
  - [x] Network / SQLite error -> inline error row + toast "Retry".
  - [x] Very long titles are ellipsized; full title appears in tooltip.

- [x] Search extras
  - [x] after load of prev conversation the cursor is still in box so for next time no one clicks or i can type again, but nothing happens there. ideally focus should move out so that user can come there again either via keyboard or click OR type should result in query
  - [x] right now after one query I have to explicitly click to get things to show again, i.e click out and then click in again.
  - [x] Search fully when only few chars is broken

- [x] post last streamed msg the scroll is moving a bit up
- [x] spinner before first response ~~(details if possible)~~
- [x] When scroll kicks in the font appears to be bolder. This is due to gpu composting issue with webkit. Put antialiasing on body to make experience consistent.

- [x] Refactor provider in prep for presets and skills flow
  - [x] Types in aiprovider
  - [x] We should store all these preset files in settings or their own domain folders?
  - [x] Remove confusion of ModelParams vs ModelSettings vs ModelPreset. What is stored is a ModelPreset. What comes in is a ModelParam.
  - [x] Default model also should be with presets, with name being "default preset".
  - [x] Check if inbuilt models have tooltip mentioning internal defaults
  - [x] Presets should have id/slug as unique, may be add id too as forward compatible thing.
  - [x] font of settings page
  - [x] api key in settings page is not validated before send and then set
  - [x] no config in providers gives empty unhandled page rejection in console
  - [x] use preset in input field and completion
  - [x] model preset can be loaded in input bar rather than just model. may need to figure out how to show/allow edits to the persona vs model preset

- [x] Integrate presets end to end
- [x] details loading is very slow as of now
- [x] request details are not visible in ui as of now.
- [x] input text box seems to cause some cpu consumption and jerkiness.
  - [x] autoresize is the issue.
  - [ ] ~~may be memoize other toolbar components too~~
- [x] Click on zoom for mermaid diagram

- [x] in modify modal, the search bar is visible i.e it doesnt get in background
- [x] Input field height should go to 16 rows most probably
- [x] prompt page table is not outlined

- [x] Add builtin with overlay
- [x] Integrate builtin into prompt template store
  - [x] fts and search pending
- [x] Integrate the store with UI
- [x] Startup issue in linux mint if sqlite file is missing.
- [x] Details expansion was giving large cpu
- [x] Ideally conversation list should be sorted based on modified time and not created time.

- [x] implement prompts and tools using dirstore+fts
- [x] set default provider is pending
- [x] Inbuilt model handling is pending in modelpresets.
- [x] May be in settings too, check.
- [x] In hybrid mode, temperature setting is not coming for sonnet etc

- [x] Add other daisyui themes to settings and then use them as a dropdown
- [x] theming adjustments for mermaid and code
- [x] scroll non needed on assistents reply, let it be where it is.
- [x] the scroll down buttons possition needs adjusting a bit
- [x] Scroll button needs to become visible as soon as some streamed message arrives

- Settings/Presets/Prompt/tool UI
  - [x] Add Auth keys for provider should be allowed for current providers, else point to add provider page
  - [x] In Add mode, it should be allowed to copy data from one of current things for all, with some mods in conflict avoiding fields
  - [x] If we try to set a inbuilt preset as default for first time there is a error as of now.

  - [x] overlay needs to have non bool flag support too, need it for default model support, currently no default model change supported for inbuilt ones
  - [x] Add default model in non bool overlay
  - [x] backend should check if atleast one of reasoning or temp is set

- [x] For new month of conversations etc, if we set default provider preset it says read partition error
  - [x] This is ok as it is read error at start and then next dr is created

- [x] Settings
  - [x] Need a settings data version and migration func
  - [x] API key should never reach frontend after save: Need to verify if some place this is remaining
  - [x] theme setting persistance and usage across installs

- [x] langchaingo seem to be moving very slow with no thinking support. better use official sdks from openai and anthropic
  - [x] Openai
    - [x] Add openai compatible provider presets: hf, deepseek, xai groq, openrouter, llamacpp as presets.
    - [x] Add Google also as openai compatible provider for now. gemini dev ai additions.

  - [x] Anthropic

  - [x] UI:
    - [x] Create provider should allow to choose the api sdk type
    - [ ] ~~Need to allow adding models to inbuilt providers, may need to list compatible ones in existing copy flow~~
    - [ ] ~~Need to check provider compatibility during "copy existing" flow for models i.e sdk should match. Add current providers models first, then add such models and then incompatable ones after that.~~
    - [x] If there was some amount of data that was streamed, but error occured at end someplace, the whole message goes away and only error is visible. Ideally you want to concat the issue at end and leave text as is. This is mostly UI issue.
    - [x] Need timeout in chat input field edit at runtime
    - [x] Need request cancel and request resend and response fetch synchronizations
    - [x] Thinking should be a collapsible card / dropdown / something inside the message block. Figure out the UX and implement.

- [x] Should not throw stack trace in fetch completion in case of abort

- [x] Conversations
  - [x] load all pages via infinite scroll for titles too.
  - [x] Support delete conversation. ~~Maybe, have soft delete then hard delete workflow.~~
  - [x] Support edit convo title too
  - [x] Check if title edits are proper and manual is preserved in second message edits too.

- [x] Add modal portals

## Tasks: Tools

- Tools Page
  - Header
    - [x] Design the header with the title "Tools."
    - [ ] ~~Implement a search bar for tool searching.~~

  - Main Content Area:
    - [ ] ~~Design the tool list in a card format.~~
    - [ ] ~~Display tool name, short description, and last modified date on each card.~~
    - [x] Add quick action icons/buttons for edit, delete, ~~and duplicate~~.
    - [x] Implement a "Create New Tool" button using a floating action button (FAB) or a clearly marked button.

- Create/Edit Tool Modal:
  - [x] Design the modal layout for creating/editing tools.
  - [x] Add a tool name field (required).
  - [ ] ~~Add a description field (optional).~~

  - [x] Usability
    - [x] Provide real-time validation and feedback for required fields.
    - [x] Use tooltips or inline messages for guidance on schema and function implementation.
    - [x] Ensure keyboard navigability.
    - [x] Implement ARIA labels and roles for screen readers.
  - [x] Action area:
    - [x] Implement a "Save & Close" option.
    - [x] Add a "Discard" button to exit without saving changes.

- Backend
  - [x] Integrate with json store for tool storage.
  - [x] Implement API endpoints for creating, retrieving, updating, and deleting tools.
  - [x] Ensure schema validation and function execution are supported on the backend.

## Tasks: Prompt templates

- Prompt list page
  - Header
    - [x] Design the header with the title "Prompt Templates."
    - [ ] ~~Implement a search bar for prompt searching.~~

  - Main Content Area:
    - [ ] ~~Design the prompt list in a card format.~~
    - [ ] ~~Display prompt name, short description, short prompt start string and last modified date on each card.~~
    - [x] Add quick action icons/buttons for edit, delete, and ~~duplicate~~.
    - [x] Implement a "Create New Prompt" button using a floating action button (FAB) or a clearly marked button.

- Create/Edit Prompt Modal:
  - [x] Design the modal layout for creating/editing prompts.
  - [x] Add a prompt name field (required).
  - [ ] ~~Add a description field (optional).~~
  - [ ] ~~Implement a large text area for entering the prompt template. This should be large enough and scrollable~~
  - [ ] ~~May have preferred llm provider config (??)~~
  - [ ] Would need:
    - [x] ~~tools~~
    - [ ] ~~KB~~
  - [x] Usability
    - [x] Provide real-time validation and feedback for required fields.
    - [x] Use tooltips or inline messages for guidance on template strings.
    - [x] Ensure keyboard navigability.
    - [x] Implement ARIA labels and roles for screen readers.
  - [x] Detect template strings and dynamically add them to a "Variables" section.
    - [x] This should be below the scrollable area
    - [x] Automatically populate variables section with detected template strings.
    - [x] Implement fields for variable name, type dropdown, and default value.
  - [x] Action area: should be below variables
    - [x] Implement a "Save & Close" option.
    - [x] Add a "Discard" button to exit without saving changes.

- Backend
  - [x] Integrate with JSON file store per prompt
  - [x] get list of prompts using the partitioned file store

- Integration of prompts in chat
  - [ ] ~~Once defined all prompts should be available to use in chat via some keystroke (??).~~
  - [ ] ~~Would also need some way to indicate if someone wants to add a prompt as a system prompt or just prompt~~
  - [ ] ~~Ideally if it is just prompt it should just expand the prompt in the input box~~
  - [ ] ~~For any prompt with vars, the vars should be injected with defaults properly so that a user can edit them and frontend can parse it to create a proper string too.~~
  - [ ] ~~Can vars be few system functions like open file (??)~~

- [x] Chat input UI
  - [x] Add system prompt array to main bar.
  - [x] Move input to editor from text field

  - [x] prompt/tool inline elements:
    - [x] slash command plumbing
    - [x] prompt templates via slash command

    - [x] prompt visual filling via toolbar
      - [x] prompt toolbar/editing implementation
      - [x] system prompt updates from prompts
      - [x] may be allow expand/decompose for prompts user message
      - [x] toolbar outline and stacking with shadows and round corners styling

    - [x] tools via slash command
    - [x] check if proper memo etc is there for usetemplates and use tools etc
    - [x] tools slash command hooking, tools should be added to bottom of the editor rather than inline

- Completions API backend
  - [x] Integrate stores with llm. decide on if you want to populate the prompt in input so that var expansion etc can be done properly.

- [x] CPU is high when "thinking". Need to debug
- [x] After system restart if app is open it should start again, it is not happening in mac.
- [x] Implement responses api for openai new models and integrate gpt 5 codex too.

- [x] input area spacing and size when streaming back from assistant needs re-look
- [x] Tool invocation in backend
  - [x] base support
  - [x] Don't add go tools at runtime i.e only compiled tools are there
  - [x] invocation tests for go
  - [x] invocation tests for http, inbuilt and externally added

- [x] Tool invoke when doing prompt template preprocess run
- [x] prompt templates to fill in full text including preprocessors and then give back message blocks to caller for exec.

- [x] prompt and tool processing for ai completion call

## Attachments and Tools

- [x] edit message should fall to the editor with attachments/tools so that resend is consistent
- [x] render images and attachments in msg ui
- [x] attachment tools buttons and shortcuts etc dropdowns ui
- [x] slow paste handling
- [x] system prompt delete is via sys dialog. need it to be a modal
- [x] request details are not very useful as of now in the ui. need what was actually sent rather than out version of completion data. may empty out some data things, but others can be given. Most probably need to check response too.

- [x] Details need to pop up as modal
- [x] need info to user on how the url or any attachment for that matter was processed. like was it finally done as text or link or bytes or what.
- [x] If api errors due to token length issues etc, the reply is fully omitted. need to make same as partial response in api abort cases.
- [x] Update ALL modals to use dialog showmodal with backdrop and escape handling.
- [x] url validation check in all modals after novalidate
- [x] Ariakit select for some select items and menus so that we get proper keyboard navigation.
- [x] Tooltips in checkboxes

- attachments
  - [x] attachments separate package.
  - [x] Need image and associated processing in one place
  - [x] any extension that can be read as text in one place
  - [x] web url that is just html fetch
  - [x] url modal needs styling

  - [x] pdf and associated processing in one place. Integrate a pdf text extractor.
  - [x] need to allow dirs too with some good boundaries.
  - [x] for attachments, need to detect mime and type of file etc is backend and ui should use it rather than infering using extension.
  - [x] Skip dot dirs inside the selected dir

  - [x] attachments from prev turns also need to be used. they may need to be freshly hydrated again, but need to be used.
  - [x] input notes should talk about attachments, turns, etc.
  - [x] get visible text from html needs to be implemented and a new mode raw vs visible text needs to be added.
  - [ ] ~~look at how we send attachment kind text as of now. if it is sent already as text block, we would have to skip them.~~
    - [ ] ~~It is sent as a text block but never propagated back to conversation store. so ok.~~

  - [x] Most probably we want atleast url content blocks, non image ones atleast to persist in conversation. May look at other text ones too.

- [x] Lots of testing needed for new attachments.
  - [x] Check debug details.
  - [x] Attachment follow through across conversations.
  - [x] cache token usage.
  - [x] url attachments.

- Usage:
  - [x] With attachments, token usage display seems more important.

- [x] check if extraction pipeline for PDF urls is handled properly.

- [x] tool enhancements
  - [x] attached tools to be given back as schema to consumers for exec
  - [x] backend of tool calls and outputs from build to fetch to back
  - [x] testing of chip bars
  - [x] tool exec bars review
  - [x] tool choices also need to be added as chips
  - [x] anthropic needs thinking blocks too in response.
  - [x] message attachments bar needs to have similar dropdown info chips rather than one long bar.
  - [x] url attach is still leading to a submit.

  - [x] check for empty msg with only toolresult in fe. seems some issue
    - [x] abort and then edit again is also some issue.
  - [x] the mode menu looks cutoff after pushed inside the composer.
  - [x] tool choices need to be added as "previous tool choices" and an option to reattach needs to be given.

- [x] 5.1 codex max, deepseek 3.2, 5.2
- [x] Can now introduce a independent per message thinking block and manage it as separate input block in the api

- [x] Need to migrate to content block array to support citations in conversation turns.
- [x] migrate to inference go
  - [x] adopt any types that can be adopted
  - [x] rewrite to remove build step in current code
  - [x] move the attachments hydration in prep step in current code
  - [x] think through input type and conversation storage type
    - [x] this and new inference api adoption would need to be hand in hand
  - [x] UIToolOutput and invoke tool needs to handle error response and bool
  - [x] Check if tool output is set properly in chat history.
  - [x] also may want to have a hydrate chat function that creates a proper chat from stored chat and then set.
  - [x] we return err from fetch completion which results in wails throwing error. what we should be doing is setting error object and then sending back with err nil.
- [x] typing ":" opens something in platejs.

- [x] reasoning content has status attached in input even though output did not have one
- [x] in readfile toolcall, if we say read pdf, llm calls text encoding. now readfile tries to read as stringed text even if no extraction or anything is done and we end up passing gibberish. need to rectify the toolcall.
  - [x] there is some tool error when doing this for openai 5.1
  - [x] haiku image worked.
- [x] in tooloutput, once we get the output, we are attaching it as text always, we want tools to return some content types and then create appropriate content blocks in output.
- [x] if there is a toolcall, and we dont input anything, it doesnt activate run and send button.
  - [x] in fast forward we seem to get some error. check if it is due to above or actual issue

- [x] UI for web search.
  - [x] citations to show.
    - [x] message citations bar needs proper tooltip
    - [x] tool details modal need to use message content card.
  - [x] tool calls to show in some way and not in composer chip.
  - [x] no callable tool call jsut chips end to end.

- [x] scroll to top button needed
- [x] Adding a small visual distinction between provider‑managed web search calls vs. “suggested” function/custom calls in message history.
- [x] edit user args modal opens and closes immediately.
- [x] web search: we may want a vanity chip, if space is there, in bottom actions bar, that has quick attach web search button, which will insert web search tool. this is just quick shortcut and should be synced with in composer behavior. if tool is already present it should show tick else cross. no other action other than add/remove.
- [x] for args edit, right now it is json with no indication of shape etc. we may want to actually render a json schema with empty and then let user edit. if he submits empty again, we may need way to see that it is default value or actual new value.
- [x] we should also trigger a scroll after adding thinking blob

- [x] frameless app
- [x] editor becomes heavy when lots of input content is there. verify and rectify.
  - [x] when edit chat is called for a message, it takes a LOT of time to get to input area
- [x] single tilda causes subscript ideally it should be double? No.
- [x] in input editor if i give something between tilda it doesnt work. may be i have disabled subscript super script plugins and not really skipped them somehow.

- [x] interleaved reasoning and tools
  - [x] reasoning text and switch between openai and anthropic is not working properly. better on switch would be to drop anything excess and then work with options.
  - [x] reverse anyways may not be possible as anthropic uses encrypted content, so would be better to add the message as user message to the chat.

- [x] tabs
  - [x] with more and more usage, atleast some tabs seem to be worthwhile to do. basically start a command in one and then switch to another. we would need to ensure that we dont really mix anything anywhere and streaming is proper but req wise it seems worthwhile to do.
  - [x] with tabs, we should put title and edit/export in proper places so that we get better space vertically.

- [x] after edit in editor, the cursor is going to end of the text and not immediately after. check.
- [x] do we need buttons/icon buttons as circle?
- [x] may be restore tabs should be present for cross page/restart navigation
- [x] shortcuts for tabs
- [x] chat page cleanup/refactor
- [x] ps1 file is causing crash.
  - [x] treat as unreadable

- [x] License additions in published binaries.
- [x] mime sniffing can be enriched in llmtools and then dropped from flexigpt
- [x] we can give option to change mermaids rendering theme inline.
- [x] View mode needed for builtins
- [x] prompt template add/edit needs enrichment. same to check for tools.
- [x] prompt modal doesn't close on outside click. all form modals in prompt and tool.

- [x] edit and save template doesn't work. says template slug already exists.
  - [x] most probably a mismatch between backend and frontend.
  - [x] behavior is that for tool and prompts we need versions to be different. once a version is created we dont allow anything except enable disable.

- [x] prompt template, role/vartype/varsource dropdown uses select

- [x] File name and details also should be sent along with file text from attachments
  - [x] ensure new line properly at end of file block.
- [x] shell tool should block all rm/curl etc. people can use specialized tools for that.
- [x] after paste cursor has again moved out someplace as of 72.
- [x] we should look at restricting sqlite search to last n days. ideally 30. Not now.
- [x] no scroll bar when no of tools/files chips become large.

- [x] tool features
  - [x] Implementation of openai/anthropic etc built-in tools that have been analyzed and filtered.
  - [x] Analysis of openai/anthropic etc given tools so that they can be invoked in api directly.
  - [ ] ~~parallel tools can be created locally~~

- tool auto execute
  - [x] tools should have a configuration which says can autoexecute i.e without user consent vs not.
    - with this config we can have a "agent loop" of sort for file edits/mods etc
    - major question remains as to what sort of tools should be auto exec vs not. write anything being human in loop is safest anycase. for write ones, we may want to see if we need to have a "keep old file as renamed" with some sessionid/tmp extension so that reverting is easier.

  - [x] backend to have flag for recommendation per tool
  - [x] at choice time ui can show autoexecute true/false and set default to provided one and then allow to change.
    - [x] Right now you can set it at insertion time (picker), and see it in details, but you can’t toggle it later from the “Tools” chip menu.
    - [x] toggle has a delayed view effect, i.e doesnt really show immediately. if you click it doesnt change, but if you click next, then it changes.
    - [x] tool menu doesnt show tick when selected. also this should be grid based display, the tick, the auto toggle aligned to some col, and version.

  - [x] UI needs to be so that we dont waste a lot of space for tool call only or tool result only cards.
    - [x] a tool call only and tool output only message may be rendered as a single line after this.
    - [x] rather that drop down in ui for toolcalls we need to show them as visible chips like citations if there is some "content" (not just thinking content)
    - [x] the type of chip shown can be specific to tool call like if toolcall is of type replacetext, we can have chip that shows diff and have run button there too. or generic if no other chip present.
    - [x] same can be for result but as of now we dont have tool results with special ui reqs.

- [x] editor is starting to look very very heavy, review to see for better refactoring and perf opt.
- [x] in message attachment bar chips, we want to not show a dropdown if there is only 1 call/result/output.

- [x] Chats vs Chat with AI from home
  - [x] Should one create a new chat, and other load recent chat or go to search?
  - [x] Do we want a home page at all? should it be direct chat landing page with redirects to other pages?

- [x] responses api integration for o3-pro.
  - [x] create model preset should allow the api function type
  - [x] The responses api seems a bit weird and sdk seems immature for it as of now. Evaluate at a later time again.
  - [x] Things not available due to not moving to responses as of now: reasoning thought process, state management on openai, prompt template usage from openai servers. Not sure if tool calls have some problems in chat completions vs this one including openai mcp servers and tools.

- [x] if there is a very large table in md, the ui hangs.
  - [x] scroll conflicts fixed
- [x] if there are attachments in previous message and then it was changed and then we resend it, the app crashes without trace.
- [x] mdx is not readable

- [x] UI tables alignment for config pages
- [x] when thinking, dont show streaming text in bottom line, do that only when streaming actual data.

- [x] when we send attachments, like go files. openai is consuming a lot of tokens, i.e if tiktoken says x, is is almost 2x generally. check if we are sending things double or something. req debug details doesnt show double atleast, but better print and check.
  - [x] xml encoding caused bloat and resend did not clean hydrated inputs.
