import { useState } from 'react';
import { FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiKey, FiPlus, FiTrash2 } from 'react-icons/fi';

import { ArtifactState } from '@/spec/artifact';

import type { ModelManagementItem, ModelProviderManagementItem } from '@/apis/model_management';

import { ModelCard } from '@/models/model_card';

interface ModelProviderCardProps {
	provider: ModelProviderManagementItem;
	models: ModelManagementItem[];
	busy: boolean;
	onViewProvider: (provider: ModelProviderManagementItem) => void;
	onEditProvider: (provider: ModelProviderManagementItem) => void;
	onToggleProvider: (provider: ModelProviderManagementItem) => void;
	onDeleteProvider: (provider: ModelProviderManagementItem) => void;
	onCredential: (provider: ModelProviderManagementItem) => void;
	onAddModel: (provider: ModelProviderManagementItem) => void;
	onViewModel: (model: ModelManagementItem) => void;
	onEditModel: (model: ModelManagementItem) => void;
	onToggleModel: (model: ModelManagementItem) => void;
	onDeleteModel: (model: ModelManagementItem) => void;
	onSetDefaultModel: (provider: ModelProviderManagementItem, model: ModelManagementItem) => void;
}

function providerDefaultName(provider: ModelProviderManagementItem): string | undefined {
	return provider.view.document.defaultModel?.name;
}

export function ModelProviderCard({
	provider,
	models,
	busy,
	onViewProvider,
	onEditProvider,
	onToggleProvider,
	onDeleteProvider,
	onCredential,
	onAddModel,
	onViewModel,
	onEditModel,
	onToggleModel,
	onDeleteModel,
	onSetDefaultModel,
}: ModelProviderCardProps) {
	const [expanded, setExpanded] = useState(false);
	const mutable = !provider.list.builtIn;
	const available = provider.list.state === ArtifactState.Available;
	const defaultModelName = providerDefaultName(provider);

	return (
		<section className="bg-base-100 rounded-2xl p-4 shadow-sm">
			<div className="flex flex-wrap items-center justify-between gap-3">
				<div className="min-w-0">
					<h2 className="truncate text-lg font-semibold">{provider.list.displayName}</h2>
					<p className="truncate font-mono text-xs opacity-70">{provider.view.document.name}</p>
				</div>

				<div className="flex flex-wrap items-center gap-2">
					<span className={`badge ${provider.list.enabled ? 'badge-success' : ''}`}>
						{provider.list.enabled ? 'enabled' : 'disabled'}
					</span>

					<span className="badge">{provider.list.builtIn ? 'built-in' : 'custom'}</span>

					<span className={`badge ${provider.list.credentialConfigured ? 'badge-success' : 'badge-warning'}`}>
						{provider.list.credentialConfigured ? 'credential configured' : 'credential missing'}
					</span>

					<button
						type="button"
						className="btn btn-xs"
						disabled={busy}
						onClick={() => {
							onToggleProvider(provider);
						}}
					>
						{provider.list.enabled ? 'Disable' : 'Enable'}
					</button>

					<button
						type="button"
						className="btn btn-xs"
						disabled={busy}
						onClick={() => {
							onCredential(provider);
						}}
					>
						<FiKey size={14} />
						Credential
					</button>

					<button
						type="button"
						className="btn btn-xs"
						disabled={busy}
						onClick={() => {
							if (mutable) {
								onEditProvider(provider);
								return;
							}
							onViewProvider(provider);
						}}
					>
						{mutable ? <FiEdit2 size={14} /> : <FiEye size={14} />}
						{mutable ? 'Edit' : 'View'}
					</button>

					<button
						type="button"
						className="btn btn-xs"
						disabled={busy || !available}
						onClick={() => {
							onAddModel(provider);
						}}
					>
						<FiPlus size={14} />
						Add Model
					</button>

					{mutable ? (
						<button
							type="button"
							className="btn btn-xs btn-error"
							disabled={busy}
							onClick={() => {
								onDeleteProvider(provider);
							}}
						>
							<FiTrash2 size={14} />
							Delete
						</button>
					) : null}

					<button
						type="button"
						className="btn btn-xs btn-ghost"
						aria-expanded={expanded}
						onClick={() => {
							setExpanded(value => !value);
						}}
					>
						{models.length} models
						{expanded ? <FiChevronUp size={14} /> : <FiChevronDown size={14} />}
					</button>
				</div>
			</div>

			{expanded ? (
				<div className="mt-4 space-y-3">
					<div className="grid gap-2 text-sm md:grid-cols-2">
						<div>
							<span className="font-medium">Adapter: </span>
							<span className="font-mono">{provider.view.document.adapter}</span>
						</div>
						<div>
							<span className="font-medium">Default model: </span>
							<span className="font-mono">{defaultModelName ?? 'none'}</span>
						</div>
					</div>

					{models.map(model => (
						<ModelCard
							key={`${model.list.ref.rootID}:${model.list.ref.artifactID}`}
							model={model}
							provider={provider}
							busy={busy}
							isProviderDefault={model.view.document.name === defaultModelName}
							onView={onViewModel}
							onEdit={onEditModel}
							onToggle={onToggleModel}
							onDelete={onDeleteModel}
							onSetDefault={m => {
								onSetDefaultModel(provider, m);
							}}
						/>
					))}

					{models.length === 0 ? <p className="text-sm opacity-70">No models are linked to this provider.</p> : null}
				</div>
			) : null}
		</section>
	);
}
