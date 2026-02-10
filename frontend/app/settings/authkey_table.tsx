import { useState } from 'react';

import { FiCheckCircle, FiDelete, FiEdit2, FiTrash2, FiXCircle } from 'react-icons/fi';

import type { AuthKeyMeta } from '@/spec/setting';

import { isBuiltInProviderAuthKeyName, useBuiltInsReady } from '@/hooks/use_builtin_provider';

import { settingstoreAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

interface AuthKeyTableProps {
	authKeys: AuthKeyMeta[];
	onEdit: (meta: AuthKeyMeta) => void;
	onChanged: () => void; // parent refetch
}

export function AuthKeyTable({ authKeys, onEdit, onChanged }: AuthKeyTableProps) {
	const builtInsReady = useBuiltInsReady();
	const [deleteTarget, setDeleteTarget] = useState<AuthKeyMeta | null>(null);
	const [alertMsg, setAlertMsg] = useState<string>('');

	if (!builtInsReady) {
		return <span className="loading loading-dots loading-sm" />;
	}

	const resetKey = async (meta: AuthKeyMeta) => {
		await settingstoreAPI.setAuthKey(meta.type, meta.keyName, '');
		onChanged();
	};

	const requestDelete = (meta: AuthKeyMeta) => {
		if (isBuiltInProviderAuthKeyName(meta.type, meta.keyName)) {
			setAlertMsg('In-built keys cannot be deleted. You can only reset them.');
		} else {
			setDeleteTarget(meta);
		}
	};

	const confirmDelete = async () => {
		if (!deleteTarget) return;
		await settingstoreAPI.deleteAuthKey(deleteTarget.type, deleteTarget.keyName);
		setDeleteTarget(null);
		onChanged();
	};

	if (!authKeys.length) return <p className="text-neutral-custom my-6 text-center text-sm">No keys defined.</p>;

	return (
		<>
			<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
				<table className="table-zebra table w-full">
					<thead className="bg-base-300 text-sm font-semibold">
						<tr className="text-sm">
							<th className="min-w-0">Type</th>
							<th className="w-full text-center">Key Name</th>
							<th className="max-w-64 text-center">SHA-256</th>
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

									<td className="text-center">{meta.keyName}</td>

									<td className="max-w-64 text-center align-middle font-mono text-xs">
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
											className="btn btn-xs btn-ghost rounded-2xl"
											onClick={() => {
												onEdit(meta);
											}}
											title="Edit"
										>
											<FiEdit2 size={16} />
										</button>

										<button
											className="btn btn-xs btn-ghost rounded-2xl"
											onClick={() => resetKey(meta)}
											title="Reset Secret"
										>
											<FiDelete size={16} />
										</button>

										<button
											className="btn btn-xs btn-ghost rounded-2xl"
											onClick={() => {
												requestDelete(meta);
											}}
											title={inbuilt ? 'Cannot delete in-built key' : 'Delete'}
											disabled={inbuilt}
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
				isOpen={!!deleteTarget}
				title="Delete Auth Key"
				message={`Delete key "${deleteTarget?.keyName}" of type "${deleteTarget?.type}"? This cannot be undone.`}
				confirmButtonText="Delete"
				onConfirm={confirmDelete}
				onClose={() => {
					setDeleteTarget(null);
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
