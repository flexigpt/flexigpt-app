import type { ArtifactRef } from '@/spec/artifact';
import type { TextMaterialization } from '@/spec/text';
import { TextInsert } from '@/spec/text';

import type { ITextStoreAPI } from '@/apis/interface';
import { enumFromWails, requiredObject } from '@/apis/wailsapi/transport';
import { MaterializeText } from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/textstorewrapper';

export class WailsTextStoreAPI implements ITextStoreAPI {
	async materializeText(text: ArtifactRef): Promise<TextMaterialization> {
		const materialization = requiredObject<TextMaterialization>(
			await MaterializeText(text as Parameters<typeof MaterializeText>[0]),
			'MaterializeText'
		);

		return {
			...materialization,
			insert: enumFromWails(materialization.insert, TextInsert, 'MaterializeText.insert'),
		};
	}
}
