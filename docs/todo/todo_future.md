# Project Future notes

## Future feature ideas

- [ ] Token count in build completion data may be wrong as it doesn't really account for attachments/tool calls etc.
  - [ ] Need to rectify the FilterMessagesByTokenCount function. Look at using gpt5 bpe encoder now rather than regex compilation. [go-tokenizer](https://github.com/tiktoken-go/tokenizer)
  - [ ] Claude has a free token counting api with some rate limits. may need to check it out too.

- [ ] Start with the noted apps support
- [ ] Observability: At least tokens and dollars need to be represented someplace
- MCP handling
  - [ ] MCP transport: mcp httpsse, stdio for mcp
  - [ ] long future: add servers for: file, git, simplemapstore
- [ ] Docs inside app
  - [ ] Have docs pages inside app itself
  - [ ] Docs should ideally have course work for very beginners that are users of genai, without any programming knowledge.
  - [ ] MS course is fantastic, but requires lots of tech know how.
- [ ] Add keyboard shortcuts and some tooltips for them to know what are these.
  - [ ] Add AWS bedrock too. May be anthropic api as of now for claude on bedrock models?
  - [ ] Check vertex ai for google: needs google dedicated sdk

- [ ] Think through and enhance tools create/edit and prompts create/edit modals
  - [ ] Implement a schema area using CodeMirror for defining the tool's/prompts schema. (deferred)
    - [ ] Configure CodeMirror to support JSON syntax highlighting and validation.
    - [ ] Allow users to define input and output parameters using JSON schema.
  - [ ] Implement a function area using CodeMirror for accepting/implementing the schema.
    - [ ] Configure CodeMirror to support TypeScript syntax highlighting and validation.
    - [ ] Allow users to write an asynchronous TypeScript function that utilizes the defined schema.

- Chat attachments
  - [ ] Future: May need to decide later that keeping conversation attachment shapshots is better than reattach etc.
  - [x] Future: Check for race between build and fetch for attachment content fetch and how to resolve. For now this is ok to have as the time between these 2 is very small. If we introduce human feedback in between, we ned to look at this closely.

  - [ ] Future: Attachments support
    - [ ] doc formats can be in one place
    - [ ] specialized urls like github/gitlab pr's
    - [ ] local git diff/commit history/etc attachments

## Laundry list

- [ ] Chat
  - [x] Details of request are not visible immediately. We use a trapper in golang ai provider package to get req/resp details, that means until the completion returns there is no way of getting the request details. Some refactoring of workflow is needed to make this happen.
  - [x] Need a stop streaming / cancel button on assistant
  - [ ] Think through:
    - [x] In AI replies, find a way to mention the model used and special params a bit more explicitly.
    - [x] Check whether current details is good enough UX or need a bit better subtle ux.
  - [ ] Support for image/audio in chat
  - [ ] Slash commands for prompts/tools/kb in chat
    - [ ] Should model selection also be in chat commands?
    - [ ] may be can be done as: @ invokes assistants. inbuilt models with current settings are default assistants.
  - [ ] support for read/attach file in chats and then write/save file (slightly different flow than download)
  - [ ] Think if needed:
    - [ ] Users can mark a query to be not included in the chat message conversation history when talking to llm.
    - [ ] This is query level control which is different from the generic control.
    - [ ] Ignore history command structure:
      - [ ] Global on/off switch like right now
      - [ ] Query level switch overrides the global switch.

- [ ] Windows
  - [ ] Signing for Win
  - [ ] Test Windows build

- [ ] When url cannot fetch content, there is no way of knowing what happened as of now. may want to see how to expose this or disable link only mode in this flow?
  - [ ] as of now content blocks are errored out only during attach tme and not during build time.
  - [ ] whenever this is to be done, we need to see what is best ux and then design thngs accordingly.
  - [ ] early thought is that errors definitly need to be consdiered during attach time, at that time, we should do as much as possible to tell user that something is wrong. but once that stage is done, i.e non error in attach time, may be assumethat things wil lbe correct and log and skip in next stage.

## Stricken out list

- [ ] ~~logger is imported in securejsondb before it is set as ipc file logger in appimage~~
- [ ] ~~Electron: currently supported only via appimagelauncher. Better do snap I suppose, but explore later.~~
- [ ] ~~Electron: see about package distribution and updates~~
- [ ] ~~Code interpreter for few languages baked in. This can be provided as a normal tool so that~~
- [ ] ~~Container build and publish~~

## Knowledge base

Implement a `DocStoreSet` that manages multiple `DocStore` instances within a Wails desktop application. Each `DocStore` can use different adapters (e.g., local file system, HTTP) and contains collections of documents. Documents are split into chunks and stored in a vector database as separate vector "documents".

### UI

- [x] Simple single page that lists servers + collections under them.
  - [x] KB server has some associated detail
  - [x] Collection row is still a visible row but placed so that it is visually known that it belongs to the server
- [x] Add/edit/remove buttons as needed
- [x] Modal to edit server details
- [x] Modal to edit collection details and Viewing documents within a collection and number of chunks info.
- [ ] Add/Edit docs inside the collections modal
- [ ] Integrate with backend

### Entity: `DocStoreSet`

- [ ] Top level entity

- [ ] **Create the `DocStoreSet` struct with the following fields:**

  ```go
  type DocStoreSet struct {
    MetadataPath string
    DocStores    map[DocStoreName]*DocStore
  }
  ```

- [ ] **Implement methods for `DocStoreSet`:**
  - Initialize the default local filesystem docstore and add it in the set.
  - Dont implement add for now. ~~`AddDocStore(name string, adapter Adapter)`: Adds a new docstore with the specified adapter.~~
  - `GetDocStore(name string) *DocStore`: Retrieves a docstore by name.
  - Dont implement add for now. ~~`RemoveDocStore(name string)`: Removes a docstore by name.~~
  - Dont implement list for now. ~~`ListDocStores() []string`: Lists all docstore names.~~

### Entity: `DocStore`

- [ ] Design the `DocStore` Interface
  - Initialize can be independent and store specific.

  ```go
  type DocStoreName string

  // DocStore interface for different storage backends
  type DocStore interface {
   GetCollections() (map[CollectionName]*Collection, error)
   AddCollection(name CollectionName) error
   AddDocument(collection string, doc *Document) error
   DeleteDocument(collection string, docID DocumentID) error
   ListDocuments(collection string) ([]*Document, error)
  }
  ```

- [ ] Implement the Default Local File Adapter
  - [ ] Create a `ChromemDocStore` struct implementing the `DocStore` interface.
  - [ ] Define the fields for `ChromemDocStore`
    - [ ] `BasePath` (string): Base file path for storing documents.
  - [ ] Implement the `DocStore` methods for `ChromemDocStore`.

- [ ] Support for HTTP-Based DocStores
  - [ ] Similarly implement a HTTPDocStore
  - [ ] Define the fields for `HTTPDocStore`:
    - `Endpoint` (string): Base URL of the HTTP docstore.
    - `Headers` (map[string]string): Optional headers for HTTP requests.

### Entity: `Collection`

- [ ] **Create the `Collection` struct with the following fields:**

  ```go
  type CollectionName string
  type Collection struct {
    Name      CollectionName
    Documents map[string]*Document
  }
  ```

- [ ] **Implement methods for `Collection`:**
  - `AddDocument(doc *Document) error`: Adds a new document.
  - `GetDocument(docID string) *Document`: Retrieves a document by ID.
  - `RemoveDocument(docID string) error`: Removes a document by ID.
  - `ListDocuments() ([]string, error)`: Lists all document IDs.

### Entity: `Document`

- [ ] Allow CRUD for docs inside a collection
- [ ] ~~May not need a separate page. docs list can be in collection page~~
- [ ] Create searchable chunked entities
- [ ] Need ways to integrate with doc type specific loader/chunker/semantic analyzer.
- [ ] A doc type is going to have specific loader and semantic parsers. E.g: PDF would need to extract text then do some semantic classification and then create chunks to store embeddings.
- [ ] Can provide basic already available loaders in app, but would need a defined hook for it.

- Document
  - Features
    - CRUD operations
    - Store and manage documents
    - Allow document upload and deletion
    - Provide search capabilities

- [ ] **Create the `Document` struct with the following fields:**
  - `ID` (string): Unique identifier for the document.
  - `Content` (string): Full content of the document.
  - `Metadata` (map[string]interface{}): Document metadata.

  ```go
  type Document struct {
    ID       string
    Content  string
    Chunks   []*Chunk
    Metadata map[string]interface{}
  }
  ```

- [ ] **Implement methods for `Document`:**
  - `SplitIntoChunks(chunkSize int) error`: Splits the document content into chunks.
  - `GetChunk(index int) (*Chunk, error)`: Retrieves a specific chunk by index.

- [ ] Define the `Chunk` Structure
  - `Index` (int): The order of the chunk in the document.
  - `Content` (string): The content of the chunk.
  - `VectorID` (string): Identifier in the vector database.

  ```go
  type Chunk struct {
    Index    int
    Content  string
    VectorID string
  }
  ```

- [ ] **Modify `Document.SplitIntoChunks` to:**
  - Split the document content into chunks of specified size.
  - For each chunk:
    - Create a `Chunk` instance.
    - Store the chunk in the vector database using `VectorStore.StoreChunk`.
    - Update the `VectorStore` in the `Chunk`.

### Entity: `VectorStore`

- Use VectoreStore interface from langchaingo and implement for chromamem for local files
