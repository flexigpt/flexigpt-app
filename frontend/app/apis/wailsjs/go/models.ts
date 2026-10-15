export namespace agent {
	
	export class AgentCapabilityOccurrence {
	    path: string;
	    type: string;
	    name?: string;
	    status: string;
	    required: boolean;
	    target?: composition.CapabilityTarget;
	    autoExecute?: boolean;
	    includeSystemPrompt?: boolean;
	    skillUseMode?: string;
	    code?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentCapabilityOccurrence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.required = source["required"];
	        this.target = this.convertValues(source["target"], composition.CapabilityTarget);
	        this.autoExecute = source["autoExecute"];
	        this.includeSystemPrompt = source["includeSystemPrompt"];
	        this.skillUseMode = source["skillUseMode"];
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
	export class AgentCapabilityPlan {
	    occurrences: AgentCapabilityOccurrence[];
	    complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentCapabilityPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.occurrences = this.convertValues(source["occurrences"], AgentCapabilityOccurrence);
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
	export class AgentExportRequest {
	    agent: model.ArtifactRef;
	
	    static createFrom(source: any = {}) {
	        return new AgentExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], model.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    artifact?: model.ArtifactRef;
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
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
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
	export class AgentRestoredMembership {
	    plugin: model.ArtifactRef;
	    path: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentRestoredMembership(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
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
	    ref: model.ArtifactRef;
	    name: string;
	    displayName: string;
	    description?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    definitionDigest?: string;
	    builtIn: boolean;
	    managed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], model.ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
	        this.definitionDigest = source["definitionDigest"];
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
	    plugin: plugin.PluginView;
	    restoredMemberships?: AgentRestoredMembership[];
	    mcpSetupDescriptors?: AgentMCPSetupDescriptor[];
	    preparedFingerprint: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportCommitResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], AgentView);
	        this.plugin = this.convertValues(source["plugin"], plugin.PluginView);
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
	    plugin: model.ArtifactRef;
	    pluginRevision: number;
	    pluginName: string;
	    pluginDisplayName: string;
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
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.pluginRevision = source["pluginRevision"];
	        this.pluginName = source["pluginName"];
	        this.pluginDisplayName = source["pluginDisplayName"];
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
	    target?: composition.CapabilityTarget;
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
	        this.target = this.convertValues(source["target"], composition.CapabilityTarget);
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
	    plugin: model.ArtifactRef;
	    expectedPluginRevision: number;
	    expectedSourceDigest?: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentImportPreviewRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.expectedPluginRevision = source["expectedPluginRevision"];
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
	
	export class AgentListItem {
	    ref: model.ArtifactRef;
	    name: string;
	    displayName: string;
	    description?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    definitionDigest?: string;
	    builtIn: boolean;
	    managed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], model.ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
	        this.definitionDigest = source["definitionDigest"];
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
	
	
	export class AgentResolution {
	    agent: AgentView;
	    capabilities: AgentCapabilityPlan;
	
	    static createFrom(source: any = {}) {
	        return new AgentResolution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], AgentView);
	        this.capabilities = this.convertValues(source["capabilities"], AgentCapabilityPlan);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    plugin?: model.ArtifactRef;
	    includeBuiltin?: boolean;
	    enabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ListAgentsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.logicalNames = source["logicalNames"];
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
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
	    agent: model.ArtifactRef;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new ManagedAgentDeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], model.ArtifactRef);
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
	export class MCPAuthStatus {
	    server: string;
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
	        this.server = source["server"];
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

export namespace composition {
	
	export class CapabilityTarget {
	    form: string;
	    type: string;
	    name: string;
	    provenance: string;
	    artifact?: model.ArtifactRef;
	    providerIdentity?: string;
	    providerLocalID?: string;
	    evidence?: string;
	
	    static createFrom(source: any = {}) {
	        return new CapabilityTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.form = source["form"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.provenance = source["provenance"];
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
	        this.providerIdentity = source["providerIdentity"];
	        this.providerLocalID = source["providerLocalID"];
	        this.evidence = source["evidence"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	    target?: CapabilityTarget;
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
	        this.target = this.convertValues(source["target"], CapabilityTarget);
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
	    rootArtifact?: model.ArtifactRef;
	    rootTarget?: CapabilityTarget;
	    rootType: string;
	    rootName: string;
	    occurrences: CapabilityOccurrence[];
	    complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CapabilityPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootArtifact = this.convertValues(source["rootArtifact"], model.ArtifactRef);
	        this.rootTarget = this.convertValues(source["rootTarget"], CapabilityTarget);
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
	
	export class ArtifactNameReference {
	    name: string;
	    scope?: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactNameReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.scope = source["scope"];
	    }
	}
	export class Authentication {
	    mode: string;
	    headerName?: string;
	    prefix?: string;
	
	    static createFrom(source: any = {}) {
	        return new Authentication(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.headerName = source["headerName"];
	        this.prefix = source["prefix"];
	    }
	}
	export class CacheControlCapabilities {
	    supportsTTL?: boolean;
	    supportedKinds?: string[];
	    supportedTTLs?: string[];
	    supportsKey?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CacheControlCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportsTTL = source["supportsTTL"];
	        this.supportedKinds = source["supportedKinds"];
	        this.supportedTTLs = source["supportedTTLs"];
	        this.supportsKey = source["supportsKey"];
	    }
	}
	export class CacheCapabilities {
	    supportsAutomaticCaching?: boolean;
	    topLevel?: CacheControlCapabilities;
	    inputOutputContent?: CacheControlCapabilities;
	    reasoningContent?: CacheControlCapabilities;
	    toolChoice?: CacheControlCapabilities;
	    toolCall?: CacheControlCapabilities;
	    toolOutput?: CacheControlCapabilities;
	
	    static createFrom(source: any = {}) {
	        return new CacheCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportsAutomaticCaching = source["supportsAutomaticCaching"];
	        this.topLevel = this.convertValues(source["topLevel"], CacheControlCapabilities);
	        this.inputOutputContent = this.convertValues(source["inputOutputContent"], CacheControlCapabilities);
	        this.reasoningContent = this.convertValues(source["reasoningContent"], CacheControlCapabilities);
	        this.toolChoice = this.convertValues(source["toolChoice"], CacheControlCapabilities);
	        this.toolCall = this.convertValues(source["toolCall"], CacheControlCapabilities);
	        this.toolOutput = this.convertValues(source["toolOutput"], CacheControlCapabilities);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CacheControl {
	    kind?: string;
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
	
	export class ParameterDialect {
	    maxOutputTokensParamName?: string;
	    toolChoiceParamStyle?: string;
	
	    static createFrom(source: any = {}) {
	        return new ParameterDialect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxOutputTokensParamName = source["maxOutputTokensParamName"];
	        this.toolChoiceParamStyle = source["toolChoiceParamStyle"];
	    }
	}
	export class ToolCapabilities {
	    supportedToolTypes?: string[];
	    supportedToolPolicyModes?: string[];
	    supportsParallelToolCalls?: boolean;
	    maxForcedTools?: number;
	    supportedClientToolOutputFormats?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ToolCapabilities(source);
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
	export class OutputCapabilities {
	    supportedOutputFormats?: string[];
	    supportsVerbosity?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OutputCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportedOutputFormats = source["supportedOutputFormats"];
	        this.supportsVerbosity = source["supportsVerbosity"];
	    }
	}
	export class StopSequenceCapabilities {
	    isSupported?: boolean;
	    disallowedWithReasoning?: boolean;
	    maxSequences?: number;
	
	    static createFrom(source: any = {}) {
	        return new StopSequenceCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isSupported = source["isSupported"];
	        this.disallowedWithReasoning = source["disallowedWithReasoning"];
	        this.maxSequences = source["maxSequences"];
	    }
	}
	export class ReasoningTokenBudgetCapabilities {
	    minAllowed?: number;
	    maxAllowed?: number;
	    zeroAllowed?: boolean;
	    minusOneAllowed?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReasoningTokenBudgetCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minAllowed = source["minAllowed"];
	        this.maxAllowed = source["maxAllowed"];
	        this.zeroAllowed = source["zeroAllowed"];
	        this.minusOneAllowed = source["minusOneAllowed"];
	    }
	}
	export class ReasoningCapabilities {
	    supportsReasoningConfig?: boolean;
	    supportedReasoningTypes?: string[];
	    supportedReasoningLevels?: string[];
	    hybridTokenBudgetCapabilities?: ReasoningTokenBudgetCapabilities;
	    supportsSummaryStyle?: boolean;
	    supportsReasoningContext?: boolean;
	    supportsReasoningMode?: boolean;
	    supportsEncryptedReasoningInput?: boolean;
	    temperatureDisallowedWhenEnabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReasoningCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supportsReasoningConfig = source["supportsReasoningConfig"];
	        this.supportedReasoningTypes = source["supportedReasoningTypes"];
	        this.supportedReasoningLevels = source["supportedReasoningLevels"];
	        this.hybridTokenBudgetCapabilities = this.convertValues(source["hybridTokenBudgetCapabilities"], ReasoningTokenBudgetCapabilities);
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
	export class CapabilitiesPatch {
	    modalitiesIn?: string[];
	    modalitiesOut?: string[];
	    reasoningCapabilities?: ReasoningCapabilities;
	    stopSequenceCapabilities?: StopSequenceCapabilities;
	    outputCapabilities?: OutputCapabilities;
	    toolCapabilities?: ToolCapabilities;
	    cacheCapabilities?: CacheCapabilities;
	    paramDialect?: ParameterDialect;
	
	    static createFrom(source: any = {}) {
	        return new CapabilitiesPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modalitiesIn = source["modalitiesIn"];
	        this.modalitiesOut = source["modalitiesOut"];
	        this.reasoningCapabilities = this.convertValues(source["reasoningCapabilities"], ReasoningCapabilities);
	        this.stopSequenceCapabilities = this.convertValues(source["stopSequenceCapabilities"], StopSequenceCapabilities);
	        this.outputCapabilities = this.convertValues(source["outputCapabilities"], OutputCapabilities);
	        this.toolCapabilities = this.convertValues(source["toolCapabilities"], ToolCapabilities);
	        this.cacheCapabilities = this.convertValues(source["cacheCapabilities"], CacheCapabilities);
	        this.paramDialect = this.convertValues(source["paramDialect"], ParameterDialect);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HeaderPatch {
	    set?: Record<string, string>;
	    remove?: string[];
	
	    static createFrom(source: any = {}) {
	        return new HeaderPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.set = source["set"];
	        this.remove = source["remove"];
	    }
	}
	export class ConnectionPatch {
	    origin?: string;
	    path?: string;
	    headers?: HeaderPatch;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.origin = source["origin"];
	        this.path = source["path"];
	        this.headers = this.convertValues(source["headers"], HeaderPatch);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OutputJSONSchemaPatch {
	    name?: string;
	    description?: string;
	    schema?: any;
	    strict?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OutputJSONSchemaPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.schema = source["schema"];
	        this.strict = source["strict"];
	    }
	}
	export class OutputFormatPatch {
	    kind?: string;
	    jsonSchema?: OutputJSONSchemaPatch;
	
	    static createFrom(source: any = {}) {
	        return new OutputFormatPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.jsonSchema = this.convertValues(source["jsonSchema"], OutputJSONSchemaPatch);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OutputPatch {
	    verbosity?: string;
	    format?: OutputFormatPatch;
	
	    static createFrom(source: any = {}) {
	        return new OutputPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.verbosity = source["verbosity"];
	        this.format = this.convertValues(source["format"], OutputFormatPatch);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReasoningDefaults {
	    type?: string;
	    level?: string;
	    tokens?: number;
	    summaryStyle?: string;
	    context?: string;
	    mode?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReasoningDefaults(source);
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
	export class DefaultsPatch {
	    stream?: boolean;
	    maxPromptTokens?: number;
	    maxOutputTokens?: number;
	    temperature?: number;
	    systemPrompt?: string;
	    timeoutMS?: number;
	    reasoning?: ReasoningDefaults;
	    cacheControl?: CacheControl;
	    output?: OutputPatch;
	    stopSequences?: string[];
	    adapterParameters?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new DefaultsPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stream = source["stream"];
	        this.maxPromptTokens = source["maxPromptTokens"];
	        this.maxOutputTokens = source["maxOutputTokens"];
	        this.temperature = source["temperature"];
	        this.systemPrompt = source["systemPrompt"];
	        this.timeoutMS = source["timeoutMS"];
	        this.reasoning = this.convertValues(source["reasoning"], ReasoningDefaults);
	        this.cacheControl = this.convertValues(source["cacheControl"], CacheControl);
	        this.output = this.convertValues(source["output"], OutputPatch);
	        this.stopSequences = source["stopSequences"];
	        this.adapterParameters = source["adapterParameters"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class MembershipPolicy {
	    mode: string;
	    allowedTypes?: string[];
	    allowedForms?: string[];
	
	    static createFrom(source: any = {}) {
	        return new MembershipPolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.allowedTypes = source["allowedTypes"];
	        this.allowedForms = source["allowedForms"];
	    }
	}
	export class ModelDocument {
	    type: string;
	    name: string;
	    displayName?: string;
	    description?: string;
	    labels?: Record<string, string>;
	    provider: ArtifactNameReference;
	    providerModelID: string;
	    defaults?: DefaultsPatch;
	    capabilities?: CapabilitiesPatch;
	    adapterParameters?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ModelDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.labels = source["labels"];
	        this.provider = this.convertValues(source["provider"], ArtifactNameReference);
	        this.providerModelID = source["providerModelID"];
	        this.defaults = this.convertValues(source["defaults"], DefaultsPatch);
	        this.capabilities = this.convertValues(source["capabilities"], CapabilitiesPatch);
	        this.adapterParameters = source["adapterParameters"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModelSettings {
	    revision: number;
	    defaults?: DefaultsPatch;
	    capabilities?: CapabilitiesPatch;
	    adapterParameters?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ModelSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.defaults = this.convertValues(source["defaults"], DefaultsPatch);
	        this.capabilities = this.convertValues(source["capabilities"], CapabilitiesPatch);
	        this.adapterParameters = source["adapterParameters"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	export class ProviderDocument {
	    type: string;
	    name: string;
	    displayName?: string;
	    description?: string;
	    labels?: Record<string, string>;
	    adapter: string;
	    connection?: ConnectionPatch;
	    authentication?: Authentication;
	    defaultModel?: ArtifactNameReference;
	    defaults?: DefaultsPatch;
	    capabilities?: CapabilitiesPatch;
	    adapterParameters?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ProviderDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.labels = source["labels"];
	        this.adapter = source["adapter"];
	        this.connection = this.convertValues(source["connection"], ConnectionPatch);
	        this.authentication = this.convertValues(source["authentication"], Authentication);
	        this.defaultModel = this.convertValues(source["defaultModel"], ArtifactNameReference);
	        this.defaults = this.convertValues(source["defaults"], DefaultsPatch);
	        this.capabilities = this.convertValues(source["capabilities"], CapabilitiesPatch);
	        this.adapterParameters = source["adapterParameters"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProviderSettings {
	    revision: number;
	    connection?: ConnectionPatch;
	    defaults?: DefaultsPatch;
	    capabilities?: CapabilitiesPatch;
	    defaultModel?: ArtifactNameReference;
	    adapterParameters?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ProviderSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.connection = this.convertValues(source["connection"], ConnectionPatch);
	        this.defaults = this.convertValues(source["defaults"], DefaultsPatch);
	        this.capabilities = this.convertValues(source["capabilities"], CapabilitiesPatch);
	        this.defaultModel = this.convertValues(source["defaultModel"], ArtifactNameReference);
	        this.adapterParameters = source["adapterParameters"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    artifact: model.Artifact;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
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
	
	export class ArtifactSkillFilter {
	    types?: string[];
	    inserts?: string[];
	    namePrefix?: string;
	    locationPrefix?: string;
	    allowArtifacts?: model.ArtifactRef[];
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
	        this.allowArtifacts = this.convertValues(source["allowArtifacts"], model.ArtifactRef);
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
	    Artifact: model.ArtifactRef;
	    IsEnabled: boolean;
	    Insert: string;
	    HasArguments: boolean;
	    HasResources: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSkillSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Artifact = this.convertValues(source["Artifact"], model.ArtifactRef);
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
	export class MCPServerDetails {
	    settings: mcp.ServerInstallationView;
	    policy: mcppolicy.Effective;
	    authorization: auth.MCPAuthHealth;
	    connection: server.MCPServerRuntimeSnapshot;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], mcp.ServerInstallationView);
	        this.policy = this.convertValues(source["policy"], mcppolicy.Effective);
	        this.authorization = this.convertValues(source["authorization"], auth.MCPAuthHealth);
	        this.connection = this.convertValues(source["connection"], server.MCPServerRuntimeSnapshot);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MCPServerRuntimeDetails {
	    ref: model.ArtifactRef;
	    connection: server.MCPServerRuntimeSnapshot;
	    authorization?: auth.MCPAuthStatus;
	    pendingAuthorization?: auth.MCPOAuthAuthorization;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerRuntimeDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], model.ArtifactRef);
	        this.connection = this.convertValues(source["connection"], server.MCPServerRuntimeSnapshot);
	        this.authorization = this.convertValues(source["authorization"], auth.MCPAuthStatus);
	        this.pendingAuthorization = this.convertValues(source["pendingAuthorization"], auth.MCPOAuthAuthorization);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MCPSettingsView {
	    settings: auth.MCPAuthSettings;
	    revision: number;
	
	    static createFrom(source: any = {}) {
	        return new MCPSettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], auth.MCPAuthSettings);
	        this.revision = source["revision"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace mcp {
	
	export class InstallationInputView {
	    value?: string;
	    secretConfigured: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InstallationInputView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.secretConfigured = source["secretConfigured"];
	    }
	}
	export class ManagedMCPCreateRequest {
	    plugin: model.ArtifactRef;
	    expectedPluginRevision: number;
	    document: server.ServerDocument;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPCreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.expectedPluginRevision = source["expectedPluginRevision"];
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
	    artifact: model.Artifact;
	    address: model.ArtifactAddress;
	    plugin: plugin.PluginView;
	    membershipCreated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPCreateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.address = this.convertValues(source["address"], model.ArtifactAddress);
	        this.plugin = this.convertValues(source["plugin"], plugin.PluginView);
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
	    plugin: model.ArtifactRef;
	    expectedPluginRevision: number;
	    name: string;
	    description?: string;
	    policy: mcppolicy.MCPPolicy;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPPolicyUpsertRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.expectedPluginRevision = source["expectedPluginRevision"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.policy = this.convertValues(source["policy"], mcppolicy.MCPPolicy);
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
	    artifact: model.Artifact;
	    address: model.ArtifactAddress;
	    plugin: plugin.PluginView;
	    membershipCreated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPPolicyUpsertResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.address = this.convertValues(source["address"], model.ArtifactAddress);
	        this.plugin = this.convertValues(source["plugin"], plugin.PluginView);
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
	    plugin: model.ArtifactRef;
	    expectedPluginRevision: number;
	    artifact: model.ArtifactRef;
	    expectedArtifactRevision: number;
	    document: server.ServerDocument;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPReplaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.expectedPluginRevision = source["expectedPluginRevision"];
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
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
	    artifact: model.Artifact;
	    address: model.ArtifactAddress;
	    plugin: plugin.PluginView;
	
	    static createFrom(source: any = {}) {
	        return new ManagedMCPReplaceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.address = this.convertValues(source["address"], model.ArtifactAddress);
	        this.plugin = this.convertValues(source["plugin"], plugin.PluginView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PluginPage {
	    items: plugin.ListItem[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], plugin.ListItem);
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
	export class PolicyListItem {
	    ref: model.ArtifactRef;
	    name: string;
	    displayName: string;
	    description?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    definitionDigest?: string;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PolicyListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], model.ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
	        this.definitionDigest = source["definitionDigest"];
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
	export class PolicyView {
	    artifact: model.Artifact;
	    body: mcppolicy.MCPPolicy;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PolicyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.body = this.convertValues(source["body"], mcppolicy.MCPPolicy);
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
	export class ServerInstallationDataView {
	    selectedConnectionProfile?: string;
	    inputs?: Record<string, InstallationInputView>;
	    additionalPolicies?: model.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new ServerInstallationDataView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selectedConnectionProfile = source["selectedConnectionProfile"];
	        this.inputs = this.convertValues(source["inputs"], InstallationInputView, true);
	        this.additionalPolicies = this.convertValues(source["additionalPolicies"], model.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    artifact: model.Artifact;
	    document: server.ServerDocument;
	    installation: ServerInstallationDataView;
	    installationRevision: number;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerInstallationView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.document = this.convertValues(source["document"], server.ServerDocument);
	        this.installation = this.convertValues(source["installation"], ServerInstallationDataView);
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
	export class ServerListItem {
	    ref: model.ArtifactRef;
	    name: string;
	    displayName: string;
	    description?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    definitionDigest?: string;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], model.ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
	        this.definitionDigest = source["definitionDigest"];
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
	    items: ServerListItem[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ServerListItem);
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
	export class ServerSecretInputView {
	    name: string;
	    label?: string;
	    description?: string;
	    required: boolean;
	    kind: string;
	    configured: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerSecretInputView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.required = source["required"];
	        this.kind = source["kind"];
	        this.configured = source["configured"];
	    }
	}
	export class ServerSecretsView {
	    inputs: ServerSecretInputView[];
	
	    static createFrom(source: any = {}) {
	        return new ServerSecretsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputs = this.convertValues(source["inputs"], ServerSecretInputView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace mcppolicy {
	
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

export namespace model {
	
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
	export class ManagedModelCreateRequest {
	    rootID: string;
	    document: domain.ModelDocument;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedModelCreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.document = this.convertValues(source["document"], domain.ModelDocument);
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
	export class ManagedModelCreateResult {
	    artifact: Artifact;
	    address: ArtifactAddress;
	
	    static createFrom(source: any = {}) {
	        return new ManagedModelCreateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], Artifact);
	        this.address = this.convertValues(source["address"], ArtifactAddress);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedModelReplaceRequest {
	    model: ArtifactRef;
	    expectedArtifactRevision: number;
	    document: domain.ModelDocument;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedModelReplaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = this.convertValues(source["model"], ArtifactRef);
	        this.expectedArtifactRevision = source["expectedArtifactRevision"];
	        this.document = this.convertValues(source["document"], domain.ModelDocument);
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
	export class ManagedModelReplaceResult {
	    artifact: Artifact;
	    address: ArtifactAddress;
	
	    static createFrom(source: any = {}) {
	        return new ManagedModelReplaceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], Artifact);
	        this.address = this.convertValues(source["address"], ArtifactAddress);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class ManagedProviderCreateRequest {
	    rootID: string;
	    document: domain.ProviderDocument;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedProviderCreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.document = this.convertValues(source["document"], domain.ProviderDocument);
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
	export class ManagedProviderCreateResult {
	    artifact: Artifact;
	    address: ArtifactAddress;
	
	    static createFrom(source: any = {}) {
	        return new ManagedProviderCreateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], Artifact);
	        this.address = this.convertValues(source["address"], ArtifactAddress);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedProviderReplaceRequest {
	    provider: ArtifactRef;
	    expectedArtifactRevision: number;
	    document: domain.ProviderDocument;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedProviderReplaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = this.convertValues(source["provider"], ArtifactRef);
	        this.expectedArtifactRevision = source["expectedArtifactRevision"];
	        this.document = this.convertValues(source["document"], domain.ProviderDocument);
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
	export class ManagedProviderReplaceResult {
	    artifact: Artifact;
	    address: ArtifactAddress;
	
	    static createFrom(source: any = {}) {
	        return new ManagedProviderReplaceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], Artifact);
	        this.address = this.convertValues(source["address"], ArtifactAddress);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModelListItem {
	    ref: ArtifactRef;
	    name: string;
	    displayName: string;
	    description?: string;
	    provider?: domain.ArtifactNameReference;
	    providerModelID?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModelListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.provider = this.convertValues(source["provider"], domain.ArtifactNameReference);
	        this.providerModelID = source["providerModelID"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
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
	export class ModelView {
	    artifact: Artifact;
	    definitionDigest: string;
	    document: domain.ModelDocument;
	    settings: domain.ModelSettings;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModelView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], Artifact);
	        this.definitionDigest = source["definitionDigest"];
	        this.document = this.convertValues(source["document"], domain.ModelDocument);
	        this.settings = this.convertValues(source["settings"], domain.ModelSettings);
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
	export class ProviderAPIKeyStatus {
	    providerRevision: number;
	    apiKeyRevision: number;
	    configured: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderAPIKeyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerRevision = source["providerRevision"];
	        this.apiKeyRevision = source["apiKeyRevision"];
	        this.configured = source["configured"];
	    }
	}
	export class ProviderListItem {
	    ref: ArtifactRef;
	    name: string;
	    displayName: string;
	    description?: string;
	    adapter?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.adapter = source["adapter"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
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
	export class ProviderView {
	    artifact: Artifact;
	    definitionDigest: string;
	    document: domain.ProviderDocument;
	    settings: domain.ProviderSettings;
	    defaultModel?: domain.ArtifactNameReference;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], Artifact);
	        this.definitionDigest = source["definitionDigest"];
	        this.document = this.convertValues(source["document"], domain.ProviderDocument);
	        this.settings = this.convertValues(source["settings"], domain.ProviderSettings);
	        this.defaultModel = this.convertValues(source["defaultModel"], domain.ArtifactNameReference);
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
	export class SaveModelSettingsRequest {
	    model: ArtifactRef;
	    expectedModelRevision: number;
	    expectedSettingsRevision: number;
	    defaults?: domain.DefaultsPatch;
	    capabilities?: domain.CapabilitiesPatch;
	    adapterParameters?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new SaveModelSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = this.convertValues(source["model"], ArtifactRef);
	        this.expectedModelRevision = source["expectedModelRevision"];
	        this.expectedSettingsRevision = source["expectedSettingsRevision"];
	        this.defaults = this.convertValues(source["defaults"], domain.DefaultsPatch);
	        this.capabilities = this.convertValues(source["capabilities"], domain.CapabilitiesPatch);
	        this.adapterParameters = source["adapterParameters"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SaveProviderSettingsRequest {
	    provider: ArtifactRef;
	    expectedProviderRevision: number;
	    expectedSettingsRevision: number;
	    connection?: domain.ConnectionPatch;
	    defaults?: domain.DefaultsPatch;
	    capabilities?: domain.CapabilitiesPatch;
	    defaultModel?: domain.ArtifactNameReference;
	    adapterParameters?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new SaveProviderSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = this.convertValues(source["provider"], ArtifactRef);
	        this.expectedProviderRevision = source["expectedProviderRevision"];
	        this.expectedSettingsRevision = source["expectedSettingsRevision"];
	        this.connection = this.convertValues(source["connection"], domain.ConnectionPatch);
	        this.defaults = this.convertValues(source["defaults"], domain.DefaultsPatch);
	        this.capabilities = this.convertValues(source["capabilities"], domain.CapabilitiesPatch);
	        this.defaultModel = this.convertValues(source["defaultModel"], domain.ArtifactNameReference);
	        this.adapterParameters = source["adapterParameters"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SetProviderAPIKeyRequest {
	    provider: ArtifactRef;
	    expectedProviderRevision: number;
	    expectedAPIKeyRevision: number;
	    apiKey: string;
	
	    static createFrom(source: any = {}) {
	        return new SetProviderAPIKeyRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = this.convertValues(source["provider"], ArtifactRef);
	        this.expectedProviderRevision = source["expectedProviderRevision"];
	        this.expectedAPIKeyRevision = source["expectedAPIKeyRevision"];
	        this.apiKey = source["apiKey"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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

export namespace plugin {
	
	export class AddArtifactMemberRequest {
	    plugin: model.ArtifactRef;
	    expectedRevision: number;
	    artifact: model.ArtifactRef;
	
	    static createFrom(source: any = {}) {
	        return new AddArtifactMemberRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.expectedRevision = source["expectedRevision"];
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    plugin: model.ArtifactRef;
	    expectedRevision: number;
	    member: MemberReference;
	
	    static createFrom(source: any = {}) {
	        return new AddMemberRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
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
	    plugin: model.ArtifactRef;
	    pluginName: string;
	    pluginRevision: number;
	    memberIndex: number;
	    member: MemberReference;
	    status: string;
	    resolvedArtifact?: model.ArtifactRef;
	    resolvedToArtifact: boolean;
	    code?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactMembershipView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.pluginName = source["pluginName"];
	        this.pluginRevision = source["pluginRevision"];
	        this.memberIndex = source["memberIndex"];
	        this.member = this.convertValues(source["member"], MemberReference);
	        this.status = source["status"];
	        this.resolvedArtifact = this.convertValues(source["resolvedArtifact"], model.ArtifactRef);
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
	export class CreateRequest {
	    rootID: string;
	    sourceID?: string;
	    name: string;
	    displayName?: string;
	    description?: string;
	    membershipPolicy?: domain.MembershipPolicy;
	
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
	        this.membershipPolicy = this.convertValues(source["membershipPolicy"], domain.MembershipPolicy);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeleteRequest {
	    plugin: model.ArtifactRef;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
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
	export class DirectSelectorMatch {
	    artifact: model.ArtifactRef;
	    status: string;
	    target?: composition.CapabilityTarget;
	    code?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new DirectSelectorMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
	        this.status = source["status"];
	        this.target = this.convertValues(source["target"], composition.CapabilityTarget);
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
	export class PluginMemberView {
	    type: string;
	    name?: string;
	    insert?: string;
	    locator?: declaration.Locator;
	    server?: string;
	    contained: boolean;
	    selector: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PluginMemberView(source);
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
	export class DirectMember {
	    index: number;
	    entry: PluginMemberView;
	    status: string;
	    target?: composition.CapabilityTarget;
	    selectorMatches?: DirectSelectorMatch[];
	    code?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new DirectMember(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.entry = this.convertValues(source["entry"], PluginMemberView);
	        this.status = source["status"];
	        this.target = this.convertValues(source["target"], composition.CapabilityTarget);
	        this.selectorMatches = this.convertValues(source["selectorMatches"], DirectSelectorMatch);
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
	export class PluginView {
	    artifact: model.Artifact;
	    name: string;
	    displayName: string;
	    description?: string;
	    members: MemberReference[];
	    entries: PluginMemberView[];
	    editable: boolean;
	    deletable: boolean;
	    baseline: boolean;
	    membership: domain.MembershipPolicy;
	
	    static createFrom(source: any = {}) {
	        return new PluginView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.members = this.convertValues(source["members"], MemberReference);
	        this.entries = this.convertValues(source["entries"], PluginMemberView);
	        this.editable = source["editable"];
	        this.deletable = source["deletable"];
	        this.baseline = source["baseline"];
	        this.membership = this.convertValues(source["membership"], domain.MembershipPolicy);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DirectMembership {
	    plugin: PluginView;
	    members: DirectMember[];
	    complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DirectMembership(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], PluginView);
	        this.members = this.convertValues(source["members"], DirectMember);
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
	
	export class ListItem {
	    ref: model.ArtifactRef;
	    sourceID: string;
	    name: string;
	    displayName: string;
	    description?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    memberCount: number;
	    builtIn: boolean;
	    editable: boolean;
	    deletable: boolean;
	    baseline: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], model.ArtifactRef);
	        this.sourceID = source["sourceID"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
	        this.memberCount = source["memberCount"];
	        this.builtIn = source["builtIn"];
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
	
	export class PluginCapabilityPlan {
	    plugin: PluginView;
	    occurrences: composition.CapabilityOccurrence[];
	    complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PluginCapabilityPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], PluginView);
	        this.occurrences = this.convertValues(source["occurrences"], composition.CapabilityOccurrence);
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
	
	
	export class RemoveMemberRequest {
	    plugin: model.ArtifactRef;
	    expectedRevision: number;
	    index: number;
	
	    static createFrom(source: any = {}) {
	        return new RemoveMemberRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
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
	    plugin: model.ArtifactRef;
	    expectedRevision: number;
	    description?: string;
	    displayName?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
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
	    generation: number;
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
	        this.generation = source["generation"];
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
	    additionalPolicies?: model.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new ServerData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.selectedConnectionProfile = source["selectedConnectionProfile"];
	        this.inputs = this.convertValues(source["inputs"], InputBinding, true);
	        this.additionalPolicies = this.convertValues(source["additionalPolicies"], model.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace session {
	
	export class ArtifactSelection {
	    rootID: string;
	    artifactID: string;
	    expectedRevision?: number;
	    expectedDigest?: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.artifactID = source["artifactID"];
	        this.expectedRevision = source["expectedRevision"];
	        this.expectedDigest = source["expectedDigest"];
	    }
	}
	export class ActiveSkillOverride {
	    key: string;
	    mode: string;
	    value?: ArtifactSelection;
	
	    static createFrom(source: any = {}) {
	        return new ActiveSkillOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.mode = source["mode"];
	        this.value = this.convertValues(source["value"], ArtifactSelection);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppContextContent {
	    type: string;
	    text?: string;
	    uri?: string;
	    name?: string;
	    description?: string;
	    mimeType?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppContextContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.text = source["text"];
	        this.uri = source["uri"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.mimeType = source["mimeType"];
	    }
	}
	export class AppContextUpdate {
	    appInstanceID: string;
	    viewID: string;
	    serverID: string;
	    resourceURI?: string;
	    content?: AppContextContent[];
	    structuredContent?: number[];
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AppContextUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appInstanceID = source["appInstanceID"];
	        this.viewID = source["viewID"];
	        this.serverID = source["serverID"];
	        this.resourceURI = source["resourceURI"];
	        this.content = this.convertValues(source["content"], AppContextContent);
	        this.structuredContent = source["structuredContent"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppInstance {
	    id: string;
	    sessionID: string;
	    runID: string;
	    originOperationID: string;
	    originToolCallID: string;
	    serverID: string;
	    resourceURI?: string;
	    // Go type: time
	    createdAt: any;
	    views?: string[];
	    unavailableReason?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInstance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.originOperationID = source["originOperationID"];
	        this.originToolCallID = source["originToolCallID"];
	        this.serverID = source["serverID"];
	        this.resourceURI = source["resourceURI"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.views = source["views"];
	        this.unavailableReason = source["unavailableReason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppResourceReadResult {
	    content: number[];
	
	    static createFrom(source: any = {}) {
	        return new AppResourceReadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	    }
	}
	export class AppRoute {
	    serverID: string;
	    resourceURI?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppRoute(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverID = source["serverID"];
	        this.resourceURI = source["resourceURI"];
	    }
	}
	export class ArtifactOverride {
	    key: string;
	    mode: string;
	    value?: ArtifactSelection;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.mode = source["mode"];
	        this.value = this.convertValues(source["value"], ArtifactSelection);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ArtifactReference {
	    rootID: string;
	    artifactID: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.artifactID = source["artifactID"];
	    }
	}
	
	export class DirectoryAttachmentReference {
	    path: string;
	    originalPath?: string;
	    name?: string;
	    scannedFileCount?: number;
	    hasMore?: boolean;
	    expanded?: AttachmentReference[];
	
	    static createFrom(source: any = {}) {
	        return new DirectoryAttachmentReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.originalPath = source["originalPath"];
	        this.name = source["name"];
	        this.scannedFileCount = source["scannedFileCount"];
	        this.hasMore = source["hasMore"];
	        this.expanded = this.convertValues(source["expanded"], AttachmentReference);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GenericAttachmentReference {
	    handle: string;
	    originalHandle?: string;
	
	    static createFrom(source: any = {}) {
	        return new GenericAttachmentReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.handle = source["handle"];
	        this.originalHandle = source["originalHandle"];
	    }
	}
	export class URLAttachmentReference {
	    url: string;
	    normalized?: string;
	    original?: string;
	
	    static createFrom(source: any = {}) {
	        return new URLAttachmentReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.normalized = source["normalized"];
	        this.original = source["original"];
	    }
	}
	export class LocalAttachmentReference {
	    path?: string;
	    originalPath?: string;
	    name?: string;
	    exists: boolean;
	    isDirectory?: boolean;
	    size?: number;
	    originalSize?: number;
	    modifiedAt?: string;
	    originalModifiedAt?: string;
	    mimeType?: string;
	    width?: number;
	    height?: number;
	    format?: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalAttachmentReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.originalPath = source["originalPath"];
	        this.name = source["name"];
	        this.exists = source["exists"];
	        this.isDirectory = source["isDirectory"];
	        this.size = source["size"];
	        this.originalSize = source["originalSize"];
	        this.modifiedAt = source["modifiedAt"];
	        this.originalModifiedAt = source["originalModifiedAt"];
	        this.mimeType = source["mimeType"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.format = source["format"];
	    }
	}
	export class AttachmentReference {
	    kind: string;
	    label?: string;
	    mode?: string;
	    availableModes?: string[];
	    file?: LocalAttachmentReference;
	    image?: LocalAttachmentReference;
	    url?: URLAttachmentReference;
	    generic?: GenericAttachmentReference;
	    directory?: DirectoryAttachmentReference;
	
	    static createFrom(source: any = {}) {
	        return new AttachmentReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.mode = source["mode"];
	        this.availableModes = source["availableModes"];
	        this.file = this.convertValues(source["file"], LocalAttachmentReference);
	        this.image = this.convertValues(source["image"], LocalAttachmentReference);
	        this.url = this.convertValues(source["url"], URLAttachmentReference);
	        this.generic = this.convertValues(source["generic"], GenericAttachmentReference);
	        this.directory = this.convertValues(source["directory"], DirectoryAttachmentReference);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AutomationHold {
	    enabled: boolean;
	    scope?: string;
	    targetOperationID?: string;
	    setByCommandID?: string;
	
	    static createFrom(source: any = {}) {
	        return new AutomationHold(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.scope = source["scope"];
	        this.targetOperationID = source["targetOperationID"];
	        this.setByCommandID = source["setByCommandID"];
	    }
	}
	export class AutomationControls {
	    automaticTools: AutomationHold;
	    automaticContinuation: AutomationHold;
	    paused: boolean;
	    cancellationRequested: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AutomationControls(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.automaticTools = this.convertValues(source["automaticTools"], AutomationHold);
	        this.automaticContinuation = this.convertValues(source["automaticContinuation"], AutomationHold);
	        this.paused = source["paused"];
	        this.cancellationRequested = source["cancellationRequested"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CommandMeta {
	    id: string;
	    expectedSessionRevision?: number;
	    expectedRunRevision?: number;
	
	    static createFrom(source: any = {}) {
	        return new CommandMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.expectedSessionRevision = source["expectedSessionRevision"];
	        this.expectedRunRevision = source["expectedRunRevision"];
	    }
	}
	export class CancelRunCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new CancelRunCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.reason = source["reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DirectCapabilitySelection {
	    providerIdentity: string;
	    providerLocalID: string;
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DirectCapabilitySelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerIdentity = source["providerIdentity"];
	        this.providerLocalID = source["providerLocalID"];
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class CapabilitySelection {
	    artifact?: ArtifactSelection;
	    direct?: DirectCapabilitySelection;
	
	    static createFrom(source: any = {}) {
	        return new CapabilitySelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], ArtifactSelection);
	        this.direct = this.convertValues(source["direct"], DirectCapabilitySelection);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ResolutionDiagnostic {
	    severity: string;
	    code: string;
	    message: string;
	    key?: string;
	
	    static createFrom(source: any = {}) {
	        return new ResolutionDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.code = source["code"];
	        this.message = source["message"];
	        this.key = source["key"];
	    }
	}
	export class ResolvedArtifact {
	    rootID: string;
	    artifactID: string;
	    revision?: number;
	    digest?: string;
	
	    static createFrom(source: any = {}) {
	        return new ResolvedArtifact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.artifactID = source["artifactID"];
	        this.revision = source["revision"];
	        this.digest = source["digest"];
	    }
	}
	export class ResolutionItem {
	    key: string;
	    kind: string;
	    owner: string;
	    state: string;
	    selected?: ArtifactSelection;
	    used?: ResolvedArtifact;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new ResolutionItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.kind = source["kind"];
	        this.owner = source["owner"];
	        this.state = source["state"];
	        this.selected = this.convertValues(source["selected"], ArtifactSelection);
	        this.used = this.convertValues(source["used"], ResolvedArtifact);
	        this.reason = source["reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OperationResolutionSummary {
	    version: number;
	    agent?: ResolutionItem;
	    workspace?: ResolutionItem;
	    model?: ResolutionItem;
	    instructions?: ResolutionItem[];
	    tools?: ResolutionItem[];
	    skills?: ResolutionItem[];
	    mcp?: ResolutionItem[];
	    attachments?: ResolutionItem[];
	    diagnostics?: ResolutionDiagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new OperationResolutionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.agent = this.convertValues(source["agent"], ResolutionItem);
	        this.workspace = this.convertValues(source["workspace"], ResolutionItem);
	        this.model = this.convertValues(source["model"], ResolutionItem);
	        this.instructions = this.convertValues(source["instructions"], ResolutionItem);
	        this.tools = this.convertValues(source["tools"], ResolutionItem);
	        this.skills = this.convertValues(source["skills"], ResolutionItem);
	        this.mcp = this.convertValues(source["mcp"], ResolutionItem);
	        this.attachments = this.convertValues(source["attachments"], ResolutionItem);
	        this.diagnostics = this.convertValues(source["diagnostics"], ResolutionDiagnostic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OperationPreparationFailureDetails {
	    code: string;
	    message: string;
	    resolution: OperationResolutionSummary;
	
	    static createFrom(source: any = {}) {
	        return new OperationPreparationFailureDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.resolution = this.convertValues(source["resolution"], OperationResolutionSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CommandReceipt {
	    commandID: string;
	    accepted: boolean;
	    duplicate?: boolean;
	    sessionID?: string;
	    runID?: string;
	    sessionRevision?: number;
	    runRevision?: number;
	    operationIDs?: string[];
	    toolCallIDs?: string[];
	    waitIDs?: string[];
	    alreadyActiveToolCallIDs?: string[];
	    starterText?: string;
	    message?: string;
	    preparationFailure?: OperationPreparationFailureDetails;
	
	    static createFrom(source: any = {}) {
	        return new CommandReceipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.commandID = source["commandID"];
	        this.accepted = source["accepted"];
	        this.duplicate = source["duplicate"];
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.sessionRevision = source["sessionRevision"];
	        this.runRevision = source["runRevision"];
	        this.operationIDs = source["operationIDs"];
	        this.toolCallIDs = source["toolCallIDs"];
	        this.waitIDs = source["waitIDs"];
	        this.alreadyActiveToolCallIDs = source["alreadyActiveToolCallIDs"];
	        this.starterText = source["starterText"];
	        this.message = source["message"];
	        this.preparationFailure = this.convertValues(source["preparationFailure"], OperationPreparationFailureDetails);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompleteCallAsErrorCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    callID: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CompleteCallAsErrorCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.callID = source["callID"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HistorySelection {
	    all?: boolean;
	    includePreviousUserTurns?: number;
	
	    static createFrom(source: any = {}) {
	        return new HistorySelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.all = source["all"];
	        this.includePreviousUserTurns = source["includePreviousUserTurns"];
	    }
	}
	export class ToolSelectionIntent {
	    key?: string;
	    capability: CapabilitySelection;
	    autoExecute?: boolean;
	    userArguments?: number[];
	
	    static createFrom(source: any = {}) {
	        return new ToolSelectionIntent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.capability = this.convertValues(source["capability"], CapabilitySelection);
	        this.autoExecute = source["autoExecute"];
	        this.userArguments = source["userArguments"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolOverride {
	    key: string;
	    mode: string;
	    value?: ToolSelectionIntent;
	
	    static createFrom(source: any = {}) {
	        return new ToolOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.mode = source["mode"];
	        this.value = this.convertValues(source["value"], ToolSelectionIntent);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InstructionSelection {
	    key?: string;
	    artifact?: ArtifactSelection;
	    text?: string;
	    label?: string;
	
	    static createFrom(source: any = {}) {
	        return new InstructionSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.artifact = this.convertValues(source["artifact"], ArtifactSelection);
	        this.text = source["text"];
	        this.label = source["label"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InstructionOverride {
	    key: string;
	    mode: string;
	    value?: InstructionSelection;
	
	    static createFrom(source: any = {}) {
	        return new InstructionOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.mode = source["mode"];
	        this.value = this.convertValues(source["value"], InstructionSelection);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModelOutputJSONSchemaOverride {
	    name?: string;
	    description?: string;
	    schema?: number[];
	    strict?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModelOutputJSONSchemaOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.schema = source["schema"];
	        this.strict = source["strict"];
	    }
	}
	export class ModelOutputFormatOverride {
	    kind?: string;
	    jsonSchema?: ModelOutputJSONSchemaOverride;
	
	    static createFrom(source: any = {}) {
	        return new ModelOutputFormatOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.jsonSchema = this.convertValues(source["jsonSchema"], ModelOutputJSONSchemaOverride);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModelOutputOverride {
	    verbosity?: string;
	    format?: ModelOutputFormatOverride;
	
	    static createFrom(source: any = {}) {
	        return new ModelOutputOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.verbosity = source["verbosity"];
	        this.format = this.convertValues(source["format"], ModelOutputFormatOverride);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModelCacheControlOverride {
	    kind?: string;
	    ttl?: string;
	    key?: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelCacheControlOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.ttl = source["ttl"];
	        this.key = source["key"];
	    }
	}
	export class ModelReasoningOverride {
	    type?: string;
	    level?: string;
	    tokens?: number;
	    summaryStyle?: string;
	    context?: string;
	    mode?: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelReasoningOverride(source);
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
	export class ModelDefaultsOverride {
	    stream?: boolean;
	    maxPromptTokens?: number;
	    maxOutputTokens?: number;
	    temperature?: number;
	    systemPrompt?: string;
	    timeoutMS?: number;
	    reasoning?: ModelReasoningOverride;
	    cacheControl?: ModelCacheControlOverride;
	    output?: ModelOutputOverride;
	    stopSequences?: string[];
	    adapterParameters?: Record<string, Array<number>>;
	    clear?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ModelDefaultsOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stream = source["stream"];
	        this.maxPromptTokens = source["maxPromptTokens"];
	        this.maxOutputTokens = source["maxOutputTokens"];
	        this.temperature = source["temperature"];
	        this.systemPrompt = source["systemPrompt"];
	        this.timeoutMS = source["timeoutMS"];
	        this.reasoning = this.convertValues(source["reasoning"], ModelReasoningOverride);
	        this.cacheControl = this.convertValues(source["cacheControl"], ModelCacheControlOverride);
	        this.output = this.convertValues(source["output"], ModelOutputOverride);
	        this.stopSequences = source["stopSequences"];
	        this.adapterParameters = source["adapterParameters"];
	        this.clear = source["clear"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    promptName: string;
	    expectedDigest?: string;
	    arguments?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPPromptSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptName = source["promptName"];
	        this.expectedDigest = source["expectedDigest"];
	        this.arguments = source["arguments"];
	    }
	}
	export class MCPResourceTemplateSelection {
	    uriTemplate: string;
	    expectedDigest?: string;
	    arguments?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourceTemplateSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uriTemplate = source["uriTemplate"];
	        this.expectedDigest = source["expectedDigest"];
	        this.arguments = source["arguments"];
	    }
	}
	export class MCPResourceSelection {
	    uri: string;
	    expectedDigest?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPResourceSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uri = source["uri"];
	        this.expectedDigest = source["expectedDigest"];
	    }
	}
	export class MCPToolSelection {
	    toolName: string;
	    expectedDigest?: string;
	    approvalRule?: string;
	    executionMode?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPToolSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolName = source["toolName"];
	        this.expectedDigest = source["expectedDigest"];
	        this.approvalRule = source["approvalRule"];
	        this.executionMode = source["executionMode"];
	    }
	}
	export class MCPServerSelection {
	    key?: string;
	    server: ArtifactSelection;
	    includeServerInstructions?: boolean;
	    toolExposure: string;
	    selectedTools?: MCPToolSelection[];
	    resources?: MCPResourceSelection[];
	    resourceTemplates?: MCPResourceTemplateSelection[];
	    prompts?: MCPPromptSelection[];
	
	    static createFrom(source: any = {}) {
	        return new MCPServerSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.server = this.convertValues(source["server"], ArtifactSelection);
	        this.includeServerInstructions = source["includeServerInstructions"];
	        this.toolExposure = source["toolExposure"];
	        this.selectedTools = this.convertValues(source["selectedTools"], MCPToolSelection);
	        this.resources = this.convertValues(source["resources"], MCPResourceSelection);
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
	export class MCPServerOverride {
	    key: string;
	    mode: string;
	    value?: MCPServerSelection;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.mode = source["mode"];
	        this.value = this.convertValues(source["value"], MCPServerSelection);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceSelectionIntent {
	    workspace: ArtifactSelection;
	    context?: ArtifactOverride[];
	    skills?: ArtifactOverride[];
	    mcp?: MCPServerOverride[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSelectionIntent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], ArtifactSelection);
	        this.context = this.convertValues(source["context"], ArtifactOverride);
	        this.skills = this.convertValues(source["skills"], ArtifactOverride);
	        this.mcp = this.convertValues(source["mcp"], MCPServerOverride);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionConfiguration {
	    agent?: ArtifactSelection;
	    workspace?: WorkspaceSelectionIntent;
	    model?: ArtifactSelection;
	    modelOverrides: ModelDefaultsOverride;
	    instructions?: InstructionOverride[];
	    tools?: ToolOverride[];
	    skills?: ArtifactOverride[];
	    activeSkills?: ActiveSkillOverride[];
	    mcp?: MCPServerOverride[];
	    history: HistorySelection;
	    metadata?: number[];
	
	    static createFrom(source: any = {}) {
	        return new SessionConfiguration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = this.convertValues(source["agent"], ArtifactSelection);
	        this.workspace = this.convertValues(source["workspace"], WorkspaceSelectionIntent);
	        this.model = this.convertValues(source["model"], ArtifactSelection);
	        this.modelOverrides = this.convertValues(source["modelOverrides"], ModelDefaultsOverride);
	        this.instructions = this.convertValues(source["instructions"], InstructionOverride);
	        this.tools = this.convertValues(source["tools"], ToolOverride);
	        this.skills = this.convertValues(source["skills"], ArtifactOverride);
	        this.activeSkills = this.convertValues(source["activeSkills"], ActiveSkillOverride);
	        this.mcp = this.convertValues(source["mcp"], MCPServerOverride);
	        this.history = this.convertValues(source["history"], HistorySelection);
	        this.metadata = source["metadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContinuationInput {
	    revision: number;
	    consumedRevision?: number;
	    text?: string;
	    attachments?: AttachmentReference[];
	    mcpAppContext?: AppContextUpdate[];
	    configuration?: SessionConfiguration;
	    metadata?: number[];
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ContinuationInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.consumedRevision = source["consumedRevision"];
	        this.text = source["text"];
	        this.attachments = this.convertValues(source["attachments"], AttachmentReference);
	        this.mcpAppContext = this.convertValues(source["mcpAppContext"], AppContextUpdate);
	        this.configuration = this.convertValues(source["configuration"], SessionConfiguration);
	        this.metadata = source["metadata"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContinueRunCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	
	    static createFrom(source: any = {}) {
	        return new ContinueRunCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateSessionCommand {
	    meta: CommandMeta;
	    sessionID?: string;
	    title?: string;
	    configuration: SessionConfiguration;
	
	    static createFrom(source: any = {}) {
	        return new CreateSessionCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.title = source["title"];
	        this.configuration = this.convertValues(source["configuration"], SessionConfiguration);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeleteSessionCommand {
	    meta: CommandMeta;
	    sessionID: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteSessionCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class DriverState {
	    kind: string;
	    version: number;
	    data?: number[];
	
	    static createFrom(source: any = {}) {
	        return new DriverState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.version = source["version"];
	        this.data = source["data"];
	    }
	}
	export class RunLimits {
	    maxModelOperations?: number;
	    maxToolAttempts?: number;
	
	    static createFrom(source: any = {}) {
	        return new RunLimits(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxModelOperations = source["maxModelOperations"];
	        this.maxToolAttempts = source["maxToolAttempts"];
	    }
	}
	export class UserInput {
	    text?: string;
	    attachments?: AttachmentReference[];
	    mcpAppContext?: AppContextUpdate[];
	    metadata?: number[];
	
	    static createFrom(source: any = {}) {
	        return new UserInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.attachments = this.convertValues(source["attachments"], AttachmentReference);
	        this.mcpAppContext = this.convertValues(source["mcpAppContext"], AppContextUpdate);
	        this.metadata = source["metadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EditAndResendCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    replaceMessageID: string;
	    runID?: string;
	    input: UserInput;
	    carryForwardToolResultIDs?: string[];
	    configuration?: SessionConfiguration;
	    limits?: RunLimits;
	
	    static createFrom(source: any = {}) {
	        return new EditAndResendCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.replaceMessageID = source["replaceMessageID"];
	        this.runID = source["runID"];
	        this.input = this.convertValues(source["input"], UserInput);
	        this.carryForwardToolResultIDs = source["carryForwardToolResultIDs"];
	        this.configuration = this.convertValues(source["configuration"], SessionConfiguration);
	        this.limits = this.convertValues(source["limits"], RunLimits);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExecuteAppToolCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    appInstanceID: string;
	    viewID: string;
	    toolName: string;
	    arguments?: number[];
	
	    static createFrom(source: any = {}) {
	        return new ExecuteAppToolCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.appInstanceID = source["appInstanceID"];
	        this.viewID = source["viewID"];
	        this.toolName = source["toolName"];
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
	export class ExecuteCallsCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    callIDs?: string[];
	    continue: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteCallsCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.callIDs = source["callIDs"];
	        this.continue = source["continue"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	export class InstructionSkillPreviewRequest {
	    skill: ArtifactSelection;
	    arguments?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new InstructionSkillPreviewRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skill = this.convertValues(source["skill"], ArtifactSelection);
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
	export class InstructionSkillPreviewResult {
	    skill: ArtifactReference;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new InstructionSkillPreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skill = this.convertValues(source["skill"], ArtifactReference);
	        this.text = source["text"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	
	export class OperationPreparation {
	    state: string;
	    fingerprint?: string;
	    // Go type: time
	    preparedAt: any;
	    resolution: OperationResolutionSummary;
	    effectiveSkills?: ArtifactReference[];
	    activeSkills?: ArtifactReference[];
	
	    static createFrom(source: any = {}) {
	        return new OperationPreparation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.fingerprint = source["fingerprint"];
	        this.preparedAt = this.convertValues(source["preparedAt"], null);
	        this.resolution = this.convertValues(source["resolution"], OperationResolutionSummary);
	        this.effectiveSkills = this.convertValues(source["effectiveSkills"], ArtifactReference);
	        this.activeSkills = this.convertValues(source["activeSkills"], ArtifactReference);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OperationError {
	    code: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new OperationError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class ToolResult {
	    id: string;
	    attemptID?: string;
	    callID: string;
	    providerCallID?: string;
	    name: string;
	    choiceID?: string;
	    type?: string;
	    kind: string;
	    text?: string;
	    contents?: spec.ToolOutputItemUnion[];
	    structured?: number[];
	    app?: number[];
	    error?: string;
	    appInstanceID?: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ToolResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.attemptID = source["attemptID"];
	        this.callID = source["callID"];
	        this.providerCallID = source["providerCallID"];
	        this.name = source["name"];
	        this.choiceID = source["choiceID"];
	        this.type = source["type"];
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.contents = this.convertValues(source["contents"], spec.ToolOutputItemUnion);
	        this.structured = source["structured"];
	        this.app = source["app"];
	        this.error = source["error"];
	        this.appInstanceID = source["appInstanceID"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TranscriptMessage {
	    id: string;
	    // Go type: time
	    createdAt: any;
	    role: string;
	    status?: string;
	    text?: string;
	    attachments?: AttachmentReference[];
	    attachmentsHydrated?: boolean;
	    protocolInputs?: number[][];
	    protocolOutputs?: number[][];
	    toolResults?: ToolResult[];
	    mcpAppContext?: AppContextUpdate[];
	    injectedContext?: string[];
	    operationID?: string;
	    error?: OperationError;
	    metadata?: number[];
	    runtimeMetadata?: number[];
	
	    static createFrom(source: any = {}) {
	        return new TranscriptMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.role = source["role"];
	        this.status = source["status"];
	        this.text = source["text"];
	        this.attachments = this.convertValues(source["attachments"], AttachmentReference);
	        this.attachmentsHydrated = source["attachmentsHydrated"];
	        this.protocolInputs = source["protocolInputs"];
	        this.protocolOutputs = source["protocolOutputs"];
	        this.toolResults = this.convertValues(source["toolResults"], ToolResult);
	        this.mcpAppContext = this.convertValues(source["mcpAppContext"], AppContextUpdate);
	        this.injectedContext = source["injectedContext"];
	        this.operationID = source["operationID"];
	        this.error = this.convertValues(source["error"], OperationError);
	        this.metadata = source["metadata"];
	        this.runtimeMetadata = source["runtimeMetadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModelOperationInput {
	    configuration: SessionConfiguration;
	    history: TranscriptMessage[];
	    current: TranscriptMessage;
	    continuationRevision?: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelOperationInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configuration = this.convertValues(source["configuration"], SessionConfiguration);
	        this.history = this.convertValues(source["history"], TranscriptMessage);
	        this.current = this.convertValues(source["current"], TranscriptMessage);
	        this.continuationRevision = source["continuationRevision"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModelOperation {
	    id: string;
	    state: string;
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    finishedAt: any;
	    automatic?: boolean;
	    request: ModelOperationInput;
	    streamedText?: string;
	    streamedThinking?: string;
	    protocolOutputs?: number[][];
	    toolCallIDs?: string[];
	    error?: OperationError;
	    preparation: OperationPreparation;
	    usable: boolean;
	    usage?: spec.Usage;
	    metadata?: number[];
	    runtimeMetadata?: number[];
	
	    static createFrom(source: any = {}) {
	        return new ModelOperation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.state = source["state"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.finishedAt = this.convertValues(source["finishedAt"], null);
	        this.automatic = source["automatic"];
	        this.request = this.convertValues(source["request"], ModelOperationInput);
	        this.streamedText = source["streamedText"];
	        this.streamedThinking = source["streamedThinking"];
	        this.protocolOutputs = source["protocolOutputs"];
	        this.toolCallIDs = source["toolCallIDs"];
	        this.error = this.convertValues(source["error"], OperationError);
	        this.preparation = this.convertValues(source["preparation"], OperationPreparation);
	        this.usable = source["usable"];
	        this.usage = this.convertValues(source["usage"], spec.Usage);
	        this.metadata = source["metadata"];
	        this.runtimeMetadata = source["runtimeMetadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	export class OpenAppViewCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    appInstanceID: string;
	    viewID: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenAppViewCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.appInstanceID = source["appInstanceID"];
	        this.viewID = source["viewID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	export class PauseRunCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	
	    static createFrom(source: any = {}) {
	        return new PauseRunCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReadAppResourceRequest {
	    sessionID: string;
	    runID: string;
	    appInstanceID: string;
	    viewID: string;
	    resourceURI?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReadAppResourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.appInstanceID = source["appInstanceID"];
	        this.viewID = source["viewID"];
	        this.resourceURI = source["resourceURI"];
	    }
	}
	export class RenameSessionCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new RenameSessionCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.title = source["title"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class ResolveWaitCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    waitID: string;
	    resolution: string;
	    input?: number[];
	
	    static createFrom(source: any = {}) {
	        return new ResolveWaitCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.waitID = source["waitID"];
	        this.resolution = source["resolution"];
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
	
	export class ResumeRunCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	
	    static createFrom(source: any = {}) {
	        return new ResumeRunCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RetryCallCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    callID: string;
	
	    static createFrom(source: any = {}) {
	        return new RetryCallCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.callID = source["callID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunOutcome {
	    kind: string;
	    code?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new RunOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class WaitingRequest {
	    id: string;
	    kind: string;
	    status: string;
	    operationID?: string;
	    toolCallID?: string;
	    summary?: number[];
	    inputSchema?: number[];
	    allowedResolutions?: string[];
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: time
	    expiresAt?: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new WaitingRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.operationID = source["operationID"];
	        this.toolCallID = source["toolCallID"];
	        this.summary = source["summary"];
	        this.inputSchema = source["inputSchema"];
	        this.allowedResolutions = source["allowedResolutions"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolAttempt {
	    id: string;
	    callID: string;
	    state: string;
	    automatic: boolean;
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    finishedAt: any;
	    error?: string;
	    uncertain?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ToolAttempt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.callID = source["callID"];
	        this.state = source["state"];
	        this.automatic = source["automatic"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.finishedAt = this.convertValues(source["finishedAt"], null);
	        this.error = source["error"];
	        this.uncertain = source["uncertain"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolBinding {
	    kind: string;
	    data?: number[];
	
	    static createFrom(source: any = {}) {
	        return new ToolBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.data = source["data"];
	    }
	}
	export class ToolCall {
	    id: string;
	    originOperationID: string;
	    providerCallID?: string;
	    name: string;
	    choiceID?: string;
	    type?: string;
	    arguments?: number[];
	    binding: ToolBinding;
	    required: boolean;
	    providerManaged?: boolean;
	    automatic?: boolean;
	    source?: string;
	    appInstanceID?: string;
	    state: string;
	    attempts?: ToolAttempt[];
	    results?: ToolResult[];
	    selectedResultID?: string;
	    waitID?: string;
	    lastFailure?: string;
	    consumedByOperationID?: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.originOperationID = source["originOperationID"];
	        this.providerCallID = source["providerCallID"];
	        this.name = source["name"];
	        this.choiceID = source["choiceID"];
	        this.type = source["type"];
	        this.arguments = source["arguments"];
	        this.binding = this.convertValues(source["binding"], ToolBinding);
	        this.required = source["required"];
	        this.providerManaged = source["providerManaged"];
	        this.automatic = source["automatic"];
	        this.source = source["source"];
	        this.appInstanceID = source["appInstanceID"];
	        this.state = source["state"];
	        this.attempts = this.convertValues(source["attempts"], ToolAttempt);
	        this.results = this.convertValues(source["results"], ToolResult);
	        this.selectedResultID = source["selectedResultID"];
	        this.waitID = source["waitID"];
	        this.lastFailure = source["lastFailure"];
	        this.consumedByOperationID = source["consumedByOperationID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunUsage {
	    modelOperations: number;
	    toolAttempts: number;
	
	    static createFrom(source: any = {}) {
	        return new RunUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelOperations = source["modelOperations"];
	        this.toolAttempts = source["toolAttempts"];
	    }
	}
	export class SkillSessionState {
	    id?: string;
	    generation: number;
	    establishedGeneration?: number;
	    enabled?: ArtifactReference[];
	    active?: ArtifactReference[];
	
	    static createFrom(source: any = {}) {
	        return new SkillSessionState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.generation = source["generation"];
	        this.establishedGeneration = source["establishedGeneration"];
	        this.enabled = this.convertValues(source["enabled"], ArtifactReference);
	        this.active = this.convertValues(source["active"], ArtifactReference);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Run {
	    id: string;
	    sessionID: string;
	    parentRunID?: string;
	    rootRunID?: string;
	    revision: number;
	    state: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    driver: DriverState;
	    configuration: SessionConfiguration;
	    skillSession: SkillSessionState;
	    controls: AutomationControls;
	    limits: RunLimits;
	    usage: RunUsage;
	    continuation: ContinuationInput;
	    modelOperations?: ModelOperation[];
	    toolCalls?: ToolCall[];
	    waitingRequests?: WaitingRequest[];
	    apps?: AppInstance[];
	    requestedCallIDs?: string[];
	    continuationRequested?: boolean;
	    activeModelOperationID?: string;
	    outcome?: RunOutcome;
	    eventSequence: number;
	
	    static createFrom(source: any = {}) {
	        return new Run(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionID = source["sessionID"];
	        this.parentRunID = source["parentRunID"];
	        this.rootRunID = source["rootRunID"];
	        this.revision = source["revision"];
	        this.state = source["state"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.driver = this.convertValues(source["driver"], DriverState);
	        this.configuration = this.convertValues(source["configuration"], SessionConfiguration);
	        this.skillSession = this.convertValues(source["skillSession"], SkillSessionState);
	        this.controls = this.convertValues(source["controls"], AutomationControls);
	        this.limits = this.convertValues(source["limits"], RunLimits);
	        this.usage = this.convertValues(source["usage"], RunUsage);
	        this.continuation = this.convertValues(source["continuation"], ContinuationInput);
	        this.modelOperations = this.convertValues(source["modelOperations"], ModelOperation);
	        this.toolCalls = this.convertValues(source["toolCalls"], ToolCall);
	        this.waitingRequests = this.convertValues(source["waitingRequests"], WaitingRequest);
	        this.apps = this.convertValues(source["apps"], AppInstance);
	        this.requestedCallIDs = source["requestedCallIDs"];
	        this.continuationRequested = source["continuationRequested"];
	        this.activeModelOperationID = source["activeModelOperationID"];
	        this.outcome = this.convertValues(source["outcome"], RunOutcome);
	        this.eventSequence = source["eventSequence"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class RunSnapshot {
	    sessionID: string;
	    sessionRevision: number;
	    run: Run;
	
	    static createFrom(source: any = {}) {
	        return new RunSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionID = source["sessionID"];
	        this.sessionRevision = source["sessionRevision"];
	        this.run = this.convertValues(source["run"], Run);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Session {
	    id: string;
	    revision: number;
	    title?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    configuration: SessionConfiguration;
	    transcript?: TranscriptMessage[];
	    runIDs?: string[];
	    eventSequence: number;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.revision = source["revision"];
	        this.title = source["title"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.configuration = this.convertValues(source["configuration"], SessionConfiguration);
	        this.transcript = this.convertValues(source["transcript"], TranscriptMessage);
	        this.runIDs = source["runIDs"];
	        this.eventSequence = source["eventSequence"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SessionListItem {
	    id: string;
	    revision: number;
	    title?: string;
	    // Go type: time
	    updatedAt: any;
	    activeRuns?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SessionListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.revision = source["revision"];
	        this.title = source["title"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.activeRuns = source["activeRuns"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionSnapshot {
	    session: Session;
	    runs: Run[];
	    sequence: number;
	
	    static createFrom(source: any = {}) {
	        return new SessionSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session = this.convertValues(source["session"], Session);
	        this.runs = this.convertValues(source["runs"], Run);
	        this.sequence = source["sequence"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SetAutomationHoldCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    tools: boolean;
	    continuation: boolean;
	    enabled: boolean;
	    scope?: string;
	
	    static createFrom(source: any = {}) {
	        return new SetAutomationHoldCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.tools = source["tools"];
	        this.continuation = source["continuation"];
	        this.enabled = source["enabled"];
	        this.scope = source["scope"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SkipCallCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    callID: string;
	
	    static createFrom(source: any = {}) {
	        return new SkipCallCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.callID = source["callID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StageAppContextCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    appInstanceID: string;
	    viewID: string;
	    update: AppContextUpdate;
	
	    static createFrom(source: any = {}) {
	        return new StageAppContextCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.appInstanceID = source["appInstanceID"];
	        this.viewID = source["viewID"];
	        this.update = this.convertValues(source["update"], AppContextUpdate);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StartRunCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID?: string;
	    input: UserInput;
	    configuration?: SessionConfiguration;
	    limits?: RunLimits;
	
	    static createFrom(source: any = {}) {
	        return new StartRunCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.input = this.convertValues(source["input"], UserInput);
	        this.configuration = this.convertValues(source["configuration"], SessionConfiguration);
	        this.limits = this.convertValues(source["limits"], RunLimits);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	export class UpdateContinuationInputCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    runID: string;
	    expectedInputRevision?: number;
	    input: ContinuationInput;
	
	    static createFrom(source: any = {}) {
	        return new UpdateContinuationInputCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.runID = source["runID"];
	        this.expectedInputRevision = source["expectedInputRevision"];
	        this.input = this.convertValues(source["input"], ContinuationInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateSessionConfigurationCommand {
	    meta: CommandMeta;
	    sessionID: string;
	    configuration: SessionConfiguration;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSessionConfigurationCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], CommandMeta);
	        this.sessionID = source["sessionID"];
	        this.configuration = this.convertValues(source["configuration"], SessionConfiguration);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class UserMessageSkillTemplatePreparationRequest {
	    sessionID: string;
	    skill: ArtifactSelection;
	    arguments: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new UserMessageSkillTemplatePreparationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionID = source["sessionID"];
	        this.skill = this.convertValues(source["skill"], ArtifactSelection);
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
	export class UserMessageSkillTemplateResult {
	    skill: ArtifactReference;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new UserMessageSkillTemplateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skill = this.convertValues(source["skill"], ArtifactReference);
	        this.text = source["text"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace sessionstore {
	
	export class ConversationProjection {
	    schemaVersion: string;
	    sessionID: string;
	    sessionRevision: number;
	    title?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    messages: session.TranscriptMessage[];
	    associatedRunIDs?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationProjection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.sessionID = source["sessionID"];
	        this.sessionRevision = source["sessionRevision"];
	        this.title = source["title"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.messages = this.convertValues(source["messages"], session.TranscriptMessage);
	        this.associatedRunIDs = source["associatedRunIDs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListSessionRunsRequest {
	    sessionID: string;
	    activeOnly?: boolean;
	    limit?: number;
	
	    static createFrom(source: any = {}) {
	        return new ListSessionRunsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionID = source["sessionID"];
	        this.activeOnly = source["activeOnly"];
	        this.limit = source["limit"];
	    }
	}
	export class ListSessionsRequest {
	    pageToken?: string;
	    pageSize?: number;
	
	    static createFrom(source: any = {}) {
	        return new ListSessionsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pageToken = source["pageToken"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class RunListItem {
	    id: string;
	    revision: number;
	    state: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    pendingWaitCount: number;
	    outcome?: session.RunOutcome;
	
	    static createFrom(source: any = {}) {
	        return new RunListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.revision = source["revision"];
	        this.state = source["state"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.pendingWaitCount = source["pendingWaitCount"];
	        this.outcome = this.convertValues(source["outcome"], session.RunOutcome);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchSessionsRequest {
	    query: string;
	    pageToken?: string;
	    pageSize?: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchSessionsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.pageToken = source["pageToken"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class SessionListItem {
	    id: string;
	    revision: number;
	    title?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    runCount: number;
	    activeRunIDs?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SessionListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.revision = source["revision"];
	        this.title = source["title"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.runCount = source["runCount"];
	        this.activeRunIDs = source["activeRunIDs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionPage {
	    items: SessionListItem[];
	    nextPageToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], SessionListItem);
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

}

export namespace setting {
	
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
	
	    static createFrom(source: any = {}) {
	        return new GetSettingsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appTheme = this.convertValues(source["appTheme"], AppTheme);
	        this.debug = this.convertValues(source["debug"], DebugSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace skill {
	
	export class ArtifactSkillResources {
	    hasResources: boolean;
	    totalCount: number;
	    locations?: string[];
	    moreLocations: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSkillResources(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasResources = source["hasResources"];
	        this.totalCount = source["totalCount"];
	        this.locations = source["locations"];
	        this.moreLocations = source["moreLocations"];
	    }
	}
	export class ArtifactSkillDetails {
	    artifact: model.Artifact;
	    document: document.SkillDocument;
	    location: string;
	    resources: ArtifactSkillResources;
	    runtimeWarnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSkillDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.document = this.convertValues(source["document"], document.SkillDocument);
	        this.location = source["location"];
	        this.resources = this.convertValues(source["resources"], ArtifactSkillResources);
	        this.runtimeWarnings = source["runtimeWarnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    plugin: model.ArtifactRef;
	    expectedPluginRevision: number;
	    skillName: string;
	    skillMD?: number[];
	    files?: model.ManagedPackageFile[];
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillCreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.expectedPluginRevision = source["expectedPluginRevision"];
	        this.skillName = source["skillName"];
	        this.skillMD = source["skillMD"];
	        this.files = this.convertValues(source["files"], model.ManagedPackageFile);
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
	    artifact: model.Artifact;
	    address: model.ArtifactAddress;
	    plugin: plugin.PluginView;
	    membershipCreated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillCreateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.address = this.convertValues(source["address"], model.ArtifactAddress);
	        this.plugin = this.convertValues(source["plugin"], plugin.PluginView);
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
	    plugin: model.ArtifactRef;
	    expectedPluginRevision: number;
	    artifact: model.ArtifactRef;
	    expectedArtifactRevision: number;
	    skillName: string;
	    skillMD?: number[];
	    files?: model.ManagedPackageFile[];
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillReplaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], model.ArtifactRef);
	        this.expectedPluginRevision = source["expectedPluginRevision"];
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
	        this.expectedArtifactRevision = source["expectedArtifactRevision"];
	        this.skillName = source["skillName"];
	        this.skillMD = source["skillMD"];
	        this.files = this.convertValues(source["files"], model.ManagedPackageFile);
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
	    artifact: model.Artifact;
	    address: model.ArtifactAddress;
	    plugin: plugin.PluginView;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillReplaceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.address = this.convertValues(source["address"], model.ArtifactAddress);
	        this.plugin = this.convertValues(source["plugin"], plugin.PluginView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class SkillListItem {
	    ref: model.ArtifactRef;
	    name: string;
	    displayName: string;
	    description?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    definitionDigest?: string;
	    builtIn: boolean;
	    managed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SkillListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], model.ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
	        this.definitionDigest = source["definitionDigest"];
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
	    source: model.Summary;
	    artifact: model.Artifact;
	
	    static createFrom(source: any = {}) {
	        return new SkillPathRegistrationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = this.convertValues(source["source"], model.Summary);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	
	export class ResolvedArtifactSkill {
	    Artifact: model.ArtifactRef;
	    Definition: provider.SkillDef;
	    Version: string;
	    Revision: number;
	    DefinitionDigest: string;
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResolvedArtifactSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Artifact = this.convertValues(source["Artifact"], model.ArtifactRef);
	        this.Definition = this.convertValues(source["Definition"], provider.SkillDef);
	        this.Version = source["Version"];
	        this.Revision = source["Revision"];
	        this.DefinitionDigest = source["DefinitionDigest"];
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

}

export namespace source {
	
	export class ManagedSkillDocument {
	    artifact: model.Artifact;
	    document: document.SkillDocument;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
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

}

export namespace spec {
	
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

}

export namespace text {
	
	export class Materialization {
	    artifact: model.ArtifactRef;
	    artifactRevision: number;
	    definitionDigest: string;
	    name: string;
	    insert: string;
	    mediaType?: string;
	    content: string;
	    locator: string;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Materialization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
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

export namespace tool {
	
	export class ToolImplementationView {
	    kind: string;
	    function?: string;
	    sdkType?: string;
	    sdkToolType?: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolImplementationView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.function = source["function"];
	        this.sdkType = source["sdkType"];
	        this.sdkToolType = source["sdkToolType"];
	    }
	}
	export class ToolView {
	    artifact: model.Artifact;
	    definitionDigest: string;
	    name: string;
	    version: string;
	    displayName: string;
	    description?: string;
	    tags?: string[];
	    autoExecute: boolean;
	    builtIn: boolean;
	    inputSchema: number[];
	    userArgSchema?: number[];
	    outputSchema?: number[];
	    implementation: ToolImplementationView;
	
	    static createFrom(source: any = {}) {
	        return new ToolView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.Artifact);
	        this.definitionDigest = source["definitionDigest"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.autoExecute = source["autoExecute"];
	        this.builtIn = source["builtIn"];
	        this.inputSchema = source["inputSchema"];
	        this.userArgSchema = source["userArgSchema"];
	        this.outputSchema = source["outputSchema"];
	        this.implementation = this.convertValues(source["implementation"], ToolImplementationView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResolvedToolView {
	    tool: ToolView;
	    plugin: plugin.PluginView;
	
	    static createFrom(source: any = {}) {
	        return new ResolvedToolView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = this.convertValues(source["tool"], ToolView);
	        this.plugin = this.convertValues(source["plugin"], plugin.PluginView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ToolListItem {
	    ref: model.ArtifactRef;
	    name: string;
	    displayName: string;
	    description?: string;
	    state: string;
	    enabled: boolean;
	    revision: number;
	    definitionDigest?: string;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ToolListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], model.ArtifactRef);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.revision = source["revision"];
	        this.definitionDigest = source["definitionDigest"];
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

}

export namespace workspace {
	
	export class WorkspaceArtifactView {
	    artifact: model.ArtifactRef;
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
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
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
	export class WorkspaceDirectoryListItem {
	    ref: WorkspaceDirectoryRef;
	    rootID: string;
	    rootDisplayName: string;
	    enabled: boolean;
	    directorySourceID: string;
	    directorySourceRevision: number;
	    policyID: string;
	    policyVersion: string;
	    policyDigest: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDirectoryListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], WorkspaceDirectoryRef);
	        this.rootID = source["rootID"];
	        this.rootDisplayName = source["rootDisplayName"];
	        this.enabled = source["enabled"];
	        this.directorySourceID = source["directorySourceID"];
	        this.directorySourceRevision = source["directorySourceRevision"];
	        this.policyID = source["policyID"];
	        this.policyVersion = source["policyVersion"];
	        this.policyDigest = source["policyDigest"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	    root: model.Root;
	    directorySource: model.Summary;
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
	        this.root = this.convertValues(source["root"], model.Root);
	        this.directorySource = this.convertValues(source["directorySource"], model.Summary);
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
	    artifact: model.ArtifactRef;
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
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
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
	    workspace: model.ArtifactRef;
	    servers: WorkspaceMCPServer[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceMCPServerLoadPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], model.ArtifactRef);
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
	    items: WorkspaceDirectoryListItem[];
	    nextCursor?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspacePage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], WorkspaceDirectoryListItem);
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
	    artifact: model.ArtifactRef;
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
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
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
	    artifact: model.ArtifactRef;
	    status: string;
	    code?: string;
	    originalBytes: number;
	    includedBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspacePromptDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
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
	    workspace: model.ArtifactRef;
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
	        this.workspace = this.convertValues(source["workspace"], model.ArtifactRef);
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
	    artifact: model.ArtifactRef;
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
	        this.artifact = this.convertValues(source["artifact"], model.ArtifactRef);
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
	    workspace: model.ArtifactRef;
	    skills: WorkspaceSkill[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkillLoadPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], model.ArtifactRef);
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
	    capabilities: composition.CapabilityPlan;
	    prompt: WorkspacePromptPlan;
	    skills: WorkspaceSkillLoadPlan;
	    mcpServers: WorkspaceMCPServerLoadPlan;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceRuntimePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], domain.WorkspaceView);
	        this.capabilities = this.convertValues(source["capabilities"], composition.CapabilityPlan);
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
	    promptArtifacts?: model.ArtifactRef[];
	    skillArtifacts?: model.ArtifactRef[];
	    mcpArtifacts?: model.ArtifactRef[];
	    requireComplete?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceRuntimeSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptArtifacts = this.convertValues(source["promptArtifacts"], model.ArtifactRef);
	        this.skillArtifacts = this.convertValues(source["skillArtifacts"], model.ArtifactRef);
	        this.mcpArtifacts = this.convertValues(source["mcpArtifacts"], model.ArtifactRef);
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

