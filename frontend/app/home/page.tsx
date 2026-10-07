import { useEffect, useMemo, useState } from 'react';
import { FiArrowRight, FiBookOpen, FiHome } from 'react-icons/fi';

import { Link } from 'react-router';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import type { AgentCatalogOption } from '@/apis/agent_management';
import type { ModelManagementSnapshot } from '@/apis/model_management';
import { agentManagementAPI, modelManagementAPI } from '@/apis/baseapi';
import { isModelProviderRunnable, isModelRunnable } from '@/apis/model_management';

import { PageFrame } from '@/components/page_frame';

import type { ProviderSetupState } from '@/home/provider_setup_status_card';
import { DocsCard } from '@/home/docs_card';
import { docsStarters } from '@/home/docs_starter';
import { PrimaryActionCard } from '@/home/nav_card';
import { ProviderSetupStatus } from '@/home/provider_setup_status_card';
import { workflowStarters } from '@/home/workflow_starter';
import { WorkflowStarterCard } from '@/home/workflow_starter_card';

function artifactRefsEqual(
	left: { rootID: string; artifactID: string } | undefined,
	right: { rootID: string; artifactID: string } | undefined
): boolean {
	return left?.rootID === right?.rootID && left?.artifactID === right?.artifactID;
}

// oxlint-disable-next-line no-restricted-exports
export default function HomePage() {
	useTitleBarContent(
		{
			center: (
				<div className="mx-auto flex items-center justify-center opacity-60">
					<FiHome size={16} />
				</div>
			),
		},
		[]
	);

	const [modelSnapshot, setModelSnapshot] = useState<ModelManagementSnapshot | null>(null);
	const [providersLoaded, setProvidersLoaded] = useState(false);
	const [providersLoadFailed, setProvidersLoadFailed] = useState(false);
	const [agentOptions, setAgentOptions] = useState<AgentCatalogOption[]>([]);
	const [agentsLoading, setAgentsLoading] = useState(true);

	useEffect(() => {
		let cancelled = false;
		// oxlint-disable-next-line react/set-state-in-effect react-you-might-not-need-an-effect/no-initialize-state
		setProvidersLoaded(false);
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-initialize-state
		setProvidersLoadFailed(false);

		void (async () => {
			try {
				const snapshot = await modelManagementAPI.loadSnapshot();
				if (!cancelled) {
					setModelSnapshot(snapshot);
				}
			} catch (err) {
				console.error('Failed to load model providers for home setup', err);
				if (!cancelled) {
					setProvidersLoadFailed(true);
				}
			} finally {
				if (!cancelled) {
					setProvidersLoaded(true);
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	}, []);

	useEffect(() => {
		let cancelled = false;

		void agentManagementAPI
			.listAgentCatalogOptions()
			.then(options => {
				if (!cancelled) {
					setAgentOptions(options);
				}
			})
			.catch((error: unknown) => {
				console.error('Failed to load home Agent starters', error);
			})
			.finally(() => {
				if (!cancelled) {
					setAgentsLoading(false);
				}
			});

		return () => {
			cancelled = true;
		};
	}, []);

	const providerSetup = useMemo(() => {
		if (!providersLoaded) {
			return {
				state: 'error' as ProviderSetupState,
				providerSummary: '',
			};
		}

		if (providersLoadFailed || !modelSnapshot) {
			return {
				state: 'error' as ProviderSetupState,
				providerSummary: '',
			};
		}

		const runnableProviders = modelSnapshot.providers.filter(isModelProviderRunnable);
		if (runnableProviders.length === 0) {
			return {
				state:
					modelSnapshot.providers.length === 0
						? ('needs-provider' as ProviderSetupState)
						: ('needs-credentials' as ProviderSetupState),
				providerSummary: '',
			};
		}

		const runnableModels = modelSnapshot.models.filter(isModelRunnable);
		if (runnableModels.length === 0) {
			return {
				state: 'needs-model' as ProviderSetupState,
				providerSummary: '',
			};
		}

		const preferredModel = modelSnapshot.defaultProvider
			? runnableModels.find(model => artifactRefsEqual(model.provider?.list.ref, modelSnapshot.defaultProvider))
			: undefined;
		const selectedProvider = preferredModel?.provider ?? runnableModels[0]?.provider;

		return {
			state: 'ready' as ProviderSetupState,
			providerSummary: selectedProvider?.list.displayName || 'Provider',
		};
	}, [modelSnapshot, providersLoadFailed, providersLoaded]);

	const resolvedWorkflowStarters = useMemo(
		() =>
			workflowStarters.map(workflow => ({
				workflow,
				agent:
					agentOptions.find(option => option.agent.builtIn && option.agent.name === workflow.agentName) ?? undefined,
			})),
		[agentOptions]
	);

	return (
		<PageFrame>
			<div className="mx-auto flex size-full max-w-6xl flex-col items-center px-4 py-8">
				<div className="mt-4 flex w-full flex-1 flex-col items-center justify-between pb-4 xl:mt-16">
					<div className="flex w-full flex-col items-center">
						<PrimaryActionCard
							title="Open Chat Workspace"
							description="Start a new chat/workflow or continue a saved local thread."
							to="/chats/"
							icon={<img src="/icon.png" alt="FlexiGPT" width={64} height={64} />}
						/>
						<ProviderSetupStatus
							providersLoaded={providersLoaded}
							state={providerSetup.state}
							providerSummary={providerSetup.providerSummary}
						/>
					</div>

					<section className="mt-8 w-full">
						<div className="mx-auto max-w-3xl text-center">
							<h2 className="text-lg font-semibold">Start with a proven workflow</h2>
							<p className="text-base-content/70 text-xs">
								Each starter loads the matching built-in Agent. Its own opening text and setup remain editable after
								Chats opens.
							</p>
						</div>

						<div className="mx-auto mt-4 grid w-full max-w-5xl grid-cols-1 gap-4 sm:grid-cols-3">
							{resolvedWorkflowStarters.map(({ workflow, agent }) => (
								<WorkflowStarterCard
									key={workflow.workflowID}
									workflow={workflow}
									agent={agent}
									loading={agentsLoading}
								/>
							))}
						</div>
					</section>

					<section className="mt-8 w-full">
						<div className="mx-auto max-w-3xl text-center">
							<Link
								to="/docs/"
								className="inline-flex items-center gap-2 text-lg font-semibold transition-opacity hover:opacity-80"
							>
								<FiBookOpen size={24} />
								Documentation
								<div className="flex justify-end">
									<FiArrowRight size={24} className="transition-transform group-hover:translate-x-1" />
								</div>
							</Link>
							<p className="text-base-content/70 text-xs">
								Bundled guide for getting started, Chats, Agents, Workspaces, reusable context, providers, privacy, and
								everyday tasks.
							</p>
						</div>

						<div className="mx-auto mt-4 grid w-full max-w-5xl grid-cols-1 gap-6 md:grid-cols-3">
							{docsStarters.map(card => (
								<DocsCard key={card.to} title={card.title} description={card.description} to={card.to} />
							))}
						</div>
					</section>
				</div>
			</div>
		</PageFrame>
	);
}
