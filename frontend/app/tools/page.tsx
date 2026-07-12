import { useCallback, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { Tool, ToolBundle } from '@/spec/tool';
import { ToolImplType } from '@/spec/tool';

import { mapWithConcurrency, throwIfAborted } from '@/lib/async_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { toolStoreAPI } from '@/apis/baseapi';
import { getAllToolBundles, getAllTools } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementBundleCreateModal } from '@/components/managementui/management_bundle_create_modal';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { ToolBundleCard } from '@/tools/tool_bundle_card';

interface BundleData {
	bundle: ToolBundle;
	tools: Tool[];
	toolLoadError?: string;
}

const getErrorMessage = (err: unknown, fallback: string) => {
	if (err instanceof Error && err.message.trim()) {
		return err.message;
	}
	return fallback;
};

async function loadToolBundleData(signal: AbortSignal): Promise<BundleData[]> {
	const toolBundles = await getAllToolBundles(undefined, true);
	throwIfAborted(signal);

	return mapWithConcurrency(
		toolBundles,
		4,
		async bundle => {
			try {
				const toolListItems = await getAllTools([bundle.id], undefined, true);
				throwIfAborted(signal);
				return {
					bundle,
					tools: toolListItems.map(item => item.toolDefinition),
				};
			} catch (error) {
				throwIfAborted(signal);
				return {
					bundle,
					tools: [],
					toolLoadError: getErrorMessage(error, 'Failed to load tools for this bundle.'),
				};
			}
		},
		signal
	);
}

