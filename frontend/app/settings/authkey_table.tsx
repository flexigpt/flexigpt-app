import { useState } from 'react';

import { FiCheckCircle, FiDelete, FiEdit2, FiTrash2, FiXCircle } from 'react-icons/fi';

import type { AuthKeyMeta } from '@/spec/setting';

import { isBuiltInProviderAuthKeyName, useBuiltInsReady } from '@/hooks/use_builtin_provider';
import { usePendingActions } from '@/hooks/use_pending_actions';

import { aggregateAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

interface AuthKeyTableProps {
	authKeys: AuthKeyMeta[];
	onEdit: (meta: AuthKeyMeta) => void;
	onChanged: () => void; // parent refetch
}

const actionKey = (action: string, meta: AuthKeyMeta) => JSON.stringify([action, meta.type, meta.keyName]);

export function AuthKeyTable({ authKeys, onEdit, onChanged }: AuthKeyTableProps) {
	const builtInsReady = useBuiltInsReady();
	const [deleteTarget, setDeleteTarget] = useState<AuthKeyMeta | null>(null);
	const [resetTarget, setResetTarget] = useState<AuthKeyMeta | null>(null);
	const [alertMsg, setAlertMsg] = useState('');
	const { isPending, runAction } = usePendingActions();

	if (!builtInsReady) {
		return <span className="loading loading-dots loading-sm" />;
	}

	const confirmReset = async () => {
		if (!resetTarget) {
			return;
		}

		const target = resetTarget;
		try {
			await runAction(actionKey('reset', target), async () => {
				await aggregateAPI.setAuthKey(target.type, target.keyName, '');
				onChanged();
			});
			setResetTarget(null);
		} catch (error) {
			setAlertMsg(error instanceof Error && error.message.trim() ? error.message : 'Failed to reset auth key.');
		}
	};

	const requestDelete = (meta: AuthKeyMeta) => {
		if (isBuiltInProviderAuthKeyName(meta.type, meta.keyName)) {
			setAlertMsg('In-built keys cannot be deleted. You can only reset them.');
		} else {
			setDeleteTarget(meta);
		}
	};

	const confirmDelete = async () => {
		if (!deleteTarget) {
			return;
		}

		const target = deleteTarget;
		try {
			await runAction(actionKey('delete', target), async () => {
				await aggregateAPI.deleteAuthKey(target.type, target.keyName);
				onChanged();
			});
			setDeleteTarget(null);
		} catch (error) {
			setAlertMsg(error instanceof Error && error.message.trim() ? error.message : 'Failed to delete auth key.');
		}
	};

	if (authKeys.length === 0) {
		return (
			<div className="border-base-300 bg-base-200/60 my-6 rounded-2xl border p-4 text-sm">
				<div className="font-semibold">No provider keys configured yet.</div>
				<p className="text-base-content/70 mt-1">
					Add a provider key before sending requests. Secrets are stored through the OS keyring, while FlexiGPT only
					shows local key metadata here.
				</p>
			</div>
		);
	}
	return (
		<>
			<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
				<table className="table-zebra table w-full">
					<thead className="bg-base-300 text-sm font-semibold">
						<tr className="text-sm">
							<th className="min-w-0">Type</th>
							<th className="max-w-48 text-center">Key Name</th>
							<th className="max-w-48 text-center">SHA-256</th>
							<th className="text-center">Secret</th>
							<th className="text-center">Actions</th>
						</tr>
					</thead>

					<tbody>
						{authKeys.map(meta => {
							const inbuilt = isBuiltInProviderAuthKeyName(meta.type, meta.keyName);

							return (
								<tr key={`${meta.type}:${meta.keyName}`} className="hover:bg-base-300 border-none shadow-none">
									<td className="capitalize">{meta.type}</td>

									<td className="max-w-32 text-center align-middle font-mono text-sm">
										<span className="block w-full truncate">{meta.keyName} </span>
									</td>

									<td className="max-w-32 text-center align-middle font-mono text-sm">
										<span className="block w-full truncate">{meta.nonEmpty ? meta.sha256 : '--'} </span>
									</td>

									<td className="text-center align-middle">
										{meta.nonEmpty ? (
											<FiCheckCircle className="text-success inline" />
										) : (
											<FiXCircle className="text-error inline" />
										)}
									</td>

									<td className="flex items-center justify-center gap-3 text-center">
										<button
											type="button"
											className="btn btn-xs btn-ghost rounded-2xl"
											onClick={() => {
												onEdit(meta);
											}}
											title="Edit"
											aria-label={`Edit ${meta.keyName}`}
											disabled={isPending(actionKey('reset', meta)) || isPending(actionKey('delete', meta))}
										>
											<FiEdit2 size={16} />
										</button>

										<button
											type="button"
											className="btn btn-xs btn-ghost rounded-2xl"
											onClick={() => {
												setResetTarget(meta);
											}}
											title="Reset Secret"
											aria-label={`Reset secret for ${meta.keyName}`}
											disabled={isPending(actionKey('reset', meta)) || isPending(actionKey('delete', meta))}
										>
											<FiDelete size={16} />
										</button>

										<button
											type="button"
											className="btn btn-xs btn-ghost rounded-2xl"
											onClick={() => {
												requestDelete(meta);
											}}
											title={inbuilt ? 'Cannot delete in-built key' : 'Delete'}
											aria-label={`Delete ${meta.keyName}`}
											disabled={inbuilt || isPending(actionKey('reset', meta)) || isPending(actionKey('delete', meta))}
										>
											<FiTrash2 size={16} />
										</button>
									</td>
								</tr>
							);
						})}
					</tbody>
				</table>
			</div>

			<DeleteConfirmationModal
				isOpen={resetTarget !== null}
				title="Reset Auth Key"
				message={`Clear the stored secret for "${resetTarget?.keyName}"? The key metadata will remain, but requests using it will stop working until a replacement is added.`}
				confirmButtonText="Reset Secret"
				onConfirm={confirmReset}
				onClose={() => {
					if (!resetTarget || !isPending(actionKey('reset', resetTarget))) {
						setResetTarget(null);
					}
				}}
			/>

			<DeleteConfirmationModal
				isOpen={!!deleteTarget}
				title="Delete Auth Key"
				message={`Delete key "${deleteTarget?.keyName}" of type "${deleteTarget?.type}"? This cannot be undone.`}
				confirmButtonText="Delete"
				onConfirm={confirmDelete}
				onClose={() => {
					if (!deleteTarget || !isPending(actionKey('delete', deleteTarget))) {
						setDeleteTarget(null);
					}
				}}
			/>

			<ActionDeniedAlertModal
				isOpen={!!alertMsg}
				message={alertMsg}
				onClose={() => {
					setAlertMsg('');
				}}
			/>
		</>
	);
}
