import { FiAlertTriangle, FiArrowRight, FiCheckCircle, FiSettings } from 'react-icons/fi';

import { Link } from 'react-router';

export type ProviderSetupState = 'ready' | 'needs-provider' | 'needs-credentials' | 'needs-model' | 'error';

export function ProviderSetupStatus({
	providersLoaded,
	state,
	providerSummary,
}: {
	providersLoaded: boolean;
	state: ProviderSetupState;
	providerSummary: string;
}) {
	if (!providersLoaded) {
		return null;
	}

	if (state === 'ready') {
		return (
			<div className="text-base-content/60 mt-3 flex flex-wrap items-center justify-center gap-2 text-xs">
				<span className="text-success inline-flex items-center gap-1">
					<FiCheckCircle size={13} />
					Provider ready:
				</span>
				<span>{providerSummary}</span>
				<span className="opacity-50">·</span>
				<Link to="/models/" className="link-hover inline-flex items-center gap-1">
					Manage providers &rarr;
				</Link>
			</div>
		);
	}

	const config =
		state === 'needs-provider'
			? {
					title: 'Add a provider to start',
					description: 'Configure a provider and model before opening Chats.',
					to: '/models/',
				}
			: state === 'needs-model'
				? {
						title: 'Add an enabled model to start',
						description: 'A provider is ready, but no enabled model can send requests yet.',
						to: '/models/',
					}
				: state === 'needs-credentials'
					? {
							title: 'No provider API key configured',
							description: 'Configure an enabled provider API key in Models to start chatting.',
							to: '/models/',
						}
					: {
							title: 'Provider configuration unavailable',
							description: 'Open Models to review provider and model configuration.',
							to: '/models/',
						};

	return (
		<Link
			to={config.to}
			aria-label={config.title}
			className="group border-warning/40 bg-warning/10 hover:border-warning/70 mt-4 block w-full max-w-lg rounded-2xl border text-left shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
		>
			<div className="flex items-center gap-3 p-4">
				<div className="flex shrink-0 items-center justify-center">
					<div className="bg-warning/20 text-warning-content rounded-xl p-2">
						<FiAlertTriangle size={18} />
					</div>
				</div>

				<div className="flex min-w-0 flex-1 flex-col">
					<div className="flex items-center justify-between gap-3">
						<span className="text-sm font-semibold">{config.title}</span>
						<FiArrowRight size={18} className="shrink-0 transition-transform group-hover:translate-x-1" />
					</div>

					<p className="text-base-content/70 mt-1 text-xs/relaxed">{config.description}</p>
					{state === 'needs-credentials' ? (
						<span className="text-base-content/60 mt-2 inline-flex items-center gap-1 text-xs">
							<FiSettings size={12} />
							Models → Provider API Keys
						</span>
					) : null}
				</div>
			</div>
		</Link>
	);
}
