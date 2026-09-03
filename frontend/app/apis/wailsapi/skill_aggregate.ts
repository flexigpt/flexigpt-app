import type { SkillBundleRef, SkillRuntimeCatalogID } from '@/spec/skill';

import type { ISkillAggregateAPI } from '@/apis/interface';
import { requireNonBlankString } from '@/apis/wailsapi/transport';
import { RuntimeCatalogIDForCollection } from '@/apis/wailsjs/go/main/SkillAggregateWrapper';

export class WailsSkillAggregateAPI implements ISkillAggregateAPI {
	async runtimeCatalogIDForCollection(bundle: SkillBundleRef): Promise<SkillRuntimeCatalogID> {
		const catalogID = await RuntimeCatalogIDForCollection(
			bundle as Parameters<typeof RuntimeCatalogIDForCollection>[0]
		);

		return requireNonBlankString(catalogID, 'RuntimeCatalogIDForCollection');
	}
}
