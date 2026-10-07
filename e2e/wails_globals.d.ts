export {};

declare global {
	interface Window {
		go?: {
			main?: {
				App?: {
					Ping(): Promise<string>;
					GetAppVersion(): Promise<string>;
					GetArtifactInitializationError(): Promise<string>;
				};
			};
		};
	}
}
