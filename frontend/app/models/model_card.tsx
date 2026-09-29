import { FiCheck, FiEdit2, FiEye, FiTrash2 } from 'react-icons/fi';

import { ArtifactState } from '@/spec/artifact';

import type { ModelManagementItem, ModelProviderManagementItem } from '@/apis/model_management';

interface ModelCardProps {
	model: ModelManagementItem;
	provider: ModelProviderManagementItem;
	busy: boolean;
	isProviderDefault: boolean;
	onView: (model: ModelManagementItem) => void;
	onEdit: (model: ModelManagementItem) => void;
	onToggle: (model: ModelManagementItem) => void;
	onDelete: (model: ModelManagementItem) => void;
	onSetDefault: (model: ModelManagementItem) => void;
}

export function ModelCard({
	model,
	provider,
	busy,
	isProviderDefault,
	onView,
	onEdit,
	onToggle,
	onDelete,
	onSetDefault,
}: ModelCardProps) {
	const mutable = !model.list.builtIn;
	const available = model.list.state === ArtifactState.Available;

	return (
		<div className="bg-base-200 flex flex-wrap items-center justify-between gap-3 rounded-xl p-3">
			<div className="min-w-0">
				<div className="truncate font-medium">{model.list.displayName}</div>
				<div className="truncate font-mono text-xs opacity-70">{model.view.document.providerModelID}</div>
			</div>

			<div className="flex flex-wrap items-center gap-2">
				<span className={`badge ${model.list.enabled ? 'badge-success' : ''}`}>
					{model.list.enabled ? 'enabled' : 'disabled'}
				</span>

				{isProviderDefault ? <span className="badge badge-info">provider default</span> : null}

				<button
					type="button"
					className="btn btn-xs"
					disabled={busy}
					onClick={() => {
						onView(model);
					}}
				>
					<FiEye size={14} />
					View
				</button>

				<button
					type="button"
					className="btn btn-xs"
					disabled={busy || !available}
					onClick={() => {
						onToggle(model);
					}}
				>
					{model.list.enabled ? 'Disable' : 'Enable'}
				</button>

				{mutable ? (
					<button
						type="button"
						className="btn btn-xs"
						disabled={busy}
						onClick={() => {
							onEdit(model);
						}}
					>
						<FiEdit2 size={14} />
						Edit
					</button>
				) : null}

				{!provider.list.builtIn && !isProviderDefault ? (
					<button
						type="button"
						className="btn btn-xs"
						disabled={busy || !model.list.enabled}
						onClick={() => {
							onSetDefault(model);
						}}
					>
						<FiCheck size={14} />
						Default
					</button>
				) : null}

				{mutable ? (
					<button
						type="button"
						className="btn btn-xs btn-error"
						disabled={busy}
						onClick={() => {
							onDelete(model);
						}}
					>
						<FiTrash2 size={14} />
						Delete
					</button>
				) : null}
			</div>
		</div>
	);
}
