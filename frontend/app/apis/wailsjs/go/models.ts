export namespace attachment {

	export class ContentBlock {
	    kind: string;
	    text?: string;
	    mimeType?: string;
	    fileName?: string;
	    base64Data?: string;
	    url?: string;

	    static createFrom(source: any = {}) {
	        return new ContentBlock(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.mimeType = source["mimeType"];
	        this.fileName = source["fileName"];
	        this.base64Data = source["base64Data"];
	        this.url = source["url"];
	    }
	}
	export class GenericRef {
	    handle: string;
	    origHandle: string;

	    static createFrom(source: any = {}) {
	        return new GenericRef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.handle = source["handle"];
	        this.origHandle = source["origHandle"];
	    }
	}
	export class URLRef {
	    url: string;
	    normalized?: string;
	    origNormalized: string;

	    static createFrom(source: any = {}) {
	        return new URLRef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.normalized = source["normalized"];
	        this.origNormalized = source["origNormalized"];
	    }
	}
	export class ImageRef {
	    path: string;
	    name: string;
	    exists: boolean;
	    isDir: boolean;
	    size?: number;
	    // Go type: time
	    modTime?: any;
	    width?: number;
	    height?: number;
	    format?: string;
	    mimeType?: string;
	    origPath: string;
	    origSize: number;
	    // Go type: time
	    origModTime: any;

	    static createFrom(source: any = {}) {
	        return new ImageRef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.exists = source["exists"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.modTime = this.convertValues(source["modTime"], null);
	        this.width = source["width"];
	        this.height = source["height"];
	        this.format = source["format"];
	        this.mimeType = source["mimeType"];
	        this.origPath = source["origPath"];
	        this.origSize = source["origSize"];
	        this.origModTime = this.convertValues(source["origModTime"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FileRef {
	    path: string;
	    name: string;
	    exists: boolean;
	    isDir: boolean;
	    size?: number;
	    // Go type: time
	    modTime?: any;
	    origPath: string;
	    origSize: number;
	    // Go type: time
	    origModTime: any;

	    static createFrom(source: any = {}) {
	        return new FileRef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.exists = source["exists"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.modTime = this.convertValues(source["modTime"], null);
	        this.origPath = source["origPath"];
	        this.origSize = source["origSize"];
	        this.origModTime = this.convertValues(source["origModTime"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Attachment {
	    kind: string;
	    label: string;
	    mode?: string;
	    availableContentBlockModes?: string[];
	    fileRef?: FileRef;
	    imageRef?: ImageRef;
	    urlRef?: URLRef;
	    genericRef?: GenericRef;
	    contentBlock?: ContentBlock;

	    static createFrom(source: any = {}) {
	        return new Attachment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.mode = source["mode"];
	        this.availableContentBlockModes = source["availableContentBlockModes"];
	        this.fileRef = this.convertValues(source["fileRef"], FileRef);
	        this.imageRef = this.convertValues(source["imageRef"], ImageRef);
	        this.urlRef = this.convertValues(source["urlRef"], URLRef);
	        this.genericRef = this.convertValues(source["genericRef"], GenericRef);
	        this.contentBlock = this.convertValues(source["contentBlock"], ContentBlock);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class DirectoryOverflowInfo {
	    dirPath: string;
	    relativePath: string;
	    fileCount: number;
	    partial: boolean;

	    static createFrom(source: any = {}) {
	        return new DirectoryOverflowInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dirPath = source["dirPath"];
	        this.relativePath = source["relativePath"];
	        this.fileCount = source["fileCount"];
	        this.partial = source["partial"];
	    }
	}
	export class DirectoryAttachmentsResult {
	    dirPath: string;
	    attachments: Attachment[];
	    overflowDirs: DirectoryOverflowInfo[];
	    maxFiles: number;
	    totalSize: number;
	    hasMore: boolean;

	    static createFrom(source: any = {}) {
	        return new DirectoryAttachmentsResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dirPath = source["dirPath"];
	        this.attachments = this.convertValues(source["attachments"], Attachment);
	        this.overflowDirs = this.convertValues(source["overflowDirs"], DirectoryOverflowInfo);
	        this.maxFiles = source["maxFiles"];
	        this.totalSize = source["totalSize"];
	        this.hasMore = source["hasMore"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class FileFilter {
	    DisplayName: string;
	    Extensions: string[];

	    static createFrom(source: any = {}) {
	        return new FileFilter(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DisplayName = source["DisplayName"];
	        this.Extensions = source["Extensions"];
	    }
	}




}

export namespace spec {

	export class AddProviderRequestBody {
	    sdkType: string;
	    origin: string;
	    chatCompletionPathPrefix: string;
	    apiKeyHeaderKey: string;
	    defaultHeaders: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new AddProviderRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sdkType = source["sdkType"];
	        this.origin = source["origin"];
	        this.chatCompletionPathPrefix = source["chatCompletionPathPrefix"];
	        this.apiKeyHeaderKey = source["apiKeyHeaderKey"];
	        this.defaultHeaders = source["defaultHeaders"];
	    }
	}
	export class AddProviderRequest {
	    Provider: string;
	    Body?: AddProviderRequestBody;

	    static createFrom(source: any = {}) {
	        return new AddProviderRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Provider = source["Provider"];
	        this.Body = this.convertValues(source["Body"], AddProviderRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class AddProviderResponse {


	    static createFrom(source: any = {}) {
	        return new AddProviderResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class AppTheme {
	    type: string;
	    name: string;

	    static createFrom(source: any = {}) {
	        return new AppTheme(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	    }
	}
	export class AuthKeyMeta {
	    type: string;
	    keyName: string;
	    sha256: string;
	    nonEmpty: boolean;

	    static createFrom(source: any = {}) {
	        return new AuthKeyMeta(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.keyName = source["keyName"];
	        this.sha256 = source["sha256"];
	        this.nonEmpty = source["nonEmpty"];
	    }
	}
	export class CacheControlEphemeral {
	    ttl: string;

	    static createFrom(source: any = {}) {
	        return new CacheControlEphemeral(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ttl = source["ttl"];
	    }
	}
	export class CacheControl {
	    kind: string;
	    cacheControlEphemeral?: CacheControlEphemeral;

	    static createFrom(source: any = {}) {
	        return new CacheControl(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.cacheControlEphemeral = this.convertValues(source["cacheControlEphemeral"], CacheControlEphemeral);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class URLCitation {
	    url: string;
	    title: string;
	    citedText: string;
	    startIndex: number;
	    endIndex: number;
	    encryptedIndex: string;

	    static createFrom(source: any = {}) {
	        return new URLCitation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.title = source["title"];
	        this.citedText = source["citedText"];
	        this.startIndex = source["startIndex"];
	        this.endIndex = source["endIndex"];
	        this.encryptedIndex = source["encryptedIndex"];
	    }
	}
	export class Citation {
	    kind: string;
	    urlCitation?: URLCitation;

	    static createFrom(source: any = {}) {
	        return new Citation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.urlCitation = this.convertValues(source["urlCitation"], URLCitation);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CitationConfig {
	    enabled: boolean;

	    static createFrom(source: any = {}) {
	        return new CitationConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	    }
	}
	export class Error {
	    code: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new Error(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class Usage {
	    inputTokensTotal: number;
	    inputTokensCached: number;
	    inputTokensUncached: number;
	    outputTokens: number;
	    reasoningTokens: number;

	    static createFrom(source: any = {}) {
	        return new Usage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputTokensTotal = source["inputTokensTotal"];
	        this.inputTokensCached = source["inputTokensCached"];
	        this.inputTokensUncached = source["inputTokensUncached"];
	        this.outputTokens = source["outputTokens"];
	        this.reasoningTokens = source["reasoningTokens"];
	    }
	}
	export class ToolStoreChoice {
	    choiceID: string;
	    bundleID: string;
	    bundleSlug?: string;
	    toolID?: string;
	    toolSlug: string;
	    toolVersion: string;
	    toolType: string;
	    description?: string;
	    displayName?: string;
	    autoExecute: boolean;
	    userArgSchemaInstance?: string;

	    static createFrom(source: any = {}) {
	        return new ToolStoreChoice(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.choiceID = source["choiceID"];
	        this.bundleID = source["bundleID"];
	        this.bundleSlug = source["bundleSlug"];
	        this.toolID = source["toolID"];
	        this.toolSlug = source["toolSlug"];
	        this.toolVersion = source["toolVersion"];
	        this.toolType = source["toolType"];
	        this.description = source["description"];
	        this.displayName = source["displayName"];
	        this.autoExecute = source["autoExecute"];
	        this.userArgSchemaInstance = source["userArgSchemaInstance"];
	    }
	}
	export class WebSearchToolChoiceItemUserLocation {
	    city: string;
	    country: string;
	    region: string;
	    timezone: string;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolChoiceItemUserLocation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.city = source["city"];
	        this.country = source["country"];
	        this.region = source["region"];
	        this.timezone = source["timezone"];
	    }
	}
	export class WebSearchToolChoiceItem {
	    max_uses: number;
	    searchContextSize: string;
	    allowed_domains: string[];
	    blocked_domains: string[];
	    user_location?: WebSearchToolChoiceItemUserLocation;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolChoiceItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.max_uses = source["max_uses"];
	        this.searchContextSize = source["searchContextSize"];
	        this.allowed_domains = source["allowed_domains"];
	        this.blocked_domains = source["blocked_domains"];
	        this.user_location = this.convertValues(source["user_location"], WebSearchToolChoiceItemUserLocation);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolChoice {
	    type: string;
	    id: string;
	    cacheControl?: CacheControl;
	    name: string;
	    description: string;
	    arguments?: Record<string, any>;
	    webSearchArguments?: WebSearchToolChoiceItem;

	    static createFrom(source: any = {}) {
	        return new ToolChoice(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.id = source["id"];
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.arguments = source["arguments"];
	        this.webSearchArguments = this.convertValues(source["webSearchArguments"], WebSearchToolChoiceItem);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OutputUnion {
	    kind: string;
	    outputMessage?: InputOutputContent;
	    reasoningMessage?: ReasoningContent;
	    functionToolCall?: ToolCall;
	    customToolCall?: ToolCall;
	    webSearchToolCall?: ToolCall;
	    webSearchToolOutput?: ToolOutput;

	    static createFrom(source: any = {}) {
	        return new OutputUnion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.outputMessage = this.convertValues(source["outputMessage"], InputOutputContent);
	        this.reasoningMessage = this.convertValues(source["reasoningMessage"], ReasoningContent);
	        this.functionToolCall = this.convertValues(source["functionToolCall"], ToolCall);
	        this.customToolCall = this.convertValues(source["customToolCall"], ToolCall);
	        this.webSearchToolCall = this.convertValues(source["webSearchToolCall"], ToolCall);
	        this.webSearchToolOutput = this.convertValues(source["webSearchToolOutput"], ToolOutput);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WebSearchToolOutputError {
	    code: string;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolOutputError(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	    }
	}
	export class WebSearchToolOutputSearch {
	    url: string;
	    title: string;
	    encryptedContent: string;
	    renderedContent: string;
	    page_age: string;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolOutputSearch(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.title = source["title"];
	        this.encryptedContent = source["encryptedContent"];
	        this.renderedContent = source["renderedContent"];
	        this.page_age = source["page_age"];
	    }
	}
	export class WebSearchToolOutputItemUnion {
	    kind: string;
	    searchItem?: WebSearchToolOutputSearch;
	    errorItem?: WebSearchToolOutputError;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolOutputItemUnion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.searchItem = this.convertValues(source["searchItem"], WebSearchToolOutputSearch);
	        this.errorItem = this.convertValues(source["errorItem"], WebSearchToolOutputError);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolOutputItemUnion {
	    kind: string;
	    textItem?: ContentItemText;
	    imageItem?: ContentItemImage;
	    fileItem?: ContentItemFile;

	    static createFrom(source: any = {}) {
	        return new ToolOutputItemUnion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.textItem = this.convertValues(source["textItem"], ContentItemText);
	        this.imageItem = this.convertValues(source["imageItem"], ContentItemImage);
	        this.fileItem = this.convertValues(source["fileItem"], ContentItemFile);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolOutput {
	    type: string;
	    choiceID: string;
	    id: string;
	    role: string;
	    status: string;
	    cacheControl?: CacheControl;
	    callID: string;
	    name: string;
	    isError: boolean;
	    signature: string;
	    contents?: ToolOutputItemUnion[];
	    webSearchToolOutputItems?: WebSearchToolOutputItemUnion[];

	    static createFrom(source: any = {}) {
	        return new ToolOutput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.choiceID = source["choiceID"];
	        this.id = source["id"];
	        this.role = source["role"];
	        this.status = source["status"];
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.callID = source["callID"];
	        this.name = source["name"];
	        this.isError = source["isError"];
	        this.signature = source["signature"];
	        this.contents = this.convertValues(source["contents"], ToolOutputItemUnion);
	        this.webSearchToolOutputItems = this.convertValues(source["webSearchToolOutputItems"], WebSearchToolOutputItemUnion);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WebSearchToolCallFind {
	    url: string;
	    pattern: string;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolCallFind(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.pattern = source["pattern"];
	    }
	}
	export class WebSearchToolCallOpenPage {
	    url: string;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolCallOpenPage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	    }
	}
	export class WebSearchToolCallSearchSource {
	    url: string;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolCallSearchSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	    }
	}
	export class WebSearchToolCallSearch {
	    query: string;
	    sources?: WebSearchToolCallSearchSource[];
	    input?: Record<string, any>;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolCallSearch(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.sources = this.convertValues(source["sources"], WebSearchToolCallSearchSource);
	        this.input = source["input"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WebSearchToolCallItemUnion {
	    kind: string;
	    searchItem?: WebSearchToolCallSearch;
	    openPageItem?: WebSearchToolCallOpenPage;
	    findItem?: WebSearchToolCallFind;

	    static createFrom(source: any = {}) {
	        return new WebSearchToolCallItemUnion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.searchItem = this.convertValues(source["searchItem"], WebSearchToolCallSearch);
	        this.openPageItem = this.convertValues(source["openPageItem"], WebSearchToolCallOpenPage);
	        this.findItem = this.convertValues(source["findItem"], WebSearchToolCallFind);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolCall {
	    type: string;
	    choiceID: string;
	    id: string;
	    role: string;
	    status: string;
	    cacheControl?: CacheControl;
	    callID: string;
	    name: string;
	    arguments?: string;
	    webSearchToolCallItems?: WebSearchToolCallItemUnion[];

	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.choiceID = source["choiceID"];
	        this.id = source["id"];
	        this.role = source["role"];
	        this.status = source["status"];
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.callID = source["callID"];
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	        this.webSearchToolCallItems = this.convertValues(source["webSearchToolCallItems"], WebSearchToolCallItemUnion);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReasoningContent {
	    id: string;
	    role: string;
	    status: string;
	    cacheControl?: CacheControl;
	    signature: string;
	    summary?: string[];
	    thinking?: string[];
	    redactedThinking?: string[];
	    encryptedContent?: string[];

	    static createFrom(source: any = {}) {
	        return new ReasoningContent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.status = source["status"];
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.signature = source["signature"];
	        this.summary = source["summary"];
	        this.thinking = source["thinking"];
	        this.redactedThinking = source["redactedThinking"];
	        this.encryptedContent = source["encryptedContent"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContentItemFile {
	    id: string;
	    fileName: string;
	    fileMIME: string;
	    fileURL: string;
	    fileData: string;
	    additionalContext: string;
	    citationConfig?: CitationConfig;

	    static createFrom(source: any = {}) {
	        return new ContentItemFile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fileName = source["fileName"];
	        this.fileMIME = source["fileMIME"];
	        this.fileURL = source["fileURL"];
	        this.fileData = source["fileData"];
	        this.additionalContext = source["additionalContext"];
	        this.citationConfig = this.convertValues(source["citationConfig"], CitationConfig);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContentItemImage {
	    id: string;
	    detail: string;
	    imageName: string;
	    imageMIME: string;
	    imageURL: string;
	    imageData: string;

	    static createFrom(source: any = {}) {
	        return new ContentItemImage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.detail = source["detail"];
	        this.imageName = source["imageName"];
	        this.imageMIME = source["imageMIME"];
	        this.imageURL = source["imageURL"];
	        this.imageData = source["imageData"];
	    }
	}
	export class ContentItemRefusal {
	    refusal: string;

	    static createFrom(source: any = {}) {
	        return new ContentItemRefusal(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.refusal = source["refusal"];
	    }
	}
	export class ContentItemText {
	    text: string;
	    citations?: Citation[];

	    static createFrom(source: any = {}) {
	        return new ContentItemText(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.citations = this.convertValues(source["citations"], Citation);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InputOutputContentItemUnion {
	    kind: string;
	    textItem?: ContentItemText;
	    refusalItem?: ContentItemRefusal;
	    imageItem?: ContentItemImage;
	    fileItem?: ContentItemFile;

	    static createFrom(source: any = {}) {
	        return new InputOutputContentItemUnion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.textItem = this.convertValues(source["textItem"], ContentItemText);
	        this.refusalItem = this.convertValues(source["refusalItem"], ContentItemRefusal);
	        this.imageItem = this.convertValues(source["imageItem"], ContentItemImage);
	        this.fileItem = this.convertValues(source["fileItem"], ContentItemFile);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InputOutputContent {
	    id: string;
	    role: string;
	    status: string;
	    cacheControl?: CacheControl;
	    contents?: InputOutputContentItemUnion[];

	    static createFrom(source: any = {}) {
	        return new InputOutputContent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.status = source["status"];
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.contents = this.convertValues(source["contents"], InputOutputContentItemUnion);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InputUnion {
	    kind: string;
	    inputMessage?: InputOutputContent;
	    outputMessage?: InputOutputContent;
	    reasoningMessage?: ReasoningContent;
	    functionToolCall?: ToolCall;
	    functionToolOutput?: ToolOutput;
	    customToolCall?: ToolCall;
	    customToolOutput?: ToolOutput;
	    webSearchToolCall?: ToolCall;
	    webSearchToolOutput?: ToolOutput;

	    static createFrom(source: any = {}) {
	        return new InputUnion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.inputMessage = this.convertValues(source["inputMessage"], InputOutputContent);
	        this.outputMessage = this.convertValues(source["outputMessage"], InputOutputContent);
	        this.reasoningMessage = this.convertValues(source["reasoningMessage"], ReasoningContent);
	        this.functionToolCall = this.convertValues(source["functionToolCall"], ToolCall);
	        this.functionToolOutput = this.convertValues(source["functionToolOutput"], ToolOutput);
	        this.customToolCall = this.convertValues(source["customToolCall"], ToolCall);
	        this.customToolOutput = this.convertValues(source["customToolOutput"], ToolOutput);
	        this.webSearchToolCall = this.convertValues(source["webSearchToolCall"], ToolCall);
	        this.webSearchToolOutput = this.convertValues(source["webSearchToolOutput"], ToolOutput);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConversationMessage {
	    id: string;
	    // Go type: time
	    createdAt: any;
	    role: string;
	    status: string;
	    modelParam?: ModelParam;
	    inputs?: InputUnion[];
	    outputs?: OutputUnion[];
	    toolChoices?: ToolChoice[];
	    toolStoreChoices?: ToolStoreChoice[];
	    attachments?: attachment.Attachment[];
	    usage?: Usage;
	    error?: Error;
	    debugDetails?: any;
	    meta?: Record<string, any>;

	    static createFrom(source: any = {}) {
	        return new ConversationMessage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.role = source["role"];
	        this.status = source["status"];
	        this.modelParam = this.convertValues(source["modelParam"], ModelParam);
	        this.inputs = this.convertValues(source["inputs"], InputUnion);
	        this.outputs = this.convertValues(source["outputs"], OutputUnion);
	        this.toolChoices = this.convertValues(source["toolChoices"], ToolChoice);
	        this.toolStoreChoices = this.convertValues(source["toolStoreChoices"], ToolStoreChoice);
	        this.attachments = this.convertValues(source["attachments"], attachment.Attachment);
	        this.usage = this.convertValues(source["usage"], Usage);
	        this.error = this.convertValues(source["error"], Error);
	        this.debugDetails = source["debugDetails"];
	        this.meta = source["meta"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JSONSchemaParam {
	    name: string;
	    description?: string;
	    schema?: Record<string, any>;
	    strict?: boolean;

	    static createFrom(source: any = {}) {
	        return new JSONSchemaParam(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.schema = source["schema"];
	        this.strict = source["strict"];
	    }
	}
	export class OutputFormat {
	    type: string;
	    jsonSchemaParam?: JSONSchemaParam;

	    static createFrom(source: any = {}) {
	        return new OutputFormat(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.jsonSchemaParam = this.convertValues(source["jsonSchemaParam"], JSONSchemaParam);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OutputParam {
	    format?: OutputFormat;
	    verbosity?: string;

	    static createFrom(source: any = {}) {
	        return new OutputParam(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = this.convertValues(source["format"], OutputFormat);
	        this.verbosity = source["verbosity"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReasoningParam {
	    type: string;
	    level: string;
	    tokens: number;
	    summaryStyle?: string;

	    static createFrom(source: any = {}) {
	        return new ReasoningParam(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.level = source["level"];
	        this.tokens = source["tokens"];
	        this.summaryStyle = source["summaryStyle"];
	    }
	}
	export class ModelParam {
	    name: string;
	    stream: boolean;
	    maxPromptLength: number;
	    maxOutputLength: number;
	    temperature?: number;
	    reasoning?: ReasoningParam;
	    systemPrompt: string;
	    timeout: number;
	    outputParam?: OutputParam;
	    stopSequences?: string[];
	    additionalParametersRawJSON?: string;

	    static createFrom(source: any = {}) {
	        return new ModelParam(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.stream = source["stream"];
	        this.maxPromptLength = source["maxPromptLength"];
	        this.maxOutputLength = source["maxOutputLength"];
	        this.temperature = source["temperature"];
	        this.reasoning = this.convertValues(source["reasoning"], ReasoningParam);
	        this.systemPrompt = source["systemPrompt"];
	        this.timeout = source["timeout"];
	        this.outputParam = this.convertValues(source["outputParam"], OutputParam);
	        this.stopSequences = source["stopSequences"];
	        this.additionalParametersRawJSON = source["additionalParametersRawJSON"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompletionRequestBody {
	    modelParam?: ModelParam;
	    history: ConversationMessage[];
	    current: ConversationMessage;
	    toolStoreChoices?: ToolStoreChoice[];

	    static createFrom(source: any = {}) {
	        return new CompletionRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelParam = this.convertValues(source["modelParam"], ModelParam);
	        this.history = this.convertValues(source["history"], ConversationMessage);
	        this.current = this.convertValues(source["current"], ConversationMessage);
	        this.toolStoreChoices = this.convertValues(source["toolStoreChoices"], ToolStoreChoice);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FetchCompletionResponse {
	    outputs?: OutputUnion[];
	    usage?: Usage;
	    error?: Error;
	    debugDetails?: any;

	    static createFrom(source: any = {}) {
	        return new FetchCompletionResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputs = this.convertValues(source["outputs"], OutputUnion);
	        this.usage = this.convertValues(source["usage"], Usage);
	        this.error = this.convertValues(source["error"], Error);
	        this.debugDetails = source["debugDetails"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompletionResponseBody {
	    inferenceResponse?: FetchCompletionResponse;
	    hydratedCurrentInputs?: InputUnion[];

	    static createFrom(source: any = {}) {
	        return new CompletionResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inferenceResponse = this.convertValues(source["inferenceResponse"], FetchCompletionResponse);
	        this.hydratedCurrentInputs = this.convertValues(source["hydratedCurrentInputs"], InputUnion);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompletionResponse {
	    Body?: CompletionResponseBody;

	    static createFrom(source: any = {}) {
	        return new CompletionResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], CompletionResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}





	export class Conversation {
	    schemaVersion: string;
	    id: string;
	    title?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    messages: ConversationMessage[];
	    meta?: Record<string, any>;

	    static createFrom(source: any = {}) {
	        return new Conversation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.messages = this.convertValues(source["messages"], ConversationMessage);
	        this.meta = source["meta"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConversationListItem {
	    id: string;
	    sanatizedTitle: string;
	    // Go type: time
	    modifiedAt?: any;

	    static createFrom(source: any = {}) {
	        return new ConversationListItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sanatizedTitle = source["sanatizedTitle"];
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class DeleteAuthKeyRequest {
	    Type: string;
	    KeyName: string;

	    static createFrom(source: any = {}) {
	        return new DeleteAuthKeyRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.KeyName = source["KeyName"];
	    }
	}
	export class DeleteAuthKeyResponse {


	    static createFrom(source: any = {}) {
	        return new DeleteAuthKeyResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeleteConversationRequest {
	    ID: string;
	    Title: string;

	    static createFrom(source: any = {}) {
	        return new DeleteConversationRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Title = source["Title"];
	    }
	}
	export class DeleteConversationResponse {


	    static createFrom(source: any = {}) {
	        return new DeleteConversationResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeleteModelPresetRequest {
	    ProviderName: string;
	    ModelPresetID: string;

	    static createFrom(source: any = {}) {
	        return new DeleteModelPresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	        this.ModelPresetID = source["ModelPresetID"];
	    }
	}
	export class DeleteModelPresetResponse {


	    static createFrom(source: any = {}) {
	        return new DeleteModelPresetResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeletePromptBundleRequest {
	    BundleID: string;

	    static createFrom(source: any = {}) {
	        return new DeletePromptBundleRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	    }
	}
	export class DeletePromptBundleResponse {


	    static createFrom(source: any = {}) {
	        return new DeletePromptBundleResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeletePromptTemplateRequest {
	    BundleID: string;
	    TemplateSlug: string;
	    Version: string;

	    static createFrom(source: any = {}) {
	        return new DeletePromptTemplateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.TemplateSlug = source["TemplateSlug"];
	        this.Version = source["Version"];
	    }
	}
	export class DeletePromptTemplateResponse {


	    static createFrom(source: any = {}) {
	        return new DeletePromptTemplateResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeleteProviderPresetRequest {
	    ProviderName: string;

	    static createFrom(source: any = {}) {
	        return new DeleteProviderPresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	    }
	}
	export class DeleteProviderPresetResponse {


	    static createFrom(source: any = {}) {
	        return new DeleteProviderPresetResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeleteProviderRequest {
	    Provider: string;

	    static createFrom(source: any = {}) {
	        return new DeleteProviderRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Provider = source["Provider"];
	    }
	}
	export class DeleteProviderResponse {


	    static createFrom(source: any = {}) {
	        return new DeleteProviderResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeleteToolBundleRequest {
	    BundleID: string;

	    static createFrom(source: any = {}) {
	        return new DeleteToolBundleRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	    }
	}
	export class DeleteToolBundleResponse {


	    static createFrom(source: any = {}) {
	        return new DeleteToolBundleResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeleteToolRequest {
	    BundleID: string;
	    ToolSlug: string;
	    Version: string;

	    static createFrom(source: any = {}) {
	        return new DeleteToolRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ToolSlug = source["ToolSlug"];
	        this.Version = source["Version"];
	    }
	}
	export class DeleteToolResponse {


	    static createFrom(source: any = {}) {
	        return new DeleteToolResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}


	export class GetAuthKeyRequest {
	    Type: string;
	    KeyName: string;

	    static createFrom(source: any = {}) {
	        return new GetAuthKeyRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.KeyName = source["KeyName"];
	    }
	}
	export class GetAuthKeyResponseBody {
	    secret: string;
	    sha256: string;
	    nonEmpty: boolean;

	    static createFrom(source: any = {}) {
	        return new GetAuthKeyResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.secret = source["secret"];
	        this.sha256 = source["sha256"];
	        this.nonEmpty = source["nonEmpty"];
	    }
	}
	export class GetAuthKeyResponse {
	    Body?: GetAuthKeyResponseBody;

	    static createFrom(source: any = {}) {
	        return new GetAuthKeyResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], GetAuthKeyResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class GetConversationRequest {
	    ID: string;
	    Title: string;
	    ForceFetch: boolean;

	    static createFrom(source: any = {}) {
	        return new GetConversationRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Title = source["Title"];
	        this.ForceFetch = source["ForceFetch"];
	    }
	}
	export class GetConversationResponse {
	    Body?: Conversation;

	    static createFrom(source: any = {}) {
	        return new GetConversationResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], Conversation);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetDefaultProviderRequest {


	    static createFrom(source: any = {}) {
	        return new GetDefaultProviderRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class GetDefaultProviderResponseBody {
	    DefaultProvider: string;

	    static createFrom(source: any = {}) {
	        return new GetDefaultProviderResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DefaultProvider = source["DefaultProvider"];
	    }
	}
	export class GetDefaultProviderResponse {
	    Body?: GetDefaultProviderResponseBody;

	    static createFrom(source: any = {}) {
	        return new GetDefaultProviderResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], GetDefaultProviderResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class GetPromptTemplateRequest {
	    BundleID: string;
	    TemplateSlug: string;
	    Version: string;

	    static createFrom(source: any = {}) {
	        return new GetPromptTemplateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.TemplateSlug = source["TemplateSlug"];
	        this.Version = source["Version"];
	    }
	}
	export class PromptVariable {
	    name: string;
	    type: string;
	    required: boolean;
	    source: string;
	    description?: string;
	    staticVal?: string;
	    enumValues?: string[];
	    default?: string;

	    static createFrom(source: any = {}) {
	        return new PromptVariable(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.required = source["required"];
	        this.source = source["source"];
	        this.description = source["description"];
	        this.staticVal = source["staticVal"];
	        this.enumValues = source["enumValues"];
	        this.default = source["default"];
	    }
	}
	export class MessageBlock {
	    id: string;
	    role: string;
	    content: string;

	    static createFrom(source: any = {}) {
	        return new MessageBlock(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class PromptTemplate {
	    schemaVersion: string;
	    id: string;
	    slug: string;
	    isEnabled: boolean;
	    displayName: string;
	    description?: string;
	    tags?: string[];
	    blocks: MessageBlock[];
	    variables?: PromptVariable[];
	    version: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    isBuiltIn: boolean;

	    static createFrom(source: any = {}) {
	        return new PromptTemplate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.isEnabled = source["isEnabled"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.blocks = this.convertValues(source["blocks"], MessageBlock);
	        this.variables = this.convertValues(source["variables"], PromptVariable);
	        this.version = source["version"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.isBuiltIn = source["isBuiltIn"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetPromptTemplateResponse {
	    Body?: PromptTemplate;

	    static createFrom(source: any = {}) {
	        return new GetPromptTemplateResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], PromptTemplate);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetSettingsRequest {
	    ForceFetch: boolean;

	    static createFrom(source: any = {}) {
	        return new GetSettingsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ForceFetch = source["ForceFetch"];
	    }
	}
	export class GetSettingsResponseBody {
	    appTheme: AppTheme;
	    authKeys: AuthKeyMeta[];

	    static createFrom(source: any = {}) {
	        return new GetSettingsResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appTheme = this.convertValues(source["appTheme"], AppTheme);
	        this.authKeys = this.convertValues(source["authKeys"], AuthKeyMeta);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetSettingsResponse {
	    Body?: GetSettingsResponseBody;

	    static createFrom(source: any = {}) {
	        return new GetSettingsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], GetSettingsResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class GetToolRequest {
	    BundleID: string;
	    ToolSlug: string;
	    Version: string;

	    static createFrom(source: any = {}) {
	        return new GetToolRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ToolSlug = source["ToolSlug"];
	        this.Version = source["Version"];
	    }
	}
	export class SDKToolImpl {
	    sdkType: string;

	    static createFrom(source: any = {}) {
	        return new SDKToolImpl(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sdkType = source["sdkType"];
	    }
	}
	export class HTTPResponse {
	    successCodes?: number[];
	    errorMode?: string;
	    bodyOutputMode?: string;

	    static createFrom(source: any = {}) {
	        return new HTTPResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.successCodes = source["successCodes"];
	        this.errorMode = source["errorMode"];
	        this.bodyOutputMode = source["bodyOutputMode"];
	    }
	}
	export class HTTPAuth {
	    type: string;
	    in?: string;
	    name?: string;
	    valueTemplate: string;

	    static createFrom(source: any = {}) {
	        return new HTTPAuth(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.in = source["in"];
	        this.name = source["name"];
	        this.valueTemplate = source["valueTemplate"];
	    }
	}
	export class HTTPRequest {
	    method?: string;
	    urlTemplate: string;
	    query?: Record<string, string>;
	    headers?: Record<string, string>;
	    body?: string;
	    auth?: HTTPAuth;
	    timeoutMS?: number;

	    static createFrom(source: any = {}) {
	        return new HTTPRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.urlTemplate = source["urlTemplate"];
	        this.query = source["query"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	        this.auth = this.convertValues(source["auth"], HTTPAuth);
	        this.timeoutMS = source["timeoutMS"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HTTPToolImpl {
	    request: HTTPRequest;
	    response: HTTPResponse;

	    static createFrom(source: any = {}) {
	        return new HTTPToolImpl(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.request = this.convertValues(source["request"], HTTPRequest);
	        this.response = this.convertValues(source["response"], HTTPResponse);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GoToolImpl {
	    func: string;

	    static createFrom(source: any = {}) {
	        return new GoToolImpl(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.func = source["func"];
	    }
	}
	export class Tool {
	    schemaVersion: string;
	    id: string;
	    slug: string;
	    version: string;
	    displayName: string;
	    description?: string;
	    tags?: string[];
	    userCallable: boolean;
	    llmCallable: boolean;
	    autoExecReco: boolean;
	    argSchema: number[];
	    userArgSchema?: number[];
	    llmToolType: string;
	    type: string;
	    goImpl?: GoToolImpl;
	    httpImpl?: HTTPToolImpl;
	    sdkImpl?: SDKToolImpl;
	    isEnabled: boolean;
	    isBuiltIn: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;

	    static createFrom(source: any = {}) {
	        return new Tool(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.version = source["version"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.userCallable = source["userCallable"];
	        this.llmCallable = source["llmCallable"];
	        this.autoExecReco = source["autoExecReco"];
	        this.argSchema = source["argSchema"];
	        this.userArgSchema = source["userArgSchema"];
	        this.llmToolType = source["llmToolType"];
	        this.type = source["type"];
	        this.goImpl = this.convertValues(source["goImpl"], GoToolImpl);
	        this.httpImpl = this.convertValues(source["httpImpl"], HTTPToolImpl);
	        this.sdkImpl = this.convertValues(source["sdkImpl"], SDKToolImpl);
	        this.isEnabled = source["isEnabled"];
	        this.isBuiltIn = source["isBuiltIn"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetToolResponse {
	    Body?: Tool;

	    static createFrom(source: any = {}) {
	        return new GetToolResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], Tool);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}








	export class InvokeGoOptions {
	    timeoutMS?: number;

	    static createFrom(source: any = {}) {
	        return new InvokeGoOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeoutMS = source["timeoutMS"];
	    }
	}
	export class InvokeHTTPOptions {
	    timeoutMS?: number;
	    extraHeaders?: Record<string, string>;
	    secrets?: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new InvokeHTTPOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeoutMS = source["timeoutMS"];
	        this.extraHeaders = source["extraHeaders"];
	        this.secrets = source["secrets"];
	    }
	}
	export class InvokeToolRequestBody {
	    args: string;
	    httpOptions?: InvokeHTTPOptions;
	    goOptions?: InvokeGoOptions;

	    static createFrom(source: any = {}) {
	        return new InvokeToolRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.args = source["args"];
	        this.httpOptions = this.convertValues(source["httpOptions"], InvokeHTTPOptions);
	        this.goOptions = this.convertValues(source["goOptions"], InvokeGoOptions);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvokeToolRequest {
	    BundleID: string;
	    ToolSlug: string;
	    Version: string;
	    Body?: InvokeToolRequestBody;

	    static createFrom(source: any = {}) {
	        return new InvokeToolRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ToolSlug = source["ToolSlug"];
	        this.Version = source["Version"];
	        this.Body = this.convertValues(source["Body"], InvokeToolRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ToolStoreOutputFile {
	    fileName: string;
	    fileMIME: string;
	    fileData: string;

	    static createFrom(source: any = {}) {
	        return new ToolStoreOutputFile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.fileMIME = source["fileMIME"];
	        this.fileData = source["fileData"];
	    }
	}
	export class ToolStoreOutputImage {
	    detail: string;
	    imageName: string;
	    imageMIME: string;
	    imageData: string;

	    static createFrom(source: any = {}) {
	        return new ToolStoreOutputImage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detail = source["detail"];
	        this.imageName = source["imageName"];
	        this.imageMIME = source["imageMIME"];
	        this.imageData = source["imageData"];
	    }
	}
	export class ToolStoreOutputText {
	    text: string;

	    static createFrom(source: any = {}) {
	        return new ToolStoreOutputText(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	    }
	}
	export class ToolStoreOutputUnion {
	    kind: string;
	    textItem?: ToolStoreOutputText;
	    imageItem?: ToolStoreOutputImage;
	    fileItem?: ToolStoreOutputFile;

	    static createFrom(source: any = {}) {
	        return new ToolStoreOutputUnion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.textItem = this.convertValues(source["textItem"], ToolStoreOutputText);
	        this.imageItem = this.convertValues(source["imageItem"], ToolStoreOutputImage);
	        this.fileItem = this.convertValues(source["fileItem"], ToolStoreOutputFile);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvokeToolResponseBody {
	    outputs?: ToolStoreOutputUnion[];
	    meta?: Record<string, any>;
	    isBuiltIn: boolean;
	    isError: boolean;
	    errorMessage: string;

	    static createFrom(source: any = {}) {
	        return new InvokeToolResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputs = this.convertValues(source["outputs"], ToolStoreOutputUnion);
	        this.meta = source["meta"];
	        this.isBuiltIn = source["isBuiltIn"];
	        this.isError = source["isError"];
	        this.errorMessage = source["errorMessage"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvokeToolResponse {
	    Body?: InvokeToolResponseBody;

	    static createFrom(source: any = {}) {
	        return new InvokeToolResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], InvokeToolResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}


	export class ListConversationsRequest {
	    PageSize: number;
	    PageToken: string;

	    static createFrom(source: any = {}) {
	        return new ListConversationsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class ListConversationsResponseBody {
	    conversationListItems: ConversationListItem[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new ListConversationsResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conversationListItems = this.convertValues(source["conversationListItems"], ConversationListItem);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListConversationsResponse {
	    Body?: ListConversationsResponseBody;

	    static createFrom(source: any = {}) {
	        return new ListConversationsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListConversationsResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ListPromptBundlesRequest {
	    BundleIDs: string[];
	    IncludeDisabled: boolean;
	    PageSize: number;
	    PageToken: string;

	    static createFrom(source: any = {}) {
	        return new ListPromptBundlesRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class PromptBundle {
	    schemaVersion: string;
	    id: string;
	    slug: string;
	    displayName?: string;
	    description?: string;
	    isEnabled: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    isBuiltIn: boolean;
	    // Go type: time
	    softDeletedAt?: any;

	    static createFrom(source: any = {}) {
	        return new PromptBundle(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.isEnabled = source["isEnabled"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.isBuiltIn = source["isBuiltIn"];
	        this.softDeletedAt = this.convertValues(source["softDeletedAt"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListPromptBundlesResponseBody {
	    promptBundles: PromptBundle[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new ListPromptBundlesResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptBundles = this.convertValues(source["promptBundles"], PromptBundle);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListPromptBundlesResponse {
	    Body?: ListPromptBundlesResponseBody;

	    static createFrom(source: any = {}) {
	        return new ListPromptBundlesResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListPromptBundlesResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ListPromptTemplatesRequest {
	    BundleIDs: string[];
	    Tags: string[];
	    IncludeDisabled: boolean;
	    RecommendedPageSize: number;
	    PageToken: string;

	    static createFrom(source: any = {}) {
	        return new ListPromptTemplatesRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.Tags = source["Tags"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.RecommendedPageSize = source["RecommendedPageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class PromptTemplateListItem {
	    bundleID: string;
	    bundleSlug: string;
	    templateSlug: string;
	    templateVersion: string;
	    isBuiltIn: boolean;

	    static createFrom(source: any = {}) {
	        return new PromptTemplateListItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.bundleSlug = source["bundleSlug"];
	        this.templateSlug = source["templateSlug"];
	        this.templateVersion = source["templateVersion"];
	        this.isBuiltIn = source["isBuiltIn"];
	    }
	}
	export class ListPromptTemplatesResponseBody {
	    promptTemplateListItems: PromptTemplateListItem[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new ListPromptTemplatesResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptTemplateListItems = this.convertValues(source["promptTemplateListItems"], PromptTemplateListItem);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListPromptTemplatesResponse {
	    Body?: ListPromptTemplatesResponseBody;

	    static createFrom(source: any = {}) {
	        return new ListPromptTemplatesResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListPromptTemplatesResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ListProviderPresetsRequest {
	    Names: string[];
	    IncludeDisabled: boolean;
	    PageSize: number;
	    PageToken: string;

	    static createFrom(source: any = {}) {
	        return new ListProviderPresetsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Names = source["Names"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class ModelPreset {
	    schemaVersion: string;
	    id: string;
	    name: string;
	    displayName: string;
	    slug: string;
	    isEnabled: boolean;
	    stream?: boolean;
	    maxPromptLength?: number;
	    maxOutputLength?: number;
	    temperature?: number;
	    reasoning?: ReasoningParam;
	    systemPrompt?: string;
	    timeout?: number;
	    additionalParametersRawJSON?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    isBuiltIn: boolean;

	    static createFrom(source: any = {}) {
	        return new ModelPreset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.slug = source["slug"];
	        this.isEnabled = source["isEnabled"];
	        this.stream = source["stream"];
	        this.maxPromptLength = source["maxPromptLength"];
	        this.maxOutputLength = source["maxOutputLength"];
	        this.temperature = source["temperature"];
	        this.reasoning = this.convertValues(source["reasoning"], ReasoningParam);
	        this.systemPrompt = source["systemPrompt"];
	        this.timeout = source["timeout"];
	        this.additionalParametersRawJSON = source["additionalParametersRawJSON"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.isBuiltIn = source["isBuiltIn"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProviderPreset {
	    schemaVersion: string;
	    name: string;
	    displayName: string;
	    sdkType: string;
	    isEnabled: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    isBuiltIn: boolean;
	    origin: string;
	    chatCompletionPathPrefix: string;
	    apiKeyHeaderKey: string;
	    defaultHeaders: Record<string, string>;
	    defaultModelPresetID: string;
	    modelPresets: Record<string, ModelPreset>;

	    static createFrom(source: any = {}) {
	        return new ProviderPreset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.sdkType = source["sdkType"];
	        this.isEnabled = source["isEnabled"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.isBuiltIn = source["isBuiltIn"];
	        this.origin = source["origin"];
	        this.chatCompletionPathPrefix = source["chatCompletionPathPrefix"];
	        this.apiKeyHeaderKey = source["apiKeyHeaderKey"];
	        this.defaultHeaders = source["defaultHeaders"];
	        this.defaultModelPresetID = source["defaultModelPresetID"];
	        this.modelPresets = this.convertValues(source["modelPresets"], ModelPreset, true);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListProviderPresetsResponseBody {
	    providers: ProviderPreset[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new ListProviderPresetsResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = this.convertValues(source["providers"], ProviderPreset);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListProviderPresetsResponse {
	    Body?: ListProviderPresetsResponseBody;

	    static createFrom(source: any = {}) {
	        return new ListProviderPresetsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListProviderPresetsResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ListToolBundlesRequest {
	    BundleIDs: string[];
	    IncludeDisabled: boolean;
	    PageSize: number;
	    PageToken: string;

	    static createFrom(source: any = {}) {
	        return new ListToolBundlesRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class ToolBundle {
	    schemaVersion: string;
	    id: string;
	    slug: string;
	    displayName?: string;
	    description?: string;
	    isEnabled: boolean;
	    isBuiltIn: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    // Go type: time
	    softDeletedAt?: any;

	    static createFrom(source: any = {}) {
	        return new ToolBundle(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.isEnabled = source["isEnabled"];
	        this.isBuiltIn = source["isBuiltIn"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.softDeletedAt = this.convertValues(source["softDeletedAt"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListToolBundlesResponseBody {
	    toolBundles: ToolBundle[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new ListToolBundlesResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolBundles = this.convertValues(source["toolBundles"], ToolBundle);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListToolBundlesResponse {
	    Body?: ListToolBundlesResponseBody;

	    static createFrom(source: any = {}) {
	        return new ListToolBundlesResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListToolBundlesResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ListToolsRequest {
	    BundleIDs: string[];
	    Tags: string[];
	    IncludeDisabled: boolean;
	    RecommendedPageSize: number;
	    PageToken: string;

	    static createFrom(source: any = {}) {
	        return new ListToolsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.Tags = source["Tags"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.RecommendedPageSize = source["RecommendedPageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class ToolListItem {
	    bundleID: string;
	    bundleSlug: string;
	    toolSlug: string;
	    toolVersion: string;
	    isBuiltIn: boolean;
	    toolDefinition: Tool;

	    static createFrom(source: any = {}) {
	        return new ToolListItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.bundleSlug = source["bundleSlug"];
	        this.toolSlug = source["toolSlug"];
	        this.toolVersion = source["toolVersion"];
	        this.isBuiltIn = source["isBuiltIn"];
	        this.toolDefinition = this.convertValues(source["toolDefinition"], Tool);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListToolsResponseBody {
	    toolListItems: ToolListItem[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new ListToolsResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolListItems = this.convertValues(source["toolListItems"], ToolListItem);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListToolsResponse {
	    Body?: ListToolsResponseBody;

	    static createFrom(source: any = {}) {
	        return new ListToolsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListToolsResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}







	export class PatchDefaultProviderRequestBody {
	    defaultProvider: string;

	    static createFrom(source: any = {}) {
	        return new PatchDefaultProviderRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultProvider = source["defaultProvider"];
	    }
	}
	export class PatchDefaultProviderRequest {
	    Body?: PatchDefaultProviderRequestBody;

	    static createFrom(source: any = {}) {
	        return new PatchDefaultProviderRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], PatchDefaultProviderRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PatchDefaultProviderResponse {


	    static createFrom(source: any = {}) {
	        return new PatchDefaultProviderResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PatchModelPresetRequestBody {
	    isEnabled: boolean;

	    static createFrom(source: any = {}) {
	        return new PatchModelPresetRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchModelPresetRequest {
	    ProviderName: string;
	    ModelPresetID: string;
	    Body?: PatchModelPresetRequestBody;

	    static createFrom(source: any = {}) {
	        return new PatchModelPresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	        this.ModelPresetID = source["ModelPresetID"];
	        this.Body = this.convertValues(source["Body"], PatchModelPresetRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PatchModelPresetResponse {


	    static createFrom(source: any = {}) {
	        return new PatchModelPresetResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PatchPromptBundleRequestBody {
	    isEnabled: boolean;

	    static createFrom(source: any = {}) {
	        return new PatchPromptBundleRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchPromptBundleRequest {
	    BundleID: string;
	    Body?: PatchPromptBundleRequestBody;

	    static createFrom(source: any = {}) {
	        return new PatchPromptBundleRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PatchPromptBundleRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PatchPromptBundleResponse {


	    static createFrom(source: any = {}) {
	        return new PatchPromptBundleResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PatchPromptTemplateRequestBody {
	    isEnabled: boolean;

	    static createFrom(source: any = {}) {
	        return new PatchPromptTemplateRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchPromptTemplateRequest {
	    BundleID: string;
	    TemplateSlug: string;
	    Version: string;
	    Body?: PatchPromptTemplateRequestBody;

	    static createFrom(source: any = {}) {
	        return new PatchPromptTemplateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.TemplateSlug = source["TemplateSlug"];
	        this.Version = source["Version"];
	        this.Body = this.convertValues(source["Body"], PatchPromptTemplateRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PatchPromptTemplateResponse {


	    static createFrom(source: any = {}) {
	        return new PatchPromptTemplateResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PatchProviderPresetRequestBody {
	    isEnabled?: boolean;
	    defaultModelPresetID?: string;

	    static createFrom(source: any = {}) {
	        return new PatchProviderPresetRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	        this.defaultModelPresetID = source["defaultModelPresetID"];
	    }
	}
	export class PatchProviderPresetRequest {
	    ProviderName: string;
	    Body?: PatchProviderPresetRequestBody;

	    static createFrom(source: any = {}) {
	        return new PatchProviderPresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	        this.Body = this.convertValues(source["Body"], PatchProviderPresetRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PatchProviderPresetResponse {


	    static createFrom(source: any = {}) {
	        return new PatchProviderPresetResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PatchToolBundleRequestBody {
	    isEnabled: boolean;

	    static createFrom(source: any = {}) {
	        return new PatchToolBundleRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchToolBundleRequest {
	    BundleID: string;
	    Body?: PatchToolBundleRequestBody;

	    static createFrom(source: any = {}) {
	        return new PatchToolBundleRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PatchToolBundleRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PatchToolBundleResponse {


	    static createFrom(source: any = {}) {
	        return new PatchToolBundleResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PatchToolRequestBody {
	    isEnabled: boolean;

	    static createFrom(source: any = {}) {
	        return new PatchToolRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchToolRequest {
	    BundleID: string;
	    ToolSlug: string;
	    Version: string;
	    Body?: PatchToolRequestBody;

	    static createFrom(source: any = {}) {
	        return new PatchToolRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ToolSlug = source["ToolSlug"];
	        this.Version = source["Version"];
	        this.Body = this.convertValues(source["Body"], PatchToolRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PatchToolResponse {


	    static createFrom(source: any = {}) {
	        return new PatchToolResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}





	export class PutConversationRequestBody {
	    title: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    messages: ConversationMessage[];
	    meta?: Record<string, any>;

	    static createFrom(source: any = {}) {
	        return new PutConversationRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.messages = this.convertValues(source["messages"], ConversationMessage);
	        this.meta = source["meta"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PutConversationRequest {
	    ID: string;
	    Body?: PutConversationRequestBody;

	    static createFrom(source: any = {}) {
	        return new PutConversationRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Body = this.convertValues(source["Body"], PutConversationRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PutConversationResponse {


	    static createFrom(source: any = {}) {
	        return new PutConversationResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PutMessagesToConversationRequestBody {
	    title: string;
	    messages: ConversationMessage[];

	    static createFrom(source: any = {}) {
	        return new PutMessagesToConversationRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.messages = this.convertValues(source["messages"], ConversationMessage);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PutMessagesToConversationRequest {
	    ID: string;
	    Body?: PutMessagesToConversationRequestBody;

	    static createFrom(source: any = {}) {
	        return new PutMessagesToConversationRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Body = this.convertValues(source["Body"], PutMessagesToConversationRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PutMessagesToConversationResponse {


	    static createFrom(source: any = {}) {
	        return new PutMessagesToConversationResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PutModelPresetRequestBody {
	    name: string;
	    slug: string;
	    displayName: string;
	    isEnabled: boolean;
	    stream?: boolean;
	    maxPromptLength?: number;
	    maxOutputLength?: number;
	    temperature?: number;
	    reasoning?: ReasoningParam;
	    systemPrompt?: string;
	    timeout?: number;
	    additionalParametersRawJSON?: string;

	    static createFrom(source: any = {}) {
	        return new PutModelPresetRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.isEnabled = source["isEnabled"];
	        this.stream = source["stream"];
	        this.maxPromptLength = source["maxPromptLength"];
	        this.maxOutputLength = source["maxOutputLength"];
	        this.temperature = source["temperature"];
	        this.reasoning = this.convertValues(source["reasoning"], ReasoningParam);
	        this.systemPrompt = source["systemPrompt"];
	        this.timeout = source["timeout"];
	        this.additionalParametersRawJSON = source["additionalParametersRawJSON"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PutModelPresetRequest {
	    ProviderName: string;
	    ModelPresetID: string;
	    Body?: PutModelPresetRequestBody;

	    static createFrom(source: any = {}) {
	        return new PutModelPresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	        this.ModelPresetID = source["ModelPresetID"];
	        this.Body = this.convertValues(source["Body"], PutModelPresetRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PutModelPresetResponse {


	    static createFrom(source: any = {}) {
	        return new PutModelPresetResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PutPromptBundleRequestBody {
	    slug: string;
	    displayName: string;
	    isEnabled: boolean;
	    description?: string;

	    static createFrom(source: any = {}) {
	        return new PutPromptBundleRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.isEnabled = source["isEnabled"];
	        this.description = source["description"];
	    }
	}
	export class PutPromptBundleRequest {
	    BundleID: string;
	    Body?: PutPromptBundleRequestBody;

	    static createFrom(source: any = {}) {
	        return new PutPromptBundleRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PutPromptBundleRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PutPromptBundleResponse {


	    static createFrom(source: any = {}) {
	        return new PutPromptBundleResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PutPromptTemplateRequestBody {
	    displayName: string;
	    isEnabled: boolean;
	    description?: string;
	    blocks: MessageBlock[];
	    tags?: string[];
	    variables?: PromptVariable[];

	    static createFrom(source: any = {}) {
	        return new PutPromptTemplateRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.isEnabled = source["isEnabled"];
	        this.description = source["description"];
	        this.blocks = this.convertValues(source["blocks"], MessageBlock);
	        this.tags = source["tags"];
	        this.variables = this.convertValues(source["variables"], PromptVariable);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PutPromptTemplateRequest {
	    BundleID: string;
	    TemplateSlug: string;
	    Version: string;
	    Body?: PutPromptTemplateRequestBody;

	    static createFrom(source: any = {}) {
	        return new PutPromptTemplateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.TemplateSlug = source["TemplateSlug"];
	        this.Version = source["Version"];
	        this.Body = this.convertValues(source["Body"], PutPromptTemplateRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PutPromptTemplateResponse {


	    static createFrom(source: any = {}) {
	        return new PutPromptTemplateResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PutProviderPresetRequestBody {
	    displayName: string;
	    sdkType: string;
	    isEnabled: boolean;
	    origin: string;
	    chatCompletionPathPrefix: string;
	    apiKeyHeaderKey?: string;
	    defaultHeaders?: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new PutProviderPresetRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.sdkType = source["sdkType"];
	        this.isEnabled = source["isEnabled"];
	        this.origin = source["origin"];
	        this.chatCompletionPathPrefix = source["chatCompletionPathPrefix"];
	        this.apiKeyHeaderKey = source["apiKeyHeaderKey"];
	        this.defaultHeaders = source["defaultHeaders"];
	    }
	}
	export class PutProviderPresetRequest {
	    ProviderName: string;
	    Body?: PutProviderPresetRequestBody;

	    static createFrom(source: any = {}) {
	        return new PutProviderPresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	        this.Body = this.convertValues(source["Body"], PutProviderPresetRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PutProviderPresetResponse {


	    static createFrom(source: any = {}) {
	        return new PutProviderPresetResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PutToolBundleRequestBody {
	    slug: string;
	    displayName: string;
	    isEnabled: boolean;
	    description?: string;

	    static createFrom(source: any = {}) {
	        return new PutToolBundleRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.isEnabled = source["isEnabled"];
	        this.description = source["description"];
	    }
	}
	export class PutToolBundleRequest {
	    BundleID: string;
	    Body?: PutToolBundleRequestBody;

	    static createFrom(source: any = {}) {
	        return new PutToolBundleRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PutToolBundleRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PutToolBundleResponse {


	    static createFrom(source: any = {}) {
	        return new PutToolBundleResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class PutToolRequestBody {
	    displayName: string;
	    description?: string;
	    tags?: string[];
	    isEnabled: boolean;
	    userCallable: boolean;
	    llmCallable: boolean;
	    autoExecReco: boolean;
	    argSchema: string;
	    type: string;
	    httpImpl?: HTTPToolImpl;

	    static createFrom(source: any = {}) {
	        return new PutToolRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.isEnabled = source["isEnabled"];
	        this.userCallable = source["userCallable"];
	        this.llmCallable = source["llmCallable"];
	        this.autoExecReco = source["autoExecReco"];
	        this.argSchema = source["argSchema"];
	        this.type = source["type"];
	        this.httpImpl = this.convertValues(source["httpImpl"], HTTPToolImpl);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PutToolRequest {
	    BundleID: string;
	    ToolSlug: string;
	    Version: string;
	    Body?: PutToolRequestBody;

	    static createFrom(source: any = {}) {
	        return new PutToolRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ToolSlug = source["ToolSlug"];
	        this.Version = source["Version"];
	        this.Body = this.convertValues(source["Body"], PutToolRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class PutToolResponse {


	    static createFrom(source: any = {}) {
	        return new PutToolResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}



	export class SearchConversationsRequest {
	    Query: string;
	    PageToken: string;
	    PageSize: number;

	    static createFrom(source: any = {}) {
	        return new SearchConversationsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Query = source["Query"];
	        this.PageToken = source["PageToken"];
	        this.PageSize = source["PageSize"];
	    }
	}
	export class SearchConversationsResponseBody {
	    conversationListItems: ConversationListItem[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new SearchConversationsResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conversationListItems = this.convertValues(source["conversationListItems"], ConversationListItem);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchConversationsResponse {
	    Body?: SearchConversationsResponseBody;

	    static createFrom(source: any = {}) {
	        return new SearchConversationsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], SearchConversationsResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class SearchPromptTemplatesRequest {
	    Query: string;
	    PageToken: string;
	    PageSize: number;
	    IncludeDisabled: boolean;

	    static createFrom(source: any = {}) {
	        return new SearchPromptTemplatesRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Query = source["Query"];
	        this.PageToken = source["PageToken"];
	        this.PageSize = source["PageSize"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	    }
	}
	export class SearchPromptTemplatesResponseBody {
	    promptTemplateListItems: PromptTemplateListItem[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new SearchPromptTemplatesResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptTemplateListItems = this.convertValues(source["promptTemplateListItems"], PromptTemplateListItem);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchPromptTemplatesResponse {
	    Body?: SearchPromptTemplatesResponseBody;

	    static createFrom(source: any = {}) {
	        return new SearchPromptTemplatesResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], SearchPromptTemplatesResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class SearchToolsRequest {
	    Query: string;
	    PageToken: string;
	    PageSize: number;
	    IncludeDisabled: boolean;

	    static createFrom(source: any = {}) {
	        return new SearchToolsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Query = source["Query"];
	        this.PageToken = source["PageToken"];
	        this.PageSize = source["PageSize"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	    }
	}
	export class SearchToolsResponseBody {
	    toolListItems: ToolListItem[];
	    nextPageToken?: string;

	    static createFrom(source: any = {}) {
	        return new SearchToolsResponseBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolListItems = this.convertValues(source["toolListItems"], ToolListItem);
	        this.nextPageToken = source["nextPageToken"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchToolsResponse {
	    Body?: SearchToolsResponseBody;

	    static createFrom(source: any = {}) {
	        return new SearchToolsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], SearchToolsResponseBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class SetAppThemeRequestBody {
	    type: string;
	    name: string;

	    static createFrom(source: any = {}) {
	        return new SetAppThemeRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	    }
	}
	export class SetAppThemeRequest {
	    Body?: SetAppThemeRequestBody;

	    static createFrom(source: any = {}) {
	        return new SetAppThemeRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], SetAppThemeRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class SetAppThemeResponse {


	    static createFrom(source: any = {}) {
	        return new SetAppThemeResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class SetAuthKeyRequestBody {
	    secret: string;

	    static createFrom(source: any = {}) {
	        return new SetAuthKeyRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.secret = source["secret"];
	    }
	}
	export class SetAuthKeyRequest {
	    Type: string;
	    KeyName: string;
	    Body?: SetAuthKeyRequestBody;

	    static createFrom(source: any = {}) {
	        return new SetAuthKeyRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.KeyName = source["KeyName"];
	        this.Body = this.convertValues(source["Body"], SetAuthKeyRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class SetAuthKeyResponse {


	    static createFrom(source: any = {}) {
	        return new SetAuthKeyResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class SetProviderAPIKeyRequestBody {
	    apiKey: string;

	    static createFrom(source: any = {}) {
	        return new SetProviderAPIKeyRequestBody(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	    }
	}
	export class SetProviderAPIKeyRequest {
	    Provider: string;
	    Body?: SetProviderAPIKeyRequestBody;

	    static createFrom(source: any = {}) {
	        return new SetProviderAPIKeyRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Provider = source["Provider"];
	        this.Body = this.convertValues(source["Body"], SetProviderAPIKeyRequestBody);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class SetProviderAPIKeyResponse {


	    static createFrom(source: any = {}) {
	        return new SetProviderAPIKeyResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
























}

