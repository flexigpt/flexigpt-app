import { useCallback, useEffect, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import { type Tool, type ToolBundle, ToolImplType } from '@/spec/tool';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { toolStoreAPI } from '@/apis/baseapi';
import { getAllToolBundles, getAllTools } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { PageFrame } from '@/components/page_frame';

import { AddToolBundleModal } from '@/tools/tool_bundle_add_modal';
import { ToolBundleCard } from '@/tools/tool_bundle_card';

interface BundleData {
	bundle: ToolBundle;
	tools: Tool[];
}

const getErrorMessage = (err: unknown, fallback: string) => {
	if (err instanceof Error && err.message.trim()) {
		return err.message;
	}
	return fallback;
};

// eslint-disable-next-line no-restricted-exports
export default function ToolsPage() {
	const [bundles, setBundles] = useState<BundleData[] | undefined>(undefined);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDelete, setBundleToDelete] = useState<ToolBundle | null>(null);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);

	const fetchAll = useCallback(async (showLoader = false) => {
		if (showLoader) {
			setBundles(undefined);
		}

		try {
			const toolBundles = await getAllToolBundles(undefined, true);
			const bundleResults: BundleData[] = await Promise.all(
				toolBundles.map(async b => {
					try {
						const toolListItems = await getAllTools([b.id], undefined, true);
						const tools = toolListItems.map(itm => itm.toolDefinition);
						return { bundle: b, tools };
					} catch {
						return { bundle: b, tools: [] };
					}
				})
			);

			setBundles(bundleResults);
		} catch (err) {
			console.error('Load tool bundles failed:', err);
			setAlertMsg('Failed to load tool bundles. Please try again.');
			setShowAlert(true);
			setBundles([]);
		}
	}, []);

	const refreshBundleTools = useCallback(async (bundleID: string) => {
		const toolListItems = await getAllTools([bundleID], undefined, true);
		const freshTools = toolListItems.map(itm => itm.toolDefinition);

		setBundles(prev => (prev ?? []).map(bd => (bd.bundle.id === bundleID ? { ...bd, tools: freshTools } : bd)));
	}, []);

	useEffect(() => {
		void fetchAll();
	}, [fetchAll]);

	const handleToggleBundleEnable = useCallback(async (bundleID: string, enabled: boolean) => {
		try {
			await toolStoreAPI.patchToolBundle(bundleID, enabled);

			setBundles(prev =>
				(prev ?? []).map(bd =>
					bd.bundle.id === bundleID ? { ...bd, bundle: { ...bd.bundle, isEnabled: enabled } } : bd
				)
			);
		} catch (err) {
			console.error('Toggle bundle enable failed:', err);
			throw new Error(getErrorMessage(err, 'Failed to toggle bundle enable state.'));
		}
	}, []);

	const handleToggleToolEnable = useCallback(async (bundleID: string, tool: Tool, enabled: boolean) => {
		try {
			await toolStoreAPI.patchTool(bundleID, tool.slug, tool.version, enabled);

			setBundles(prev =>
				(prev ?? []).map(bd =>
					bd.bundle.id === bundleID
						? {
								...bd,
								tools: bd.tools.map(t => (t.id === tool.id ? { ...t, isEnabled: enabled } : t)),
							}
						: bd
				)
			);
		} catch (err) {
			console.error('Toggle tool failed:', err);
			throw new Error(getErrorMessage(err, 'Failed to toggle tool.'));
		}
	}, []);

	const handleDeleteTool = useCallback(async (bundleID: string, tool: Tool) => {
		try {
			await toolStoreAPI.deleteTool(bundleID, tool.slug, tool.version);

			setBundles(prev =>
				(prev ?? []).map(bd =>
					bd.bundle.id === bundleID
						? {
								...bd,
								tools: bd.tools.filter(t => t.id !== tool.id),
							}
						: bd
				)
			);
		} catch (err) {
			console.error('Delete tool failed:', err);
			throw new Error(getErrorMessage(err, 'Failed to delete tool.'));
		}
	}, []);

	const handleSubmitTool = useCallback(
		async (bundleID: string, partial: Partial<Tool>, toolToEdit?: Tool) => {
			const bundleData = (bundles ?? []).find(bd => bd.bundle.id === bundleID);
			if (!bundleData) {
				throw new Error('Tool bundle not found.');
			}

			const slug = (toolToEdit?.slug ?? partial.slug ?? '').trim();
			const version = (partial.version ?? '').trim();

			if (!slug) {
				throw new Error('Missing tool slug.');
			}
			if (!version) {
				throw new Error('Version is required.');
			}

			const exists = bundleData.tools.some(t => t.slug === slug && t.version === version);
			if (exists) {
				throw new Error(`Version "${version}" already exists for slug "${slug}". Create a different version.`);
			}

			try {
				if (toolToEdit) {
					await toolStoreAPI.putTool(
						bundleID,
						toolToEdit.slug,
						version,
						partial.displayName ?? toolToEdit.displayName,
						partial.isEnabled ?? toolToEdit.isEnabled,
						partial.userCallable ?? toolToEdit.userCallable,
						partial.llmCallable ?? toolToEdit.llmCallable,
						partial.autoExecReco ?? toolToEdit.autoExecReco,
						partial.argSchema ?? toolToEdit.argSchema,
						partial.type ?? toolToEdit.type,
						partial.httpImpl ?? toolToEdit.httpImpl,
						partial.description ?? toolToEdit.description,
						partial.tags ?? toolToEdit.tags
					);
				} else {
					const display = partial.displayName?.trim() ?? '';

					await toolStoreAPI.putTool(
						bundleID,
						slug,
						version,
						display,
						partial.isEnabled ?? true,
						partial.userCallable ?? true,
						partial.llmCallable ?? true,
						partial.autoExecReco ?? false,
						partial.argSchema ?? {},
						partial.type ?? ToolImplType.HTTP,
						partial.httpImpl,
						partial.description,
						partial.tags
					);
				}

				await refreshBundleTools(bundleID);
			} catch (err) {
				console.error('Save tool failed:', err);
				throw new Error(getErrorMessage(err, 'Failed to save tool.'));
			}
		},
		[bundles, refreshBundleTools]
	);

	const handleBundleDelete = async () => {
		if (!bundleToDelete) return;

		try {
			await toolStoreAPI.deleteToolBundle(bundleToDelete.id);
			setBundles(prev => (prev ?? []).filter(bd => bd.bundle.id !== bundleToDelete.id));
		} catch (err) {
			console.error('Delete tool bundle failed:', err);
			setAlertMsg('Failed to delete tool bundle.');
			setShowAlert(true);
		} finally {
			setBundleToDelete(null);
		}
	};

	const handleAddBundle = async (slug: string, display: string, description?: string) => {
		try {
			const id = getUUIDv7();
			await toolStoreAPI.putToolBundle(id, slug, display, true, description);
			setIsAddModalOpen(false);
			await fetchAll(true);
		} catch (err) {
			console.error('Add tool bundle failed:', err);
			setAlertMsg('Failed to add tool bundle.');
			setShowAlert(true);
		}
	};

	if (bundles === undefined) {
		return <Loader text="Loading tool bundles…" />;
	}

	return (
		<PageFrame>
			<div className="flex h-full w-full flex-col items-center">
				<div className="fixed mt-8 flex w-10/12 items-center p-2 lg:w-2/3">
					<h1 className="flex grow items-center justify-center text-xl font-semibold">Tool Bundles</h1>
					<button
						className="btn btn-ghost flex items-center rounded-2xl"
						onClick={() => {
							setIsAddModalOpen(true);
						}}
					>
						<FiPlus size={20} /> <span className="ml-1">Add Bundle</span>
					</button>
				</div>

				<div
					className="mt-24 flex w-full grow flex-col items-center overflow-y-auto"
					style={{ maxHeight: `calc(100vh - 128px)` }}
				>
					<div className="flex w-5/6 flex-col space-y-4 xl:w-2/3">
						{bundles.length === 0 && <p className="mt-8 text-center text-sm">No tool bundles configured yet.</p>}

						{bundles.map(bd => (
							<ToolBundleCard
								key={bd.bundle.id}
								bundle={bd.bundle}
								tools={bd.tools}
								onToggleBundleEnable={handleToggleBundleEnable}
								onToggleToolEnable={handleToggleToolEnable}
								onDeleteTool={handleDeleteTool}
								onSubmitTool={handleSubmitTool}
								onRequestDeleteBundle={b => {
									setBundleToDelete(b);
								}}
							/>
						))}
					</div>
				</div>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						setBundleToDelete(null);
					}}
					onConfirm={handleBundleDelete}
					title="Delete Tool Bundle"
					message={`Delete bundle "${bundleToDelete?.displayName ?? ''}" and all its tools?`}
					confirmButtonText="Delete"
				/>

				<AddToolBundleModal
					isOpen={isAddModalOpen}
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={bundles.map(b => b.bundle.slug)}
				/>

				<ActionDeniedAlertModal
					isOpen={showAlert}
					onClose={() => {
						setShowAlert(false);
						setAlertMsg('');
					}}
					message={alertMsg}
				/>
			</div>
		</PageFrame>
	);
}
