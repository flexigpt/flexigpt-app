import { useCallback, useEffect, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { PromptBundle, PromptTemplate } from '@/spec/prompt';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { promptStoreAPI } from '@/apis/baseapi';
import { getAllPromptBundles, getAllPromptTemplates } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { PageFrame } from '@/components/page_frame';

import {
	derivePromptTemplateKind,
	derivePromptTemplateResolved,
	type PromptTemplateUpsertInput,
} from '@/prompts/lib/prompt_template_utils';
import { AddBundleModal } from '@/prompts/prompt_bundle_add_modal';
import { PromptBundleCard } from '@/prompts/prompt_bundle_card';

interface BundleData {
	bundle: PromptBundle;
	templates: PromptTemplate[];
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

// eslint-disable-next-line no-restricted-exports
export default function PromptsPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [loading, setLoading] = useState(true);

	/* alerts */
	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	/* bundle deletion modal */
	const [bundleToDeleteID, setBundleToDeleteID] = useState<string | null>(null);

	/* add-bundle modal */
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);

	const bundleToDelete =
		bundleToDeleteID === null
			? null
			: (bundles.find(bundleData => bundleData.bundle.id === bundleToDeleteID)?.bundle ?? null);

	const loadTemplatesForBundle = useCallback(async (bundleID: string): Promise<PromptTemplate[]> => {
		const promptTemplateListItems = await getAllPromptTemplates([bundleID], undefined, true);

		const templatePromises = promptTemplateListItems.map(item =>
			promptStoreAPI.getPromptTemplate(item.bundleID, item.templateSlug, item.templateVersion)
		);

		return (await Promise.all(templatePromises)).filter(
			(template): template is PromptTemplate => template !== undefined
		);
	}, []);

	const refreshBundleTemplates = useCallback(
		async (bundleID: string) => {
			const freshTemplates = await loadTemplatesForBundle(bundleID);

			setBundles(prev =>
				prev.map(bundleData =>
					bundleData.bundle.id === bundleID ? { ...bundleData, templates: freshTemplates } : bundleData
				)
			);
		},
		[loadTemplatesForBundle]
	);

	const fetchAll = useCallback(async () => {
		setLoading(true);

		try {
			const promptBundles = await getAllPromptBundles(undefined, true);

			const bundleResults: BundleData[] = await Promise.all(
				promptBundles.map(async bundle => {
					try {
						const templates = await loadTemplatesForBundle(bundle.id);
						return { bundle, templates };
					} catch {
						return { bundle, templates: [] };
					}
				})
			);

			setBundles(bundleResults);
		} catch (error) {
			console.error('Failed to load bundles:', error);
			setAlertMsg(getErrorMessage(error, 'Failed to load bundles. Please try again.'));
			setShowAlert(true);
		} finally {
			setLoading(false);
		}
	}, [loadTemplatesForBundle]);

	useEffect(() => {
		void fetchAll();
	}, [fetchAll]);

	const handleToggleBundleEnabled = useCallback(
		async (bundleID: string, enabled: boolean) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('Bundle not found.');
			}

			await promptStoreAPI.patchPromptBundle(bundleID, enabled);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								bundle: {
									...item.bundle,
									isEnabled: enabled,
								},
							}
						: item
				)
			);
		},
		[bundles]
	);

	const handleToggleTemplateEnabled = useCallback(
		async (bundleID: string, templateID: string) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('Bundle not found.');
			}

			if (!bundleData.bundle.isEnabled) {
				throw new Error('Enable the bundle before enabling or disabling templates.');
			}

			const template = bundleData.templates.find(item => item.id === templateID);

			if (!template) {
				throw new Error('Template not found.');
			}

			const nextEnabled = !template.isEnabled;

			await promptStoreAPI.patchPromptTemplate(bundleID, template.slug, template.version, nextEnabled);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								templates: item.templates.map(existingTemplate =>
									existingTemplate.id === templateID
										? {
												...existingTemplate,
												isEnabled: nextEnabled,
											}
										: existingTemplate
								),
							}
						: item
				)
			);
		},
		[bundles]
	);

	const handleDeleteTemplate = useCallback(
		async (bundleID: string, templateID: string) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('Bundle not found.');
			}

			if (bundleData.bundle.isBuiltIn) {
				throw new Error('Cannot delete templates from a built-in bundle.');
			}

			const template = bundleData.templates.find(item => item.id === templateID);

			if (!template) {
				throw new Error('Template not found.');
			}

			if (template.isBuiltIn) {
				throw new Error('Cannot delete built-in template.');
			}

			await promptStoreAPI.deletePromptTemplate(bundleID, template.slug, template.version);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								templates: item.templates.filter(existingTemplate => existingTemplate.id !== templateID),
							}
						: item
				)
			);
		},
		[bundles]
	);

	const handleSubmitTemplate = useCallback(
		async (bundleID: string, templateToEditID: string | undefined, partial: PromptTemplateUpsertInput) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('Bundle not found.');
			}

			if (bundleData.bundle.isBuiltIn) {
				throw new Error('Cannot add or edit templates in a built-in bundle.');
			}

			if (!bundleData.bundle.isEnabled) {
				throw new Error('Enable the bundle before adding or editing templates.');
			}

			const templateToEdit =
				templateToEditID === undefined ? undefined : bundleData.templates.find(item => item.id === templateToEditID);

			if (templateToEditID !== undefined && !templateToEdit) {
				throw new Error('Template not found.');
			}

			if (templateToEdit?.isBuiltIn) {
				throw new Error('Built-in templates cannot be edited.');
			}

			const slug = (templateToEdit?.slug ?? partial.slug).trim();
			const version = partial.version.trim();
			const nextBlocks = partial.blocks;
			const nextVariables = partial.variables;
			const nextKind = derivePromptTemplateKind(nextBlocks);
			const nextIsResolved = derivePromptTemplateResolved(nextBlocks, nextVariables);

			if (!slug) {
				throw new Error('Missing template slug.');
			}

			if (!version) {
				throw new Error('Version is required.');
			}

			const exists = bundleData.templates.some(template => template.slug === slug && template.version === version);

			if (exists) {
				throw new Error(`Version "${version}" already exists for slug "${slug}". Create a different version.`);
			}

			if (templateToEdit) {
				await promptStoreAPI.putPromptTemplate(
					nextKind,
					bundleID,
					templateToEdit.slug,
					partial.displayName,
					partial.isEnabled,
					nextBlocks,
					version,
					nextIsResolved,
					partial.description ?? templateToEdit.description,
					partial.tags ?? templateToEdit.tags,
					nextVariables
				);
			} else {
				const displayName = partial.displayName?.trim();

				await promptStoreAPI.putPromptTemplate(
					nextKind,
					bundleID,
					slug,
					displayName,
					partial.isEnabled,
					nextBlocks,
					version,
					nextIsResolved,
					partial.description,
					partial.tags,
					nextVariables
				);
			}

			await refreshBundleTemplates(bundleID);
		},
		[bundles, refreshBundleTemplates]
	);

	const handleBundleDelete = useCallback(async () => {
		if (!bundleToDeleteID) {
			return;
		}

		try {
			await promptStoreAPI.deletePromptBundle(bundleToDeleteID);

			setBundles(prev => prev.filter(bundleData => bundleData.bundle.id !== bundleToDeleteID));
		} catch (error) {
			console.error('Delete bundle failed:', error);
			setAlertMsg(getErrorMessage(error, 'Failed to delete bundle.'));
			setShowAlert(true);
		} finally {
			setBundleToDeleteID(null);
		}
	}, [bundleToDeleteID]);

	const handleAddBundle = useCallback(
		async (slug: string, display: string, description?: string) => {
			try {
				const id = getUUIDv7();
				await promptStoreAPI.putPromptBundle(id, slug, display, true, description);
				setIsAddModalOpen(false);
				await fetchAll();
			} catch (error) {
				console.error('Add bundle failed:', error);
				setAlertMsg(getErrorMessage(error, 'Failed to add bundle.'));
				setShowAlert(true);
			}
		},
		[fetchAll]
	);

	if (loading) {
		return <Loader text="Loading bundles…" />;
	}

	return (
		<PageFrame>
			<div className="flex h-full w-full flex-col items-center">
				<div className="fixed mt-8 flex w-10/12 items-center p-2 lg:w-2/3">
					<h1 className="flex grow items-center justify-center text-xl font-semibold">Prompt Bundles</h1>
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
					<div className="flex w-11/12 flex-col space-y-4 xl:w-2/3">
						{bundles.length === 0 && <p className="mt-8 text-center text-sm">No bundles configured yet.</p>}

						{bundles.map(bundleData => (
							<PromptBundleCard
								key={bundleData.bundle.id}
								bundle={bundleData.bundle}
								templates={bundleData.templates}
								onToggleBundleEnabled={handleToggleBundleEnabled}
								onToggleTemplateEnabled={handleToggleTemplateEnabled}
								onDeleteTemplate={handleDeleteTemplate}
								onSubmitTemplate={handleSubmitTemplate}
								onDeleteBundleRequested={bundleID => {
									setBundleToDeleteID(bundleID);
								}}
							/>
						))}
					</div>
				</div>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						setBundleToDeleteID(null);
					}}
					onConfirm={handleBundleDelete}
					title="Delete Prompt Bundle"
					message={`Delete empty bundle "${bundleToDelete?.displayName ?? ''}"? Remove all templates first.`}
					confirmButtonText="Delete"
				/>

				<AddBundleModal
					isOpen={isAddModalOpen}
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={bundles.map(bundleData => bundleData.bundle.slug)}
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