// oxlint-disable-next-line no-restricted-exports
export default function ToolsPage() {
	const loadPageData = useCallback((signal: AbortSignal) => loadToolBundleData(signal), []);
	const {
		data: bundles,
		error: pageLoadError,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setBundles,
	} = useAsyncResource(loadPageData, { initialData: [] as BundleData[] });

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDelete, setBundleToDelete] = useState<ToolBundle | null>(null);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);

	const refreshBundleTools = useCallback(
		async (bundleID: string) => {
			try {
				const toolListItems = await getAllTools([bundleID], undefined, true);
				const freshTools = toolListItems.map(itm => itm.toolDefinition);

				setBundles(prev =>
					(prev ?? []).map(bd =>
						bd.bundle.id === bundleID
							? Object.assign({}, bd, {
									tools: freshTools,
									toolLoadError: undefined,
								})
							: bd
					)
				);
			} catch (error) {
				setBundles(prev =>
					(prev ?? []).map(bd =>
						bd.bundle.id === bundleID
							? Object.assign({}, bd, {
									toolLoadError: getErrorMessage(error, 'Failed to load tools for this bundle.'),
								})
							: bd
					)
				);

				throw error;
			}
		},
		[setBundles]
	);

	const handleToggleBundleEnable = useCallback(
		async (bundleID: string, enabled: boolean) => {
			try {
				await toolStoreAPI.patchToolBundle(bundleID, enabled);

				setBundles(prev =>
					(prev ?? []).map(bd =>
						bd.bundle.id === bundleID
							? Object.assign({}, bd, { bundle: Object.assign({}, bd.bundle, { isEnabled: enabled }) })
							: bd
					)
				);
			} catch (err) {
				console.error('Toggle bundle enable failed:', err);
				throw new Error(getErrorMessage(err, 'Failed to toggle bundle enable state.'), { cause: err });
			}
		},
		[setBundles]
	);

	const handleToggleToolEnable = useCallback(
		async (bundleID: string, tool: Tool, enabled: boolean) => {
			try {
				await toolStoreAPI.patchTool(bundleID, tool.slug, tool.version, enabled);

				setBundles(prev =>
					(prev ?? []).map(bd =>
						bd.bundle.id === bundleID
							? Object.assign({}, bd, {
									tools: bd.tools.map(t => (t.id === tool.id ? Object.assign({}, t, { isEnabled: enabled }) : t)),
								})
							: bd
					)
				);
			} catch (err) {
				console.error('Toggle tool failed:', err);
				throw new Error(getErrorMessage(err, 'Failed to toggle tool.'), { cause: err });
			}
		},
		[setBundles]
	);

	const handleDeleteTool = useCallback(
		async (bundleID: string, tool: Tool) => {
			try {
				await toolStoreAPI.deleteTool(bundleID, tool.slug, tool.version);

				setBundles(prev =>
					(prev ?? []).map(bd =>
						bd.bundle.id === bundleID
							? Object.assign({}, bd, {
									tools: bd.tools.filter(t => t.id !== tool.id),
								})
							: bd
					)
				);
			} catch (err: unknown) {
				console.error('Delete tool failed:', err);
				throw new Error(getErrorMessage(err, 'Failed to delete tool.'), { cause: err });
			}
		},
		[setBundles]
	);

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
				throw new Error(getErrorMessage(err, 'Failed to save tool.'), { cause: err });
			}
		},
		[bundles, refreshBundleTools]
	);

	const handleBundleDelete = async () => {
		if (!bundleToDelete) {
			return;
		}
		const bundleData = (bundles ?? []).find(item => item.bundle.id === bundleToDelete.id);
		if (!bundleData || bundleData.toolLoadError || bundleData.tools.length > 0) {
			setAlertMsg(
				bundleData?.toolLoadError
					? 'Reload this bundle before deleting it.'
					: 'Remove all tools before deleting the bundle.'
			);
			setShowAlert(true);
			return;
		}

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
		const id = getUUIDv7();
		await toolStoreAPI.putToolBundle(id, slug, display, true, description);

		try {
			await reloadOrThrow();
		} catch (error) {
			console.error('Tool bundle was created but refresh failed:', error);
			setAlertMsg(
				'Tool bundle was created, but the page could not be refreshed. Reload before making destructive changes.'
			);
			setShowAlert(true);
		}
	};

	if (isLoading && !hasResolved) {
		return <Loader text="Loading tool bundles…" />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Tool Bundles"
					description="Manage versioned tools, argument schemas, execution recommendations, and HTTP integrations."
					actions={
						<button
							type="button"
							className="btn btn-ghost rounded-xl"
							onClick={() => {
								setIsAddModalOpen(true);
							}}
						>
							<FiPlus size={18} />
							<span>Add Bundle</span>
						</button>
					}
				/>

				<ManagementPageContent>
					{pageLoadError ? (
						<ManagementResourceError
							title="Tool bundles could not be loaded"
							error={pageLoadError}
							isRetrying={isRefreshing}
							onRetry={async () => {
								await reloadOrThrow();
							}}
						/>
					) : null}

					{bundles.length === 0 && <p className="mt-8 text-center text-sm">No tool bundles configured yet.</p>}

					{bundles.map(bd => (
						<ToolBundleCard
							key={bd.bundle.id}
							bundle={bd.bundle}
							tools={bd.tools}
							toolLoadError={bd.toolLoadError}
							onRefreshTools={() => {
								return refreshBundleTools(bd.bundle.id);
							}}
							onToggleBundleEnable={handleToggleBundleEnable}
							onToggleToolEnable={handleToggleToolEnable}
							onDeleteTool={handleDeleteTool}
							onSubmitTool={handleSubmitTool}
							onRequestDeleteBundle={b => {
								setBundleToDelete(b);
							}}
						/>
					))}
				</ManagementPageContent>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						setBundleToDelete(null);
					}}
					onConfirm={handleBundleDelete}
					title="Delete Tool Bundle"
					message={`Delete empty bundle "${bundleToDelete?.displayName ?? ''}"? Remove all tools first.`}
					confirmButtonText="Delete"
				/>

				<ManagementBundleCreateModal
					isOpen={isAddModalOpen}
					title="Add Tool Bundle"
					entityLabel="Tool bundle"
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={bundles.map(b => b.bundle.slug)}
					existingDisplayNames={bundles.map(bundleData => bundleData.bundle.displayName || bundleData.bundle.slug)}
					failureMessage="Failed to create tool bundle."
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
