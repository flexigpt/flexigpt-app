export namespace artifact {
	
	export class SourceBinding {
	    sourceID: string;
	    locator: string;
	    subresourceLocator?: string;
	    expectedKind: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	        this.expectedKind = source["expectedKind"];
	    }
	}
	export class Artifact {
	    id: string;
	    rootID: string;
	    collectionID: string;
	    binding: SourceBinding;
	    kind: string;
	    name: string;
	    enabled: boolean;
	    adoption: string;
	    resolvedDefinition?: string;
	    state: string;
	    diagnostics?: diagnostic.Diagnostic[];
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
	        this.collectionID = source["collectionID"];
	        this.binding = this.convertValues(source["binding"], SourceBinding);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.adoption = source["adoption"];
	        this.resolvedDefinition = source["resolvedDefinition"];
	        this.state = source["state"];
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
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
	    collectionID: string;
	    artifactID: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactAddress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.collectionID = source["collectionID"];
	        this.artifactID = source["artifactID"];
	        this.kind = source["kind"];
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

export namespace artifactstore {
	
	export class ArtifactSourceDraft {
	    id: string;
	    storageKey: string;
	    kind: string;
	    displayName: string;
	    enabled: boolean;
	    config: number[];
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSourceDraft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.storageKey = source["storageKey"];
	        this.kind = source["kind"];
	        this.displayName = source["displayName"];
	        this.enabled = source["enabled"];
	        this.config = source["config"];
	    }
	}
	export class CreateArtifactRootRequest {
	    Body?: root.RootDraft;
	
	    static createFrom(source: any = {}) {
	        return new CreateArtifactRootRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], root.RootDraft);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateArtifactRootResponse {
	    Body?: root.Root;
	
	    static createFrom(source: any = {}) {
	        return new CreateArtifactRootResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], root.Root);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateArtifactSourceRequest {
	    RootID: string;
	    Body?: ArtifactSourceDraft;
	
	    static createFrom(source: any = {}) {
	        return new CreateArtifactSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.Body = this.convertValues(source["Body"], ArtifactSourceDraft);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateArtifactSourceResponse {
	    Body?: source.Summary;
	
	    static createFrom(source: any = {}) {
	        return new CreateArtifactSourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetArtifactRootRequest {
	    RootID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetArtifactRootRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	    }
	}
	export class GetArtifactRootResponse {
	    Body?: root.Root;
	
	    static createFrom(source: any = {}) {
	        return new GetArtifactRootResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], root.Root);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetArtifactSourceRequest {
	    RootID: string;
	    SourceID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetArtifactSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.SourceID = source["SourceID"];
	    }
	}
	export class GetArtifactSourceResponse {
	    Body?: source.Summary;
	
	    static createFrom(source: any = {}) {
	        return new GetArtifactSourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetManagedSourceStateRequest {
	    rootID: string;
	    sourceID: string;
	
	    static createFrom(source: any = {}) {
	        return new GetManagedSourceStateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.sourceID = source["sourceID"];
	    }
	}
	export class GetManagedSourceStateResponseBody {
	    generation: string;
	    source: source.Summary;
	
	    static createFrom(source: any = {}) {
	        return new GetManagedSourceStateResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generation = source["generation"];
	        this.source = this.convertValues(source["source"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GetManagedSourceStateResponse {
	    Body?: GetManagedSourceStateResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new GetManagedSourceStateResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], GetManagedSourceStateResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ListArtifactRootsRequest {
	
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactRootsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class ListArtifactRootsResponseBody {
	    roots: root.Root[];
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactRootsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roots = this.convertValues(source["roots"], root.Root);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListArtifactRootsResponse {
	    Body?: ListArtifactRootsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactRootsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListArtifactRootsResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ListArtifactSourceKindsRequest {
	
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactSourceKindsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class ListArtifactSourceKindsResponseBody {
	    kinds: string[];
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactSourceKindsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kinds = source["kinds"];
	    }
	}
	export class ListArtifactSourceKindsResponse {
	    Body?: ListArtifactSourceKindsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactSourceKindsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListArtifactSourceKindsResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ListArtifactSourcesRequest {
	    RootID: string;
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactSourcesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	    }
	}
	export class ListArtifactSourcesResponseBody {
	    sources: source.Summary[];
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactSourcesResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sources = this.convertValues(source["sources"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListArtifactSourcesResponse {
	    Body?: ListArtifactSourcesResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListArtifactSourcesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListArtifactSourcesResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PublishManagedSourcePackageRequestBody {
	    expectedSourceRevision: number;
	    address: source.ManagedPackageAddress;
	    expectedGeneration?: string;
	    files: source.ManagedPackageFile[];
	
	    static createFrom(source: any = {}) {
	        return new PublishManagedSourcePackageRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedSourceRevision = source["expectedSourceRevision"];
	        this.address = this.convertValues(source["address"], source.ManagedPackageAddress);
	        this.expectedGeneration = source["expectedGeneration"];
	        this.files = this.convertValues(source["files"], source.ManagedPackageFile);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PublishManagedSourcePackageRequest {
	    rootID: string;
	    sourceID: string;
	    Body?: PublishManagedSourcePackageRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PublishManagedSourcePackageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.sourceID = source["sourceID"];
	        this.Body = this.convertValues(source["Body"], PublishManagedSourcePackageRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PublishManagedSourcePackageResponseBody {
	    generation: string;
	    source: source.Summary;
	
	    static createFrom(source: any = {}) {
	        return new PublishManagedSourcePackageResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generation = source["generation"];
	        this.source = this.convertValues(source["source"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PublishManagedSourcePackageResponse {
	    Body?: PublishManagedSourcePackageResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new PublishManagedSourcePackageResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], PublishManagedSourcePackageResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PurgeArtifactRootRequest {
	    RootID: string;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new PurgeArtifactRootRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.expectedRevision = source["expectedRevision"];
	    }
	}
	export class PurgeArtifactRootResponse {
	    rootID: string;
	
	    static createFrom(source: any = {}) {
	        return new PurgeArtifactRootResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	    }
	}
	export class PurgeArtifactSourceRequest {
	    RootID: string;
	    SourceID: string;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new PurgeArtifactSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.SourceID = source["SourceID"];
	        this.expectedRevision = source["expectedRevision"];
	    }
	}
	export class PurgeArtifactSourceResponse {
	    rootID: string;
	    sourceID: string;
	
	    static createFrom(source: any = {}) {
	        return new PurgeArtifactSourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.sourceID = source["sourceID"];
	    }
	}
	export class RemoveManagedSourcePackageRequest {
	    rootID: string;
	    sourceID: string;
	    expectedSourceRevision: number;
	    address: source.ManagedPackageAddress;
	    expectedGeneration: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoveManagedSourcePackageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.sourceID = source["sourceID"];
	        this.expectedSourceRevision = source["expectedSourceRevision"];
	        this.address = this.convertValues(source["address"], source.ManagedPackageAddress);
	        this.expectedGeneration = source["expectedGeneration"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RemoveManagedSourcePackageResponseBody {
	    generation: string;
	    source: source.Summary;
	
	    static createFrom(source: any = {}) {
	        return new RemoveManagedSourcePackageResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generation = source["generation"];
	        this.source = this.convertValues(source["source"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RemoveManagedSourcePackageResponse {
	    Body?: RemoveManagedSourcePackageResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new RemoveManagedSourcePackageResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], RemoveManagedSourcePackageResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RetireArtifactRootRequest {
	    RootID: string;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new RetireArtifactRootRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.expectedRevision = source["expectedRevision"];
	    }
	}
	export class RetireArtifactRootResponse {
	    Body?: root.Root;
	
	    static createFrom(source: any = {}) {
	        return new RetireArtifactRootResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], root.Root);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RetireArtifactSourceRequest {
	    RootID: string;
	    SourceID: string;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new RetireArtifactSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.SourceID = source["SourceID"];
	        this.expectedRevision = source["expectedRevision"];
	    }
	}
	export class RetireArtifactSourceResponse {
	    Body?: source.Summary;
	
	    static createFrom(source: any = {}) {
	        return new RetireArtifactSourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateArtifactRootRequest {
	    RootID: string;
	    Body?: root.RootUpdate;
	
	    static createFrom(source: any = {}) {
	        return new UpdateArtifactRootRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.Body = this.convertValues(source["Body"], root.RootUpdate);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateArtifactRootResponse {
	    Body?: root.Root;
	
	    static createFrom(source: any = {}) {
	        return new UpdateArtifactRootResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], root.Root);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateArtifactSourceRequestBody {
	    expectedRevision: number;
	    displayName: string;
	    enabled: boolean;
	    config?: number[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateArtifactSourceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.displayName = source["displayName"];
	        this.enabled = source["enabled"];
	        this.config = source["config"];
	    }
	}
	export class UpdateArtifactSourceRequest {
	    RootID: string;
	    SourceID: string;
	    Body?: UpdateArtifactSourceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new UpdateArtifactSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.SourceID = source["SourceID"];
	        this.Body = this.convertValues(source["Body"], UpdateArtifactSourceRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class UpdateArtifactSourceResponse {
	    Body?: source.Summary;
	
	    static createFrom(source: any = {}) {
	        return new UpdateArtifactSourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	    server: artifact.ArtifactRef;
	    authorizationURL: string;
	    expiresAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPOAuthAuthorization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
	        this.authorizationURL = source["authorizationURL"];
	        this.expiresAt = source["expiresAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace bundle {
	
	export class AdoptSkillRequest {
	    Bundle: collection.CollectionRef;
	    Occurrence: catalog.OccurrenceKey;
	    ArtifactID: string;
	    ExpectedCatalogRevision: number;
	    Name: string;
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AdoptSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Bundle = this.convertValues(source["Bundle"], collection.CollectionRef);
	        this.Occurrence = this.convertValues(source["Occurrence"], catalog.OccurrenceKey);
	        this.ArtifactID = source["ArtifactID"];
	        this.ExpectedCatalogRevision = source["ExpectedCatalogRevision"];
	        this.Name = source["Name"];
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
	export class AttachmentDraft {
	    SourceID: string;
	    Role: string;
	    Enabled: boolean;
	    DiscoveryRoot: string;
	    ExpectedMemberDigests: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new AttachmentDraft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SourceID = source["SourceID"];
	        this.Role = source["Role"];
	        this.Enabled = source["Enabled"];
	        this.DiscoveryRoot = source["DiscoveryRoot"];
	        this.ExpectedMemberDigests = source["ExpectedMemberDigests"];
	    }
	}
	export class CollectionData {
	    schemaVersion: string;
	    discoveryPolicyRevision: string;
	    logicalName: string;
	    logicalVersion?: string;
	    labels?: Record<string, string>;
	    managedSourceID?: string;
	
	    static createFrom(source: any = {}) {
	        return new CollectionData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.discoveryPolicyRevision = source["discoveryPolicyRevision"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.labels = source["labels"];
	        this.managedSourceID = source["managedSourceID"];
	    }
	}
	export class Bundle {
	    Collection: collection.Collection;
	    Data: CollectionData;
	    Attachments: collection.Attachment[];
	    Sources: source.Summary[];
	
	    static createFrom(source: any = {}) {
	        return new Bundle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Collection = this.convertValues(source["Collection"], collection.Collection);
	        this.Data = this.convertValues(source["Data"], CollectionData);
	        this.Attachments = this.convertValues(source["Attachments"], collection.Attachment);
	        this.Sources = this.convertValues(source["Sources"], source.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BundleExtension {
	    servers?: Record<string, server.ServerExtension>;
	    policies?: Record<string, policy.PolicyDocument>;
	
	    static createFrom(source: any = {}) {
	        return new BundleExtension(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], server.ServerExtension, true);
	        this.policies = this.convertValues(source["policies"], policy.PolicyDocument, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BundleDocument {
	    kind: string;
	    schemaID: string;
	    schemaVersion: string;
	    digest?: string;
	    logicalName: string;
	    logicalVersion?: string;
	    displayName?: string;
	    description?: string;
	    labels?: Record<string, string>;
	    mcpServers: Record<string, server.CoreServer>;
	    bundleExtension: BundleExtension;
	
	    static createFrom(source: any = {}) {
	        return new BundleDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.schemaID = source["schemaID"];
	        this.schemaVersion = source["schemaVersion"];
	        this.digest = source["digest"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.labels = source["labels"];
	        this.mcpServers = this.convertValues(source["mcpServers"], server.CoreServer, true);
	        this.bundleExtension = this.convertValues(source["bundleExtension"], BundleExtension);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class BundleInstallationView {
	    bundle: collection.CollectionRef;
	    builtIn: boolean;
	    collectionRevision: number;
	    overlayRevision: number;
	    runtimeEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BundleInstallationView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bundle = this.convertValues(source["bundle"], collection.CollectionRef);
	        this.builtIn = source["builtIn"];
	        this.collectionRevision = source["collectionRevision"];
	        this.overlayRevision = source["overlayRevision"];
	        this.runtimeEnabled = source["runtimeEnabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CreateBundleRequest {
	    RootID: string;
	    CollectionID: string;
	    ManagedSourceID: string;
	    ManagedSourceStorageKey: string;
	    DisplayName: string;
	    Description: string;
	    Enabled: boolean;
	    LogicalName: string;
	    LogicalVersion: string;
	    Labels: Record<string, string>;
	    Attachments: AttachmentDraft[];
	
	    static createFrom(source: any = {}) {
	        return new CreateBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.CollectionID = source["CollectionID"];
	        this.ManagedSourceID = source["ManagedSourceID"];
	        this.ManagedSourceStorageKey = source["ManagedSourceStorageKey"];
	        this.DisplayName = source["DisplayName"];
	        this.Description = source["Description"];
	        this.Enabled = source["Enabled"];
	        this.LogicalName = source["LogicalName"];
	        this.LogicalVersion = source["LogicalVersion"];
	        this.Labels = source["Labels"];
	        this.Attachments = this.convertValues(source["Attachments"], AttachmentDraft);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateManagedSkillRequest {
	    Bundle: collection.CollectionRef;
	    ExpectedCollectionRevision: number;
	    ArtifactID: string;
	    SkillName: string;
	    SKILLMD: number[];
	    ExpectedArtifactRevision: number;
	    Document?: spec.SkillDocument;
	    Files: source.ManagedPackageFile[];
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateManagedSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Bundle = this.convertValues(source["Bundle"], collection.CollectionRef);
	        this.ExpectedCollectionRevision = source["ExpectedCollectionRevision"];
	        this.ArtifactID = source["ArtifactID"];
	        this.SkillName = source["SkillName"];
	        this.SKILLMD = source["SKILLMD"];
	        this.ExpectedArtifactRevision = source["ExpectedArtifactRevision"];
	        this.Document = this.convertValues(source["Document"], spec.SkillDocument);
	        this.Files = this.convertValues(source["Files"], source.ManagedPackageFile);
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
	export class CreateManagedSkillResponse {
	    Artifact: artifact.Artifact;
	    Address: artifact.ArtifactAddress;
	
	    static createFrom(source: any = {}) {
	        return new CreateManagedSkillResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Artifact = this.convertValues(source["Artifact"], artifact.Artifact);
	        this.Address = this.convertValues(source["Address"], artifact.ArtifactAddress);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Registration {
	    ArtifactID: string;
	    Subresource: string;
	    Kind: string;
	    Enabled: boolean;
	    Data: number[];
	
	    static createFrom(source: any = {}) {
	        return new Registration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ArtifactID = source["ArtifactID"];
	        this.Subresource = source["Subresource"];
	        this.Kind = source["Kind"];
	        this.Enabled = source["Enabled"];
	        this.Data = source["Data"];
	    }
	}
	export class CreateRequest {
	    RootID: string;
	    CollectionID: string;
	    SourceID: string;
	    SourceStorageKey: string;
	    Document: number[];
	    Registrations: Registration[];
	
	    static createFrom(source: any = {}) {
	        return new CreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootID = source["RootID"];
	        this.CollectionID = source["CollectionID"];
	        this.SourceID = source["SourceID"];
	        this.SourceStorageKey = source["SourceStorageKey"];
	        this.Document = source["Document"];
	        this.Registrations = this.convertValues(source["Registrations"], Registration);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ManagedSkillDocument {
	    Artifact: artifact.Artifact;
	    Document: spec.SkillDocument;
	
	    static createFrom(source: any = {}) {
	        return new ManagedSkillDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Artifact = this.convertValues(source["Artifact"], artifact.Artifact);
	        this.Document = this.convertValues(source["Document"], spec.SkillDocument);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PinSkillRequest {
	    Bundle: collection.CollectionRef;
	    ExpectedCollectionRevision: number;
	    ArtifactID: string;
	    Binding: artifact.SourceBinding;
	    Name: string;
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PinSkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Bundle = this.convertValues(source["Bundle"], collection.CollectionRef);
	        this.ExpectedCollectionRevision = source["ExpectedCollectionRevision"];
	        this.ArtifactID = source["ArtifactID"];
	        this.Binding = this.convertValues(source["Binding"], artifact.SourceBinding);
	        this.Name = source["Name"];
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
	export class PolicyView {
	    artifact: artifact.Artifact;
	    collection: collection.CollectionRef;
	    catalogRevision: number;
	    definition: definition.Definition;
	    body: policy.MCPPolicy;
	    effectiveEnabled: boolean;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PolicyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.collection = this.convertValues(source["collection"], collection.CollectionRef);
	        this.catalogRevision = source["catalogRevision"];
	        this.definition = this.convertValues(source["definition"], definition.Definition);
	        this.body = this.convertValues(source["body"], policy.MCPPolicy);
	        this.effectiveEnabled = source["effectiveEnabled"];
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
	
	export class ReplaceDocumentRequest {
	    Bundle: collection.CollectionRef;
	    ExpectedCollectionRevision: number;
	    Document: number[];
	    Registrations: Registration[];
	    AllowProtected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReplaceDocumentRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Bundle = this.convertValues(source["Bundle"], collection.CollectionRef);
	        this.ExpectedCollectionRevision = source["ExpectedCollectionRevision"];
	        this.Document = source["Document"];
	        this.Registrations = this.convertValues(source["Registrations"], Registration);
	        this.AllowProtected = source["AllowProtected"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    collection: collection.CollectionRef;
	    catalogRevision: number;
	    document: server.ServerDocument;
	    installation: server.ServerData;
	    installationRevision: number;
	    runtimeEnabled: boolean;
	    builtIn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerInstallationView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.Artifact);
	        this.collection = this.convertValues(source["collection"], collection.CollectionRef);
	        this.catalogRevision = source["catalogRevision"];
	        this.document = this.convertValues(source["document"], server.ServerDocument);
	        this.installation = this.convertValues(source["installation"], server.ServerData);
	        this.installationRevision = source["installationRevision"];
	        this.runtimeEnabled = source["runtimeEnabled"];
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
	export class UpdateBundleRequest {
	    Bundle: collection.CollectionRef;
	    ExpectedRevision: number;
	    DisplayName: string;
	    Description: string;
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateBundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Bundle = this.convertValues(source["Bundle"], collection.CollectionRef);
	        this.ExpectedRevision = source["ExpectedRevision"];
	        this.DisplayName = source["DisplayName"];
	        this.Description = source["Description"];
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

export namespace catalog {
	
	export class OccurrenceKey {
	    collectionID: string;
	    sourceID: string;
	    locator: string;
	    subresourceLocator?: string;
	
	    static createFrom(source: any = {}) {
	        return new OccurrenceKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collectionID = source["collectionID"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	    }
	}

}

export namespace collection {
	
	export class Attachment {
	    rootID: string;
	    collectionID: string;
	    sourceID: string;
	    role: string;
	    enabled: boolean;
	    revision: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Attachment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.collectionID = source["collectionID"];
	        this.sourceID = source["sourceID"];
	        this.role = source["role"];
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
	export class Collection {
	    id: string;
	    rootID: string;
	    kind: string;
	    displayName: string;
	    description?: string;
	    enabled: boolean;
	    revision: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    // Go type: time
	    retiredAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new Collection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rootID = source["rootID"];
	        this.kind = source["kind"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.enabled = source["enabled"];
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
	export class CollectionRef {
	    rootID: string;
	    collectionID: string;
	
	    static createFrom(source: any = {}) {
	        return new CollectionRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
	        this.collectionID = source["collectionID"];
	    }
	}

}

export namespace definition {
	
	export class Selector {
	    kind: string;
	    logicalName?: string;
	    versionConstraint?: string;
	    labels?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Selector(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.logicalName = source["logicalName"];
	        this.versionConstraint = source["versionConstraint"];
	        this.labels = source["labels"];
	    }
	}
	export class Definition {
	    digest: string;
	    kind: string;
	    schemaID: string;
	    schemaVersion: string;
	    logicalName: string;
	    logicalVersion?: string;
	    displayName?: string;
	    description?: string;
	    labels?: Record<string, string>;
	    body: number[];
	    dependencies?: Selector[];
	
	    static createFrom(source: any = {}) {
	        return new Definition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.digest = source["digest"];
	        this.kind = source["kind"];
	        this.schemaID = source["schemaID"];
	        this.schemaVersion = source["schemaVersion"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.labels = source["labels"];
	        this.body = source["body"];
	        this.dependencies = this.convertValues(source["dependencies"], Selector);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace diagnostic {
	
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

export namespace main {
	
	export class MCPGlobalSettingsView {
	    settings: auth.MCPAuthSettings;
	    revision: number;
	    oauthRedirectURL?: string;
	    oauthLoopbackListenAddr?: string;
	    oauthRestartRequired: boolean;
	
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
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MCPSecretWriteResult {
	    secretRef: string;
	    sha256?: string;
	    nonEmpty: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPSecretWriteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.secretRef = source["secretRef"];
	        this.sha256 = source["sha256"];
	        this.nonEmpty = source["nonEmpty"];
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
	
	
	
	
	export class PolicyDocument {
	    kind: string;
	    schemaID: string;
	    schemaVersion: string;
	    digest?: string;
	    logicalName: string;
	    logicalVersion?: string;
	    displayName?: string;
	    description?: string;
	    labels?: Record<string, string>;
	    body: MCPPolicy;
	
	    static createFrom(source: any = {}) {
	        return new PolicyDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.schemaID = source["schemaID"];
	        this.schemaVersion = source["schemaVersion"];
	        this.digest = source["digest"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.labels = source["labels"];
	        this.body = this.convertValues(source["body"], MCPPolicy);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class RootDraft {
	    id: string;
	    storageKey: string;
	    displayName: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new RootDraft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.storageKey = source["storageKey"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	    }
	}
	export class RootUpdate {
	    expectedRevision: number;
	    displayName: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new RootUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
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
	    allowArtifacts?: artifact.ArtifactRef[];
	    activeArtifacts?: artifact.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new CreateSkillSessionRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.closeSessionID = source["closeSessionID"];
	        this.maxActivePerSession = source["maxActivePerSession"];
	        this.allowArtifacts = this.convertValues(source["allowArtifacts"], artifact.ArtifactRef);
	        this.activeArtifacts = this.convertValues(source["activeArtifacts"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    activeArtifacts: artifact.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new CreateSkillSessionResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionID = source["sessionID"];
	        this.activeArtifacts = this.convertValues(source["activeArtifacts"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	
	export class RuntimeSkillFilter {
	    types?: string[];
	    inserts?: string[];
	    locationPrefix?: string;
	    allowArtifacts?: artifact.ArtifactRef[];
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
	    server: artifact.ArtifactRef;
	    collection: collection.CollectionRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
	        this.collection = this.convertValues(source["collection"], collection.CollectionRef);
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
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	    skillRef: artifact.ArtifactRef;
	    type?: string;
	    name?: string;
	    displayName?: string;
	    description?: string;
	    digest?: string;
	    insert?: string;
	    arguments?: spec.SkillArgument[];
	    sourceTags?: string[];
	    resources: spec.SkillResourceInfo;
	    rawFrontmatter?: Record<string, any>;
	    warnings?: string[];
	    isActive?: boolean;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeSkillListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillRef = this.convertValues(source["skillRef"], artifact.ArtifactRef);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.digest = source["digest"];
	        this.insert = source["insert"];
	        this.arguments = this.convertValues(source["arguments"], spec.SkillArgument);
	        this.sourceTags = source["sourceTags"];
	        this.resources = this.convertValues(source["resources"], spec.SkillResourceInfo);
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
	
	export class MCPAppModelContextUpdate {
	    instanceID?: string;
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	export class MCPApprovalSummary {
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
	        this.serverDisplayName = source["serverDisplayName"];
	        this.toolName = source["toolName"];
	        this.toolDigest = source["toolDigest"];
	        this.risk = source["risk"];
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
	
	export class MCPPromptSelection {
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	export class MCPResourceTemplateSelection {
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
	        this.toolName = source["toolName"];
	        this.providerToolName = source["providerToolName"];
	        this.choiceID = source["choiceID"];
	        this.digest = source["digest"];
	        this.approvalRule = source["approvalRule"];
	        this.executionMode = source["executionMode"];
	        this.appResourceUri = source["appResourceUri"];
	        this.visibility = source["visibility"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MCPServerSelection {
	    server: artifact.ArtifactRef;
	    snapshotDigest?: string;
	    toolExposure: string;
	    selectedTools?: MCPToolSelection[];
	    includeServerInstructions?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	    server: artifact.ArtifactRef;
	    promptName: string;
	    description?: string;
	    messages?: MCPPromptMessage[];
	
	    static createFrom(source: any = {}) {
	        return new MCPGetPromptResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	
	export class MCPProviderToolMapping {
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
	        this.providerToolName = source["providerToolName"];
	        this.choiceID = source["choiceID"];
	        this.toolName = source["toolName"];
	        this.toolDigest = source["toolDigest"];
	        this.approvalRule = source["approvalRule"];
	        this.executionMode = source["executionMode"];
	        this.appResourceUri = source["appResourceUri"];
	        this.visibility = source["visibility"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    server: artifact.ArtifactRef;
	    uri: string;
	    contents?: MCPContent[];
	
	    static createFrom(source: any = {}) {
	        return new MCPReadResourceResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	
	
	export class MCPResourceTemplateRef {
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	    server: artifact.ArtifactRef;
	    collection: collection.CollectionRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
	        this.collection = this.convertValues(source["collection"], collection.CollectionRef);
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
	    server: artifact.ArtifactRef;
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
	        this.server = this.convertValues(source["server"], artifact.ArtifactRef);
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
	
	export class RenderSkillRequestBody {
	    artifact: artifact.ArtifactRef;
	    arguments?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RenderSkillRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
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
	    resources: spec.SkillResourceInfo;
	    arguments?: spec.SkillArgument[];
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
	        this.resources = this.convertValues(source["resources"], spec.SkillResourceInfo);
	        this.arguments = this.convertValues(source["arguments"], spec.SkillArgument);
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
	
	

}

export namespace selection {
	
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
	    workspace: collection.CollectionRef;
	    displayName?: string;
	    workspaceRevision?: number;
	    catalogRevision?: number;
	    contextRefs?: ConversationResourceSelectionRef[];
	    skillRefs?: ConversationResourceSelectionRef[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.displayName = source["displayName"];
	        this.workspaceRevision = source["workspaceRevision"];
	        this.catalogRevision = source["catalogRevision"];
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
	    workspace: collection.CollectionRef;
	    displayName?: string;
	    workspaceRevision?: number;
	    catalogRevision?: number;
	    status: string;
	    contexts?: ConversationContextUsage[];
	    skills?: ConversationSkillUsage[];
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.displayName = source["displayName"];
	        this.workspaceRevision = source["workspaceRevision"];
	        this.catalogRevision = source["catalogRevision"];
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
	export class PolicyReference {
	    ref: string;
	    required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PolicyReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = source["ref"];
	        this.required = source["required"];
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
	export class ServerExtension {
	    logicalVersion?: string;
	    displayName?: string;
	    description?: string;
	    timeoutMS?: number;
	    labels?: Record<string, string>;
	    auth: AuthenticationDeclaration;
	    install: InstallationDeclaration;
	    connectionProfiles?: Record<string, ConnectionProfile>;
	    policy?: PolicyReference;
	
	    static createFrom(source: any = {}) {
	        return new ServerExtension(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logicalVersion = source["logicalVersion"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.timeoutMS = source["timeoutMS"];
	        this.labels = source["labels"];
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
	export class ServerDocument {
	    kind: string;
	    schemaID: string;
	    schemaVersion: string;
	    digest?: string;
	    logicalName: string;
	    logicalVersion?: string;
	    displayName?: string;
	    description?: string;
	    labels?: Record<string, string>;
	    mcpServer: CoreServer;
	    extension: ServerExtension;
	
	    static createFrom(source: any = {}) {
	        return new ServerDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.schemaID = source["schemaID"];
	        this.schemaVersion = source["schemaVersion"];
	        this.digest = source["digest"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.labels = source["labels"];
	        this.mcpServer = this.convertValues(source["mcpServer"], CoreServer);
	        this.extension = this.convertValues(source["extension"], ServerExtension);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Resolved {
	    Server: artifact.ArtifactRef;
	    Collection: collection.CollectionRef;
	    ArtifactRevision: number;
	    CatalogRevision: number;
	    DefinitionDigest: string;
	    SourceContentDigest: string;
	    SourceGeneration: string;
	    Document: ServerDocument;
	    Installation: ServerData;
	    Policy: policy.Effective;
	    InstallationRevision: number;
	    RuntimeEnabled: boolean;
	    BuiltIn: boolean;
	    Version: string;
	
	    static createFrom(source: any = {}) {
	        return new Resolved(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Server = this.convertValues(source["Server"], artifact.ArtifactRef);
	        this.Collection = this.convertValues(source["Collection"], collection.CollectionRef);
	        this.ArtifactRevision = source["ArtifactRevision"];
	        this.CatalogRevision = source["CatalogRevision"];
	        this.DefinitionDigest = source["DefinitionDigest"];
	        this.SourceContentDigest = source["SourceContentDigest"];
	        this.SourceGeneration = source["SourceGeneration"];
	        this.Document = this.convertValues(source["Document"], ServerDocument);
	        this.Installation = this.convertValues(source["Installation"], ServerData);
	        this.Policy = this.convertValues(source["Policy"], policy.Effective);
	        this.InstallationRevision = source["InstallationRevision"];
	        this.RuntimeEnabled = source["RuntimeEnabled"];
	        this.BuiltIn = source["BuiltIn"];
	        this.Version = source["Version"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	
	export class ManagedPackageAddress {
	    kind: string;
	    name: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new ManagedPackageAddress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.version = source["version"];
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
	export class ArtifactSkillSelection {
	    artifact: artifact.ArtifactRef;
	    preLoadAsActive: boolean;
	    useAsInstructions: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSkillSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
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
	    startingSkillSelections?: ArtifactSkillSelection[];
	    startingMCPContext?: runtime.MCPConversationContext;
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
	        this.startingSkillSelections = this.convertValues(source["startingSkillSelections"], ArtifactSkillSelection);
	        this.startingMCPContext = this.convertValues(source["startingMCPContext"], runtime.MCPConversationContext);
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
	    mcpContext?: runtime.MCPConversationContext;
	    mcpToolMappings?: runtime.MCPProviderToolMapping[];
	    mcpAppContextUpdates?: runtime.MCPAppModelContextUpdate[];
	    workspaceSelection?: selection.ConversationSelection;
	    workspaceUsage?: selection.ConversationUsage;
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
	        this.mcpContext = this.convertValues(source["mcpContext"], runtime.MCPConversationContext);
	        this.mcpToolMappings = this.convertValues(source["mcpToolMappings"], runtime.MCPProviderToolMapping);
	        this.mcpAppContextUpdates = this.convertValues(source["mcpAppContextUpdates"], runtime.MCPAppModelContextUpdate);
	        this.workspaceSelection = this.convertValues(source["workspaceSelection"], selection.ConversationSelection);
	        this.workspaceUsage = this.convertValues(source["workspaceUsage"], selection.ConversationUsage);
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
	    mcpContext?: runtime.MCPConversationContext;
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
	        this.mcpContext = this.convertValues(source["mcpContext"], runtime.MCPConversationContext);
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
	    mcpToolMappings?: runtime.MCPProviderToolMapping[];
	    workspaceUsage?: selection.ConversationUsage;
	
	    static createFrom(source: any = {}) {
	        return new CompletionResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inferenceResponse = this.convertValues(source["inferenceResponse"], FetchCompletionResponse);
	        this.hydratedCurrentInputs = this.convertValues(source["hydratedCurrentInputs"], InputUnion);
	        this.mcpToolMappings = this.convertValues(source["mcpToolMappings"], runtime.MCPProviderToolMapping);
	        this.workspaceUsage = this.convertValues(source["workspaceUsage"], selection.ConversationUsage);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    startingSkillSelections?: ArtifactSkillSelection[];
	    startingMCPContext?: runtime.MCPConversationContext;
	
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
	        this.startingSkillSelections = this.convertValues(source["startingSkillSelections"], ArtifactSkillSelection);
	        this.startingMCPContext = this.convertValues(source["startingMCPContext"], runtime.MCPConversationContext);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	
	export class WorkspaceArtifactSettings {
	    runtimeDisabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceArtifactSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runtimeDisabled = source["runtimeDisabled"];
	    }
	}
	export class WorkspaceOccurrenceRef {
	    sourceID: string;
	    locator: string;
	    subresourceLocator?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceOccurrenceRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	    }
	}
	export class AdoptWorkspaceOccurrenceRequestBody {
	    expectedCatalogRevision: number;
	    occurrence: WorkspaceOccurrenceRef;
	    artifactID: string;
	    name?: string;
	    enabled: boolean;
	    settings: WorkspaceArtifactSettings;
	
	    static createFrom(source: any = {}) {
	        return new AdoptWorkspaceOccurrenceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedCatalogRevision = source["expectedCatalogRevision"];
	        this.occurrence = this.convertValues(source["occurrence"], WorkspaceOccurrenceRef);
	        this.artifactID = source["artifactID"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.settings = this.convertValues(source["settings"], WorkspaceArtifactSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AdoptWorkspaceOccurrenceRequest {
	    workspace: collection.CollectionRef;
	    Body?: AdoptWorkspaceOccurrenceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new AdoptWorkspaceOccurrenceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.Body = this.convertValues(source["Body"], AdoptWorkspaceOccurrenceRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    name: string;
	    kind: string;
	    enabled: boolean;
	    state: string;
	    adoption: string;
	    resolvedDefinition?: string;
	    sourceID: string;
	    locator: string;
	    subresourceLocator?: string;
	    runtimeDisabled: boolean;
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceArtifactView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.revision = source["revision"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.enabled = source["enabled"];
	        this.state = source["state"];
	        this.adoption = source["adoption"];
	        this.resolvedDefinition = source["resolvedDefinition"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.subresourceLocator = source["subresourceLocator"];
	        this.runtimeDisabled = source["runtimeDisabled"];
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
	export class AdoptWorkspaceOccurrenceResponse {
	    Body?: WorkspaceArtifactView;
	
	    static createFrom(source: any = {}) {
	        return new AdoptWorkspaceOccurrenceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceArtifactView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
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
	    expectedCollectionRevision: number;
	    sourceID: string;
	    role: string;
	    enabled: boolean;
	    settings: WorkspaceAttachmentSettings;
	
	    static createFrom(source: any = {}) {
	        return new AttachWorkspaceSourceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.sourceID = source["sourceID"];
	        this.role = source["role"];
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
	    workspace: collection.CollectionRef;
	    Body?: AttachWorkspaceSourceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new AttachWorkspaceSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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
	    enabled: boolean;
	    sourceDisplayName?: string;
	    sourceKind?: string;
	    path?: string;
	    settings: WorkspaceAttachmentSettings;
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceAttachmentView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceID = source["sourceID"];
	        this.revision = source["revision"];
	        this.role = source["role"];
	        this.enabled = source["enabled"];
	        this.sourceDisplayName = source["sourceDisplayName"];
	        this.sourceKind = source["sourceKind"];
	        this.path = source["path"];
	        this.settings = this.convertValues(source["settings"], WorkspaceAttachmentSettings);
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
	    workspace: collection.CollectionRef;
	    revision: number;
	    displayName: string;
	    description?: string;
	    enabled: boolean;
	    mode: string;
	    primarySourceID?: string;
	    primaryPath?: string;
	    discovery: WorkspaceDiscovery;
	    attachments: WorkspaceAttachmentView[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.revision = source["revision"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.enabled = source["enabled"];
	        this.mode = source["mode"];
	        this.primarySourceID = source["primarySourceID"];
	        this.primaryPath = source["primaryPath"];
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
	    artifacts?: artifact.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new ComposeWorkspaceContextRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifacts = this.convertValues(source["artifacts"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ComposeWorkspaceContextRequest {
	    workspace: collection.CollectionRef;
	    Body?: ComposeWorkspaceContextRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new ComposeWorkspaceContextRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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
	    artifact: artifact.ArtifactRef;
	    status: string;
	    code?: string;
	    originalBytes: number;
	    includedBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextDecision(source);
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
	export class WorkspaceContextContribution {
	    artifact: artifact.ArtifactRef;
	    recordRevision: number;
	    definitionDigest: string;
	    sourceID: string;
	    locator: string;
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
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.recordRevision = source["recordRevision"];
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.name = source["name"];
	        this.role = source["role"];
	        this.mediaType = source["mediaType"];
	        this.content = source["content"];
	        this.conventionOrder = source["conventionOrder"];
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
	export class WorkspaceContextLoadPlan {
	    workspace: collection.CollectionRef;
	    catalogRevision: number;
	    contributions: WorkspaceContextContribution[];
	    prompt: string;
	    diagnostics?: diagnostic.Diagnostic[];
	    decisions: WorkspaceContextDecision[];
	    promptBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextLoadPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.catalogRevision = source["catalogRevision"];
	        this.contributions = this.convertValues(source["contributions"], WorkspaceContextContribution);
	        this.prompt = source["prompt"];
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
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
	export class ComposeWorkspaceLoadPlanRequestBody {
	    artifacts: artifact.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new ComposeWorkspaceLoadPlanRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifacts = this.convertValues(source["artifacts"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ComposeWorkspaceLoadPlanRequest {
	    workspace: collection.CollectionRef;
	    Body?: ComposeWorkspaceLoadPlanRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new ComposeWorkspaceLoadPlanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.Body = this.convertValues(source["Body"], ComposeWorkspaceLoadPlanRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class WorkspaceDefinitionView {
	    digest: string;
	    kind: string;
	    schemaID: string;
	    schemaVersion: string;
	    logicalName: string;
	    logicalVersion?: string;
	    displayName?: string;
	    description?: string;
	    labels?: Record<string, string>;
	    body: number[];
	    dependencies?: definition.Selector[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDefinitionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.digest = source["digest"];
	        this.kind = source["kind"];
	        this.schemaID = source["schemaID"];
	        this.schemaVersion = source["schemaVersion"];
	        this.logicalName = source["logicalName"];
	        this.logicalVersion = source["logicalVersion"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.labels = source["labels"];
	        this.body = source["body"];
	        this.dependencies = this.convertValues(source["dependencies"], definition.Selector);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceLoadPlanItemView {
	    artifact: WorkspaceArtifactView;
	    definition: WorkspaceDefinitionView;
	    definitionDigest: string;
	    sourceID: string;
	    sourceKind: string;
	    locator: string;
	    catalogCurrent: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceLoadPlanItemView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], WorkspaceArtifactView);
	        this.definition = this.convertValues(source["definition"], WorkspaceDefinitionView);
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.sourceKind = source["sourceKind"];
	        this.locator = source["locator"];
	        this.catalogCurrent = source["catalogCurrent"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceLoadPlanView {
	    workspace: collection.CollectionRef;
	    catalogRevision: number;
	    items: WorkspaceLoadPlanItemView[];
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceLoadPlanView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.catalogRevision = source["catalogRevision"];
	        this.items = this.convertValues(source["items"], WorkspaceLoadPlanItemView);
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
	export class ComposeWorkspaceLoadPlanResponse {
	    Body?: WorkspaceLoadPlanView;
	
	    static createFrom(source: any = {}) {
	        return new ComposeWorkspaceLoadPlanResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceLoadPlanView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    workspaceID: string;
	    displayName: string;
	    description?: string;
	    discovery: WorkspaceDiscovery;
	
	    static createFrom(source: any = {}) {
	        return new CreateEmptyWorkspaceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspaceID = source["workspaceID"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
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
	    rootID?: string;
	    Body?: CreateEmptyWorkspaceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new CreateEmptyWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
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
	    workspaceID: string;
	    sourceID: string;
	    sourceStorageKey: string;
	    displayName: string;
	    description?: string;
	    rootPath: string;
	    discovery: WorkspaceDiscovery;
	
	    static createFrom(source: any = {}) {
	        return new CreateFilesystemWorkspaceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspaceID = source["workspaceID"];
	        this.sourceID = source["sourceID"];
	        this.sourceStorageKey = source["sourceStorageKey"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.rootPath = source["rootPath"];
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
	    rootID?: string;
	    Body?: CreateFilesystemWorkspaceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new CreateFilesystemWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
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
	export class DetachWorkspaceSourceRequest {
	    workspace: collection.CollectionRef;
	    sourceID: string;
	    expectedCollectionRevision: number;
	    expectedAttachmentRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new DetachWorkspaceSourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.sourceID = source["sourceID"];
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.expectedAttachmentRevision = source["expectedAttachmentRevision"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	export class GetWorkspaceArtifactRequest {
	    workspace: collection.CollectionRef;
	    artifact: artifact.ArtifactRef;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceArtifactRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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
	export class GetWorkspaceArtifactResponse {
	    Body?: WorkspaceArtifactView;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceArtifactResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceArtifactView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    workspace: collection.CollectionRef;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceCatalogRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	    artifact?: artifact.ArtifactRef;
	    diagnostics?: diagnostic.Diagnostic[];
	
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
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
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
	    artifact: WorkspaceArtifactView;
	    definitionDigest: string;
	    sourceID: string;
	    locator: string;
	    catalogCurrent: boolean;
	    projectionValid: boolean;
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceResourceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], WorkspaceArtifactView);
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.projectionValid = source["projectionValid"];
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
	export class WorkspaceCatalogView {
	    workspace: WorkspaceView;
	    catalogRevision: number;
	    catalogCurrent: boolean;
	    diagnostics?: diagnostic.Diagnostic[];
	    resources: WorkspaceResourceView[];
	    groups: WorkspaceResourceGroupView[];
	    occurrences: WorkspaceOccurrenceView[];
	    validOccurrences: WorkspaceOccurrenceView[];
	    invalidOccurrences: WorkspaceOccurrenceView[];
	    missingOccurrences: WorkspaceOccurrenceView[];
	    unrecordedOccurrences: WorkspaceOccurrenceView[];
	    unresolvedArtifacts: WorkspaceArtifactView[];
	    unrecordedCount: number;
	    unresolvedArtifactCount: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceCatalogView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], WorkspaceView);
	        this.catalogRevision = source["catalogRevision"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
	        this.resources = this.convertValues(source["resources"], WorkspaceResourceView);
	        this.groups = this.convertValues(source["groups"], WorkspaceResourceGroupView);
	        this.occurrences = this.convertValues(source["occurrences"], WorkspaceOccurrenceView);
	        this.validOccurrences = this.convertValues(source["validOccurrences"], WorkspaceOccurrenceView);
	        this.invalidOccurrences = this.convertValues(source["invalidOccurrences"], WorkspaceOccurrenceView);
	        this.missingOccurrences = this.convertValues(source["missingOccurrences"], WorkspaceOccurrenceView);
	        this.unrecordedOccurrences = this.convertValues(source["unrecordedOccurrences"], WorkspaceOccurrenceView);
	        this.unresolvedArtifacts = this.convertValues(source["unresolvedArtifacts"], WorkspaceArtifactView);
	        this.unrecordedCount = source["unrecordedCount"];
	        this.unresolvedArtifactCount = source["unresolvedArtifactCount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class GetWorkspaceRequest {
	    workspace: collection.CollectionRef;
	
	    static createFrom(source: any = {}) {
	        return new GetWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	export class ListWorkspaceArtifactsRequest {
	    workspace: collection.CollectionRef;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceArtifactsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListWorkspaceArtifactsResponseBody {
	    artifacts: WorkspaceArtifactView[];
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceArtifactsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifacts = this.convertValues(source["artifacts"], WorkspaceArtifactView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListWorkspaceArtifactsResponse {
	    Body?: ListWorkspaceArtifactsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceArtifactsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListWorkspaceArtifactsResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    workspace: collection.CollectionRef;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceContextsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceContextView {
	    artifact: artifact.ArtifactRef;
	    recordRevision: number;
	    definitionDigest: string;
	    sourceID: string;
	    locator: string;
	    name: string;
	    role: string;
	    mediaType: string;
	    enabled: boolean;
	    state: string;
	    catalogCurrent: boolean;
	    projectionValid: boolean;
	    runtimeDisabled: boolean;
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.recordRevision = source["recordRevision"];
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.name = source["name"];
	        this.role = source["role"];
	        this.mediaType = source["mediaType"];
	        this.enabled = source["enabled"];
	        this.state = source["state"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.projectionValid = source["projectionValid"];
	        this.runtimeDisabled = source["runtimeDisabled"];
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
	    workspace: collection.CollectionRef;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceSkillsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	    workspace: collection.CollectionRef;
	    artifact: artifact.ArtifactRef;
	    definitionDigest: string;
	    sourceID: string;
	    locator: string;
	    skill: WorkspaceSkillSummary;
	    markdownBody?: string;
	    recordRevision: number;
	    state: string;
	    projectionValid: boolean;
	    catalogCurrent: boolean;
	    runtimeDisabled: boolean;
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkillView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.definitionDigest = source["definitionDigest"];
	        this.sourceID = source["sourceID"];
	        this.locator = source["locator"];
	        this.skill = this.convertValues(source["skill"], WorkspaceSkillSummary);
	        this.markdownBody = source["markdownBody"];
	        this.recordRevision = source["recordRevision"];
	        this.state = source["state"];
	        this.projectionValid = source["projectionValid"];
	        this.catalogCurrent = source["catalogCurrent"];
	        this.runtimeDisabled = source["runtimeDisabled"];
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
	
	export class ListWorkspaceSuppressionsRequest {
	    workspace: collection.CollectionRef;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceSuppressionsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceSuppressionView {
	    workspace: collection.CollectionRef;
	    binding: artifact.SourceBinding;
	    revision: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSuppressionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.binding = this.convertValues(source["binding"], artifact.SourceBinding);
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
	export class ListWorkspaceSuppressionsResponseBody {
	    suppressions: WorkspaceSuppressionView[];
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceSuppressionsResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.suppressions = this.convertValues(source["suppressions"], WorkspaceSuppressionView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListWorkspaceSuppressionsResponse {
	    Body?: ListWorkspaceSuppressionsResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspaceSuppressionsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ListWorkspaceSuppressionsResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    rootID?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListWorkspacesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootID = source["rootID"];
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
	    artifacts?: artifact.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceContextsRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifacts = this.convertValues(source["artifacts"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LoadWorkspaceContextsRequest {
	    workspace: collection.CollectionRef;
	    Body?: LoadWorkspaceContextsRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceContextsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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
	    workspace: collection.CollectionRef;
	    catalogRevision: number;
	    contributions: WorkspaceContextContribution[];
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceContextInspectionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.catalogRevision = source["catalogRevision"];
	        this.contributions = this.convertValues(source["contributions"], WorkspaceContextContribution);
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
	    artifacts: artifact.ArtifactRef[];
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceSkillsRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifacts = this.convertValues(source["artifacts"], artifact.ArtifactRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LoadWorkspaceSkillsRequest {
	    workspace: collection.CollectionRef;
	    Body?: LoadWorkspaceSkillsRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new LoadWorkspaceSkillsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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
	    workspace: collection.CollectionRef;
	    catalogRevision: number;
	    skills: WorkspaceSkillView[];
	    diagnostics?: diagnostic.Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSkillLoadView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.catalogRevision = source["catalogRevision"];
	        this.skills = this.convertValues(source["skills"], WorkspaceSkillView);
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
	export class PinWorkspaceArtifactRequestBody {
	    expectedCollectionRevision: number;
	    binding: artifact.SourceBinding;
	    artifactID: string;
	    name: string;
	    enabled: boolean;
	    settings: WorkspaceArtifactSettings;
	
	    static createFrom(source: any = {}) {
	        return new PinWorkspaceArtifactRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.binding = this.convertValues(source["binding"], artifact.SourceBinding);
	        this.artifactID = source["artifactID"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.settings = this.convertValues(source["settings"], WorkspaceArtifactSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PinWorkspaceArtifactRequest {
	    workspace: collection.CollectionRef;
	    Body?: PinWorkspaceArtifactRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new PinWorkspaceArtifactRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.Body = this.convertValues(source["Body"], PinWorkspaceArtifactRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PinWorkspaceArtifactResponse {
	    Body?: WorkspaceArtifactView;
	
	    static createFrom(source: any = {}) {
	        return new PinWorkspaceArtifactResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceArtifactView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PurgeWorkspaceArtifactRequest {
	    workspace: collection.CollectionRef;
	    artifact: artifact.ArtifactRef;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new PurgeWorkspaceArtifactRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
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
	export class PurgeWorkspaceArtifactResponseBody {
	    artifact: artifact.ArtifactRef;
	
	    static createFrom(source: any = {}) {
	        return new PurgeWorkspaceArtifactResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
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
	export class PurgeWorkspaceArtifactResponse {
	    Body?: PurgeWorkspaceArtifactResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new PurgeWorkspaceArtifactResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], PurgeWorkspaceArtifactResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PurgeWorkspaceRequest {
	    workspace: collection.CollectionRef;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new PurgeWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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
	export class PurgeWorkspaceResponseBody {
	    workspace: collection.CollectionRef;
	
	    static createFrom(source: any = {}) {
	        return new PurgeWorkspaceResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PurgeWorkspaceResponse {
	    Body?: PurgeWorkspaceResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new PurgeWorkspaceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], PurgeWorkspaceResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    workspace: collection.CollectionRef;
	
	    static createFrom(source: any = {}) {
	        return new RefreshWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceRefreshResult {
	    workspace: collection.CollectionRef;
	    catalogRevision: number;
	    createdArtifacts: artifact.ArtifactRef[];
	    updatedArtifacts: artifact.ArtifactRef[];
	    diagnostics?: diagnostic.Diagnostic[];
	    candidates: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceRefreshResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.catalogRevision = source["catalogRevision"];
	        this.createdArtifacts = this.convertValues(source["createdArtifacts"], artifact.ArtifactRef);
	        this.updatedArtifacts = this.convertValues(source["updatedArtifacts"], artifact.ArtifactRef);
	        this.diagnostics = this.convertValues(source["diagnostics"], diagnostic.Diagnostic);
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
	export class ReplaceWorkspacePrimarySourceRequestBody {
	    expectedCollectionRevision: number;
	    previousSourceID: string;
	    expectedPreviousAttachmentRevision: number;
	    sourceID: string;
	
	    static createFrom(source: any = {}) {
	        return new ReplaceWorkspacePrimarySourceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.previousSourceID = source["previousSourceID"];
	        this.expectedPreviousAttachmentRevision = source["expectedPreviousAttachmentRevision"];
	        this.sourceID = source["sourceID"];
	    }
	}
	export class ReplaceWorkspacePrimarySourceRequest {
	    workspace: collection.CollectionRef;
	    Body?: ReplaceWorkspacePrimarySourceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new ReplaceWorkspacePrimarySourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.Body = this.convertValues(source["Body"], ReplaceWorkspacePrimarySourceRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ReplaceWorkspacePrimarySourceResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new ReplaceWorkspacePrimarySourceResponse(source);
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
	export class ResolveWorkspaceResourceRequestBody {
	    artifact?: artifact.ArtifactRef;
	    selector?: definition.Selector;
	
	    static createFrom(source: any = {}) {
	        return new ResolveWorkspaceResourceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.selector = this.convertValues(source["selector"], definition.Selector);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResolveWorkspaceResourceRequest {
	    workspace: collection.CollectionRef;
	    Body?: ResolveWorkspaceResourceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new ResolveWorkspaceResourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.Body = this.convertValues(source["Body"], ResolveWorkspaceResourceRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ResolveWorkspaceResourceResponseBody {
	    resource: WorkspaceResourceView;
	    definition: WorkspaceDefinitionView;
	
	    static createFrom(source: any = {}) {
	        return new ResolveWorkspaceResourceResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resource = this.convertValues(source["resource"], WorkspaceResourceView);
	        this.definition = this.convertValues(source["definition"], WorkspaceDefinitionView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResolveWorkspaceResourceResponse {
	    Body?: ResolveWorkspaceResourceResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new ResolveWorkspaceResourceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], ResolveWorkspaceResourceResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RetireWorkspaceRequest {
	    workspace: collection.CollectionRef;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new RetireWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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
	export class RetireWorkspaceResponseBody {
	    workspace: collection.CollectionRef;
	    revision: number;
	
	    static createFrom(source: any = {}) {
	        return new RetireWorkspaceResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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
	export class RetireWorkspaceResponse {
	    Body?: RetireWorkspaceResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new RetireWorkspaceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], RetireWorkspaceResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SetWorkspaceArtifactEnabledRequestBody {
	    expectedRevision: number;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceArtifactEnabledRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.enabled = source["enabled"];
	    }
	}
	export class SetWorkspaceArtifactEnabledRequest {
	    workspace: collection.CollectionRef;
	    artifact: artifact.ArtifactRef;
	    Body?: SetWorkspaceArtifactEnabledRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceArtifactEnabledRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.Body = this.convertValues(source["Body"], SetWorkspaceArtifactEnabledRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SetWorkspaceArtifactEnabledResponse {
	    Body?: WorkspaceArtifactView;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceArtifactEnabledResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceArtifactView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SetWorkspaceArtifactRuntimeDisabledRequestBody {
	    expectedRevision: number;
	    runtimeDisabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceArtifactRuntimeDisabledRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedRevision = source["expectedRevision"];
	        this.runtimeDisabled = source["runtimeDisabled"];
	    }
	}
	export class SetWorkspaceArtifactRuntimeDisabledRequest {
	    workspace: collection.CollectionRef;
	    artifact: artifact.ArtifactRef;
	    Body?: SetWorkspaceArtifactRuntimeDisabledRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceArtifactRuntimeDisabledRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.Body = this.convertValues(source["Body"], SetWorkspaceArtifactRuntimeDisabledRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SetWorkspaceArtifactRuntimeDisabledResponse {
	    Body?: WorkspaceArtifactView;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspaceArtifactRuntimeDisabledResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceArtifactView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SetWorkspacePrimarySourceRequestBody {
	    expectedCollectionRevision: number;
	    previousSourceID?: string;
	    expectedPreviousAttachmentRevision?: number;
	    sourceID?: string;
	    clear?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspacePrimarySourceRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.previousSourceID = source["previousSourceID"];
	        this.expectedPreviousAttachmentRevision = source["expectedPreviousAttachmentRevision"];
	        this.sourceID = source["sourceID"];
	        this.clear = source["clear"];
	    }
	}
	export class SetWorkspacePrimarySourceRequest {
	    workspace: collection.CollectionRef;
	    Body?: SetWorkspacePrimarySourceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspacePrimarySourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.Body = this.convertValues(source["Body"], SetWorkspacePrimarySourceRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SetWorkspacePrimarySourceResponse {
	    Body?: WorkspaceView;
	
	    static createFrom(source: any = {}) {
	        return new SetWorkspacePrimarySourceResponse(source);
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
	export class SuppressWorkspaceBindingRequestBody {
	    expectedCollectionRevision: number;
	    binding: artifact.SourceBinding;
	
	    static createFrom(source: any = {}) {
	        return new SuppressWorkspaceBindingRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.binding = this.convertValues(source["binding"], artifact.SourceBinding);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SuppressWorkspaceBindingRequest {
	    workspace: collection.CollectionRef;
	    Body?: SuppressWorkspaceBindingRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new SuppressWorkspaceBindingRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.Body = this.convertValues(source["Body"], SuppressWorkspaceBindingRequestBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SuppressWorkspaceBindingResponse {
	    Body?: WorkspaceSuppressionView;
	
	    static createFrom(source: any = {}) {
	        return new SuppressWorkspaceBindingResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], WorkspaceSuppressionView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UnadoptWorkspaceArtifactRequest {
	    workspace: collection.CollectionRef;
	    artifact: artifact.ArtifactRef;
	    expectedRevision: number;
	    suppress: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UnadoptWorkspaceArtifactRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.artifact = this.convertValues(source["artifact"], artifact.ArtifactRef);
	        this.expectedRevision = source["expectedRevision"];
	        this.suppress = source["suppress"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UnadoptWorkspaceArtifactResponseBody {
	    artifact: artifact.ArtifactRef;
	
	    static createFrom(source: any = {}) {
	        return new UnadoptWorkspaceArtifactResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
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
	export class UnadoptWorkspaceArtifactResponse {
	    Body?: UnadoptWorkspaceArtifactResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new UnadoptWorkspaceArtifactResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], UnadoptWorkspaceArtifactResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class UnsuppressWorkspaceBindingRequest {
	    workspace: collection.CollectionRef;
	    binding: artifact.SourceBinding;
	    expectedRevision: number;
	
	    static createFrom(source: any = {}) {
	        return new UnsuppressWorkspaceBindingRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.binding = this.convertValues(source["binding"], artifact.SourceBinding);
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
	export class UnsuppressWorkspaceBindingResponseBody {
	    workspace: collection.CollectionRef;
	    binding: artifact.SourceBinding;
	
	    static createFrom(source: any = {}) {
	        return new UnsuppressWorkspaceBindingResponseBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.binding = this.convertValues(source["binding"], artifact.SourceBinding);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UnsuppressWorkspaceBindingResponse {
	    Body?: UnsuppressWorkspaceBindingResponseBody;
	
	    static createFrom(source: any = {}) {
	        return new UnsuppressWorkspaceBindingResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Body = this.convertValues(source["Body"], UnsuppressWorkspaceBindingResponseBody);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    expectedCollectionRevision: number;
	    expectedAttachmentRevision: number;
	    role: string;
	    enabled: boolean;
	    settings: WorkspaceAttachmentSettings;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceAttachmentRequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expectedCollectionRevision = source["expectedCollectionRevision"];
	        this.expectedAttachmentRevision = source["expectedAttachmentRevision"];
	        this.role = source["role"];
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
	    workspace: collection.CollectionRef;
	    sourceID: string;
	    Body?: UpdateWorkspaceAttachmentRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceAttachmentRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
	        this.sourceID = source["sourceID"];
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
	export class UpdateWorkspaceRequestBody {
	    expectedRevision: number;
	    displayName: string;
	    description?: string;
	    enabled: boolean;
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
	    workspace: collection.CollectionRef;
	    Body?: UpdateWorkspaceRequestBody;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWorkspaceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], collection.CollectionRef);
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

