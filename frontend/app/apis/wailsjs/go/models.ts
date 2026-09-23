export namespace aggregate {
	
	export class ArtifactSkillFilter {
	    types?: string[];
	    inserts?: string[];
	    namePrefix?: string;
	    locationPrefix?: string;
	    allowArtifacts?: artifact.ArtifactRef[];
	    sessionID?: string;
	    activity?: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSkillFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.types = source["types"];
	        this.inserts = source["inserts"];
	        this.namePrefix = source["namePrefix"];
	        this.locationPrefix = source["locationPrefix"];
	        this.allowArtifacts = this.convertValues(source["allowArtifacts"], artifact.ArtifactRef);
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
	export class ArtifactSkillSummary {
	    Artifact: artifact.ArtifactRef;
	    IsEnabled: boolean;
	    Insert: string;
	    HasArguments: boolean;
	    HasResources: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSkillSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Artifact = this.convertValues(source["Artifact"], artifact.ArtifactRef);
	        this.IsEnabled = source["IsEnabled"];
	        this.Insert = source["Insert"];
	        this.HasArguments = source["HasArguments"];
	        this.HasResources = source["HasResources"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResolvedArtifactSkill {
	    Artifact: artifact.ArtifactRef;
	    Definition: provider.SkillDef;
	    Version: string;
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResolvedArtifactSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Artifact = this.convertValues(source["Artifact"], artifact.ArtifactRef);
	        this.Definition = this.convertValues(source["Definition"], provider.SkillDef);
	        this.Version = source["Version"];
	        this.Enabled = source["Enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SecretWriteResult {
	    secretRef: string;
	    sha256?: string;
	    nonEmpty: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SecretWriteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.secretRef = source["secretRef"];
	        this.sha256 = source["sha256"];
	        this.nonEmpty = source["nonEmpty"];
	    }
	}

}

export namespace artifact {
	
	export class SourceBinding {
	    sourceID: string;
	    locator: string;
	    subresourceLocator?: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	    }
	}
	export class Artifact {
	    id: string;
	    rootID: string;
	    binding: SourceBinding;
	    kind: string;
	    logicalName: string;
	    logicalVersion?: string;
	    resolvedDefinition?: string;
	    sourceContentDigest?: string;
	    state: string;
	    diagnostics?: diagnostic.Diagnostic[];
	    displayName: string;
	    enabled: boolean;
	    revision: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Artifact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rootID = source["rootID"];
	        this.binding = this.convertValues(source["binding"], SourceBinding);
	        this.kind = source["kind"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.resolvedDefinition = source["resolvedDefinition"];
	        this.sourceContentDigest = source["sourceContentDigest"];
	        this.state = source["state"];
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
	        this.displayName = source["displayName"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
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
	export class ArtifactAddress {
	    rootID: string;
	    artifactID: string;
	    kind: string;
	    logicalName: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactAddress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.artifactID = source["artifactID"];
	        this.kind = source["kind"];
	        this.logicalName = source["logicalName"];
	    }
	}
	export class ArtifactRef {
	    rootID: string;
	    artifactID: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.artifactID = source["artifactID"];
	    }
	}

}

export namespace artifactfallback {
	
	export class ResolveTargetRequest {
	    target: resolve.MappedTarget;
	
	    static createFrom(source: any = {}) {
	        return new ResolveTargetRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = this.convertValues(source["target"], resolve.MappedTarget);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResolveTargetResponseBody {
	    modelPresetRef: spec.ModelPresetRef;
	
	    static createFrom(source: any = {}) {
	        return new ResolveTargetResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelPresetRef = this.convertValues(source["modelPresetRef"], spec.ModelPresetRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResolveTargetResponse {
	    Body?: ResolveTargetResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ResolveTargetResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ResolveTargetResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace auth {
	
	export class MCPAuthHealth {
	    server: string;
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
	    oauthLoopbackReady?: boolean;
	    oauthLoopbackError?: string;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPAuthHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
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
	        this.oauthLoopbackReady = source["oauthLoopbackReady"];
	        this.oauthLoopbackError = source["oauthLoopbackError"];
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
	export class MCPAuthSettings {
	    oauthLoopbackListenAddr?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPAuthSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oauthLoopbackListenAddr = source["oauthLoopbackListenAddr"];
	    }
	}
	export class MCPOAuthAuthorization {
	    server: string;
	    authorizationURL: string;
	    expiresAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPOAuthAuthorization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.authorizationURL = source["authorizationURL"];
	        this.expiresAt = source["expiresAt"];
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
	    supportsReasoningContext?: boolean;
	    supportsReasoningMode?: boolean;
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
	        this.supportsReasoningContext = source["supportsReasoningContext"];
	        this.supportsReasoningMode = source["supportsReasoningMode"];
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

export namespace collection {
	
	export class AddArtifactMemberRequest {
	    collection: artifact.ArtifactRef;
	    expectedRevision: number;
	    artifact: artifact.ArtifactRef;
	
	    static createFrom(source: any = {}) {
	        return new AddArtifactMemberRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedRevision = source["expectedRevision"];
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MemberReference {
	    type: string;
	    name: string;
	    insert?: string;
	    locator?: declaration.Locator;
	    scope?: string;
	    server?: string;
	
	    static createFrom(source: any = {}) {
	        return new MemberReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.insert = source["insert"];
	        this.locator = this.convertValues(source["locator"], declaration.Locator);
	        this.scope = source["scope"];
	        this.server = source["server"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AddMemberRequest {
	    collection: artifact.ArtifactRef;
	    expectedRevision: number;
	    member: MemberReference;
	
	    static createFrom(source: any = {}) {
	        return new AddMemberRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedRevision = source["expectedRevision"];
	        this.member = this.convertValues(source["member"], MemberReference);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ArtifactMembershipView {
	    collection: artifact.ArtifactRef;
	    collectionName: string;
	    collectionRevision: number;
	    memberIndex: number;
	    member: MemberReference;
	    status: string;
	    resolvedArtifact?: artifact.ArtifactRef;
	    resolvedToArtifact: boolean;
	    code?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactMembershipView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.collectionName = source["collectionName"];
	        this.collectionRevision = source["collectionRevision"];
	        this.memberIndex = source["memberIndex"];
	        this.member = this.convertValues(source["member"], MemberReference);
	        this.status = source["status"];
	        this.resolvedArtifact = this.convertValues(source["resolvedArtifact"], artifact.ArtifactRef);
	        this.resolvedToArtifact = source["resolvedToArtifact"];
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CollectionMemberView {
	    type: string;
	    name?: string;
	    insert?: string;
	    locator?: declaration.Locator;
	    server?: string;
	    contained: boolean;
	    selector: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CollectionMemberView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.insert = source["insert"];
	        this.locator = this.convertValues(source["locator"], declaration.Locator);
	        this.server = source["server"];
	        this.contained = source["contained"];
	        this.selector = source["selector"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CollectionView {
	    artifact: artifact.Artifact;
	    name: string;
	    displayName: string;
	    description?: string;
	    members: MemberReference[];
	    entries: CollectionMemberView[];
	    editable: boolean;
	    deletable: boolean;
	    baseline: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CollectionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.members = this.convertValues(source["members"], MemberReference);
	        this.entries = this.convertValues(source["entries"], CollectionMemberView);
	        this.editable = source["editable"];
	        this.deletable = source["deletable"];
	        this.baseline = source["baseline"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CollectionCapabilityPlan {
	    collection: CollectionView;
	    occurrences: resolve.CapabilityOccurrence[];
	    complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CollectionCapabilityPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], CollectionView);
	        this.occurrences = this.convertValues(source["occurrences"], resolve.CapabilityOccurrence);
	        this.complete = source["complete"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class CreateRequest {
	    rootID: string;
	    sourceID?: string;
	    name: string;
	    displayName?: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.sourceID = source["sourceID"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	    }
	}
	export class DeleteRequest {
	    collection: artifact.ArtifactRef;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedRevision = source["expectedRevision"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RemoveMemberRequest {
	    collection: artifact.ArtifactRef;
	    expectedRevision: number;
	    index: number;
	
	    static createFrom(source: any = {}) {
	        return new RemoveMemberRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedRevision = source["expectedRevision"];
	        this.index = source["index"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateRequest {
	    collection: artifact.ArtifactRef;
	    expectedRevision: number;
	    description?: string;
	    displayName?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedRevision = source["expectedRevision"];
	        this.description = source["description"];
	        this.displayName = source["displayName"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace consumerapi {
	
	export class AgentExportRequest {
	    agent: artifact.ArtifactRef;
	
	    static createFrom(source: any = {}) {
	        return new AgentExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentMCPSetupInput {
	    name: string;
	    kind: string;
	    label?: string;
	    description?: string;
	    required: boolean;
	    clientSecretRequired: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentMCPSetupInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.required = source["required"];
	        this.clientSecretRequired = source["clientSecretRequired"];
	    }
	}
	export class AgentMCPSetupDescriptor {
	    occurrencePath: string;
	    name: string;
	    artifact?: artifact.ArtifactRef;
	    transport?: string;
	    command?: string;
	    url?: string;
	    authMode?: string;
	    inputs?: AgentMCPSetupInput[];
	
	    static createFrom(source: any = {}) {
	        return new AgentMCPSetupDescriptor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.occurrencePath = source["occurrencePath"];
	        this.name = source["name"];
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.transport = source["transport"];
	        this.command = source["command"];
	        this.url = source["url"];
	        this.authMode = source["authMode"];
	        this.inputs = this.convertValues(source["inputs"], AgentMCPSetupInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentExportResult {
	    type: string;
	    name: string;
	    mediaType: string;
	    suggestedFileName: string;
	    content: string;
	    contentDigest: string;
	    definitionDigest: string;
	    artifactRevision: number;
	    builtIn: boolean;
	    managed: boolean;
	    resolution?: resolve.CapabilityPlan;
	    resolutionIssue?: resolve.ResolutionIssue;
	    mcpSetupDescriptors?: AgentMCPSetupDescriptor[];
	
	    static createFrom(source: any = {}) {
	        return new AgentExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.mediaType = source["mediaType"];
	        this.suggestedFileName = source["suggestedFileName"];
	        this.content = source["content"];
	        this.contentDigest = source["contentDigest"];
	        this.definitionDigest = source["definitionDigest"];
	        this.artifactRevision = source["artifactRevision"];
	        this.builtIn = source["builtIn"];
	        this.managed = source["managed"];
	        this.resolution = this.convertValues(source["resolution"], resolve.CapabilityPlan);
	        this.resolutionIssue = this.convertValues(source["resolutionIssue"], resolve.ResolutionIssue);
	        this.mcpSetupDescriptors = this.convertValues(source["mcpSetupDescriptors"], AgentMCPSetupDescriptor);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentImportArtifactPreview {
	    occurrencePath: string;
	    type: string;
	    name: string;
	    logicalVersion?: string;
	    definitionDigest: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportArtifactPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.occurrencePath = source["occurrencePath"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.logicalVersion = source["logicalVersion"];
	        this.definitionDigest = source["definitionDigest"];
	    }
	}
	export class AgentImportCommitRequest {
	    prepared: string;
	    preparedFingerprint: string;
	    acceptedConfirmationCodes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new AgentImportCommitRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prepared = source["prepared"];
	        this.preparedFingerprint = source["preparedFingerprint"];
	        this.acceptedConfirmationCodes = source["acceptedConfirmationCodes"];
	    }
	}
	export class AgentRestoredMembership {
	    collection: artifact.ArtifactRef;
	    path: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentRestoredMembership(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.path = source["path"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentView {
	    artifact: artifact.Artifact;
	    name: string;
	    displayName: string;
	    description?: string;
	    builtIn: boolean;
	    managed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.builtIn = source["builtIn"];
	        this.managed = source["managed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentImportCommitResult {
	    agent: AgentView;
	    collection: collection.CollectionView;
	    restoredMemberships?: AgentRestoredMembership[];
	    mcpSetupDescriptors?: AgentMCPSetupDescriptor[];
	    preparedFingerprint: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportCommitResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], AgentView);
	        this.collection = this.convertValues(source["collection"], collection.CollectionView);
	        this.restoredMemberships = this.convertValues(source["restoredMemberships"], AgentRestoredMembership);
	        this.mcpSetupDescriptors = this.convertValues(source["mcpSetupDescriptors"], AgentMCPSetupDescriptor);
	        this.preparedFingerprint = source["preparedFingerprint"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentImportConflict {
	    code: string;
	    path?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.path = source["path"];
	        this.message = source["message"];
	    }
	}
	export class AgentImportDestination {
	    rootID: string;
	    rootDisplayName?: string;
	    sourceID: string;
	    collection: collection.CollectionView;
	    collectionRevision: number;
	    collectionName: string;
	    collectionDisplayName: string;
	    baseline: boolean;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportDestination(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.rootDisplayName = source["rootDisplayName"];
	        this.sourceID = source["sourceID"];
	        this.collection = this.convertValues(source["collection"], collection.CollectionView);
	        this.collectionRevision = source["collectionRevision"];
	        this.collectionName = source["collectionName"];
	        this.collectionDisplayName = source["collectionDisplayName"];
	        this.baseline = source["baseline"];
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentImportIssue {
	    code: string;
	    severity: string;
	    path?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.severity = source["severity"];
	        this.path = source["path"];
	        this.message = source["message"];
	    }
	}
	export class AgentImportRelationship {
	    path: string;
	    type: string;
	    name: string;
	    scope?: string;
	    status: string;
	    artifact?: artifact.ArtifactRef;
	    mapped?: resolve.MappedTarget;
	    code?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportRelationship(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.scope = source["scope"];
	        this.status = source["status"];
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.mapped = this.convertValues(source["mapped"], resolve.MappedTarget);
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentImportPreview {
	    prepared?: string;
	    preparedFingerprint?: string;
	    // Go type: time
	    expiresAt: any;
	    sourceDigest?: string;
	    definitionDigest?: string;
	    normalizedYAML?: string;
	    agent?: AgentImportArtifactPreview;
	    destination: AgentImportDestination;
	    projectedArtifacts?: AgentImportArtifactPreview[];
	    relationships?: AgentImportRelationship[];
	    conflicts?: AgentImportConflict[];
	    restoredMemberships?: AgentRestoredMembership[];
	    mcpSetupDescriptors?: AgentMCPSetupDescriptor[];
	    canImport: boolean;
	    requiresConfirmation: boolean;
	    requiredConfirmationCodes?: string[];
	    issues?: AgentImportIssue[];
	
	    static createFrom(source: any = {}) {
	        return new AgentImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prepared = source["prepared"];
	        this.preparedFingerprint = source["preparedFingerprint"];
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
	        this.sourceDigest = source["sourceDigest"];
	        this.definitionDigest = source["definitionDigest"];
	        this.normalizedYAML = source["normalizedYAML"];
	        this.agent = this.convertValues(source["agent"], AgentImportArtifactPreview);
	        this.destination = this.convertValues(source["destination"], AgentImportDestination);
	        this.projectedArtifacts = this.convertValues(source["projectedArtifacts"], AgentImportArtifactPreview);
	        this.relationships = this.convertValues(source["relationships"], AgentImportRelationship);
	        this.conflicts = this.convertValues(source["conflicts"], AgentImportConflict);
	        this.restoredMemberships = this.convertValues(source["restoredMemberships"], AgentRestoredMembership);
	        this.mcpSetupDescriptors = this.convertValues(source["mcpSetupDescriptors"], AgentMCPSetupDescriptor);
	        this.canImport = source["canImport"];
	        this.requiresConfirmation = source["requiresConfirmation"];
	        this.requiredConfirmationCodes = source["requiredConfirmationCodes"];
	        this.issues = this.convertValues(source["issues"], AgentImportIssue);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentImportPreviewRequest {
	    path: string;
	    collection: artifact.ArtifactRef;
	    expectedCollectionRevision: number;
	    expectedSourceDigest?: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportPreviewRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.expectedSourceDigest = source["expectedSourceDigest"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class AgentResolution {
	    agent: AgentView;
	    capabilities: resolve.CapabilityPlan;
	
	    static createFrom(source: any = {}) {
	        return new AgentResolution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], AgentView);
	        this.capabilities = this.convertValues(source["capabilities"], resolve.CapabilityPlan);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class AgentTextMaterialization {
	    artifact: artifact.ArtifactRef;
	    artifactRevision: number;
	    definitionDigest: string;
	    name: string;
	    insert: string;
	    mediaType?: string;
	    content: string;
	    locator: string;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentTextMaterialization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.artifactRevision = source["artifactRevision"];
	        this.definitionDigest = source["definitionDigest"];
	        this.name = source["name"];
	        this.insert = source["insert"];
	        this.mediaType = source["mediaType"];
	        this.content = source["content"];
	        this.locator = source["locator"];
	        this.builtIn = source["builtIn"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CollectionPage {
	    items: collection.CollectionView[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new CollectionPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], collection.CollectionView);
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
	export class ListAgentsRequest {
	    rootID: string;
	    logicalNames?: string[];
	    collection?: artifact.ArtifactRef;
	    includeBuiltin?: boolean;
	    enabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ListAgentsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.logicalNames = source["logicalNames"];
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.includeBuiltin = source["includeBuiltin"];
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedAgentDeleteRequest {
	    agent: artifact.ArtifactRef;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new ManagedAgentDeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], artifact.ArtifactRef);
	        this.expectedRevision = source["expectedRevision"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedMCPCreateRequest {
	    collection: artifact.ArtifactRef;
	    expectedCollectionRevision: number;
	    document: server.ServerDocument;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPCreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.document = this.convertValues(source["document"], server.ServerDocument);
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedMCPCreateResult {
	    artifact: artifact.Artifact;
	    address: artifact.ArtifactAddress;
	    collection: collection.CollectionView;
	    membershipCreated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPCreateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.address = this.convertValues(source["address"], artifact.ArtifactAddress);
	        this.collection = this.convertValues(source["collection"], collection.CollectionView);
	        this.membershipCreated = source["membershipCreated"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedMCPPolicyUpsertRequest {
	    collection: artifact.ArtifactRef;
	    expectedCollectionRevision: number;
	    name: string;
	    description?: string;
	    policy: policy.MCPPolicy;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPPolicyUpsertRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.policy = this.convertValues(source["policy"], policy.MCPPolicy);
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedMCPPolicyUpsertResult {
	    artifact: artifact.Artifact;
	    address: artifact.ArtifactAddress;
	    collection: collection.CollectionView;
	    membershipCreated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPPolicyUpsertResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.address = this.convertValues(source["address"], artifact.ArtifactAddress);
	        this.collection = this.convertValues(source["collection"], collection.CollectionView);
	        this.membershipCreated = source["membershipCreated"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedMCPReplaceRequest {
	    collection: artifact.ArtifactRef;
	    expectedCollectionRevision: number;
	    artifact: artifact.ArtifactRef;
	    expectedArtifactRevision: number;
	    document: server.ServerDocument;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPReplaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.expectedArtifactRevision = source["expectedArtifactRevision"];
	        this.document = this.convertValues(source["document"], server.ServerDocument);
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedMCPReplaceResult {
	    artifact: artifact.Artifact;
	    address: artifact.ArtifactAddress;
	    collection: collection.CollectionView;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPReplaceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.address = this.convertValues(source["address"], artifact.ArtifactAddress);
	        this.collection = this.convertValues(source["collection"], collection.CollectionView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedSkillCreateRequest {
	    collection: artifact.ArtifactRef;
	    expectedCollectionRevision: number;
	    skillName: string;
	    skillMD?: number[];
	    files?: source.ManagedPackageFile[];
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillCreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.skillName = source["skillName"];
	        this.skillMD = source["skillMD"];
	        this.files = this.convertValues(source["files"], source.ManagedPackageFile);
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedSkillCreateResult {
	    artifact: artifact.Artifact;
	    address: artifact.ArtifactAddress;
	    collection: collection.CollectionView;
	    membershipCreated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillCreateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.address = this.convertValues(source["address"], artifact.ArtifactAddress);
	        this.collection = this.convertValues(source["collection"], collection.CollectionView);
	        this.membershipCreated = source["membershipCreated"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedSkillReplaceRequest {
	    collection: artifact.ArtifactRef;
	    expectedCollectionRevision: number;
	    artifact: artifact.ArtifactRef;
	    expectedArtifactRevision: number;
	    skillName: string;
	    skillMD?: number[];
	    files?: source.ManagedPackageFile[];
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillReplaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collection = this.convertValues(source["collection"], artifact.ArtifactRef);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.expectedArtifactRevision = source["expectedArtifactRevision"];
	        this.skillName = source["skillName"];
	        this.skillMD = source["skillMD"];
	        this.files = this.convertValues(source["files"], source.ManagedPackageFile);
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedSkillReplaceResult {
	    artifact: artifact.Artifact;
	    address: artifact.ArtifactAddress;
	    collection: collection.CollectionView;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillReplaceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.address = this.convertValues(source["address"], artifact.ArtifactAddress);
	        this.collection = this.convertValues(source["collection"], collection.CollectionView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PolicyView {
	    artifact: artifact.Artifact;
	    body: policy.MCPPolicy;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PolicyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.body = this.convertValues(source["body"], policy.MCPPolicy);
	        this.builtIn = source["builtIn"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ServerInstallationView {
	    artifact: artifact.Artifact;
	    document: server.ServerDocument;
	    installation: server.ServerData;
	    installationRevision: number;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerInstallationView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.document = this.convertValues(source["document"], server.ServerDocument);
	        this.installation = this.convertValues(source["installation"], server.ServerData);
	        this.installationRevision = source["installationRevision"];
	        this.builtIn = source["builtIn"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ServerPage {
	    items: artifact.Artifact[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], artifact.Artifact);
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
	export class SkillDirectoryRegistration {
	    rootID: string;
	    rootPath: string;
	    sourceDisplayName: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillDirectoryRegistration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.rootPath = source["rootPath"];
	        this.sourceDisplayName = source["sourceDisplayName"];
	    }
	}
	export class SkillPathRegistration {
	    rootID: string;
	    path: string;
	    sourceDisplayName?: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SkillPathRegistration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.path = source["path"];
	        this.sourceDisplayName = source["sourceDisplayName"];
	        this.enabled = source["enabled"];
	    }
	}
	export class SkillPathRegistrationResult {
	    source: source.Summary;
	    artifact: artifact.Artifact;
	
	    static createFrom(source: any = {}) {
	        return new SkillPathRegistrationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = this.convertValues(source["source"], source.Summary);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceArtifactView {
	    artifact: artifact.ArtifactRef;
	    revision: number;
	    displayName: string;
	    kind: string;
	    logicalName: string;
	    logicalVersion?: string;
	    enabled: boolean;
	    state: string;
	    sourceID: string;
	    locator: string;
	    subresourceLocator?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceArtifactView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.revision = source["revision"];
	        this.displayName = source["displayName"];
	        this.kind = source["kind"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.enabled = source["enabled"];
	        this.state = source["state"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceDefaultPolicyView {
	    policyID: string;
	    policyVersion: string;
	    policyDigest: string;
	    yaml: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDefaultPolicyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.policyID = source["policyID"];
	        this.policyVersion = source["policyVersion"];
	        this.policyDigest = source["policyDigest"];
	        this.yaml = source["yaml"];
	    }
	}
	export class WorkspaceDirectoryRef {
	    rootID: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDirectoryRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	    }
	}
	export class WorkspaceDirectoryWorkspace {
	    workspace: domain.WorkspaceView;
	    origin: string;
	    manifestLocator?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDirectoryWorkspace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], domain.WorkspaceView);
	        this.origin = source["origin"];
	        this.manifestLocator = source["manifestLocator"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceDirectoryView {
	    ref: WorkspaceDirectoryRef;
	    root: root.Root;
	    directorySource: source.Summary;
	    enabled: boolean;
	    policyID: string;
	    policyVersion: string;
	    policyDigest: string;
	    workspaces: WorkspaceDirectoryWorkspace[];
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDirectoryView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], WorkspaceDirectoryRef);
	        this.root = this.convertValues(source["root"], root.Root);
	        this.directorySource = this.convertValues(source["directorySource"], source.Summary);
	        this.enabled = source["enabled"];
	        this.policyID = source["policyID"];
	        this.policyVersion = source["policyVersion"];
	        this.policyDigest = source["policyDigest"];
	        this.workspaces = this.convertValues(source["workspaces"], WorkspaceDirectoryWorkspace);
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class WorkspaceMCPServer {
	    artifact: artifact.ArtifactRef;
	    artifactRevision: number;
	    definitionDigest: string;
	    name: string;
	    displayName?: string;
	    builtIn: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceMCPServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.artifactRevision = source["artifactRevision"];
	        this.definitionDigest = source["definitionDigest"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.builtIn = source["builtIn"];
	        this.version = source["version"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceMCPServerLoadPlan {
	    workspace: artifact.ArtifactRef;
	    servers: WorkspaceMCPServer[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceMCPServerLoadPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], artifact.ArtifactRef);
	        this.servers = this.convertValues(source["servers"], WorkspaceMCPServer);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspacePage {
	    items: WorkspaceDirectoryView[];
	    nextCursor?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspacePage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], WorkspaceDirectoryView);
	        this.nextCursor = source["nextCursor"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspacePageRequest {
	    cursor?: string;
	    limit?: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspacePageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cursor = source["cursor"];
	        this.limit = source["limit"];
	    }
	}
	export class WorkspacePromptContribution {
	    artifact: artifact.ArtifactRef;
	    artifactRevision: number;
	    definitionDigest: string;
	    kind: string;
	    name: string;
	    insert: string;
	    mediaType?: string;
	    locator?: string;
	    originalBytes: number;
	    includedBytes: number;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspacePromptContribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.artifactRevision = source["artifactRevision"];
	        this.definitionDigest = source["definitionDigest"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.insert = source["insert"];
	        this.mediaType = source["mediaType"];
	        this.locator = source["locator"];
	        this.originalBytes = source["originalBytes"];
	        this.includedBytes = source["includedBytes"];
	        this.truncated = source["truncated"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspacePromptDecision {
	    artifact: artifact.ArtifactRef;
	    status: string;
	    code?: string;
	    originalBytes: number;
	    includedBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspacePromptDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.status = source["status"];
	        this.code = source["code"];
	        this.originalBytes = source["originalBytes"];
	        this.includedBytes = source["includedBytes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspacePromptPlan {
	    workspace: artifact.ArtifactRef;
	    contributions: WorkspacePromptContribution[];
	    instructions: string;
	    userMessage: string;
	    diagnostics?: diagnostic.Diagnostic[];
	    decisions: WorkspacePromptDecision[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspacePromptPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], artifact.ArtifactRef);
	        this.contributions = this.convertValues(source["contributions"], WorkspacePromptContribution);
	        this.instructions = source["instructions"];
	        this.userMessage = source["userMessage"];
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
	        this.decisions = this.convertValues(source["decisions"], WorkspacePromptDecision);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceSkill {
	    artifact: artifact.ArtifactRef;
	    artifactRevision: number;
	    definitionDigest: string;
	    name: string;
	    displayName?: string;
	    insert?: string;
	    locator?: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.artifactRevision = source["artifactRevision"];
	        this.definitionDigest = source["definitionDigest"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.insert = source["insert"];
	        this.locator = source["locator"];
	        this.version = source["version"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceSkillLoadPlan {
	    workspace: artifact.ArtifactRef;
	    skills: WorkspaceSkill[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkillLoadPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], artifact.ArtifactRef);
	        this.skills = this.convertValues(source["skills"], WorkspaceSkill);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceRuntimePlan {
	    workspace: domain.WorkspaceView;
	    capabilities: resolve.CapabilityPlan;
	    prompt: WorkspacePromptPlan;
	    skills: WorkspaceSkillLoadPlan;
	    mcpServers: WorkspaceMCPServerLoadPlan;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceRuntimePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], domain.WorkspaceView);
	        this.capabilities = this.convertValues(source["capabilities"], resolve.CapabilityPlan);
	        this.prompt = this.convertValues(source["prompt"], WorkspacePromptPlan);
	        this.skills = this.convertValues(source["skills"], WorkspaceSkillLoadPlan);
	        this.mcpServers = this.convertValues(source["mcpServers"], WorkspaceMCPServerLoadPlan);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceRuntimeSelection {
	    promptArtifacts?: artifact.ArtifactRef[];
	    skillArtifacts?: artifact.ArtifactRef[];
	    mcpArtifacts?: artifact.ArtifactRef[];
	    requireComplete?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceRuntimeSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptArtifacts = this.convertValues(source["promptArtifacts"], artifact.ArtifactRef);
	        this.skillArtifacts = this.convertValues(source["skillArtifacts"], artifact.ArtifactRef);
	        this.mcpArtifacts = this.convertValues(source["mcpArtifacts"], artifact.ArtifactRef);
	        this.requireComplete = source["requireComplete"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace conversation {
	
	export class ConversationContextUsage {
	    artifact: artifact.ArtifactRef;
	    name?: string;
	    locator?: string;
	    selectedDefinitionDigest?: string;
	    usedDefinitionDigest?: string;
	    usedArtifactRevision?: number;
	    status: string;
	    code?: string;
	    originalBytes?: number;
	    includedBytes?: number;
	    changed?: boolean;
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationContextUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.name = source["name"];
	        this.locator = source["locator"];
	        this.selectedDefinitionDigest = source["selectedDefinitionDigest"];
	        this.usedDefinitionDigest = source["usedDefinitionDigest"];
	        this.usedArtifactRevision = source["usedArtifactRevision"];
	        this.status = source["status"];
	        this.code = source["code"];
	        this.originalBytes = source["originalBytes"];
	        this.includedBytes = source["includedBytes"];
	        this.changed = source["changed"];
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConversationResourceSelectionRef {
	    artifact: artifact.ArtifactRef;
	    name?: string;
	    locator?: string;
	    definitionDigest?: string;
	    artifactRevision?: number;
	
	    static createFrom(source: any = {}) {
	        return new ConversationResourceSelectionRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.name = source["name"];
	        this.locator = source["locator"];
	        this.definitionDigest = source["definitionDigest"];
	        this.artifactRevision = source["artifactRevision"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConversationSelection {
	    workspace: artifact.ArtifactRef;
	    displayName?: string;
	    workspaceRevision?: number;
	    contextRefs?: ConversationResourceSelectionRef[];
	    skillRefs?: ConversationResourceSelectionRef[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], artifact.ArtifactRef);
	        this.displayName = source["displayName"];
	        this.workspaceRevision = source["workspaceRevision"];
	        this.contextRefs = this.convertValues(source["contextRefs"], ConversationResourceSelectionRef);
	        this.skillRefs = this.convertValues(source["skillRefs"], ConversationResourceSelectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConversationSkillUsage {
	    artifact: artifact.ArtifactRef;
	    name?: string;
	    displayName?: string;
	    locator?: string;
	    selectedDefinitionDigest?: string;
	    usedDefinitionDigest?: string;
	    usedArtifactRevision?: number;
	    status: string;
	    changed?: boolean;
	    sessionAvailable?: boolean;
	    active?: boolean;
	    advertised?: boolean;
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationSkillUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.locator = source["locator"];
	        this.selectedDefinitionDigest = source["selectedDefinitionDigest"];
	        this.usedDefinitionDigest = source["usedDefinitionDigest"];
	        this.usedArtifactRevision = source["usedArtifactRevision"];
	        this.status = source["status"];
	        this.changed = source["changed"];
	        this.sessionAvailable = source["sessionAvailable"];
	        this.active = source["active"];
	        this.advertised = source["advertised"];
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConversationUsage {
	    workspace: artifact.ArtifactRef;
	    displayName?: string;
	    workspaceRevision?: number;
	    status: string;
	    contexts?: ConversationContextUsage[];
	    skills?: ConversationSkillUsage[];
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], artifact.ArtifactRef);
	        this.displayName = source["displayName"];
	        this.workspaceRevision = source["workspaceRevision"];
	        this.status = source["status"];
	        this.contexts = this.convertValues(source["contexts"], ConversationContextUsage);
	        this.skills = this.convertValues(source["skills"], ConversationSkillUsage);
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    server: string;
	    resourceUri?: string;
	    content?: server.MCPContent[];
	    structuredContent?: any;
	    updatedAt?: string;
	    rawArguments?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPAppModelContextUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.instanceID = source["instanceID"];
	        this.server = source["server"];
	        this.resourceUri = source["resourceUri"];
	        this.content = this.convertValues(source["content"], server.MCPContent);
	        this.structuredContent = source["structuredContent"];
	        this.updatedAt = source["updatedAt"];
	        this.rawArguments = source["rawArguments"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MCPPromptSelection {
	    server: string;
	    promptName: string;
	    title?: string;
	    displayName: string;
	    description?: string;
	    arguments?: Record<string, server.MCPArgumentDefinition>;
	    digest?: string;
	    argumentValues?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPPromptSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.promptName = source["promptName"];
	        this.title = source["title"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.arguments = this.convertValues(source["arguments"], server.MCPArgumentDefinition, true);
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
	export class MCPResourceTemplateSelection {
	    server: string;
	    uriTemplate: string;
	    name?: string;
	    title?: string;
	    displayName: string;
	    description?: string;
	    mimeType?: string;
	    arguments?: Record<string, server.MCPArgumentDefinition>;
	    annotations?: Record<string, any>;
	    digest?: string;
	    argumentValues?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourceTemplateSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.uriTemplate = source["uriTemplate"];
	        this.name = source["name"];
	        this.title = source["title"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.mimeType = source["mimeType"];
	        this.arguments = this.convertValues(source["arguments"], server.MCPArgumentDefinition, true);
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
	export class MCPToolSelection {
	    server: string;
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
	        this.server = source["server"];
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
	    server: string;
	    snapshotDigest?: string;
	    toolExposure: string;
	    selectedTools?: MCPToolSelection[];
	    includeServerInstructions?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
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
	    resources?: server.MCPResourceRef[];
	    resourceTemplates?: MCPResourceTemplateSelection[];
	    prompts?: MCPPromptSelection[];
	
	    static createFrom(source: any = {}) {
	        return new MCPConversationContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], MCPServerSelection);
	        this.resources = this.convertValues(source["resources"], server.MCPResourceRef);
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
	
	export class MCPProviderToolMapping {
	    server: string;
	    providerToolName: string;
	    choiceID: string;
	    toolName: string;
	    toolDigest: string;
	    approvalRule?: string;
	    executionMode?: string;
	    appResourceUri?: string;
	    visibility?: string[];
	
	    static createFrom(source: any = {}) {
	        return new MCPProviderToolMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.providerToolName = source["providerToolName"];
	        this.choiceID = source["choiceID"];
	        this.toolName = source["toolName"];
	        this.toolDigest = source["toolDigest"];
	        this.approvalRule = source["approvalRule"];
	        this.executionMode = source["executionMode"];
	        this.appResourceUri = source["appResourceUri"];
	        this.visibility = source["visibility"];
	    }
	}
	
	

}

export namespace declaration {
	
	export class Locator {
	    kind?: string;
	    path?: string;
	    url?: string;
	    integrity?: string;
	    repository?: string;
	    revision?: string;
	    manager?: string;
	    package?: string;
	    version?: string;
	    registry?: string;
	    command?: string;
	
	    static createFrom(source: any = {}) {
	        return new Locator(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.url = source["url"];
	        this.integrity = source["integrity"];
	        this.repository = source["repository"];
	        this.revision = source["revision"];
	        this.manager = source["manager"];
	        this.package = source["package"];
	        this.version = source["version"];
	        this.registry = source["registry"];
	        this.command = source["command"];
	    }
	}

}

export namespace diagnostic {
	
	export class Location {
	    locator?: string;
	    subresourceLocator?: string;
	    line?: number;
	    column?: number;
	
	    static createFrom(source: any = {}) {
	        return new Location(source);
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
	    location?: Location;
	
	    static createFrom(source: any = {}) {
	        return new Diagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.code = source["code"];
	        this.message = source["message"];
	        this.location = this.convertValues(source["location"], Location);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace document {
	
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
	export class SkillDocument {
	    name: string;
	    displayName?: string;
	    description: string;
	    insert: string;
	    arguments?: SkillArgument[];
	    tags?: string[];
	    markdownBody: string;
	    rawFrontmatter?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new SkillDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], SkillArgument);
	        this.tags = source["tags"];
	        this.markdownBody = source["markdownBody"];
	        this.rawFrontmatter = source["rawFrontmatter"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace domain {
	
	export class ManagedSkillDocument {
	    artifact: artifact.Artifact;
	    document: document.SkillDocument;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.document = this.convertValues(source["document"], document.SkillDocument);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    artifact: artifact.Artifact;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.description = source["description"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace main {
	
	export class MCPGlobalSettingsView {
	    settings: auth.MCPAuthSettings;
	    revision: number;
	    oauthRedirectURL?: string;
	    oauthLoopbackListenAddr?: string;
	    oauthRestartRequired: boolean;
	    oauthLoopbackReady: boolean;
	    oauthLoopbackError?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPGlobalSettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], auth.MCPAuthSettings);
	        this.revision = source["revision"];
	        this.oauthRedirectURL = source["oauthRedirectURL"];
	        this.oauthLoopbackListenAddr = source["oauthLoopbackListenAddr"];
	        this.oauthRestartRequired = source["oauthRestartRequired"];
	        this.oauthLoopbackReady = source["oauthLoopbackReady"];
	        this.oauthLoopbackError = source["oauthLoopbackError"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace policy {
	
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
	export class MCPPolicy {
	    trustLevel: string;
	    defaultPolicy: MCPServerPolicy;
	    toolPolicies?: Record<string, MCPToolPolicyOverride>;
	    appsPolicy: MCPAppsPolicy;
	
	    static createFrom(source: any = {}) {
	        return new MCPPolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trustLevel = source["trustLevel"];
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
	export class Effective {
	    body: MCPPolicy;
	    conflicts?: Record<string, string>;
	    digest: string;
	
	    static createFrom(source: any = {}) {
	        return new Effective(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.body = this.convertValues(source["body"], MCPPolicy);
	        this.conflicts = source["conflicts"];
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
	
	
	

}

export namespace provider {
	
	export class SkillDef {
	    type: string;
	    name: string;
	    location: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillDef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.location = source["location"];
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

}

export namespace resolve {
	
	export class MappedTarget {
	    provider: string;
	    identifier: string;
	    type: string;
	    name: string;
	    builtin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MappedTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.identifier = source["identifier"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.builtin = source["builtin"];
	    }
	}
	export class CapabilityOccurrence {
	    path: string;
	    kind: string;
	    type: string;
	    name?: string;
	    status: string;
	    required: boolean;
	    scope?: string;
	    artifact?: artifact.ArtifactRef;
	    mapped?: MappedTarget;
	    overrides?: Record<string, Array<number>>;
	    use?: Record<string, Array<number>>;
	    code?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new CapabilityOccurrence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.required = source["required"];
	        this.scope = source["scope"];
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.mapped = this.convertValues(source["mapped"], MappedTarget);
	        this.overrides = source["overrides"];
	        this.use = source["use"];
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CapabilityPlan {
	    rootArtifact?: artifact.ArtifactRef;
	    rootMapped?: MappedTarget;
	    rootType: string;
	    rootName: string;
	    occurrences: CapabilityOccurrence[];
	    complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CapabilityPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootArtifact = this.convertValues(source["rootArtifact"], artifact.ArtifactRef);
	        this.rootMapped = this.convertValues(source["rootMapped"], MappedTarget);
	        this.rootType = source["rootType"];
	        this.rootName = source["rootName"];
	        this.occurrences = this.convertValues(source["occurrences"], CapabilityOccurrence);
	        this.complete = source["complete"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ResolutionIssue {
	    code: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ResolutionIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}

}

export namespace root {
	
	export class Root {
	    id: string;
	    storageKey: string;
	    displayName: string;
	    description?: string;
	    revision: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    // Go type: time
	    retiredAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new Root(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.storageKey = source["storageKey"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.revision = source["revision"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.retiredAt = this.convertValues(source["retiredAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace runtime {
	
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
	export class CreateSkillSessionRequestBody {
	    closeSessionID?: string;
	    maxActivePerSession?: number;
	    allowedSkills?: provider.SkillDef[];
	    activeSkills?: provider.SkillDef[];
	
	    static createFrom(source: any = {}) {
	        return new CreateSkillSessionRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.closeSessionID = source["closeSessionID"];
	        this.maxActivePerSession = source["maxActivePerSession"];
	        this.allowedSkills = this.convertValues(source["allowedSkills"], provider.SkillDef);
	        this.activeSkills = this.convertValues(source["activeSkills"], provider.SkillDef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    activeSkills: provider.SkillDef[];
	
	    static createFrom(source: any = {}) {
	        return new CreateSkillSessionResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionID = source["sessionID"];
	        this.activeSkills = this.convertValues(source["activeSkills"], provider.SkillDef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	
	export class SkillPromptFilter {
	    types?: string[];
	    namePrefix?: string;
	    locationPrefix?: string;
	    allowSkills?: provider.SkillDef[];
	    sessionID?: string;
	    activity?: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillPromptFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.types = source["types"];
	        this.namePrefix = source["namePrefix"];
	        this.locationPrefix = source["locationPrefix"];
	        this.allowSkills = this.convertValues(source["allowSkills"], provider.SkillDef);
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
	    filter?: SkillPromptFilter;
	
	    static createFrom(source: any = {}) {
	        return new GetSkillsPromptRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = this.convertValues(source["filter"], SkillPromptFilter);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	
	export class InvokeSkillToolResponseBody {
	    outputs?: spec.ToolOutputUnion[];
	    meta?: Record<string, any>;
	    isBuiltIn: boolean;
	    isError?: boolean;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new InvokeSkillToolResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputs = this.convertValues(source["outputs"], spec.ToolOutputUnion);
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
	
	export class SkillListFilter {
	    types?: string[];
	    namePrefix?: string;
	    locationPrefix?: string;
	    allowSkills?: provider.SkillDef[];
	    inserts?: string[];
	    sessionID?: string;
	    activity?: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillListFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.types = source["types"];
	        this.namePrefix = source["namePrefix"];
	        this.locationPrefix = source["locationPrefix"];
	        this.allowSkills = this.convertValues(source["allowSkills"], provider.SkillDef);
	        this.inserts = source["inserts"];
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
	export class ListSkillsRequestBody {
	    filter?: SkillListFilter;
	
	    static createFrom(source: any = {}) {
	        return new ListSkillsRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = this.convertValues(source["filter"], SkillListFilter);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    Body?: ListSkillsRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new ListSkillsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListSkillsRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    skills: spec.SkillRecord[];
	
	    static createFrom(source: any = {}) {
	        return new ListSkillsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skills = this.convertValues(source["skills"], spec.SkillRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	
	export class RenderSkillOut {
	    name: string;
	    description?: string;
	    displayName?: string;
	    insert: string;
	    tags?: string[];
	    text: string;
	    arguments?: document.SkillArgument[];
	    appliedArguments?: Record<string, string>;
	    rawFrontmatter?: Record<string, any>;
	    warnings?: string[];
	    resources: provider.SkillResourceInfo;
	
	    static createFrom(source: any = {}) {
	        return new RenderSkillOut(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.displayName = source["displayName"];
	        this.insert = source["insert"];
	        this.tags = source["tags"];
	        this.text = source["text"];
	        this.arguments = this.convertValues(source["arguments"], document.SkillArgument);
	        this.appliedArguments = source["appliedArguments"];
	        this.rawFrontmatter = source["rawFrontmatter"];
	        this.warnings = source["warnings"];
	        this.resources = this.convertValues(source["resources"], provider.SkillResourceInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    definition: provider.SkillDef;
	    arguments?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RenderSkillRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.definition = this.convertValues(source["definition"], provider.SkillDef);
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
	
	export class RenderSkillResponse {
	    Body?: RenderSkillOut;
	
	    static createFrom(source: any = {}) {
	        return new RenderSkillResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], RenderSkillOut);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace server {
	
	export class AuthenticationDeclaration {
	    mode: string;
	    clientCredentialsInput?: string;
	    clientIDMetadataDocumentURL?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthenticationDeclaration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.clientCredentialsInput = source["clientCredentialsInput"];
	        this.clientIDMetadataDocumentURL = source["clientIDMetadataDocumentURL"];
	    }
	}
	export class HTTPProfile {
	    url?: string;
	    headers?: Record<string, string>;
	    removeHeaders?: string[];
	
	    static createFrom(source: any = {}) {
	        return new HTTPProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.headers = source["headers"];
	        this.removeHeaders = source["removeHeaders"];
	    }
	}
	export class StdioProfile {
	    command?: string;
	    args?: string[];
	    env?: Record<string, string>;
	    removeEnv?: string[];
	
	    static createFrom(source: any = {}) {
	        return new StdioProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	        this.removeEnv = source["removeEnv"];
	    }
	}
	export class ConnectionProfile {
	    platforms?: string[];
	    stdio?: StdioProfile;
	    http?: HTTPProfile;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platforms = source["platforms"];
	        this.stdio = this.convertValues(source["stdio"], StdioProfile);
	        this.http = this.convertValues(source["http"], HTTPProfile);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CoreServer {
	    type?: string;
	    command?: string;
	    args?: string[];
	    env?: Record<string, string>;
	    url?: string;
	    headers?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new CoreServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	        this.url = source["url"];
	        this.headers = source["headers"];
	    }
	}
	
	export class Include {
	    tools?: string[];
	    resources?: string[];
	    prompts?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Include(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tools = source["tools"];
	        this.resources = source["resources"];
	        this.prompts = source["prompts"];
	    }
	}
	export class InputBinding {
	    value?: string;
	    secretRef?: string;
	
	    static createFrom(source: any = {}) {
	        return new InputBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.secretRef = source["secretRef"];
	    }
	}
	export class InputDeclaration {
	    kind: string;
	    label?: string;
	    description?: string;
	    note?: string;
	    placeholder?: string;
	    required?: boolean;
	    default?: string;
	    clientSecretRequired?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InputDeclaration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.note = source["note"];
	        this.placeholder = source["placeholder"];
	        this.required = source["required"];
	        this.default = source["default"];
	        this.clientSecretRequired = source["clientSecretRequired"];
	    }
	}
	export class InstallationDeclaration {
	    note?: string;
	    inputs?: Record<string, InputDeclaration>;
	    allowEnvironment?: string[];
	
	    static createFrom(source: any = {}) {
	        return new InstallationDeclaration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.note = source["note"];
	        this.inputs = this.convertValues(source["inputs"], InputDeclaration, true);
	        this.allowEnvironment = source["allowEnvironment"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvokeMCPToolRequestBody {
	    source: string;
	    toolName: string;
	    providerToolName?: string;
	    choiceID?: string;
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
	        this.choiceID = source["choiceID"];
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
	    server: string;
	    catalog: string;
	    serverDisplayName?: string;
	    toolName: string;
	    providerToolName: string;
	    toolDigest?: string;
	    choiceID?: string;
	    toolUseID?: string;
	    approvalID?: string;
	    appResourceUri?: string;
	    appInstanceID?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPToolCallProvenance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.catalog = source["catalog"];
	        this.serverDisplayName = source["serverDisplayName"];
	        this.toolName = source["toolName"];
	        this.providerToolName = source["providerToolName"];
	        this.toolDigest = source["toolDigest"];
	        this.choiceID = source["choiceID"];
	        this.toolUseID = source["toolUseID"];
	        this.approvalID = source["approvalID"];
	        this.appResourceUri = source["appResourceUri"];
	        this.appInstanceID = source["appInstanceID"];
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
	export class InvokeMCPToolResponseBody {
	    server: string;
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
	        this.server = source["server"];
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
	export class MCPApprovalSummary {
	    server: string;
	    serverDisplayName?: string;
	    source: string;
	    appInstanceID?: string;
	    toolName: string;
	    toolDigest?: string;
	    risk: string;
	    arguments?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPApprovalSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.serverDisplayName = source["serverDisplayName"];
	        this.source = source["source"];
	        this.appInstanceID = source["appInstanceID"];
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
	export class MCPApprovalResolutionResult {
	    approvalID: string;
	    resolution: string;
	    decision: string;
	    rememberedForSession?: boolean;
	    token?: string;
	    expiresAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPApprovalResolutionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.approvalID = source["approvalID"];
	        this.resolution = source["resolution"];
	        this.decision = source["decision"];
	        this.rememberedForSession = source["rememberedForSession"];
	        this.token = source["token"];
	        this.expiresAt = source["expiresAt"];
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
	    server: string;
	    promptName: string;
	    description?: string;
	    messages?: MCPPromptMessage[];
	
	    static createFrom(source: any = {}) {
	        return new MCPGetPromptResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
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
	
	export class MCPPromptRef {
	    server: string;
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
	        this.server = source["server"];
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
	export class MCPPromptPage {
	    items: MCPPromptRef[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPPromptPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], MCPPromptRef);
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
	
	export class MCPReadResourceResponseBody {
	    server: string;
	    uri: string;
	    contents?: MCPContent[];
	
	    static createFrom(source: any = {}) {
	        return new MCPReadResourceResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
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
	
	export class MCPResourceRef {
	    server: string;
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
	        this.server = source["server"];
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
	export class MCPResourcePage {
	    items: MCPResourceRef[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourcePage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], MCPResourceRef);
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
	
	export class MCPResourceTemplateRef {
	    server: string;
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
	        this.server = source["server"];
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
	export class MCPResourceTemplatePage {
	    items: MCPResourceTemplateRef[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourceTemplatePage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], MCPResourceTemplateRef);
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
	export class MCPServerRuntimeSnapshot {
	    server: string;
	    catalog: string;
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
	        this.server = source["server"];
	        this.catalog = source["catalog"];
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
	
	
	export class MCPToolCapability {
	    server: string;
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
	        this.server = source["server"];
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
	export class MCPToolCapabilityPage {
	    items: MCPToolCapability[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPToolCapabilityPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], MCPToolCapability);
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
	export class PolicyReference {
	    name: string;
	    required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PolicyReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.required = source["required"];
	    }
	}
	export class ServerConfiguration {
	    timeoutMS?: number;
	    auth: AuthenticationDeclaration;
	    install: InstallationDeclaration;
	    connectionProfiles?: Record<string, ConnectionProfile>;
	    policy?: PolicyReference;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfiguration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeoutMS = source["timeoutMS"];
	        this.auth = this.convertValues(source["auth"], AuthenticationDeclaration);
	        this.install = this.convertValues(source["install"], InstallationDeclaration);
	        this.connectionProfiles = this.convertValues(source["connectionProfiles"], ConnectionProfile, true);
	        this.policy = this.convertValues(source["policy"], PolicyReference);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ServerData {
	    schemaVersion: string;
	    selectedConnectionProfile?: string;
	    inputs?: Record<string, InputBinding>;
	    additionalPolicies?: artifact.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new ServerData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.selectedConnectionProfile = source["selectedConnectionProfile"];
	        this.inputs = this.convertValues(source["inputs"], InputBinding, true);
	        this.additionalPolicies = this.convertValues(source["additionalPolicies"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ServerDocument {
	    logicalName: string;
	    logicalVersion?: string;
	    displayName?: string;
	    description?: string;
	    labels?: Record<string, string>;
	    mcpServer: CoreServer;
	    include?: Include;
	    configuration: ServerConfiguration;
	
	    static createFrom(source: any = {}) {
	        return new ServerDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.labels = source["labels"];
	        this.mcpServer = this.convertValues(source["mcpServer"], CoreServer);
	        this.include = this.convertValues(source["include"], Include);
	        this.configuration = this.convertValues(source["configuration"], ServerConfiguration);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace source {
	
	export class DecoderHint {
	    locator: string;
	    recursive?: boolean;
	    decoderIDs: string[];
	
	    static createFrom(source: any = {}) {
	        return new DecoderHint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.locator = source["locator"];
	        this.recursive = source["recursive"];
	        this.decoderIDs = source["decoderIDs"];
	    }
	}
	export class DirectoryRoot {
	    root: string;
	    recursive?: boolean;
	    includePatterns?: string[];
	    excludePatterns?: string[];
	
	    static createFrom(source: any = {}) {
	        return new DirectoryRoot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.recursive = source["recursive"];
	        this.includePatterns = source["includePatterns"];
	        this.excludePatterns = source["excludePatterns"];
	    }
	}
	export class DiscoverySpec {
	    explicitLocators?: string[];
	    directoryRoots?: DirectoryRoot[];
	    decoderHints?: DecoderHint[];
	    allowedDecoderIDs?: string[];
	    expectedContentDigests?: Record<string, string>;
	    authoritative?: boolean;
	    maxCandidateBytes?: number;
	    maxTotalBytes?: number;
	    maxCandidates?: number;
	    maxEntries?: number;
	    maxDepth?: number;
	
	    static createFrom(source: any = {}) {
	        return new DiscoverySpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.explicitLocators = source["explicitLocators"];
	        this.directoryRoots = this.convertValues(source["directoryRoots"], DirectoryRoot);
	        this.decoderHints = this.convertValues(source["decoderHints"], DecoderHint);
	        this.allowedDecoderIDs = source["allowedDecoderIDs"];
	        this.expectedContentDigests = source["expectedContentDigests"];
	        this.authoritative = source["authoritative"];
	        this.maxCandidateBytes = source["maxCandidateBytes"];
	        this.maxTotalBytes = source["maxTotalBytes"];
	        this.maxCandidates = source["maxCandidates"];
	        this.maxEntries = source["maxEntries"];
	        this.maxDepth = source["maxDepth"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedPackageFile {
	    locator: string;
	    content: number[];
	
	    static createFrom(source: any = {}) {
	        return new ManagedPackageFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.locator = source["locator"];
	        this.content = source["content"];
	    }
	}
	export class Summary {
	    id: string;
	    rootID: string;
	    rootStorageKey: string;
	    storageKey: string;
	    kind: string;
	    displayName: string;
	    enabled: boolean;
	    discovery: DiscoverySpec;
	    revision: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    // Go type: time
	    retiredAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rootID = source["rootID"];
	        this.rootStorageKey = source["rootStorageKey"];
	        this.storageKey = source["storageKey"];
	        this.kind = source["kind"];
	        this.displayName = source["displayName"];
	        this.enabled = source["enabled"];
	        this.discovery = this.convertValues(source["discovery"], DiscoverySpec);
	        this.revision = source["revision"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.retiredAt = this.convertValues(source["retiredAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    continuationFingerprint?: string;
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
	        this.continuationFingerprint = source["continuationFingerprint"];
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
	    mcpContext?: conversation.MCPConversationContext;
	    mcpToolMappings?: conversation.MCPProviderToolMapping[];
	    mcpAppContextUpdates?: conversation.MCPAppModelContextUpdate[];
	    workspaceSelection?: conversation.ConversationSelection;
	    workspaceUsage?: conversation.ConversationUsage;
	    attachments?: attachment.Attachment[];
	    enabledSkillRefs?: artifact.ArtifactRef[];
	    activeSkillRefs?: artifact.ArtifactRef[];
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
	        this.mcpContext = this.convertValues(source["mcpContext"], conversation.MCPConversationContext);
	        this.mcpToolMappings = this.convertValues(source["mcpToolMappings"], conversation.MCPProviderToolMapping);
	        this.mcpAppContextUpdates = this.convertValues(source["mcpAppContextUpdates"], conversation.MCPAppModelContextUpdate);
	        this.workspaceSelection = this.convertValues(source["workspaceSelection"], conversation.ConversationSelection);
	        this.workspaceUsage = this.convertValues(source["workspaceUsage"], conversation.ConversationUsage);
	        this.attachments = this.convertValues(source["attachments"], attachment.Attachment);
	        this.enabledSkillRefs = this.convertValues(source["enabledSkillRefs"], artifact.ArtifactRef);
	        this.activeSkillRefs = this.convertValues(source["activeSkillRefs"], artifact.ArtifactRef);
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
	    context?: string;
	    mode?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReasoningParam(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.level = source["level"];
	        this.tokens = source["tokens"];
	        this.summaryStyle = source["summaryStyle"];
	        this.context = source["context"];
	        this.mode = source["mode"];
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
	    mcpContext?: conversation.MCPConversationContext;
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
	        this.mcpContext = this.convertValues(source["mcpContext"], conversation.MCPConversationContext);
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
	    mcpToolMappings?: conversation.MCPProviderToolMapping[];
	    workspaceUsage?: conversation.ConversationUsage;
	
	    static createFrom(source: any = {}) {
	        return new CompletionResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inferenceResponse = this.convertValues(source["inferenceResponse"], FetchCompletionResponse);
	        this.hydratedCurrentInputs = this.convertValues(source["hydratedCurrentInputs"], InputUnion);
	        this.mcpToolMappings = this.convertValues(source["mcpToolMappings"], conversation.MCPProviderToolMapping);
	        this.workspaceUsage = this.convertValues(source["workspaceUsage"], conversation.ConversationUsage);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    autoExecute: boolean;
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
	        this.autoExecute = source["autoExecute"];
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
	    autoExecute: boolean;
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
	        this.autoExecute = source["autoExecute"];
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
	export class SkillRecord {
	    def: provider.SkillDef;
	    name: string;
	    description: string;
	    displayName?: string;
	    insert: string;
	    arguments?: document.SkillArgument[];
	    tags?: string[];
	    resources: provider.SkillResourceInfo;
	    rawFrontmatter?: Record<string, any>;
	    warnings?: string[];
	    digest?: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.def = this.convertValues(source["def"], provider.SkillDef);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.displayName = source["displayName"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], document.SkillArgument);
	        this.tags = source["tags"];
	        this.resources = this.convertValues(source["resources"], provider.SkillResourceInfo);
	        this.rawFrontmatter = source["rawFrontmatter"];
	        this.warnings = source["warnings"];
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

