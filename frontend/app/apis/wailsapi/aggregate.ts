import type { AuthKeyName, AuthKeyType } from '@/spec/setting';

import type { IAggregateAPI } from '@/apis/interface';
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';
import { DeleteAuthKey, SetAuthKey } from '@/apis/wailsjs/go/main/AggregrateWrapper';

export class WailsAggregateAPI implements IAggregateAPI {
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
