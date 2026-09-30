import type { ArtifactRef } from '@/spec/artifact';

import type { IModelAggregateAPI } from '@/apis/interface';
import { requiredObject, requireWailsString } from '@/apis/wailsapi/transport';
import {
	GetDefaultModelProvider,
	GetModelProviderDefaultModel,
	SetDefaultModelProvider,
} from '@/apis/wailsjs/go/main/ModelAggregateWrapper';

function artifactRefFromWails(value: unknown, operation: string): ArtifactRef {
	const ref = requiredObject<Record<string, unknown>>(value, operation);

	return {
		rootID: requireWailsString(ref.rootID, `${operation}.rootID`),
		artifactID: requireWailsString(ref.artifactID, `${operation}.artifactID`),
	};
}

function optionalArtifactRefFromWails(value: unknown, operation: string): ArtifactRef | undefined {
	if (value === null || value === undefined) {
		return undefined;
	}
	return artifactRefFromWails(value, operation);
}

export class WailsModelAggregateAPI implements IModelAggregateAPI {
	async getDefaultModelProvider(): Promise<ArtifactRef | undefined> {
		const result = await GetDefaultModelProvider();
		return optionalArtifactRefFromWails(result, 'GetDefaultModelProvider');
	}

	async setDefaultModelProvider(provider?: ArtifactRef): Promise<void> {
		// The Go method accepts *artifact.ArtifactRef. Wails generation currently
		// represents that pointer as a required ArtifactRef in its declaration.
		// Send null explicitly so Go receives a nil pointer and clears preference.
		await SetDefaultModelProvider((provider ?? null) as never);
	}

	async getModelProviderDefaultModel(provider: ArtifactRef): Promise<ArtifactRef> {
		const result = await GetModelProviderDefaultModel(provider as never);
		return artifactRefFromWails(result, 'GetModelProviderDefaultModel');
	}
}
