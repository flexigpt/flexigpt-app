import { useEffect, useMemo, useState } from 'react';
import { FiArrowRight, FiBookOpen, FiHome } from 'react-icons/fi';

import { Link } from 'react-router';

import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import type { AgentCatalogOption } from '@/apis/agent_management';
import { agentManagementAPI, settingstoreAPI } from '@/apis/baseapi';
import { getAllProviderPresetsMap } from '@/apis/list_helper';

import { PageFrame } from '@/components/page_frame';

import { DocsCard } from '@/home/docs_card';
import { docsStarters } from '@/home/docs_starter';
import {
	formatConfiguredProviderSummary,
	getConfiguredProviderNames,
	pickDefaultProviderName,
} from '@/home/home_utils';
import { PrimaryActionCard } from '@/home/nav_card';
import { HomeAuthKeyModalIntro, ProviderSetupStatus } from '@/home/provider_setup_status_card';
import { workflowStarters } from '@/home/workflow_starter';
import { WorkflowStarterCard } from '@/home/workflow_starter_card';
import { AddEditAuthKeyModal } from '@/settings/authkey_add_edit_modal';

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

	const [authKeys, setAuthKeys] = useState<AuthKeyMeta[]>([]);
	const [settingsLoadRequestId, setSettingsLoadRequestId] = useState(0);
	const [settingsLoadedRequestId, setSettingsLoadedRequestId] = useState<number | null>(null);
	const [providerPresets, setProviderPresets] = useState<Record<ProviderName, ProviderPreset>>({});
	const [apiKeyModalOpen, setApiKeyModalOpen] = useState(false);
	const [agentOptions, setAgentOptions] = useState<AgentCatalogOption[]>([]);
	const [agentsLoading, setAgentsLoading] = useState(true);

	const settingsLoaded = settingsLoadedRequestId === settingsLoadRequestId;

	useEffect(() => {
		let cancelled = false;
		const requestId = settingsLoadRequestId;

		void (async () => {
			try {
				const settings = await settingstoreAPI.getSettings();
				if (!cancelled) {
					setAuthKeys(settings.authKeys);
				}
			} catch (err) {
				console.error('Failed to load home auth-key setup state', err);
			} finally {
				if (!cancelled) {
					setSettingsLoadedRequestId(requestId);
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [settingsLoadRequestId]);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			try {
				const providers = await getAllProviderPresetsMap(true);
				if (!cancelled) {
					setProviderPresets(providers);
				}
			} catch (err) {
				console.error('Failed to load provider presets for home setup', err);
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

	const configuredProviderNames = useMemo(() => getConfiguredProviderNames(authKeys), [authKeys]);
	const hasUsableProviderKey = configuredProviderNames.length > 0;
	const providerSummary = useMemo(
		() => formatConfiguredProviderSummary(configuredProviderNames, providerPresets),
		[configuredProviderNames, providerPresets]
	);
	const defaultProviderName = useMemo(() => pickDefaultProviderName(providerPresets), [providerPresets]);

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
							settingsLoaded={settingsLoaded}
							hasUsableProviderKey={hasUsableProviderKey}
							providerSummary={providerSummary}
							onAddKey={() => {
								setApiKeyModalOpen(true);
							}}
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

			<AddEditAuthKeyModal
				isOpen={apiKeyModalOpen}
				initial={null}
				existing={authKeys}
				providerOnly={true}
				defaultKeyName={defaultProviderName}
				intro={
					<HomeAuthKeyModalIntro
						onNavigateAway={() => {
							setApiKeyModalOpen(false);
						}}
					/>
				}
				onClose={() => {
					setApiKeyModalOpen(false);
				}}
				onChanged={() => {
					setSettingsLoadRequestId(requestId => requestId + 1);
				}}
			/>
		</PageFrame>
	);
}
