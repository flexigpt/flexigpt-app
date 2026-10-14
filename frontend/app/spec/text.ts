import type { ArtifactDigest, ArtifactLocator, ArtifactRef } from '@/spec/artifact';

enum TextInsert {
	Instructions = 'instructions',
	UserMessage = 'user-message',
}

export interface TextMaterialization {
	artifact: ArtifactRef;
	artifactRevision: number;
	definitionDigest: ArtifactDigest;
	name: string;
	insert: TextInsert;
	mediaType?: string;
	content: string;
	locator: ArtifactLocator;
	builtIn: boolean;
}
