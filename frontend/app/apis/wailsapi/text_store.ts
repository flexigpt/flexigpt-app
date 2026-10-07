import type { ArtifactRef } from '@/spec/artifact';
import type { TextMaterialization } from '@/spec/text';

import type { ITextStoreAPI } from '@/apis/interface';
import { requiredObject } from '@/apis/wailsapi/transport';
import { MaterializeText } from '@/apis/wailsjs/go/main/TextStoreWrapper';

export class WailsTextStoreAPI implements ITextStoreAPI {
	async materializeText(text: ArtifactRef): Promise<TextMaterialization> {
		return requiredObject<TextMaterialization>(
			await MaterializeText(text as Parameters<typeof MaterializeText>[0]),
			'MaterializeText'
		);
	}
}
