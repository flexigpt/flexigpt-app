import type { AuthKeyName, AuthKeyType } from '@/spec/setting';
import type { ApplyUnifiedDiffArgs, ApplyUnifiedDiffOut } from '@/spec/unified_diff';

import type { IAggregateAPI } from '@/apis/interface';
import type { texttool as texttoolSpec, spec as wailsSpec } from '@/apis/wailsjs/go/models';
import { requireWailsBody } from '@/apis/wailsapi/transport';
import { ApplyUnifiedDiff, DeleteAuthKey, SetAuthKey } from '@/apis/wailsjs/go/main/AggregrateWrapper';

export class WailsAggregateAPI implements IAggregateAPI {
	async applyUnifiedDiff(args: ApplyUnifiedDiffArgs): Promise<ApplyUnifiedDiffOut> {
		const resp = await ApplyUnifiedDiff(args as texttoolSpec.ApplyUnifiedDiffArgs);
		return requireWailsBody(resp as ApplyUnifiedDiffOut | null | undefined, 'ApplyUnifiedDiff');
	}

	async deleteAuthKey(type: AuthKeyType, keyName: AuthKeyName): Promise<void> {
		const r = {
			Type: type,
			KeyName: keyName,
		};
		await DeleteAuthKey(r as wailsSpec.DeleteAuthKeyRequest);
	}

	async setAuthKey(type: AuthKeyType, keyName: AuthKeyName, secret: string): Promise<void> {
		const r = {
			Type: type,
			KeyName: keyName,
			Body: {
				secret: secret,
			},
		};
		await SetAuthKey(r as wailsSpec.SetAuthKeyRequest);
	}
}
