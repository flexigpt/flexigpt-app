export namespace artifactstore {
	
	export class DiagnosticLocation {
	    locator?: string;
	    subresourceLocator?: string;
	    line?: number;
	    column?: number;
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticLocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	        this.line = source["line"];
	        this.column = source["column"];
	    }
	}
	export class Diagnostic {
	    severity: string;
	    code: string;
	    message: string;
	    location?: DiagnosticLocation;
	
	    static createFrom(source: any = {}) {
	        return new Diagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.code = source["code"];
	        this.message = source["message"];
	        this.location = this.convertValues(source["location"], DiagnosticLocation);
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

}

export namespace attachment {
	
	export class ContentBlock {
	    kind: string;
	    text?: string;
	    mimeType?: string;
	    fileName?: string;
	    filePath?: string;
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
	        this.filePath = source["filePath"];
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
	
	
	
	export class PathAttachmentsResult {
	    fileAttachments: Attachment[];
	    dirAttachments: DirectoryAttachmentsResult[];
	    errors?: string[];
	
	    static createFrom(source: any = {}) {
	        return new PathAttachmentsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileAttachments = this.convertValues(source["fileAttachments"], Attachment);
	        this.dirAttachments = this.convertValues(source["dirAttachments"], DirectoryAttachmentsResult);
	        this.errors = source["errors"];
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

}

export namespace capabilityoverride {
	
	export class CacheControlCapabilitiesOverride {
	    supportsTTL?: boolean;
	    supportedKinds?: string[];
	    supportedTTLs?: string[];
	    supportsKey?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CacheControlCapabilitiesOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportsTTL = source["supportsTTL"];
	        this.supportedKinds = source["supportedKinds"];
	        this.supportedTTLs = source["supportedTTLs"];
	        this.supportsKey = source["supportsKey"];
	    }
	}
	export class CacheCapabilitiesOverride {
	    supportsAutomaticCaching?: boolean;
	    topLevel?: CacheControlCapabilitiesOverride;
	    inputOutputContent?: CacheControlCapabilitiesOverride;
	    reasoningContent?: CacheControlCapabilitiesOverride;
	    toolChoice?: CacheControlCapabilitiesOverride;
	    toolCall?: CacheControlCapabilitiesOverride;
	    toolOutput?: CacheControlCapabilitiesOverride;
	
	    static createFrom(source: any = {}) {
	        return new CacheCapabilitiesOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportsAutomaticCaching = source["supportsAutomaticCaching"];
	        this.topLevel = this.convertValues(source["topLevel"], CacheControlCapabilitiesOverride);
	        this.inputOutputContent = this.convertValues(source["inputOutputContent"], CacheControlCapabilitiesOverride);
	        this.reasoningContent = this.convertValues(source["reasoningContent"], CacheControlCapabilitiesOverride);
	        this.toolChoice = this.convertValues(source["toolChoice"], CacheControlCapabilitiesOverride);
	        this.toolCall = this.convertValues(source["toolCall"], CacheControlCapabilitiesOverride);
	        this.toolOutput = this.convertValues(source["toolOutput"], CacheControlCapabilitiesOverride);
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
	
	export class ParamDialectOverride {
	    maxOutputTokensParamName?: string;
	    toolChoiceParamStyle?: string;
	
	    static createFrom(source: any = {}) {
	        return new ParamDialectOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxOutputTokensParamName = source["maxOutputTokensParamName"];
	        this.toolChoiceParamStyle = source["toolChoiceParamStyle"];
	    }
	}
	export class ToolCapabilitiesOverride {
	    supportedToolTypes?: string[];
	    supportedToolPolicyModes?: string[];
	    supportsParallelToolCalls?: boolean;
	    maxForcedTools?: number;
	    supportedClientToolOutputFormats?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ToolCapabilitiesOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportedToolTypes = source["supportedToolTypes"];
	        this.supportedToolPolicyModes = source["supportedToolPolicyModes"];
	        this.supportsParallelToolCalls = source["supportsParallelToolCalls"];
	        this.maxForcedTools = source["maxForcedTools"];
	        this.supportedClientToolOutputFormats = source["supportedClientToolOutputFormats"];
	    }
	}
	export class OutputCapabilitiesOverride {
	    supportedOutputFormats?: string[];
	    supportsVerbosity?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OutputCapabilitiesOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportedOutputFormats = source["supportedOutputFormats"];
	        this.supportsVerbosity = source["supportsVerbosity"];
	    }
	}
	export class StopSequenceCapabilitiesOverride {
	    isSupported?: boolean;
	    disallowedWithReasoning?: boolean;
	    maxSequences?: number;
	
	    static createFrom(source: any = {}) {
	        return new StopSequenceCapabilitiesOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isSupported = source["isSupported"];
	        this.disallowedWithReasoning = source["disallowedWithReasoning"];
	        this.maxSequences = source["maxSequences"];
	    }
	}
	export class ReasoningTokenBudgetCapabilitiesOverride {
	    minAllowed?: number;
	    maxAllowed?: number;
	    zeroAllowed?: boolean;
	    minusOneAllowed?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReasoningTokenBudgetCapabilitiesOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minAllowed = source["minAllowed"];
	        this.maxAllowed = source["maxAllowed"];
	        this.zeroAllowed = source["zeroAllowed"];
	        this.minusOneAllowed = source["minusOneAllowed"];
	    }
	}
	export class ReasoningCapabilitiesOverride {
	    supportsReasoningConfig?: boolean;
	    supportedReasoningTypes?: string[];
	    supportedReasoningLevels?: string[];
	    hybridTokenBudgetCapabilities?: ReasoningTokenBudgetCapabilitiesOverride;
	    supportsSummaryStyle?: boolean;
	    supportsEncryptedReasoningInput?: boolean;
	    temperatureDisallowedWhenEnabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReasoningCapabilitiesOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportsReasoningConfig = source["supportsReasoningConfig"];
	        this.supportedReasoningTypes = source["supportedReasoningTypes"];
	        this.supportedReasoningLevels = source["supportedReasoningLevels"];
	        this.hybridTokenBudgetCapabilities = this.convertValues(source["hybridTokenBudgetCapabilities"], ReasoningTokenBudgetCapabilitiesOverride);
	        this.supportsSummaryStyle = source["supportsSummaryStyle"];
	        this.supportsEncryptedReasoningInput = source["supportsEncryptedReasoningInput"];
	        this.temperatureDisallowedWhenEnabled = source["temperatureDisallowedWhenEnabled"];
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
	export class ModelCapabilitiesOverride {
	    modalitiesIn?: string[];
	    modalitiesOut?: string[];
	    reasoningCapabilities?: ReasoningCapabilitiesOverride;
	    stopSequenceCapabilities?: StopSequenceCapabilitiesOverride;
	    outputCapabilities?: OutputCapabilitiesOverride;
	    toolCapabilities?: ToolCapabilitiesOverride;
	    cacheCapabilities?: CacheCapabilitiesOverride;
	    paramDialect?: ParamDialectOverride;
	
	    static createFrom(source: any = {}) {
	        return new ModelCapabilitiesOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modalitiesIn = source["modalitiesIn"];
	        this.modalitiesOut = source["modalitiesOut"];
	        this.reasoningCapabilities = this.convertValues(source["reasoningCapabilities"], ReasoningCapabilitiesOverride);
	        this.stopSequenceCapabilities = this.convertValues(source["stopSequenceCapabilities"], StopSequenceCapabilitiesOverride);
	        this.outputCapabilities = this.convertValues(source["outputCapabilities"], OutputCapabilitiesOverride);
	        this.toolCapabilities = this.convertValues(source["toolCapabilities"], ToolCapabilitiesOverride);
	        this.cacheCapabilities = this.convertValues(source["cacheCapabilities"], CacheCapabilitiesOverride);
	        this.paramDialect = this.convertValues(source["paramDialect"], ParamDialectOverride);
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
	
	
	
	
	

}

export namespace skillruntime {
	
	export class ListProvidedSkillsRequest {
	    workspaceRootID?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListProvidedSkillsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspaceRootID = source["workspaceRootID"];
	    }
	}
	export class Skill {
	    identity: string;
	    origin: string;
	    installedRef?: spec.SkillRef;
	    workspaceRootID?: string;
	    workspaceRecordID?: string;
	    recordRevision?: number;
	    name: string;
	    displayName: string;
	    description: string;
	    insert: string;
	    arguments?: spec.SkillArgument[];
	    tags?: string[];
	    enabled: boolean;
	    available: boolean;
	    runtimeAllowed: boolean;
	    builtIn: boolean;
	    priority: number;
	    catalogCurrent: boolean;
	    state?: string;
	    shadowed: boolean;
	    shadowedBy?: string;
	    definitionDigest?: string;
	    sourceID?: string;
	    locator?: string;
	    diagnostics?: artifactstore.Diagnostic[];
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.identity = source["identity"];
	        this.origin = source["origin"];
	        this.installedRef = this.convertValues(source["installedRef"], spec.SkillRef);
	        this.workspaceRootID = source["workspaceRootID"];
	        this.workspaceRecordID = source["workspaceRecordID"];
	        this.recordRevision = source["recordRevision"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], spec.SkillArgument);
	        this.tags = source["tags"];
	        this.enabled = source["enabled"];
	        this.available = source["available"];
	        this.runtimeAllowed = source["runtimeAllowed"];
	        this.builtIn = source["builtIn"];
	        this.priority = source["priority"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.state = source["state"];
	        this.shadowed = source["shadowed"];
	        this.shadowedBy = source["shadowedBy"];
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class ListProvidedSkillsResponseBody {
	    skills: Skill[];
	
	    static createFrom(source: any = {}) {
	        return new ListProvidedSkillsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skills = this.convertValues(source["skills"], Skill);
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
	export class ListProvidedSkillsResponse {
	    Body?: ListProvidedSkillsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListProvidedSkillsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListProvidedSkillsResponseBody);
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
	
	export class RenderProvidedSkillRequestBody {
	    workspaceRootID?: string;
	    identity: string;
	    arguments?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RenderProvidedSkillRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspaceRootID = source["workspaceRootID"];
	        this.identity = source["identity"];
	        this.arguments = source["arguments"];
	    }
	}
	export class RenderProvidedSkillRequest {
	    Body?: RenderProvidedSkillRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new RenderProvidedSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], RenderProvidedSkillRequestBody);
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
	
	export class RenderedSkill {
	    skill: Skill;
	    available: boolean;
	    text?: string;
	    insert?: string;
	    arguments?: spec.SkillArgument[];
	    appliedArguments?: Record<string, string>;
	    diagnostics?: artifactstore.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new RenderedSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skill = this.convertValues(source["skill"], Skill);
	        this.available = source["available"];
	        this.text = source["text"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], spec.SkillArgument);
	        this.appliedArguments = source["appliedArguments"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class RenderProvidedSkillResponse {
	    Body?: RenderedSkill;
	
	    static createFrom(source: any = {}) {
	        return new RenderProvidedSkillResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], RenderedSkill);
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
	

}

export namespace spec {
	
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
	export class MCPPromptSelection {
	    bundleID: string;
	    serverID: string;
	    promptName: string;
	    title?: string;
	    displayName: string;
	    description?: string;
	    arguments?: Record<string, MCPArgumentDefinition>;
	    digest?: string;
	    argumentValues?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPPromptSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.promptName = source["promptName"];
	        this.title = source["title"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.arguments = this.convertValues(source["arguments"], MCPArgumentDefinition, true);
	        this.digest = source["digest"];
	        this.argumentValues = source["argumentValues"];
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
	export class MCPArgumentDefinition {
	    name: string;
	    title?: string;
	    description?: string;
	    required?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPArgumentDefinition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.required = source["required"];
	    }
	}
	export class MCPResourceTemplateSelection {
	    bundleID: string;
	    serverID: string;
	    uriTemplate: string;
	    name?: string;
	    title?: string;
	    displayName: string;
	    description?: string;
	    mimeType?: string;
	    arguments?: Record<string, MCPArgumentDefinition>;
	    annotations?: Record<string, any>;
	    digest?: string;
	    argumentValues?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourceTemplateSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.uriTemplate = source["uriTemplate"];
	        this.name = source["name"];
	        this.title = source["title"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.mimeType = source["mimeType"];
	        this.arguments = this.convertValues(source["arguments"], MCPArgumentDefinition, true);
	        this.annotations = source["annotations"];
	        this.digest = source["digest"];
	        this.argumentValues = source["argumentValues"];
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
	export class MCPResourceRef {
	    bundleID: string;
	    serverID: string;
	    uri: string;
	    name?: string;
	    title?: string;
	    displayName: string;
	    description?: string;
	    mimeType?: string;
	    size?: number;
	    annotations?: Record<string, any>;
	    digest?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourceRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.uri = source["uri"];
	        this.name = source["name"];
	        this.title = source["title"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.mimeType = source["mimeType"];
	        this.size = source["size"];
	        this.annotations = source["annotations"];
	        this.digest = source["digest"];
	    }
	}
	export class MCPToolSelection {
	    bundleID: string;
	    serverID: string;
	    toolName: string;
	    providerToolName?: string;
	    choiceID?: string;
	    digest?: string;
	    approvalRule?: string;
	    executionMode?: string;
	    appResourceUri?: string;
	    visibility?: string[];
	
	    static createFrom(source: any = {}) {
	        return new MCPToolSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.toolName = source["toolName"];
	        this.providerToolName = source["providerToolName"];
	        this.choiceID = source["choiceID"];
	        this.digest = source["digest"];
	        this.approvalRule = source["approvalRule"];
	        this.executionMode = source["executionMode"];
	        this.appResourceUri = source["appResourceUri"];
	        this.visibility = source["visibility"];
	    }
	}
	export class MCPServerSelection {
	    bundleID: string;
	    serverID: string;
	    snapshotDigest?: string;
	    toolExposure: string;
	    selectedTools?: MCPToolSelection[];
	    includeServerInstructions?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.snapshotDigest = source["snapshotDigest"];
	        this.toolExposure = source["toolExposure"];
	        this.selectedTools = this.convertValues(source["selectedTools"], MCPToolSelection);
	        this.includeServerInstructions = source["includeServerInstructions"];
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
	export class MCPConversationContext {
	    servers: MCPServerSelection[];
	    resources?: MCPResourceRef[];
	    resourceTemplates?: MCPResourceTemplateSelection[];
	    prompts?: MCPPromptSelection[];
	
	    static createFrom(source: any = {}) {
	        return new MCPConversationContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], MCPServerSelection);
	        this.resources = this.convertValues(source["resources"], MCPResourceRef);
	        this.resourceTemplates = this.convertValues(source["resourceTemplates"], MCPResourceTemplateSelection);
	        this.prompts = this.convertValues(source["prompts"], MCPPromptSelection);
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
	export class SkillRef {
	    bundleID: string;
	    skillSlug: string;
	    skillID: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.skillSlug = source["skillSlug"];
	        this.skillID = source["skillID"];
	    }
	}
	export class SkillSelection {
	    skillRef: SkillRef;
	    preLoadAsActive: boolean;
	    useAsInstructions: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SkillSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillRef = this.convertValues(source["skillRef"], SkillRef);
	        this.preLoadAsActive = source["preLoadAsActive"];
	        this.useAsInstructions = source["useAsInstructions"];
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
	export class ToolChoicePatch {
	    autoExecute?: boolean;
	    userArgSchemaInstance?: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolChoicePatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoExecute = source["autoExecute"];
	        this.userArgSchemaInstance = source["userArgSchemaInstance"];
	    }
	}
	export class ToolRef {
	    bundleID: string;
	    toolSlug: string;
	    toolVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.toolSlug = source["toolSlug"];
	        this.toolVersion = source["toolVersion"];
	    }
	}
	export class ToolSelection {
	    toolRef: ToolRef;
	    toolChoicePatch?: ToolChoicePatch;
	
	    static createFrom(source: any = {}) {
	        return new ToolSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolRef = this.convertValues(source["toolRef"], ToolRef);
	        this.toolChoicePatch = this.convertValues(source["toolChoicePatch"], ToolChoicePatch);
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
	export class ModelPresetRef {
	    providerName: string;
	    modelPresetID: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelPresetRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerName = source["providerName"];
	        this.modelPresetID = source["modelPresetID"];
	    }
	}
	export class AssistantPreset {
	    schemaVersion: string;
	    id: string;
	    slug: string;
	    version: string;
	    displayName: string;
	    description?: string;
	    isEnabled: boolean;
	    isBuiltIn: boolean;
	    startingText?: string;
	    startingModelPresetRef?: ModelPresetRef;
	    startingIncludeModelSystemPrompt?: boolean;
	    startingToolSelections?: ToolSelection[];
	    startingSkillSelections?: SkillSelection[];
	    startingMCPContext?: MCPConversationContext;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AssistantPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.version = source["version"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.isEnabled = source["isEnabled"];
	        this.isBuiltIn = source["isBuiltIn"];
	        this.startingText = source["startingText"];
	        this.startingModelPresetRef = this.convertValues(source["startingModelPresetRef"], ModelPresetRef);
	        this.startingIncludeModelSystemPrompt = source["startingIncludeModelSystemPrompt"];
	        this.startingToolSelections = this.convertValues(source["startingToolSelections"], ToolSelection);
	        this.startingSkillSelections = this.convertValues(source["startingSkillSelections"], SkillSelection);
	        this.startingMCPContext = this.convertValues(source["startingMCPContext"], MCPConversationContext);
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
	export class AssistantPresetBundle {
	    schemaVersion: string;
	    id: string;
	    slug: string;
	    displayName: string;
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
	        return new AssistantPresetBundle(source);
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
	export class AssistantPresetListItem {
	    bundleID: string;
	    bundleSlug: string;
	    assistantPresetSlug: string;
	    assistantPresetVersion: string;
	    displayName: string;
	    description?: string;
	    isEnabled: boolean;
	    isBuiltIn: boolean;
	    // Go type: time
	    modifiedAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new AssistantPresetListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.bundleSlug = source["bundleSlug"];
	        this.assistantPresetSlug = source["assistantPresetSlug"];
	        this.assistantPresetVersion = source["assistantPresetVersion"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.isEnabled = source["isEnabled"];
	        this.isBuiltIn = source["isBuiltIn"];
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
	export class CacheControl {
	    kind: string;
	    ttl?: string;
	    key?: string;
	
	    static createFrom(source: any = {}) {
	        return new CacheControl(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.ttl = source["ttl"];
	        this.key = source["key"];
	    }
	}
	export class CancelPendingMCPOAuthAuthorizationRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new CancelPendingMCPOAuthAuthorizationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class CancelPendingMCPOAuthAuthorizationResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new CancelPendingMCPOAuthAuthorizationResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
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
	export class CloseSkillSessionRequest {
	    SessionID: string;
	
	    static createFrom(source: any = {}) {
	        return new CloseSkillSessionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SessionID = source["SessionID"];
	    }
	}
	export class CloseSkillSessionResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new CloseSkillSessionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
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
	export class MCPIcon {
	    src: string;
	    mimeType?: string;
	    sizes?: string[];
	    theme?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPIcon(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.src = source["src"];
	        this.mimeType = source["mimeType"];
	        this.sizes = source["sizes"];
	        this.theme = source["theme"];
	    }
	}
	export class MCPResourceContents {
	    uri: string;
	    mimeType?: string;
	    text?: string;
	    blob?: number[];
	    _meta?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourceContents(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uri = source["uri"];
	        this.mimeType = source["mimeType"];
	        this.text = source["text"];
	        this.blob = source["blob"];
	        this._meta = source["_meta"];
	    }
	}
	export class MCPContent {
	    type: string;
	    text?: string;
	    data?: number[];
	    mimeType?: string;
	    uri?: string;
	    name?: string;
	    title?: string;
	    description?: string;
	    size?: number;
	    resource?: MCPResourceContents;
	    annotations?: Record<string, any>;
	    _meta?: Record<string, any>;
	    icons?: MCPIcon[];
	
	    static createFrom(source: any = {}) {
	        return new MCPContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.text = source["text"];
	        this.data = source["data"];
	        this.mimeType = source["mimeType"];
	        this.uri = source["uri"];
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.size = source["size"];
	        this.resource = this.convertValues(source["resource"], MCPResourceContents);
	        this.annotations = source["annotations"];
	        this._meta = source["_meta"];
	        this.icons = this.convertValues(source["icons"], MCPIcon);
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
	export class MCPAppModelContextUpdate {
	    instanceID?: string;
	    bundleID?: string;
	    serverID?: string;
	    resourceUri?: string;
	    content?: MCPContent[];
	    structuredContent?: any;
	    updatedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPAppModelContextUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.instanceID = source["instanceID"];
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.resourceUri = source["resourceUri"];
	        this.content = this.convertValues(source["content"], MCPContent);
	        this.structuredContent = source["structuredContent"];
	        this.updatedAt = source["updatedAt"];
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
	    maxUses: number;
	    searchContextSize: string;
	    allowedDomains: string[];
	    blockedDomains: string[];
	    userLocation?: WebSearchToolChoiceItemUserLocation;
	
	    static createFrom(source: any = {}) {
	        return new WebSearchToolChoiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxUses = source["maxUses"];
	        this.searchContextSize = source["searchContextSize"];
	        this.allowedDomains = source["allowedDomains"];
	        this.blockedDomains = source["blockedDomains"];
	        this.userLocation = this.convertValues(source["userLocation"], WebSearchToolChoiceItemUserLocation);
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
	    pageAge: string;
	
	    static createFrom(source: any = {}) {
	        return new WebSearchToolOutputSearch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.title = source["title"];
	        this.encryptedContent = source["encryptedContent"];
	        this.renderedContent = source["renderedContent"];
	        this.pageAge = source["pageAge"];
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
	    signature: string;
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
	        this.signature = source["signature"];
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
	    signature: string;
	
	    static createFrom(source: any = {}) {
	        return new ContentItemText(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.citations = this.convertValues(source["citations"], Citation);
	        this.signature = source["signature"];
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
	    modelPresetRef?: ModelPresetRef;
	    inputs?: InputUnion[];
	    outputs?: OutputUnion[];
	    toolChoices?: ToolChoice[];
	    toolStoreChoices?: ToolStoreChoice[];
	    mcpContext?: MCPConversationContext;
	    mcpAppContextUpdates?: MCPAppModelContextUpdate[];
	    attachments?: attachment.Attachment[];
	    enabledSkillRefs?: SkillRef[];
	    activeSkillRefs?: SkillRef[];
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
	        this.modelPresetRef = this.convertValues(source["modelPresetRef"], ModelPresetRef);
	        this.inputs = this.convertValues(source["inputs"], InputUnion);
	        this.outputs = this.convertValues(source["outputs"], OutputUnion);
	        this.toolChoices = this.convertValues(source["toolChoices"], ToolChoice);
	        this.toolStoreChoices = this.convertValues(source["toolStoreChoices"], ToolStoreChoice);
	        this.mcpContext = this.convertValues(source["mcpContext"], MCPConversationContext);
	        this.mcpAppContextUpdates = this.convertValues(source["mcpAppContextUpdates"], MCPAppModelContextUpdate);
	        this.attachments = this.convertValues(source["attachments"], attachment.Attachment);
	        this.enabledSkillRefs = this.convertValues(source["enabledSkillRefs"], SkillRef);
	        this.activeSkillRefs = this.convertValues(source["activeSkillRefs"], SkillRef);
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
	    kind: string;
	    jsonSchemaParam?: JSONSchemaParam;
	
	    static createFrom(source: any = {}) {
	        return new OutputFormat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
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
	    cacheControl?: CacheControl;
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
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
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
	    mcpContext?: MCPConversationContext;
	    skillSessionID?: string;
	
	    static createFrom(source: any = {}) {
	        return new CompletionRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelParam = this.convertValues(source["modelParam"], ModelParam);
	        this.history = this.convertValues(source["history"], ConversationMessage);
	        this.current = this.convertValues(source["current"], ConversationMessage);
	        this.toolStoreChoices = this.convertValues(source["toolStoreChoices"], ToolStoreChoice);
	        this.mcpContext = this.convertValues(source["mcpContext"], MCPConversationContext);
	        this.skillSessionID = source["skillSessionID"];
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
	export class Warning {
	    code: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Warning(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class FetchCompletionResponse {
	    outputs?: OutputUnion[];
	    usage?: Usage;
	    error?: Error;
	    warnings?: Warning[];
	    debugDetails?: any;
	
	    static createFrom(source: any = {}) {
	        return new FetchCompletionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputs = this.convertValues(source["outputs"], OutputUnion);
	        this.usage = this.convertValues(source["usage"], Usage);
	        this.error = this.convertValues(source["error"], Error);
	        this.warnings = this.convertValues(source["warnings"], Warning);
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
	
	export class ConnectMCPServerRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectMCPServerRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class MCPServerCapabilitiesSummary {
	    tools?: boolean;
	    toolsListChanged?: boolean;
	    resources?: boolean;
	    resourcesSubscribe?: boolean;
	    resourcesListChanged?: boolean;
	    prompts?: boolean;
	    promptsListChanged?: boolean;
	    completions?: boolean;
	    experimental?: Record<string, any>;
	    extensions?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerCapabilitiesSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tools = source["tools"];
	        this.toolsListChanged = source["toolsListChanged"];
	        this.resources = source["resources"];
	        this.resourcesSubscribe = source["resourcesSubscribe"];
	        this.resourcesListChanged = source["resourcesListChanged"];
	        this.prompts = source["prompts"];
	        this.promptsListChanged = source["promptsListChanged"];
	        this.completions = source["completions"];
	        this.experimental = source["experimental"];
	        this.extensions = source["extensions"];
	    }
	}
	export class MCPImplementationInfo {
	    name?: string;
	    version?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPImplementationInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	    }
	}
	export class MCPServerRuntimeSnapshot {
	    bundleID: string;
	    serverID: string;
	    status: string;
	    negotiatedProtocolVersion?: string;
	    serverInfo?: MCPImplementationInfo;
	    serverCapabilities?: MCPServerCapabilitiesSummary;
	    instructions?: string;
	    lastError?: string;
	    lastConnectedAt?: string;
	    lastSyncedAt?: string;
	    toolCount: number;
	    resourceCount: number;
	    resourceTemplateCount: number;
	    promptCount: number;
	    snapshotDigest?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerRuntimeSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.status = source["status"];
	        this.negotiatedProtocolVersion = source["negotiatedProtocolVersion"];
	        this.serverInfo = this.convertValues(source["serverInfo"], MCPImplementationInfo);
	        this.serverCapabilities = this.convertValues(source["serverCapabilities"], MCPServerCapabilitiesSummary);
	        this.instructions = source["instructions"];
	        this.lastError = source["lastError"];
	        this.lastConnectedAt = source["lastConnectedAt"];
	        this.lastSyncedAt = source["lastSyncedAt"];
	        this.toolCount = source["toolCount"];
	        this.resourceCount = source["resourceCount"];
	        this.resourceTemplateCount = source["resourceTemplateCount"];
	        this.promptCount = source["promptCount"];
	        this.snapshotDigest = source["snapshotDigest"];
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
	export class ConnectMCPServerResponse {
	    Body?: MCPServerRuntimeSnapshot;
	
	    static createFrom(source: any = {}) {
	        return new ConnectMCPServerResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPServerRuntimeSnapshot);
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
	
	export class CreateSkillSessionRequestBody {
	    closeSessionID?: string;
	    maxActivePerSession?: number;
	    allowSkillRefs?: SkillRef[];
	    activeSkillRefs?: SkillRef[];
	
	    static createFrom(source: any = {}) {
	        return new CreateSkillSessionRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.closeSessionID = source["closeSessionID"];
	        this.maxActivePerSession = source["maxActivePerSession"];
	        this.allowSkillRefs = this.convertValues(source["allowSkillRefs"], SkillRef);
	        this.activeSkillRefs = this.convertValues(source["activeSkillRefs"], SkillRef);
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
	export class CreateSkillSessionRequest {
	    Body?: CreateSkillSessionRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new CreateSkillSessionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], CreateSkillSessionRequestBody);
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
	
	export class CreateSkillSessionResponseBody {
	    sessionID: string;
	    activeSkillRefs: SkillRef[];
	
	    static createFrom(source: any = {}) {
	        return new CreateSkillSessionResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionID = source["sessionID"];
	        this.activeSkillRefs = this.convertValues(source["activeSkillRefs"], SkillRef);
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
	export class CreateSkillSessionResponse {
	    Body?: CreateSkillSessionResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new CreateSkillSessionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], CreateSkillSessionResponseBody);
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
	
	export class DebugSettings {
	    logLLMReqResp: boolean;
	    disableContentStripping: boolean;
	    logLevel: string;
	
	    static createFrom(source: any = {}) {
	        return new DebugSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logLLMReqResp = source["logLLMReqResp"];
	        this.disableContentStripping = source["disableContentStripping"];
	        this.logLevel = source["logLevel"];
	    }
	}
	export class DeleteAssistantPresetBundleRequest {
	    BundleID: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteAssistantPresetBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	    }
	}
	export class DeleteAssistantPresetBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new DeleteAssistantPresetBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class DeleteAssistantPresetRequest {
	    BundleID: string;
	    AssistantPresetSlug: string;
	    Version: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteAssistantPresetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.AssistantPresetSlug = source["AssistantPresetSlug"];
	        this.Version = source["Version"];
	    }
	}
	export class DeleteAssistantPresetResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new DeleteAssistantPresetResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
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
	export class DeleteMCPBundleRequest {
	    BundleID: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteMCPBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	    }
	}
	export class DeleteMCPBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new DeleteMCPBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class DeleteMCPServerRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteMCPServerRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class DeleteMCPServerResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new DeleteMCPServerResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class DeleteMCPServerSecretRequest {
	    BundleID: string;
	    ServerID: string;
	    Kind: string;
	    Slot: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteMCPServerSecretRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Kind = source["Kind"];
	        this.Slot = source["Slot"];
	    }
	}
	export class DeleteMCPServerSecretResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new DeleteMCPServerSecretResponse(source);
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
	export class DeleteSkillBundleRequest {
	    BundleID: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteSkillBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	    }
	}
	export class DeleteSkillBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new DeleteSkillBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class DeleteSkillRequest {
	    BundleID: string;
	    SkillSlug: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.SkillSlug = source["SkillSlug"];
	    }
	}
	export class DeleteSkillResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new DeleteSkillResponse(source);
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
	export class DisconnectMCPServerRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new DisconnectMCPServerRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class DisconnectMCPServerResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new DisconnectMCPServerResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	
	export class InvokeMCPToolRequestBody {
	    source: string;
	    toolName: string;
	    providerToolName?: string;
	    toolDigest?: string;
	    arguments?: Record<string, any>;
	    approvalID?: string;
	    approvalToken?: string;
	    conversationID?: string;
	    messageID?: string;
	    toolUseID?: string;
	    appInstanceID?: string;
	
	    static createFrom(source: any = {}) {
	        return new InvokeMCPToolRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.toolName = source["toolName"];
	        this.providerToolName = source["providerToolName"];
	        this.toolDigest = source["toolDigest"];
	        this.arguments = source["arguments"];
	        this.approvalID = source["approvalID"];
	        this.approvalToken = source["approvalToken"];
	        this.conversationID = source["conversationID"];
	        this.messageID = source["messageID"];
	        this.toolUseID = source["toolUseID"];
	        this.appInstanceID = source["appInstanceID"];
	    }
	}
	export class EvaluateMCPToolCallRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: InvokeMCPToolRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new EvaluateMCPToolCallRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], InvokeMCPToolRequestBody);
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
	export class MCPApprovalSummary {
	    bundleID: string;
	    serverID: string;
	    serverDisplayName?: string;
	    toolName: string;
	    toolDigest?: string;
	    risk: string;
	    arguments?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPApprovalSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.serverDisplayName = source["serverDisplayName"];
	        this.toolName = source["toolName"];
	        this.toolDigest = source["toolDigest"];
	        this.risk = source["risk"];
	        this.arguments = source["arguments"];
	    }
	}
	export class MCPApprovalEvaluation {
	    decision: string;
	    reason?: string;
	    approvalID?: string;
	    summary?: MCPApprovalSummary;
	
	    static createFrom(source: any = {}) {
	        return new MCPApprovalEvaluation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.decision = source["decision"];
	        this.reason = source["reason"];
	        this.approvalID = source["approvalID"];
	        this.summary = this.convertValues(source["summary"], MCPApprovalSummary);
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
	export class EvaluateMCPToolCallResponse {
	    Body?: MCPApprovalEvaluation;
	
	    static createFrom(source: any = {}) {
	        return new EvaluateMCPToolCallResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPApprovalEvaluation);
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
	
	export class GetAssistantPresetRequest {
	    BundleID: string;
	    AssistantPresetSlug: string;
	    Version: string;
	
	    static createFrom(source: any = {}) {
	        return new GetAssistantPresetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.AssistantPresetSlug = source["AssistantPresetSlug"];
	        this.Version = source["Version"];
	    }
	}
	export class GetAssistantPresetResponse {
	    Body?: AssistantPreset;
	
	    static createFrom(source: any = {}) {
	        return new GetAssistantPresetResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], AssistantPreset);
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
	    defaultProvider: string;
	
	    static createFrom(source: any = {}) {
	        return new GetDefaultProviderResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultProvider = source["defaultProvider"];
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
	
	export class GetMCPServerAuthHealthRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPServerAuthHealthRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class MCPAuthHealth {
	    bundleID?: string;
	    serverID: string;
	    authMode: string;
	    state: string;
	    configured: boolean;
	    resource?: string;
	    scopes?: string[];
	    // Go type: time
	    expiresAt?: any;
	    authorizationPending?: boolean;
	    authorizationURL?: string;
	    authorizationExpiresAt?: string;
	    oauthRedirectURL?: string;
	    oauthLoopbackListenAddr?: string;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPAuthHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.authMode = source["authMode"];
	        this.state = source["state"];
	        this.configured = source["configured"];
	        this.resource = source["resource"];
	        this.scopes = source["scopes"];
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
	        this.authorizationPending = source["authorizationPending"];
	        this.authorizationURL = source["authorizationURL"];
	        this.authorizationExpiresAt = source["authorizationExpiresAt"];
	        this.oauthRedirectURL = source["oauthRedirectURL"];
	        this.oauthLoopbackListenAddr = source["oauthLoopbackListenAddr"];
	        this.lastError = source["lastError"];
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
	export class GetMCPServerAuthHealthResponse {
	    Body?: MCPAuthHealth;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPServerAuthHealthResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPAuthHealth);
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
	export class GetMCPServerAuthStatusRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPServerAuthStatusRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class MCPAuthStatus {
	    bundleID: string;
	    serverID: string;
	    authMode: string;
	    state: string;
	    scopes?: string[];
	    // Go type: time
	    expiresAt?: any;
	    lastError?: string;
	    authorizationServer?: string;
	    resource?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPAuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.authMode = source["authMode"];
	        this.state = source["state"];
	        this.scopes = source["scopes"];
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
	        this.lastError = source["lastError"];
	        this.authorizationServer = source["authorizationServer"];
	        this.resource = source["resource"];
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
	export class GetMCPServerAuthStatusResponse {
	    Body?: MCPAuthStatus;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPServerAuthStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPAuthStatus);
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
	export class GetMCPServerRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPServerRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class MCPSetupClientIDMetadataDocumentURLInput {
	
	
	    static createFrom(source: any = {}) {
	        return new MCPSetupClientIDMetadataDocumentURLInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class MCPSetupStreamableHTTPURLInput {
	
	
	    static createFrom(source: any = {}) {
	        return new MCPSetupStreamableHTTPURLInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class MCPSetupStdioEnvInput {
	    envName: string;
	    secret?: boolean;
	    valuePrefix?: string;
	    valueSuffix?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPSetupStdioEnvInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.envName = source["envName"];
	        this.secret = source["secret"];
	        this.valuePrefix = source["valuePrefix"];
	        this.valueSuffix = source["valueSuffix"];
	    }
	}
	export class MCPSetupHTTPHeaderInput {
	    headerName: string;
	    secret?: boolean;
	    valuePrefix?: string;
	    valueSuffix?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPSetupHTTPHeaderInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headerName = source["headerName"];
	        this.secret = source["secret"];
	        this.valuePrefix = source["valuePrefix"];
	        this.valueSuffix = source["valueSuffix"];
	    }
	}
	export class MCPSetupOAuthClientCredentialsInput {
	    clientSecretRequired?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPSetupOAuthClientCredentialsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clientSecretRequired = source["clientSecretRequired"];
	    }
	}
	export class MCPServerSetupInput {
	    id: string;
	    kind: string;
	    label?: string;
	    description?: string;
	    note?: string;
	    placeholder?: string;
	    required?: boolean;
	    oauthClientCredentials?: MCPSetupOAuthClientCredentialsInput;
	    httpHeader?: MCPSetupHTTPHeaderInput;
	    stdioEnv?: MCPSetupStdioEnvInput;
	    // Go type: MCPSetupStreamableHTTPURLInput
	    streamableHttpUrl?: any;
	    // Go type: MCPSetupClientIDMetadataDocumentURLInput
	    clientIDMetadataDocumentURL?: any;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerSetupInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.note = source["note"];
	        this.placeholder = source["placeholder"];
	        this.required = source["required"];
	        this.oauthClientCredentials = this.convertValues(source["oauthClientCredentials"], MCPSetupOAuthClientCredentialsInput);
	        this.httpHeader = this.convertValues(source["httpHeader"], MCPSetupHTTPHeaderInput);
	        this.stdioEnv = this.convertValues(source["stdioEnv"], MCPSetupStdioEnvInput);
	        this.streamableHttpUrl = this.convertValues(source["streamableHttpUrl"], null);
	        this.clientIDMetadataDocumentURL = this.convertValues(source["clientIDMetadataDocumentURL"], null);
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
	export class MCPServerSetup {
	    note?: string;
	    inputs?: MCPServerSetupInput[];
	
	    static createFrom(source: any = {}) {
	        return new MCPServerSetup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.note = source["note"];
	        this.inputs = this.convertValues(source["inputs"], MCPServerSetupInput);
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
	export class MCPAppsPolicy {
	    enabled: boolean;
	    allowAppInitiatedToolCalls: boolean;
	    requireApprovalForOpenLink: boolean;
	    requireApprovalForContextUpdates: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPAppsPolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.allowAppInitiatedToolCalls = source["allowAppInitiatedToolCalls"];
	        this.requireApprovalForOpenLink = source["requireApprovalForOpenLink"];
	        this.requireApprovalForContextUpdates = source["requireApprovalForContextUpdates"];
	    }
	}
	export class MCPToolPolicyOverride {
	    toolName: string;
	    approvalRule?: string;
	    executionMode?: string;
	    allowStaleDigest?: boolean;
	    expectedDigest?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPToolPolicyOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolName = source["toolName"];
	        this.approvalRule = source["approvalRule"];
	        this.executionMode = source["executionMode"];
	        this.allowStaleDigest = source["allowStaleDigest"];
	        this.expectedDigest = source["expectedDigest"];
	    }
	}
	export class MCPServerPolicy {
	    defaultApprovalRule: string;
	    defaultExecutionMode: string;
	    requireApprovalForUnknownRisk: boolean;
	    requireApprovalForWrite: boolean;
	    requireApprovalForDestructive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerPolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultApprovalRule = source["defaultApprovalRule"];
	        this.defaultExecutionMode = source["defaultExecutionMode"];
	        this.requireApprovalForUnknownRisk = source["requireApprovalForUnknownRisk"];
	        this.requireApprovalForWrite = source["requireApprovalForWrite"];
	        this.requireApprovalForDestructive = source["requireApprovalForDestructive"];
	    }
	}
	export class MCPStreamableHTTPConfig {
	    url: string;
	    timeoutMS?: number;
	    authMode: string;
	    headers?: Record<string, string>;
	    secretHeaderRefs?: Record<string, string>;
	    clientCredentialRef?: string;
	    clientIDMetadataDocumentURL?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPStreamableHTTPConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.timeoutMS = source["timeoutMS"];
	        this.authMode = source["authMode"];
	        this.headers = source["headers"];
	        this.secretHeaderRefs = source["secretHeaderRefs"];
	        this.clientCredentialRef = source["clientCredentialRef"];
	        this.clientIDMetadataDocumentURL = source["clientIDMetadataDocumentURL"];
	    }
	}
	export class MCPStdioConfig {
	    command: string;
	    args?: string[];
	    workingDir?: string;
	    env?: Record<string, string>;
	    secretEnvRefs?: Record<string, string>;
	    startupTimeoutMS?: number;
	
	    static createFrom(source: any = {}) {
	        return new MCPStdioConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.args = source["args"];
	        this.workingDir = source["workingDir"];
	        this.env = source["env"];
	        this.secretEnvRefs = source["secretEnvRefs"];
	        this.startupTimeoutMS = source["startupTimeoutMS"];
	    }
	}
	export class MCPServerConfig {
	    schemaVersion: string;
	    bundleID: string;
	    id: string;
	    displayName: string;
	    enabled: boolean;
	    transport: string;
	    trustLevel: string;
	    stdio?: MCPStdioConfig;
	    streamableHttp?: MCPStreamableHTTPConfig;
	    defaultPolicy: MCPServerPolicy;
	    toolPolicies?: Record<string, MCPToolPolicyOverride>;
	    appsPolicy?: MCPAppsPolicy;
	    setup?: MCPServerSetup;
	    isBuiltIn: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.bundleID = source["bundleID"];
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.enabled = source["enabled"];
	        this.transport = source["transport"];
	        this.trustLevel = source["trustLevel"];
	        this.stdio = this.convertValues(source["stdio"], MCPStdioConfig);
	        this.streamableHttp = this.convertValues(source["streamableHttp"], MCPStreamableHTTPConfig);
	        this.defaultPolicy = this.convertValues(source["defaultPolicy"], MCPServerPolicy);
	        this.toolPolicies = this.convertValues(source["toolPolicies"], MCPToolPolicyOverride, true);
	        this.appsPolicy = this.convertValues(source["appsPolicy"], MCPAppsPolicy);
	        this.setup = this.convertValues(source["setup"], MCPServerSetup);
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
	export class GetMCPServerResponse {
	    Body?: MCPServerConfig;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPServerResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPServerConfig);
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
	export class GetMCPServerStatusRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPServerStatusRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class GetMCPServerStatusResponse {
	    Body?: MCPServerRuntimeSnapshot;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPServerStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPServerRuntimeSnapshot);
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
	export class GetMCPSettingsRequest {
	
	
	    static createFrom(source: any = {}) {
	        return new GetMCPSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class MCPSettings {
	    oauthLoopbackListenAddr?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oauthLoopbackListenAddr = source["oauthLoopbackListenAddr"];
	    }
	}
	export class MCPSettingsView {
	    settings: MCPSettings;
	    oauthRedirectURL?: string;
	    oauthLoopbackListenAddr?: string;
	    oauthRestartRequired?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPSettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], MCPSettings);
	        this.oauthRedirectURL = source["oauthRedirectURL"];
	        this.oauthLoopbackListenAddr = source["oauthLoopbackListenAddr"];
	        this.oauthRestartRequired = source["oauthRestartRequired"];
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
	export class GetMCPSettingsResponse {
	    Body?: MCPSettingsView;
	
	    static createFrom(source: any = {}) {
	        return new GetMCPSettingsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPSettingsView);
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
	export class GetModelPresetRequest {
	    ProviderName: string;
	    ModelPresetID: string;
	    IncludeDisabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GetModelPresetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	        this.ModelPresetID = source["ModelPresetID"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	    }
	}
	export class ModelPreset {
	    stream?: boolean;
	    maxPromptLength?: number;
	    maxOutputLength?: number;
	    temperature?: number;
	    reasoning?: ReasoningParam;
	    systemPrompt?: string;
	    timeout?: number;
	    cacheControl?: CacheControl;
	    outputParam?: OutputParam;
	    stopSequences?: string[];
	    additionalParametersRawJSON?: string;
	    capabilitiesOverride?: capabilityoverride.ModelCapabilitiesOverride;
	    schemaVersion: string;
	    id: string;
	    name: string;
	    displayName: string;
	    slug: string;
	    isEnabled: boolean;
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
	        this.stream = source["stream"];
	        this.maxPromptLength = source["maxPromptLength"];
	        this.maxOutputLength = source["maxOutputLength"];
	        this.temperature = source["temperature"];
	        this.reasoning = this.convertValues(source["reasoning"], ReasoningParam);
	        this.systemPrompt = source["systemPrompt"];
	        this.timeout = source["timeout"];
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.outputParam = this.convertValues(source["outputParam"], OutputParam);
	        this.stopSequences = source["stopSequences"];
	        this.additionalParametersRawJSON = source["additionalParametersRawJSON"];
	        this.capabilitiesOverride = this.convertValues(source["capabilitiesOverride"], capabilityoverride.ModelCapabilitiesOverride);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.slug = source["slug"];
	        this.isEnabled = source["isEnabled"];
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
	    capabilitiesOverride?: capabilityoverride.ModelCapabilitiesOverride;
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
	        this.capabilitiesOverride = this.convertValues(source["capabilitiesOverride"], capabilityoverride.ModelCapabilitiesOverride);
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
	export class GetModelPresetResponseBody {
	    provider: ProviderPreset;
	    model: ModelPreset;
	
	    static createFrom(source: any = {}) {
	        return new GetModelPresetResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = this.convertValues(source["provider"], ProviderPreset);
	        this.model = this.convertValues(source["model"], ModelPreset);
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
	export class GetModelPresetResponse {
	    Body?: GetModelPresetResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new GetModelPresetResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], GetModelPresetResponseBody);
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
	    debug: DebugSettings;
	    authKeys: AuthKeyMeta[];
	
	    static createFrom(source: any = {}) {
	        return new GetSettingsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appTheme = this.convertValues(source["appTheme"], AppTheme);
	        this.debug = this.convertValues(source["debug"], DebugSettings);
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
	
	export class GetSkillRequest {
	    BundleID: string;
	    SkillSlug: string;
	    IncludeDisabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GetSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.SkillSlug = source["SkillSlug"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	    }
	}
	export class SkillPresence {
	    status: string;
	    // Go type: time
	    lastCheckedAt?: any;
	    // Go type: time
	    lastSeenAt?: any;
	    // Go type: time
	    missingSince?: any;
	    lastCheckError?: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillPresence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.lastCheckedAt = this.convertValues(source["lastCheckedAt"], null);
	        this.lastSeenAt = this.convertValues(source["lastSeenAt"], null);
	        this.missingSince = this.convertValues(source["missingSince"], null);
	        this.lastCheckError = source["lastCheckError"];
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
	export class SkillResourceInfo {
	    hasResources: boolean;
	    totalCount: number;
	    locations?: string[];
	    moreLocations: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SkillResourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasResources = source["hasResources"];
	        this.totalCount = source["totalCount"];
	        this.locations = source["locations"];
	        this.moreLocations = source["moreLocations"];
	    }
	}
	export class SkillArgument {
	    name: string;
	    description?: string;
	    default?: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillArgument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.default = source["default"];
	    }
	}
	export class Skill {
	    schemaVersion: string;
	    id: string;
	    slug: string;
	    type: string;
	    location: string;
	    name: string;
	    displayName?: string;
	    description?: string;
	    tags?: string[];
	    insert?: string;
	    arguments?: SkillArgument[];
	    resources: SkillResourceInfo;
	    rawFrontmatter?: Record<string, any>;
	    runtimeWarnings?: string[];
	    digest?: string;
	    presence?: SkillPresence;
	    isEnabled: boolean;
	    isBuiltIn: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.type = source["type"];
	        this.location = source["location"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], SkillArgument);
	        this.resources = this.convertValues(source["resources"], SkillResourceInfo);
	        this.rawFrontmatter = source["rawFrontmatter"];
	        this.runtimeWarnings = source["runtimeWarnings"];
	        this.digest = source["digest"];
	        this.presence = this.convertValues(source["presence"], SkillPresence);
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
	export class GetSkillResponse {
	    Body?: Skill;
	
	    static createFrom(source: any = {}) {
	        return new GetSkillResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], Skill);
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
	export class RuntimeSkillFilter {
	    types?: string[];
	    inserts?: string[];
	    locationPrefix?: string;
	    allowSkillRefs?: SkillRef[];
	    sessionID?: string;
	    activity?: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeSkillFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.types = source["types"];
	        this.inserts = source["inserts"];
	        this.locationPrefix = source["locationPrefix"];
	        this.allowSkillRefs = this.convertValues(source["allowSkillRefs"], SkillRef);
	        this.sessionID = source["sessionID"];
	        this.activity = source["activity"];
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
	export class GetSkillsPromptRequestBody {
	    filter?: RuntimeSkillFilter;
	
	    static createFrom(source: any = {}) {
	        return new GetSkillsPromptRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = this.convertValues(source["filter"], RuntimeSkillFilter);
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
	export class GetSkillsPromptRequest {
	    Body?: GetSkillsPromptRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new GetSkillsPromptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], GetSkillsPromptRequestBody);
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
	
	export class GetSkillsPromptResponseBody {
	    prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new GetSkillsPromptResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prompt = source["prompt"];
	    }
	}
	export class GetSkillsPromptResponse {
	    Body?: GetSkillsPromptResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new GetSkillsPromptResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], GetSkillsPromptResponseBody);
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
	export class InvokeMCPToolRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: InvokeMCPToolRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new InvokeMCPToolRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], InvokeMCPToolRequestBody);
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
	
	export class MCPToolAppRenderInfo {
	    resourceUri?: string;
	    mimeType?: string;
	    content?: MCPContent[];
	    structuredContent?: any;
	    isError?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPToolAppRenderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resourceUri = source["resourceUri"];
	        this.mimeType = source["mimeType"];
	        this.content = this.convertValues(source["content"], MCPContent);
	        this.structuredContent = source["structuredContent"];
	        this.isError = source["isError"];
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
	export class MCPToolCallProvenance {
	    bundleID: string;
	    serverID: string;
	    serverDisplayName?: string;
	    toolName: string;
	    providerToolName: string;
	    toolDigest?: string;
	    toolUseID?: string;
	    approvalID?: string;
	    appResourceUri?: string;
	    appInstanceID?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPToolCallProvenance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.serverDisplayName = source["serverDisplayName"];
	        this.toolName = source["toolName"];
	        this.providerToolName = source["providerToolName"];
	        this.toolDigest = source["toolDigest"];
	        this.toolUseID = source["toolUseID"];
	        this.approvalID = source["approvalID"];
	        this.appResourceUri = source["appResourceUri"];
	        this.appInstanceID = source["appInstanceID"];
	    }
	}
	export class InvokeMCPToolResponseBody {
	    bundleID: string;
	    serverID: string;
	    toolName: string;
	    providerToolName?: string;
	    content?: MCPContent[];
	    structuredContent?: any;
	    isError?: boolean;
	    provenance: MCPToolCallProvenance;
	    app?: MCPToolAppRenderInfo;
	
	    static createFrom(source: any = {}) {
	        return new InvokeMCPToolResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.toolName = source["toolName"];
	        this.providerToolName = source["providerToolName"];
	        this.content = this.convertValues(source["content"], MCPContent);
	        this.structuredContent = source["structuredContent"];
	        this.isError = source["isError"];
	        this.provenance = this.convertValues(source["provenance"], MCPToolCallProvenance);
	        this.app = this.convertValues(source["app"], MCPToolAppRenderInfo);
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
	export class InvokeMCPToolResponse {
	    Body?: InvokeMCPToolResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new InvokeMCPToolResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], InvokeMCPToolResponseBody);
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
	
	export class InvokeSkillToolRequestBody {
	    sessionID: string;
	    toolName: string;
	    args?: string;
	
	    static createFrom(source: any = {}) {
	        return new InvokeSkillToolRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionID = source["sessionID"];
	        this.toolName = source["toolName"];
	        this.args = source["args"];
	    }
	}
	export class InvokeSkillToolRequest {
	    Body?: InvokeSkillToolRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new InvokeSkillToolRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], InvokeSkillToolRequestBody);
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
	
	export class ToolOutputFile {
	    fileName: string;
	    fileMIME: string;
	    fileData: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolOutputFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.fileMIME = source["fileMIME"];
	        this.fileData = source["fileData"];
	    }
	}
	export class ToolOutputImage {
	    detail: string;
	    imageName: string;
	    imageMIME: string;
	    imageData: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolOutputImage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detail = source["detail"];
	        this.imageName = source["imageName"];
	        this.imageMIME = source["imageMIME"];
	        this.imageData = source["imageData"];
	    }
	}
	export class ToolOutputText {
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolOutputText(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	    }
	}
	export class ToolOutputUnion {
	    kind: string;
	    textItem?: ToolOutputText;
	    imageItem?: ToolOutputImage;
	    fileItem?: ToolOutputFile;
	
	    static createFrom(source: any = {}) {
	        return new ToolOutputUnion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.textItem = this.convertValues(source["textItem"], ToolOutputText);
	        this.imageItem = this.convertValues(source["imageItem"], ToolOutputImage);
	        this.fileItem = this.convertValues(source["fileItem"], ToolOutputFile);
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
	export class InvokeSkillToolResponseBody {
	    outputs?: ToolOutputUnion[];
	    meta?: Record<string, any>;
	    isBuiltIn: boolean;
	    isError?: boolean;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new InvokeSkillToolResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputs = this.convertValues(source["outputs"], ToolOutputUnion);
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
	export class InvokeSkillToolResponse {
	    Body?: InvokeSkillToolResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new InvokeSkillToolResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], InvokeSkillToolResponseBody);
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
	
	export class InvokeToolResponseBody {
	    outputs?: ToolOutputUnion[];
	    meta?: Record<string, any>;
	    isBuiltIn: boolean;
	    isError: boolean;
	    errorMessage: string;
	
	    static createFrom(source: any = {}) {
	        return new InvokeToolResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputs = this.convertValues(source["outputs"], ToolOutputUnion);
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
	
	
	export class ListAssistantPresetBundlesRequest {
	    BundleIDs: string[];
	    IncludeDisabled: boolean;
	    PageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListAssistantPresetBundlesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class ListAssistantPresetBundlesResponseBody {
	    assistantPresetBundles: AssistantPresetBundle[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListAssistantPresetBundlesResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assistantPresetBundles = this.convertValues(source["assistantPresetBundles"], AssistantPresetBundle);
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
	export class ListAssistantPresetBundlesResponse {
	    Body?: ListAssistantPresetBundlesResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListAssistantPresetBundlesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListAssistantPresetBundlesResponseBody);
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
	
	export class ListAssistantPresetsRequest {
	    BundleIDs: string[];
	    IncludeDisabled: boolean;
	    RecommendedPageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListAssistantPresetsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.RecommendedPageSize = source["RecommendedPageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class ListAssistantPresetsResponseBody {
	    assistantPresetListItems: AssistantPresetListItem[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListAssistantPresetsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assistantPresetListItems = this.convertValues(source["assistantPresetListItems"], AssistantPresetListItem);
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
	export class ListAssistantPresetsResponse {
	    Body?: ListAssistantPresetsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListAssistantPresetsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListAssistantPresetsResponseBody);
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
	
	export class ListMCPBundlesRequest {
	    BundleIDs: string[];
	    IncludeDisabled: boolean;
	    PageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPBundlesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class MCPBundle {
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
	        return new MCPBundle(source);
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
	export class ListMCPBundlesResponseBody {
	    bundles: MCPBundle[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPBundlesResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundles = this.convertValues(source["bundles"], MCPBundle);
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
	export class ListMCPBundlesResponse {
	    Body?: ListMCPBundlesResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPBundlesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListMCPBundlesResponseBody);
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
	
	export class ListMCPServerPromptsRequest {
	    BundleID: string;
	    ServerID: string;
	    PageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerPromptsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class MCPPromptRef {
	    bundleID: string;
	    serverID: string;
	    promptName: string;
	    title?: string;
	    displayName: string;
	    description?: string;
	    arguments?: Record<string, MCPArgumentDefinition>;
	    digest?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPPromptRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.promptName = source["promptName"];
	        this.title = source["title"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.arguments = this.convertValues(source["arguments"], MCPArgumentDefinition, true);
	        this.digest = source["digest"];
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
	export class ListMCPServerPromptsResponseBody {
	    prompts: MCPPromptRef[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerPromptsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prompts = this.convertValues(source["prompts"], MCPPromptRef);
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
	export class ListMCPServerPromptsResponse {
	    Body?: ListMCPServerPromptsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerPromptsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListMCPServerPromptsResponseBody);
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
	
	export class ListMCPServerResourceTemplatesRequest {
	    BundleID: string;
	    ServerID: string;
	    PageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerResourceTemplatesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class MCPResourceTemplateRef {
	    bundleID: string;
	    serverID: string;
	    uriTemplate: string;
	    name?: string;
	    title?: string;
	    displayName: string;
	    description?: string;
	    mimeType?: string;
	    arguments?: Record<string, MCPArgumentDefinition>;
	    annotations?: Record<string, any>;
	    digest?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourceTemplateRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.uriTemplate = source["uriTemplate"];
	        this.name = source["name"];
	        this.title = source["title"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.mimeType = source["mimeType"];
	        this.arguments = this.convertValues(source["arguments"], MCPArgumentDefinition, true);
	        this.annotations = source["annotations"];
	        this.digest = source["digest"];
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
	export class ListMCPServerResourceTemplatesResponseBody {
	    resourceTemplates: MCPResourceTemplateRef[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerResourceTemplatesResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resourceTemplates = this.convertValues(source["resourceTemplates"], MCPResourceTemplateRef);
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
	export class ListMCPServerResourceTemplatesResponse {
	    Body?: ListMCPServerResourceTemplatesResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerResourceTemplatesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListMCPServerResourceTemplatesResponseBody);
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
	
	export class ListMCPServerResourcesRequest {
	    BundleID: string;
	    ServerID: string;
	    PageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerResourcesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class ListMCPServerResourcesResponseBody {
	    resources: MCPResourceRef[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerResourcesResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resources = this.convertValues(source["resources"], MCPResourceRef);
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
	export class ListMCPServerResourcesResponse {
	    Body?: ListMCPServerResourcesResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerResourcesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListMCPServerResourcesResponseBody);
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
	
	export class ListMCPServerToolsRequest {
	    BundleID: string;
	    ServerID: string;
	    PageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerToolsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class MCPToolAppInfo {
	    resourceUri?: string;
	    visibility?: string[];
	
	    static createFrom(source: any = {}) {
	        return new MCPToolAppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resourceUri = source["resourceUri"];
	        this.visibility = source["visibility"];
	    }
	}
	export class MCPToolAnnotations {
	    destructiveHint?: boolean;
	    idempotentHint: boolean;
	    openWorldHint?: boolean;
	    readOnlyHint: boolean;
	    title?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPToolAnnotations(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.destructiveHint = source["destructiveHint"];
	        this.idempotentHint = source["idempotentHint"];
	        this.openWorldHint = source["openWorldHint"];
	        this.readOnlyHint = source["readOnlyHint"];
	        this.title = source["title"];
	    }
	}
	export class MCPToolCapability {
	    bundleID: string;
	    serverID: string;
	    toolName: string;
	    providerToolName: string;
	    choiceID: string;
	    title?: string;
	    displayName: string;
	    description?: string;
	    inputSchema?: Record<string, any>;
	    outputSchema?: Record<string, any>;
	    annotations?: MCPToolAnnotations;
	    inferredRisk: string;
	    approvalRule: string;
	    executionMode: string;
	    taskSupport: string;
	    app?: MCPToolAppInfo;
	    digest: string;
	    enabled: boolean;
	    stale?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPToolCapability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.toolName = source["toolName"];
	        this.providerToolName = source["providerToolName"];
	        this.choiceID = source["choiceID"];
	        this.title = source["title"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.inputSchema = source["inputSchema"];
	        this.outputSchema = source["outputSchema"];
	        this.annotations = this.convertValues(source["annotations"], MCPToolAnnotations);
	        this.inferredRisk = source["inferredRisk"];
	        this.approvalRule = source["approvalRule"];
	        this.executionMode = source["executionMode"];
	        this.taskSupport = source["taskSupport"];
	        this.app = this.convertValues(source["app"], MCPToolAppInfo);
	        this.digest = source["digest"];
	        this.enabled = source["enabled"];
	        this.stale = source["stale"];
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
	export class ListMCPServerToolsResponseBody {
	    tools: MCPToolCapability[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerToolsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tools = this.convertValues(source["tools"], MCPToolCapability);
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
	export class ListMCPServerToolsResponse {
	    Body?: ListMCPServerToolsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServerToolsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListMCPServerToolsResponseBody);
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
	
	export class ListMCPServersRequest {
	    BundleID: string;
	    ServerIDs: string[];
	    Enabled?: boolean;
	    IncludeDisabled: boolean;
	    PageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServersRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerIDs = source["ServerIDs"];
	        this.Enabled = source["Enabled"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class ListMCPServersResponseBody {
	    servers: MCPServerConfig[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServersResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], MCPServerConfig);
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
	export class ListMCPServersResponse {
	    Body?: ListMCPServersResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListMCPServersResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListMCPServersResponseBody);
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
	
	export class ListPendingMCPOAuthAuthorizationsRequest {
	
	
	    static createFrom(source: any = {}) {
	        return new ListPendingMCPOAuthAuthorizationsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class MCPOAuthAuthorization {
	    bundleID: string;
	    serverID: string;
	    authorizationURL: string;
	    expiresAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPOAuthAuthorization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.authorizationURL = source["authorizationURL"];
	        this.expiresAt = source["expiresAt"];
	    }
	}
	export class ListPendingMCPOAuthAuthorizationsResponseBody {
	    authorizations: MCPOAuthAuthorization[];
	
	    static createFrom(source: any = {}) {
	        return new ListPendingMCPOAuthAuthorizationsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authorizations = this.convertValues(source["authorizations"], MCPOAuthAuthorization);
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
	export class ListPendingMCPOAuthAuthorizationsResponse {
	    Body?: ListPendingMCPOAuthAuthorizationsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListPendingMCPOAuthAuthorizationsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListPendingMCPOAuthAuthorizationsResponseBody);
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
	
	export class ListRuntimeSkillsRequestBody {
	    filter?: RuntimeSkillFilter;
	
	    static createFrom(source: any = {}) {
	        return new ListRuntimeSkillsRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = this.convertValues(source["filter"], RuntimeSkillFilter);
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
	export class ListRuntimeSkillsRequest {
	    Body?: ListRuntimeSkillsRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new ListRuntimeSkillsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListRuntimeSkillsRequestBody);
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
	
	export class RuntimeSkillListItem {
	    skillRef: SkillRef;
	    type?: string;
	    name?: string;
	    displayName?: string;
	    description?: string;
	    digest?: string;
	    insert?: string;
	    arguments?: SkillArgument[];
	    sourceTags?: string[];
	    resources: SkillResourceInfo;
	    rawFrontmatter?: Record<string, any>;
	    warnings?: string[];
	    isActive?: boolean;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeSkillListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillRef = this.convertValues(source["skillRef"], SkillRef);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.digest = source["digest"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], SkillArgument);
	        this.sourceTags = source["sourceTags"];
	        this.resources = this.convertValues(source["resources"], SkillResourceInfo);
	        this.rawFrontmatter = source["rawFrontmatter"];
	        this.warnings = source["warnings"];
	        this.isActive = source["isActive"];
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
	export class ListRuntimeSkillsResponseBody {
	    skills: RuntimeSkillListItem[];
	
	    static createFrom(source: any = {}) {
	        return new ListRuntimeSkillsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skills = this.convertValues(source["skills"], RuntimeSkillListItem);
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
	export class ListRuntimeSkillsResponse {
	    Body?: ListRuntimeSkillsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListRuntimeSkillsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListRuntimeSkillsResponseBody);
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
	
	export class ListSkillBundlesRequest {
	    BundleIDs: string[];
	    IncludeDisabled: boolean;
	    PageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListSkillBundlesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.PageSize = source["PageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class SkillBundle {
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
	        return new SkillBundle(source);
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
	export class ListSkillBundlesResponseBody {
	    skillBundles: SkillBundle[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListSkillBundlesResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillBundles = this.convertValues(source["skillBundles"], SkillBundle);
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
	export class ListSkillBundlesResponse {
	    Body?: ListSkillBundlesResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListSkillBundlesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListSkillBundlesResponseBody);
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
	
	export class ListSkillsRequest {
	    BundleIDs: string[];
	    Types: string[];
	    Inserts: string[];
	    Tags: string[];
	    IncludeDisabled: boolean;
	    IncludeMissing: boolean;
	    RecommendedPageSize: number;
	    PageToken: string;
	
	    static createFrom(source: any = {}) {
	        return new ListSkillsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleIDs = source["BundleIDs"];
	        this.Types = source["Types"];
	        this.Inserts = source["Inserts"];
	        this.Tags = source["Tags"];
	        this.IncludeDisabled = source["IncludeDisabled"];
	        this.IncludeMissing = source["IncludeMissing"];
	        this.RecommendedPageSize = source["RecommendedPageSize"];
	        this.PageToken = source["PageToken"];
	    }
	}
	export class SkillListItem {
	    bundleID: string;
	    bundleSlug: string;
	    skillSlug: string;
	    isBuiltIn: boolean;
	    skillDefinition: Skill;
	
	    static createFrom(source: any = {}) {
	        return new SkillListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.bundleSlug = source["bundleSlug"];
	        this.skillSlug = source["skillSlug"];
	        this.isBuiltIn = source["isBuiltIn"];
	        this.skillDefinition = this.convertValues(source["skillDefinition"], Skill);
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
	export class ListSkillsResponseBody {
	    skillListItems: SkillListItem[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListSkillsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillListItems = this.convertValues(source["skillListItems"], SkillListItem);
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
	export class ListSkillsResponse {
	    Body?: ListSkillsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListSkillsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListSkillsResponseBody);
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
	
	
	
	
	export class MCPApprovalToken {
	    approvalID: string;
	    token: string;
	    expiresAt: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPApprovalToken(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.approvalID = source["approvalID"];
	        this.token = source["token"];
	        this.expiresAt = source["expiresAt"];
	    }
	}
	
	
	
	
	
	export class MCPCompleteArgumentRequestBody {
	    refType: string;
	    name: string;
	    argumentName: string;
	    argumentValue?: string;
	    context?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPCompleteArgumentRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.refType = source["refType"];
	        this.name = source["name"];
	        this.argumentName = source["argumentName"];
	        this.argumentValue = source["argumentValue"];
	        this.context = source["context"];
	    }
	}
	export class MCPCompleteArgumentRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: MCPCompleteArgumentRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new MCPCompleteArgumentRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], MCPCompleteArgumentRequestBody);
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
	
	export class MCPCompletionResult {
	    values?: string[];
	    total?: number;
	    hasMore?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPCompletionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.values = source["values"];
	        this.total = source["total"];
	        this.hasMore = source["hasMore"];
	    }
	}
	
	
	export class MCPGetPromptRequestBody {
	    promptName: string;
	    arguments?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPGetPromptRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptName = source["promptName"];
	        this.arguments = source["arguments"];
	    }
	}
	export class MCPGetPromptRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: MCPGetPromptRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new MCPGetPromptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], MCPGetPromptRequestBody);
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
	
	export class MCPPromptMessage {
	    role: string;
	    content: MCPContent;
	
	    static createFrom(source: any = {}) {
	        return new MCPPromptMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = this.convertValues(source["content"], MCPContent);
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
	export class MCPGetPromptResponseBody {
	    bundleID: string;
	    serverID: string;
	    promptName: string;
	    description?: string;
	    messages?: MCPPromptMessage[];
	
	    static createFrom(source: any = {}) {
	        return new MCPGetPromptResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.promptName = source["promptName"];
	        this.description = source["description"];
	        this.messages = this.convertValues(source["messages"], MCPPromptMessage);
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
	export class MCPGetPromptResponse {
	    Body?: MCPGetPromptResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new MCPGetPromptResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPGetPromptResponseBody);
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
	
	
	
	
	
	
	
	export class MCPReadResourceRequestBody {
	    uri: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPReadResourceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uri = source["uri"];
	    }
	}
	export class MCPReadResourceRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: MCPReadResourceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new MCPReadResourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], MCPReadResourceRequestBody);
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
	
	export class MCPReadResourceResponseBody {
	    bundleID: string;
	    serverID: string;
	    uri: string;
	    contents?: MCPContent[];
	
	    static createFrom(source: any = {}) {
	        return new MCPReadResourceResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundleID = source["bundleID"];
	        this.serverID = source["serverID"];
	        this.uri = source["uri"];
	        this.contents = this.convertValues(source["contents"], MCPContent);
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
	export class MCPReadResourceResponse {
	    Body?: MCPReadResourceResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new MCPReadResourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPReadResourceResponseBody);
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
	
	
	
	
	
	
	
	
	
	
	
	
	export class MCPServerSetupInputValue {
	    value?: string;
	    clientID?: string;
	    clientSecret?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerSetupInputValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.clientID = source["clientID"];
	        this.clientSecret = source["clientSecret"];
	    }
	}
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	export class PatchAssistantPresetBundleRequestBody {
	    isEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PatchAssistantPresetBundleRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchAssistantPresetBundleRequest {
	    BundleID: string;
	    Body?: PatchAssistantPresetBundleRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PatchAssistantPresetBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PatchAssistantPresetBundleRequestBody);
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
	
	export class PatchAssistantPresetBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PatchAssistantPresetBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PatchAssistantPresetRequestBody {
	    isEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PatchAssistantPresetRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchAssistantPresetRequest {
	    BundleID: string;
	    AssistantPresetSlug: string;
	    Version: string;
	    Body?: PatchAssistantPresetRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PatchAssistantPresetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.AssistantPresetSlug = source["AssistantPresetSlug"];
	        this.Version = source["Version"];
	        this.Body = this.convertValues(source["Body"], PatchAssistantPresetRequestBody);
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
	
	export class PatchAssistantPresetResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PatchAssistantPresetResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
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
	export class PatchMCPBundleRequestBody {
	    isEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPBundleRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchMCPBundleRequest {
	    BundleID: string;
	    Body?: PatchMCPBundleRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PatchMCPBundleRequestBody);
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
	
	export class PatchMCPBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PatchMCPServerEnabledRequestBody {
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerEnabledRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	    }
	}
	export class PatchMCPServerEnabledRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: PatchMCPServerEnabledRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerEnabledRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], PatchMCPServerEnabledRequestBody);
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
	
	export class PatchMCPServerEnabledResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerEnabledResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PatchMCPServerPolicyPayload {
	    defaultPolicy?: MCPServerPolicy;
	    toolPolicies?: Record<string, MCPToolPolicyOverride>;
	    appsPolicy?: MCPAppsPolicy;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerPolicyPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultPolicy = this.convertValues(source["defaultPolicy"], MCPServerPolicy);
	        this.toolPolicies = this.convertValues(source["toolPolicies"], MCPToolPolicyOverride, true);
	        this.appsPolicy = this.convertValues(source["appsPolicy"], MCPAppsPolicy);
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
	export class PatchMCPServerPolicyRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: PatchMCPServerPolicyPayload;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerPolicyRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], PatchMCPServerPolicyPayload);
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
	export class PatchMCPServerPolicyResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerPolicyResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PatchMCPServerSetupRequestBody {
	    reset?: boolean;
	    inputValues?: Record<string, MCPServerSetupInputValue>;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerSetupRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reset = source["reset"];
	        this.inputValues = this.convertValues(source["inputValues"], MCPServerSetupInputValue, true);
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
	export class PatchMCPServerSetupRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: PatchMCPServerSetupRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerSetupRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], PatchMCPServerSetupRequestBody);
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
	
	export class PatchMCPServerSetupResponse {
	    Body?: MCPServerConfig;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPServerSetupResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPServerConfig);
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
	export class PatchMCPSettingsRequestBody {
	    oauthLoopbackListenAddr?: string;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPSettingsRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oauthLoopbackListenAddr = source["oauthLoopbackListenAddr"];
	    }
	}
	export class PatchMCPSettingsRequest {
	    Body?: PatchMCPSettingsRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], PatchMCPSettingsRequestBody);
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
	
	export class PatchMCPSettingsResponse {
	    Body?: MCPSettingsView;
	
	    static createFrom(source: any = {}) {
	        return new PatchMCPSettingsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPSettingsView);
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
	export class PatchModelPresetRequestBody {
	    stream?: boolean;
	    maxPromptLength?: number;
	    maxOutputLength?: number;
	    temperature?: number;
	    reasoning?: ReasoningParam;
	    systemPrompt?: string;
	    timeout?: number;
	    cacheControl?: CacheControl;
	    outputParam?: OutputParam;
	    stopSequences?: string[];
	    additionalParametersRawJSON?: string;
	    capabilitiesOverride?: capabilityoverride.ModelCapabilitiesOverride;
	    name?: string;
	    slug?: string;
	    displayName?: string;
	    isEnabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PatchModelPresetRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stream = source["stream"];
	        this.maxPromptLength = source["maxPromptLength"];
	        this.maxOutputLength = source["maxOutputLength"];
	        this.temperature = source["temperature"];
	        this.reasoning = this.convertValues(source["reasoning"], ReasoningParam);
	        this.systemPrompt = source["systemPrompt"];
	        this.timeout = source["timeout"];
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.outputParam = this.convertValues(source["outputParam"], OutputParam);
	        this.stopSequences = source["stopSequences"];
	        this.additionalParametersRawJSON = source["additionalParametersRawJSON"];
	        this.capabilitiesOverride = this.convertValues(source["capabilitiesOverride"], capabilityoverride.ModelCapabilitiesOverride);
	        this.name = source["name"];
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.isEnabled = source["isEnabled"];
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
	export class PatchProviderPresetRequestBody {
	    displayName?: string;
	    sdkType?: string;
	    isEnabled?: boolean;
	    origin?: string;
	    chatCompletionPathPrefix?: string;
	    apiKeyHeaderKey?: string;
	    defaultHeaders?: Record<string, string>;
	    defaultModelPresetID?: string;
	    capabilitiesOverride?: capabilityoverride.ModelCapabilitiesOverride;
	
	    static createFrom(source: any = {}) {
	        return new PatchProviderPresetRequestBody(source);
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
	        this.defaultModelPresetID = source["defaultModelPresetID"];
	        this.capabilitiesOverride = this.convertValues(source["capabilitiesOverride"], capabilityoverride.ModelCapabilitiesOverride);
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
	export class PatchSkillBundleRequestBody {
	    isEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PatchSkillBundleRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PatchSkillBundleRequest {
	    BundleID: string;
	    Body?: PatchSkillBundleRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PatchSkillBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PatchSkillBundleRequestBody);
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
	
	export class PatchSkillBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PatchSkillBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PatchSkillRequestBody {
	    isEnabled?: boolean;
	    location?: string;
	    displayName?: string;
	    description?: string;
	    tags?: string[];
	
	    static createFrom(source: any = {}) {
	        return new PatchSkillRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isEnabled = source["isEnabled"];
	        this.location = source["location"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	    }
	}
	export class PatchSkillRequest {
	    BundleID: string;
	    SkillSlug: string;
	    Body?: PatchSkillRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PatchSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.SkillSlug = source["SkillSlug"];
	        this.Body = this.convertValues(source["Body"], PatchSkillRequestBody);
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
	
	export class PatchSkillResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PatchSkillResponse(source);
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
	export class PostModelPresetRequestBody {
	    stream?: boolean;
	    maxPromptLength?: number;
	    maxOutputLength?: number;
	    temperature?: number;
	    reasoning?: ReasoningParam;
	    systemPrompt?: string;
	    timeout?: number;
	    cacheControl?: CacheControl;
	    outputParam?: OutputParam;
	    stopSequences?: string[];
	    additionalParametersRawJSON?: string;
	    capabilitiesOverride?: capabilityoverride.ModelCapabilitiesOverride;
	    name: string;
	    slug: string;
	    displayName: string;
	    isEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PostModelPresetRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stream = source["stream"];
	        this.maxPromptLength = source["maxPromptLength"];
	        this.maxOutputLength = source["maxOutputLength"];
	        this.temperature = source["temperature"];
	        this.reasoning = this.convertValues(source["reasoning"], ReasoningParam);
	        this.systemPrompt = source["systemPrompt"];
	        this.timeout = source["timeout"];
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.outputParam = this.convertValues(source["outputParam"], OutputParam);
	        this.stopSequences = source["stopSequences"];
	        this.additionalParametersRawJSON = source["additionalParametersRawJSON"];
	        this.capabilitiesOverride = this.convertValues(source["capabilitiesOverride"], capabilityoverride.ModelCapabilitiesOverride);
	        this.name = source["name"];
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.isEnabled = source["isEnabled"];
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
	export class PostModelPresetRequest {
	    ProviderName: string;
	    ModelPresetID: string;
	    Body?: PostModelPresetRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PostModelPresetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	        this.ModelPresetID = source["ModelPresetID"];
	        this.Body = this.convertValues(source["Body"], PostModelPresetRequestBody);
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
	
	export class PostModelPresetResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PostModelPresetResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PostProviderPresetRequestBody {
	    displayName: string;
	    sdkType: string;
	    isEnabled: boolean;
	    origin: string;
	    chatCompletionPathPrefix: string;
	    apiKeyHeaderKey?: string;
	    defaultHeaders?: Record<string, string>;
	    capabilitiesOverride?: capabilityoverride.ModelCapabilitiesOverride;
	
	    static createFrom(source: any = {}) {
	        return new PostProviderPresetRequestBody(source);
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
	        this.capabilitiesOverride = this.convertValues(source["capabilitiesOverride"], capabilityoverride.ModelCapabilitiesOverride);
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
	export class PostProviderPresetRequest {
	    ProviderName: string;
	    Body?: PostProviderPresetRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PostProviderPresetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProviderName = source["ProviderName"];
	        this.Body = this.convertValues(source["Body"], PostProviderPresetRequestBody);
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
	
	export class PostProviderPresetResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PostProviderPresetResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	
	export class PutAssistantPresetBundleRequestBody {
	    slug: string;
	    displayName: string;
	    description?: string;
	    isEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PutAssistantPresetBundleRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.isEnabled = source["isEnabled"];
	    }
	}
	export class PutAssistantPresetBundleRequest {
	    BundleID: string;
	    Body?: PutAssistantPresetBundleRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PutAssistantPresetBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PutAssistantPresetBundleRequestBody);
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
	
	export class PutAssistantPresetBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PutAssistantPresetBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PutAssistantPresetRequestBody {
	    displayName: string;
	    description?: string;
	    isEnabled: boolean;
	    startingText?: string;
	    startingModelPresetRef?: ModelPresetRef;
	    startingIncludeModelSystemPrompt?: boolean;
	    startingToolSelections?: ToolSelection[];
	    startingSkillSelections?: SkillSelection[];
	    startingMCPContext?: MCPConversationContext;
	
	    static createFrom(source: any = {}) {
	        return new PutAssistantPresetRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.isEnabled = source["isEnabled"];
	        this.startingText = source["startingText"];
	        this.startingModelPresetRef = this.convertValues(source["startingModelPresetRef"], ModelPresetRef);
	        this.startingIncludeModelSystemPrompt = source["startingIncludeModelSystemPrompt"];
	        this.startingToolSelections = this.convertValues(source["startingToolSelections"], ToolSelection);
	        this.startingSkillSelections = this.convertValues(source["startingSkillSelections"], SkillSelection);
	        this.startingMCPContext = this.convertValues(source["startingMCPContext"], MCPConversationContext);
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
	export class PutAssistantPresetRequest {
	    BundleID: string;
	    AssistantPresetSlug: string;
	    Version: string;
	    Body?: PutAssistantPresetRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PutAssistantPresetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.AssistantPresetSlug = source["AssistantPresetSlug"];
	        this.Version = source["Version"];
	        this.Body = this.convertValues(source["Body"], PutAssistantPresetRequestBody);
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
	
	export class PutAssistantPresetResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PutAssistantPresetResponse(source);
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
	export class PutMCPBundleRequestBody {
	    slug: string;
	    displayName: string;
	    isEnabled: boolean;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new PutMCPBundleRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.isEnabled = source["isEnabled"];
	        this.description = source["description"];
	    }
	}
	export class PutMCPBundleRequest {
	    BundleID: string;
	    Body?: PutMCPBundleRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PutMCPBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PutMCPBundleRequestBody);
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
	
	export class PutMCPBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PutMCPBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PutMCPServerPayload {
	    displayName: string;
	    enabled: boolean;
	    transport: string;
	    trustLevel?: string;
	    stdio?: MCPStdioConfig;
	    streamableHttp?: MCPStreamableHTTPConfig;
	    defaultPolicy?: MCPServerPolicy;
	    toolPolicies?: Record<string, MCPToolPolicyOverride>;
	    appsPolicy?: MCPAppsPolicy;
	    setup?: MCPServerSetup;
	
	    static createFrom(source: any = {}) {
	        return new PutMCPServerPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.enabled = source["enabled"];
	        this.transport = source["transport"];
	        this.trustLevel = source["trustLevel"];
	        this.stdio = this.convertValues(source["stdio"], MCPStdioConfig);
	        this.streamableHttp = this.convertValues(source["streamableHttp"], MCPStreamableHTTPConfig);
	        this.defaultPolicy = this.convertValues(source["defaultPolicy"], MCPServerPolicy);
	        this.toolPolicies = this.convertValues(source["toolPolicies"], MCPToolPolicyOverride, true);
	        this.appsPolicy = this.convertValues(source["appsPolicy"], MCPAppsPolicy);
	        this.setup = this.convertValues(source["setup"], MCPServerSetup);
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
	export class PutMCPServerRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: PutMCPServerPayload;
	
	    static createFrom(source: any = {}) {
	        return new PutMCPServerRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], PutMCPServerPayload);
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
	export class PutMCPServerResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PutMCPServerResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PutMCPServerSecretRequestBody {
	    kind: string;
	    slot: string;
	    secret: string;
	
	    static createFrom(source: any = {}) {
	        return new PutMCPServerSecretRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.slot = source["slot"];
	        this.secret = source["secret"];
	    }
	}
	export class PutMCPServerSecretRequest {
	    BundleID: string;
	    ServerID: string;
	    Body?: PutMCPServerSecretRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PutMCPServerSecretRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	        this.Body = this.convertValues(source["Body"], PutMCPServerSecretRequestBody);
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
	
	export class PutMCPServerSecretResponseBody {
	    secretRef: string;
	    sha256?: string;
	    nonEmpty: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PutMCPServerSecretResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.secretRef = source["secretRef"];
	        this.sha256 = source["sha256"];
	        this.nonEmpty = source["nonEmpty"];
	    }
	}
	export class PutMCPServerSecretResponse {
	    Body?: PutMCPServerSecretResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new PutMCPServerSecretResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], PutMCPServerSecretResponseBody);
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
	export class PutSkillArtifactRequestBody {
	    name?: string;
	    isEnabled: boolean;
	    displayName?: string;
	    description?: string;
	    insert?: string;
	    arguments?: SkillArgument[];
	    tags?: string[];
	    markdownBody: string;
	
	    static createFrom(source: any = {}) {
	        return new PutSkillArtifactRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isEnabled = source["isEnabled"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], SkillArgument);
	        this.tags = source["tags"];
	        this.markdownBody = source["markdownBody"];
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
	export class PutSkillArtifactRequest {
	    BundleID: string;
	    SkillSlug: string;
	    Body?: PutSkillArtifactRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PutSkillArtifactRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.SkillSlug = source["SkillSlug"];
	        this.Body = this.convertValues(source["Body"], PutSkillArtifactRequestBody);
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
	
	export class PutSkillArtifactResponseBody {
	    skill: Skill;
	
	    static createFrom(source: any = {}) {
	        return new PutSkillArtifactResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skill = this.convertValues(source["skill"], Skill);
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
	export class PutSkillArtifactResponse {
	    Body?: PutSkillArtifactResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new PutSkillArtifactResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], PutSkillArtifactResponseBody);
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
	
	export class PutSkillBundleRequestBody {
	    slug: string;
	    displayName: string;
	    isEnabled: boolean;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new PutSkillBundleRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.displayName = source["displayName"];
	        this.isEnabled = source["isEnabled"];
	        this.description = source["description"];
	    }
	}
	export class PutSkillBundleRequest {
	    BundleID: string;
	    Body?: PutSkillBundleRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PutSkillBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.Body = this.convertValues(source["Body"], PutSkillBundleRequestBody);
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
	
	export class PutSkillBundleResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PutSkillBundleResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class PutSkillRequestBody {
	    skillType: string;
	    location: string;
	    name: string;
	    isEnabled: boolean;
	    displayName?: string;
	    description?: string;
	    tags?: string[];
	
	    static createFrom(source: any = {}) {
	        return new PutSkillRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillType = source["skillType"];
	        this.location = source["location"];
	        this.name = source["name"];
	        this.isEnabled = source["isEnabled"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	    }
	}
	export class PutSkillRequest {
	    BundleID: string;
	    SkillSlug: string;
	    Body?: PutSkillRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PutSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.SkillSlug = source["SkillSlug"];
	        this.Body = this.convertValues(source["Body"], PutSkillRequestBody);
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
	
	export class PutSkillResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new PutSkillResponse(source);
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
	
	
	export class RefreshMCPServerRequest {
	    BundleID: string;
	    ServerID: string;
	
	    static createFrom(source: any = {}) {
	        return new RefreshMCPServerRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BundleID = source["BundleID"];
	        this.ServerID = source["ServerID"];
	    }
	}
	export class RefreshMCPServerResponse {
	    Body?: MCPServerRuntimeSnapshot;
	
	    static createFrom(source: any = {}) {
	        return new RefreshMCPServerResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPServerRuntimeSnapshot);
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
	export class RenderSkillRequestBody {
	    skillRef: SkillRef;
	    arguments?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RenderSkillRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillRef = this.convertValues(source["skillRef"], SkillRef);
	        this.arguments = source["arguments"];
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
	export class RenderSkillRequest {
	    Body?: RenderSkillRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new RenderSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], RenderSkillRequestBody);
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
	
	export class RenderSkillResponseBody {
	    text: string;
	    insert: string;
	    name: string;
	    description?: string;
	    displayName?: string;
	    sourceTags?: string[];
	    resources: SkillResourceInfo;
	    arguments?: SkillArgument[];
	    appliedArguments?: Record<string, string>;
	    rawFrontmatter?: Record<string, any>;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new RenderSkillResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.insert = source["insert"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.displayName = source["displayName"];
	        this.sourceTags = source["sourceTags"];
	        this.resources = this.convertValues(source["resources"], SkillResourceInfo);
	        this.arguments = this.convertValues(source["arguments"], SkillArgument);
	        this.appliedArguments = source["appliedArguments"];
	        this.rawFrontmatter = source["rawFrontmatter"];
	        this.warnings = source["warnings"];
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
	export class RenderSkillResponse {
	    Body?: RenderSkillResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new RenderSkillResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], RenderSkillResponseBody);
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
	
	export class ResolveMCPApprovalRequestBody {
	    approvalID: string;
	    resolution: string;
	
	    static createFrom(source: any = {}) {
	        return new ResolveMCPApprovalRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.approvalID = source["approvalID"];
	        this.resolution = source["resolution"];
	    }
	}
	export class ResolveMCPApprovalRequest {
	    Body?: ResolveMCPApprovalRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new ResolveMCPApprovalRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ResolveMCPApprovalRequestBody);
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
	
	export class ResolveMCPApprovalResponse {
	    Body?: MCPApprovalToken;
	
	    static createFrom(source: any = {}) {
	        return new ResolveMCPApprovalResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], MCPApprovalToken);
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
	export class SetDebugSettingsRequestBody {
	    logLLMReqResp: boolean;
	    disableContentStripping: boolean;
	    logLevel: string;
	
	    static createFrom(source: any = {}) {
	        return new SetDebugSettingsRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logLLMReqResp = source["logLLMReqResp"];
	        this.disableContentStripping = source["disableContentStripping"];
	        this.logLevel = source["logLevel"];
	    }
	}
	export class SetDebugSettingsRequest {
	    Body?: SetDebugSettingsRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new SetDebugSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], SetDebugSettingsRequestBody);
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
	
	export class SetDebugSettingsResponse {
	
	
	    static createFrom(source: any = {}) {
	        return new SetDebugSettingsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	

}

export namespace texttool {
	
	export class ApplyUnifiedDiffFileTarget {
	    fileKey?: string;
	    oldPath?: string;
	    newPath?: string;
	    targetPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ApplyUnifiedDiffFileTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileKey = source["fileKey"];
	        this.oldPath = source["oldPath"];
	        this.newPath = source["newPath"];
	        this.targetPath = source["targetPath"];
	    }
	}
	export class ApplyUnifiedDiffArgs {
	    diffText: string;
	    dryRun?: boolean;
	    strict?: boolean;
	    fileTargets?: ApplyUnifiedDiffFileTarget[];
	    candidatePaths?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ApplyUnifiedDiffArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.diffText = source["diffText"];
	        this.dryRun = source["dryRun"];
	        this.strict = source["strict"];
	        this.fileTargets = this.convertValues(source["fileTargets"], ApplyUnifiedDiffFileTarget);
	        this.candidatePaths = source["candidatePaths"];
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
	export class ApplyUnifiedDiffDiagnostic {
	    level: string;
	    code?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ApplyUnifiedDiffDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class ApplyUnifiedDiffFileOut {
	    ok: boolean;
	    fileKey: string;
	    oldPath?: string;
	    newPath?: string;
	    targetPath?: string;
	    resolvedPath?: string;
	    status: string;
	    message?: string;
	    candidatePaths?: string[];
	    diagnostics?: ApplyUnifiedDiffDiagnostic[];
	    hunks: number;
	    appliedHunks: number;
	    alreadyAppliedHunks: number;
	    addedLines: number;
	    deletedLines: number;
	
	    static createFrom(source: any = {}) {
	        return new ApplyUnifiedDiffFileOut(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.fileKey = source["fileKey"];
	        this.oldPath = source["oldPath"];
	        this.newPath = source["newPath"];
	        this.targetPath = source["targetPath"];
	        this.resolvedPath = source["resolvedPath"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.candidatePaths = source["candidatePaths"];
	        this.diagnostics = this.convertValues(source["diagnostics"], ApplyUnifiedDiffDiagnostic);
	        this.hunks = source["hunks"];
	        this.appliedHunks = source["appliedHunks"];
	        this.alreadyAppliedHunks = source["alreadyAppliedHunks"];
	        this.addedLines = source["addedLines"];
	        this.deletedLines = source["deletedLines"];
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
	
	export class ApplyUnifiedDiffSummary {
	    files: number;
	    hunks: number;
	    appliedHunks: number;
	    alreadyAppliedHunks: number;
	    addedLines: number;
	    deletedLines: number;
	
	    static createFrom(source: any = {}) {
	        return new ApplyUnifiedDiffSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = source["files"];
	        this.hunks = source["hunks"];
	        this.appliedHunks = source["appliedHunks"];
	        this.alreadyAppliedHunks = source["alreadyAppliedHunks"];
	        this.addedLines = source["addedLines"];
	        this.deletedLines = source["deletedLines"];
	    }
	}
	export class ApplyUnifiedDiffOut {
	    ok: boolean;
	    dryRun: boolean;
	    status: string;
	    message?: string;
	    diagnostics?: ApplyUnifiedDiffDiagnostic[];
	    summary: ApplyUnifiedDiffSummary;
	    fileTargets?: ApplyUnifiedDiffFileTarget[];
	    files?: ApplyUnifiedDiffFileOut[];
	
	    static createFrom(source: any = {}) {
	        return new ApplyUnifiedDiffOut(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.dryRun = source["dryRun"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.diagnostics = this.convertValues(source["diagnostics"], ApplyUnifiedDiffDiagnostic);
	        this.summary = this.convertValues(source["summary"], ApplyUnifiedDiffSummary);
	        this.fileTargets = this.convertValues(source["fileTargets"], ApplyUnifiedDiffFileTarget);
	        this.files = this.convertValues(source["files"], ApplyUnifiedDiffFileOut);
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

}

export namespace workspace {
	
	export class WorkspaceAttachmentSettings {
	    recursive?: boolean;
	    authoritative?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceAttachmentSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recursive = source["recursive"];
	        this.authoritative = source["authoritative"];
	    }
	}
	export class AttachWorkspaceSourceRequestBody {
	    expectedRootRevision: number;
	    sourceID: string;
	    role: string;
	    priority: number;
	    enabled: boolean;
	    settings: WorkspaceAttachmentSettings;
	
	    static createFrom(source: any = {}) {
	        return new AttachWorkspaceSourceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRootRevision = source["expectedRootRevision"];
	        this.sourceID = source["sourceID"];
	        this.role = source["role"];
	        this.priority = source["priority"];
	        this.enabled = source["enabled"];
	        this.settings = this.convertValues(source["settings"], WorkspaceAttachmentSettings);
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
	export class AttachWorkspaceSourceRequest {
	    RootID: string;
	    Body?: AttachWorkspaceSourceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new AttachWorkspaceSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.Body = this.convertValues(source["Body"], AttachWorkspaceSourceRequestBody);
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
	
	export class WorkspaceAttachmentView {
	    sourceID: string;
	    revision: number;
	    role: string;
	    priority: number;
	    enabled: boolean;
	    sourceDisplayName?: string;
	    sourceKind?: string;
	    path?: string;
	    settings: WorkspaceAttachmentSettings;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceAttachmentView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceID = source["sourceID"];
	        this.revision = source["revision"];
	        this.role = source["role"];
	        this.priority = source["priority"];
	        this.enabled = source["enabled"];
	        this.sourceDisplayName = source["sourceDisplayName"];
	        this.sourceKind = source["sourceKind"];
	        this.path = source["path"];
	        this.settings = this.convertValues(source["settings"], WorkspaceAttachmentSettings);
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
	export class WorkspaceDiscoveryRoot {
	    root: string;
	    recursive: boolean;
	    includePatterns?: string[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDiscoveryRoot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.recursive = source["recursive"];
	        this.includePatterns = source["includePatterns"];
	    }
	}
	export class WorkspaceDiscovery {
	    additionalLocators?: string[];
	    additionalRoots?: WorkspaceDiscoveryRoot[];
	    includeReadme?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDiscovery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.additionalLocators = source["additionalLocators"];
	        this.additionalRoots = this.convertValues(source["additionalRoots"], WorkspaceDiscoveryRoot);
	        this.includeReadme = source["includeReadme"];
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
	export class WorkspaceView {
	    rootID: string;
	    revision: number;
	    displayName: string;
	    description?: string;
	    enabled: boolean;
	    mode: string;
	    primarySourceID?: string;
	    primaryPath?: string;
	    hasTrustReference: boolean;
	    discovery: WorkspaceDiscovery;
	    attachments: WorkspaceAttachmentView[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.revision = source["revision"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.enabled = source["enabled"];
	        this.mode = source["mode"];
	        this.primarySourceID = source["primarySourceID"];
	        this.primaryPath = source["primaryPath"];
	        this.hasTrustReference = source["hasTrustReference"];
	        this.discovery = this.convertValues(source["discovery"], WorkspaceDiscovery);
	        this.attachments = this.convertValues(source["attachments"], WorkspaceAttachmentView);
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
	export class AttachWorkspaceSourceResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new AttachWorkspaceSourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceView);
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
	export class ComposeWorkspaceContextRequestBody {
	    recordIDs?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ComposeWorkspaceContextRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordIDs = source["recordIDs"];
	    }
	}
	export class ComposeWorkspaceContextRequest {
	    RootID: string;
	    Body?: ComposeWorkspaceContextRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new ComposeWorkspaceContextRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.Body = this.convertValues(source["Body"], ComposeWorkspaceContextRequestBody);
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
	
	export class WorkspaceContextDecision {
	    recordID: string;
	    status: string;
	    code?: string;
	    originalBytes: number;
	    includedBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordID = source["recordID"];
	        this.status = source["status"];
	        this.code = source["code"];
	        this.originalBytes = source["originalBytes"];
	        this.includedBytes = source["includedBytes"];
	    }
	}
	export class WorkspaceContextContribution {
	    recordID: string;
	    definitionDigest: string;
	    sourceID: string;
	    locator: string;
	    priority: number;
	    name: string;
	    role: string;
	    mediaType: string;
	    content: string;
	    conventionOrder: number;
	    originalBytes: number;
	    includedBytes: number;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextContribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordID = source["recordID"];
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.priority = source["priority"];
	        this.name = source["name"];
	        this.role = source["role"];
	        this.mediaType = source["mediaType"];
	        this.content = source["content"];
	        this.conventionOrder = source["conventionOrder"];
	        this.originalBytes = source["originalBytes"];
	        this.includedBytes = source["includedBytes"];
	        this.truncated = source["truncated"];
	    }
	}
	export class WorkspaceContextLoadPlan {
	    rootID: string;
	    catalogRevision: number;
	    contributions: WorkspaceContextContribution[];
	    prompt: string;
	    diagnostics?: artifactstore.Diagnostic[];
	    decisions: WorkspaceContextDecision[];
	    promptBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextLoadPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.catalogRevision = source["catalogRevision"];
	        this.contributions = this.convertValues(source["contributions"], WorkspaceContextContribution);
	        this.prompt = source["prompt"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
	        this.decisions = this.convertValues(source["decisions"], WorkspaceContextDecision);
	        this.promptBytes = source["promptBytes"];
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
	export class ComposeWorkspaceContextResponse {
	    Body?: WorkspaceContextLoadPlan;
	
	    static createFrom(source: any = {}) {
	        return new ComposeWorkspaceContextResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceContextLoadPlan);
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
	export class CreateEmptyWorkspaceRequestBody {
	    displayName: string;
	    description?: string;
	    trustReference?: string;
	    discovery: WorkspaceDiscovery;
	
	    static createFrom(source: any = {}) {
	        return new CreateEmptyWorkspaceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.trustReference = source["trustReference"];
	        this.discovery = this.convertValues(source["discovery"], WorkspaceDiscovery);
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
	export class CreateEmptyWorkspaceRequest {
	    Body?: CreateEmptyWorkspaceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new CreateEmptyWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], CreateEmptyWorkspaceRequestBody);
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
	
	export class CreateEmptyWorkspaceResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new CreateEmptyWorkspaceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceView);
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
	export class CreateFilesystemWorkspaceRequestBody {
	    displayName: string;
	    description?: string;
	    rootPath: string;
	    trustReference?: string;
	    discovery: WorkspaceDiscovery;
	
	    static createFrom(source: any = {}) {
	        return new CreateFilesystemWorkspaceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.rootPath = source["rootPath"];
	        this.trustReference = source["trustReference"];
	        this.discovery = this.convertValues(source["discovery"], WorkspaceDiscovery);
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
	export class CreateFilesystemWorkspaceRequest {
	    Body?: CreateFilesystemWorkspaceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new CreateFilesystemWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], CreateFilesystemWorkspaceRequestBody);
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
	
	export class CreateFilesystemWorkspaceResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new CreateFilesystemWorkspaceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceView);
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
	export class DeleteWorkspaceRecordRequest {
	    RootID: string;
	    RecordID: string;
	    ExpectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteWorkspaceRecordRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.RecordID = source["RecordID"];
	        this.ExpectedRevision = source["ExpectedRevision"];
	    }
	}
	export class DeleteWorkspaceRecordResponseBody {
	    recordID: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteWorkspaceRecordResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordID = source["recordID"];
	    }
	}
	export class DeleteWorkspaceRecordResponse {
	    Body?: DeleteWorkspaceRecordResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new DeleteWorkspaceRecordResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], DeleteWorkspaceRecordResponseBody);
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
	
	export class DeleteWorkspaceRequest {
	    RootID: string;
	    ExpectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.ExpectedRevision = source["ExpectedRevision"];
	    }
	}
	export class DeleteWorkspaceResponseBody {
	    rootID: string;
	    revision: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteWorkspaceResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.revision = source["revision"];
	    }
	}
	export class DeleteWorkspaceResponse {
	    Body?: DeleteWorkspaceResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new DeleteWorkspaceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], DeleteWorkspaceResponseBody);
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
	
	export class DetachWorkspaceSourceRequest {
	    RootID: string;
	    SourceID: string;
	    ExpectedRootRevision: number;
	    ExpectedAttachmentRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new DetachWorkspaceSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.SourceID = source["SourceID"];
	        this.ExpectedRootRevision = source["ExpectedRootRevision"];
	        this.ExpectedAttachmentRevision = source["ExpectedAttachmentRevision"];
	    }
	}
	export class DetachWorkspaceSourceResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new DetachWorkspaceSourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceView);
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
	export class FollowWorkspaceRecordRequestBody {
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new FollowWorkspaceRecordRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	    }
	}
	export class FollowWorkspaceRecordRequest {
	    RootID: string;
	    RecordID: string;
	    Body?: FollowWorkspaceRecordRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new FollowWorkspaceRecordRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.RecordID = source["RecordID"];
	        this.Body = this.convertValues(source["Body"], FollowWorkspaceRecordRequestBody);
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
	
	export class WorkspaceRecordView {
	    id: string;
	    revision: number;
	    name: string;
	    kind: string;
	    enabled: boolean;
	    state: string;
	    mode: string;
	    pinnedDefinition?: string;
	    resolvedDefinition?: string;
	    sourceID: string;
	    locator: string;
	    subresourceLocator?: string;
	    runtimeAllowed: boolean;
	    diagnostics?: artifactstore.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceRecordView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.revision = source["revision"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.enabled = source["enabled"];
	        this.state = source["state"];
	        this.mode = source["mode"];
	        this.pinnedDefinition = source["pinnedDefinition"];
	        this.resolvedDefinition = source["resolvedDefinition"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	        this.runtimeAllowed = source["runtimeAllowed"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class FollowWorkspaceRecordResponse {
	    Body?: WorkspaceRecordView;
	
	    static createFrom(source: any = {}) {
	        return new FollowWorkspaceRecordResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceRecordView);
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
	export class GetWorkspaceCatalogRequest {
	    RootID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceCatalogRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	    }
	}
	export class WorkspaceOccurrenceView {
	    sourceID: string;
	    locator: string;
	    subresourceLocator?: string;
	    kind?: string;
	    logicalName?: string;
	    logicalVersion?: string;
	    definitionDigest?: string;
	    sourceContentDigest?: string;
	    state: string;
	    recorded: boolean;
	    recordID?: string;
	    diagnostics?: artifactstore.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceOccurrenceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	        this.kind = source["kind"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceContentDigest = source["sourceContentDigest"];
	        this.state = source["state"];
	        this.recorded = source["recorded"];
	        this.recordID = source["recordID"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class WorkspaceResourceGroupView {
	    kind: string;
	    resources: WorkspaceResourceView[];
	    unrecorded: WorkspaceOccurrenceView[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceResourceGroupView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.resources = this.convertValues(source["resources"], WorkspaceResourceView);
	        this.unrecorded = this.convertValues(source["unrecorded"], WorkspaceOccurrenceView);
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
	export class WorkspaceResourceView {
	    record: WorkspaceRecordView;
	    definitionDigest: string;
	    sourceID: string;
	    locator: string;
	    catalogCurrent: boolean;
	    projectionValid: boolean;
	    diagnostics?: artifactstore.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceResourceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record = this.convertValues(source["record"], WorkspaceRecordView);
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.projectionValid = source["projectionValid"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class WorkspaceCatalogView {
	    workspace: WorkspaceView;
	    catalogRevision: number;
	    catalogCurrent: boolean;
	    diagnostics?: artifactstore.Diagnostic[];
	    resources: WorkspaceResourceView[];
	    groups: WorkspaceResourceGroupView[];
	    occurrences: WorkspaceOccurrenceView[];
	    validOccurrences: WorkspaceOccurrenceView[];
	    invalidOccurrences: WorkspaceOccurrenceView[];
	    missingOccurrences: WorkspaceOccurrenceView[];
	    unrecordedOccurrences: WorkspaceOccurrenceView[];
	    unresolvedRecords: WorkspaceRecordView[];
	    unrecordedCount: number;
	    unresolvedRecordCount: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceCatalogView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], WorkspaceView);
	        this.catalogRevision = source["catalogRevision"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
	        this.resources = this.convertValues(source["resources"], WorkspaceResourceView);
	        this.groups = this.convertValues(source["groups"], WorkspaceResourceGroupView);
	        this.occurrences = this.convertValues(source["occurrences"], WorkspaceOccurrenceView);
	        this.validOccurrences = this.convertValues(source["validOccurrences"], WorkspaceOccurrenceView);
	        this.invalidOccurrences = this.convertValues(source["invalidOccurrences"], WorkspaceOccurrenceView);
	        this.missingOccurrences = this.convertValues(source["missingOccurrences"], WorkspaceOccurrenceView);
	        this.unrecordedOccurrences = this.convertValues(source["unrecordedOccurrences"], WorkspaceOccurrenceView);
	        this.unresolvedRecords = this.convertValues(source["unresolvedRecords"], WorkspaceRecordView);
	        this.unrecordedCount = source["unrecordedCount"];
	        this.unresolvedRecordCount = source["unresolvedRecordCount"];
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
	export class GetWorkspaceCatalogResponse {
	    Body?: WorkspaceCatalogView;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceCatalogResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceCatalogView);
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
	export class GetWorkspaceRecordRequest {
	    RootID: string;
	    RecordID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceRecordRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.RecordID = source["RecordID"];
	    }
	}
	export class GetWorkspaceRecordResponse {
	    Body?: WorkspaceRecordView;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceRecordResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceRecordView);
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
	export class GetWorkspaceRequest {
	    RootID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	    }
	}
	export class GetWorkspaceResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceView);
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
	export class ListWorkspaceContextsRequest {
	    RootID: string;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceContextsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	    }
	}
	export class WorkspaceContextView {
	    recordID: string;
	    recordRevision: number;
	    definitionDigest: string;
	    sourceID: string;
	    locator: string;
	    priority: number;
	    name: string;
	    role: string;
	    mediaType: string;
	    enabled: boolean;
	    state: string;
	    catalogCurrent: boolean;
	    runtimeAllowed: boolean;
	    diagnostics?: artifactstore.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordID = source["recordID"];
	        this.recordRevision = source["recordRevision"];
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.priority = source["priority"];
	        this.name = source["name"];
	        this.role = source["role"];
	        this.mediaType = source["mediaType"];
	        this.enabled = source["enabled"];
	        this.state = source["state"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.runtimeAllowed = source["runtimeAllowed"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class ListWorkspaceContextsResponseBody {
	    contexts: WorkspaceContextView[];
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceContextsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.contexts = this.convertValues(source["contexts"], WorkspaceContextView);
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
	export class ListWorkspaceContextsResponse {
	    Body?: ListWorkspaceContextsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceContextsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListWorkspaceContextsResponseBody);
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
	
	export class ListWorkspaceSkillsRequest {
	    RootID: string;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceSkillsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	    }
	}
	export class WorkspaceSkillArgument {
	    name: string;
	    description?: string;
	    default?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkillArgument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.default = source["default"];
	    }
	}
	export class WorkspaceSkillSummary {
	    schemaVersion: string;
	    id: string;
	    slug: string;
	    name: string;
	    displayName: string;
	    description: string;
	    tags?: string[];
	    insert: string;
	    arguments?: WorkspaceSkillArgument[];
	    isEnabled: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkillSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], WorkspaceSkillArgument);
	        this.isEnabled = source["isEnabled"];
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
	export class WorkspaceSkillView {
	    rootID: string;
	    recordID: string;
	    definitionDigest: string;
	    sourceID: string;
	    locator: string;
	    skill: WorkspaceSkillSummary;
	    markdownBody?: string;
	    priority: number;
	    recordRevision: number;
	    state: string;
	    catalogCurrent: boolean;
	    runtimeAllowed: boolean;
	    diagnostics?: artifactstore.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkillView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.recordID = source["recordID"];
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.skill = this.convertValues(source["skill"], WorkspaceSkillSummary);
	        this.markdownBody = source["markdownBody"];
	        this.priority = source["priority"];
	        this.recordRevision = source["recordRevision"];
	        this.state = source["state"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.runtimeAllowed = source["runtimeAllowed"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class ListWorkspaceSkillsResponseBody {
	    skills: WorkspaceSkillView[];
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceSkillsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skills = this.convertValues(source["skills"], WorkspaceSkillView);
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
	export class ListWorkspaceSkillsResponse {
	    Body?: ListWorkspaceSkillsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceSkillsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListWorkspaceSkillsResponseBody);
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
	
	export class ListWorkspacesRequest {
	
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspacesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class ListWorkspacesResponseBody {
	    workspaces: WorkspaceView[];
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspacesResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspaces = this.convertValues(source["workspaces"], WorkspaceView);
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
	export class ListWorkspacesResponse {
	    Body?: ListWorkspacesResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspacesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListWorkspacesResponseBody);
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
	
	export class LoadWorkspaceContextsRequestBody {
	    recordIDs?: string[];
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceContextsRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordIDs = source["recordIDs"];
	    }
	}
	export class LoadWorkspaceContextsRequest {
	    RootID: string;
	    Body?: LoadWorkspaceContextsRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceContextsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.Body = this.convertValues(source["Body"], LoadWorkspaceContextsRequestBody);
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
	
	export class WorkspaceContextInspectionView {
	    rootID: string;
	    catalogRevision: number;
	    contributions: WorkspaceContextContribution[];
	    diagnostics?: artifactstore.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextInspectionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.catalogRevision = source["catalogRevision"];
	        this.contributions = this.convertValues(source["contributions"], WorkspaceContextContribution);
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class LoadWorkspaceContextsResponse {
	    Body?: WorkspaceContextInspectionView;
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceContextsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceContextInspectionView);
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
	export class LoadWorkspaceSkillsRequestBody {
	    recordIDs: string[];
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceSkillsRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordIDs = source["recordIDs"];
	    }
	}
	export class LoadWorkspaceSkillsRequest {
	    RootID: string;
	    Body?: LoadWorkspaceSkillsRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceSkillsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.Body = this.convertValues(source["Body"], LoadWorkspaceSkillsRequestBody);
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
	
	export class WorkspaceSkillLoadView {
	    rootID: string;
	    catalogRevision: number;
	    skills: WorkspaceSkillView[];
	    diagnostics?: artifactstore.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkillLoadView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.catalogRevision = source["catalogRevision"];
	        this.skills = this.convertValues(source["skills"], WorkspaceSkillView);
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
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
	export class LoadWorkspaceSkillsResponse {
	    Body?: WorkspaceSkillLoadView;
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceSkillsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceSkillLoadView);
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
	export class PinWorkspaceRecordRequestBody {
	    expectedRevision: number;
	    definitionDigest: string;
	
	    static createFrom(source: any = {}) {
	        return new PinWorkspaceRecordRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.definitionDigest = source["definitionDigest"];
	    }
	}
	export class PinWorkspaceRecordRequest {
	    RootID: string;
	    RecordID: string;
	    Body?: PinWorkspaceRecordRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PinWorkspaceRecordRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.RecordID = source["RecordID"];
	        this.Body = this.convertValues(source["Body"], PinWorkspaceRecordRequestBody);
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
	
	export class PinWorkspaceRecordResponse {
	    Body?: WorkspaceRecordView;
	
	    static createFrom(source: any = {}) {
	        return new PinWorkspaceRecordResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceRecordView);
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
	export class RefreshWorkspaceRequest {
	    RootID: string;
	
	    static createFrom(source: any = {}) {
	        return new RefreshWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	    }
	}
	export class WorkspaceRefreshResult {
	    rootID: string;
	    catalogRevision: number;
	    createdRecords: string[];
	    updatedRecords: string[];
	    diagnostics?: artifactstore.Diagnostic[];
	    candidates: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceRefreshResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.catalogRevision = source["catalogRevision"];
	        this.createdRecords = source["createdRecords"];
	        this.updatedRecords = source["updatedRecords"];
	        this.diagnostics = this.convertValues(source["diagnostics"], artifactstore.Diagnostic);
	        this.candidates = source["candidates"];
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
	export class RefreshWorkspaceResponse {
	    Body?: WorkspaceRefreshResult;
	
	    static createFrom(source: any = {}) {
	        return new RefreshWorkspaceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceRefreshResult);
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
	export class SetWorkspaceRecordEnabledRequestBody {
	    expectedRevision: number;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceRecordEnabledRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.enabled = source["enabled"];
	    }
	}
	export class SetWorkspaceRecordEnabledRequest {
	    RootID: string;
	    RecordID: string;
	    Body?: SetWorkspaceRecordEnabledRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceRecordEnabledRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.RecordID = source["RecordID"];
	        this.Body = this.convertValues(source["Body"], SetWorkspaceRecordEnabledRequestBody);
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
	
	export class SetWorkspaceRecordEnabledResponse {
	    Body?: WorkspaceRecordView;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceRecordEnabledResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceRecordView);
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
	export class UpdateWorkspaceAttachmentRequestBody {
	    expectedRootRevision: number;
	    expectedAttachmentRevision: number;
	    role: string;
	    priority: number;
	    enabled: boolean;
	    settings: WorkspaceAttachmentSettings;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceAttachmentRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRootRevision = source["expectedRootRevision"];
	        this.expectedAttachmentRevision = source["expectedAttachmentRevision"];
	        this.role = source["role"];
	        this.priority = source["priority"];
	        this.enabled = source["enabled"];
	        this.settings = this.convertValues(source["settings"], WorkspaceAttachmentSettings);
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
	export class UpdateWorkspaceAttachmentRequest {
	    RootID: string;
	    SourceID: string;
	    Body?: UpdateWorkspaceAttachmentRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceAttachmentRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.SourceID = source["SourceID"];
	        this.Body = this.convertValues(source["Body"], UpdateWorkspaceAttachmentRequestBody);
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
	
	export class UpdateWorkspaceAttachmentResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceAttachmentResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceView);
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
	export class UpdateWorkspaceRecordDataRequestBody {
	    expectedRevision: number;
	    runtimeAllowed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceRecordDataRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.runtimeAllowed = source["runtimeAllowed"];
	    }
	}
	export class UpdateWorkspaceRecordDataRequest {
	    RootID: string;
	    RecordID: string;
	    Body?: UpdateWorkspaceRecordDataRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceRecordDataRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.RecordID = source["RecordID"];
	        this.Body = this.convertValues(source["Body"], UpdateWorkspaceRecordDataRequestBody);
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
	
	export class UpdateWorkspaceRecordDataResponse {
	    Body?: WorkspaceRecordView;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceRecordDataResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceRecordView);
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
	export class UpdateWorkspaceRequestBody {
	    expectedRevision: number;
	    displayName: string;
	    description?: string;
	    enabled: boolean;
	    trustReference?: string;
	    discovery: WorkspaceDiscovery;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.enabled = source["enabled"];
	        this.trustReference = source["trustReference"];
	        this.discovery = this.convertValues(source["discovery"], WorkspaceDiscovery);
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
	export class UpdateWorkspaceRequest {
	    RootID: string;
	    Body?: UpdateWorkspaceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.Body = this.convertValues(source["Body"], UpdateWorkspaceRequestBody);
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
	
	export class UpdateWorkspaceResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceView);
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
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	

}

