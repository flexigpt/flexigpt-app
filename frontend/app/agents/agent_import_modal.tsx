import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiCheck, FiFileText, FiUpload } from 'react-icons/fi';

import type {
	AgentImportCommitResult,
	AgentImportDestination,
	AgentImportPreview,
	AgentImportRelationship,
} from '@/spec/agent';
import { AgentImportIssueSeverity } from '@/spec/agent';

import { getErrorMessage } from '@/lib/error_utils';

import { agentCollectionKey, collectionDisplayName } from '@/apis/agent_management';
import { agentStoreAPI, backendAPI } from '@/apis/baseapi';

import { Dropdown } from '@/components/dropdown';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { formatArtifactRef, formatDateish, getAgentRelationshipBadgeClass } from '@/agents/lib/agent_management_utils';

interface AgentImportModalProps {
	isOpen: boolean;
	destinations: AgentImportDestination[];
	initialDestination?: AgentImportDestination | null;
	onClose: () => void;
	onCommitted: (result: AgentImportCommitResult) => Promise<void>;
}

function isSupportedAgentImportPath(value: string): boolean {
	return /\.(?:json|ya?ml)$/i.test(value);
}

function getIssueClass(severity: AgentImportIssueSeverity): string {
	switch (severity) {
		case AgentImportIssueSeverity.Error:
			return 'alert-error';
		case AgentImportIssueSeverity.Confirmation:
			return 'alert-warning';
		case AgentImportIssueSeverity.Warning:
			return 'alert-warning';
		default:
			return 'alert-info';
	}
}

function relationshipTarget(relationship: AgentImportRelationship): string | undefined {
	if (relationship.artifact) {
		return formatArtifactRef(relationship.artifact);
	}

	if (relationship.mapped) {
		return `${relationship.mapped.provider}/${relationship.mapped.identifier}`;
	}

	return undefined;
}

export function AgentImportModal({
	isOpen,
	destinations,
	initialDestination,
	onClose,
	onCommitted,
}: AgentImportModalProps) {
	const initialDestinationKey = initialDestination
		? agentCollectionKey(initialDestination.collection)
		: destinations[0]
			? agentCollectionKey(destinations[0].collection)
			: '';

	const [destinationKey, setDestinationKey] = useState(initialDestinationKey);
	const [path, setPath] = useState('');
	const [preview, setPreview] = useState<AgentImportPreview | null>(null);
	const [acceptedConfirmationCodes, setAcceptedConfirmationCodes] = useState<Set<string>>(new Set());
	const [error, setError] = useState('');
	const [isPickingFile, setIsPickingFile] = useState(false);
	const [isPreviewing, setIsPreviewing] = useState(false);
	const [isCommitting, setIsCommitting] = useState(false);

	useEffect(() => {
		if (!isOpen) {
			return;
		}

		// oxlint-disable-next-line react/set-state-in-effect react-you-might-not-need-an-effect/no-derived-state
		setDestinationKey(initialDestinationKey);
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setPath('');

		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setPreview(null);

		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setAcceptedConfirmationCodes(new Set());

		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setError('');
	}, [initialDestinationKey, isOpen]);

	const destinationByKey = useMemo(
		() => new Map(destinations.map(destination => [agentCollectionKey(destination.collection), destination] as const)),
		[destinations]
	);

	const dropdownItems = useMemo<Record<string, { isEnabled: boolean }>>(
		() =>
			Object.fromEntries(
				destinations.map(destination => [agentCollectionKey(destination.collection), { isEnabled: true }] as const)
			),
		[destinations]
	);

	const selectedDestination = destinationByKey.get(destinationKey);
	const requiredConfirmationCodes = preview?.requiredConfirmationCodes ?? [];
	const allConfirmationsAccepted = requiredConfirmationCodes.every(code => acceptedConfirmationCodes.has(code));

	const resetPreview = () => {
		setPreview(null);
		setAcceptedConfirmationCodes(new Set());
		setError('');
	};

	const chooseFile = async () => {
		if (isPickingFile || isPreviewing || isCommitting) {
			return;
		}

		setIsPickingFile(true);
		setError('');

		try {
			const paths = await backendAPI.pickFilePaths(false);
			const selectedPath = paths[0]?.trim();

			if (selectedPath) {
				resetPreview();

				if (!isSupportedAgentImportPath(selectedPath)) {
					setPath('');
					setError('Select a JSON or YAML Agent declaration file (.json, .yaml, or .yml).');
					return;
				}

				setPath(selectedPath);
			}
		} catch (pickError) {
			setError(getErrorMessage(pickError, 'Unable to select an Agent JSON or YAML file.'));
		} finally {
			setIsPickingFile(false);
		}
	};

	const previewImport = async () => {
		if (!selectedDestination) {
			setError('Select an Agent Collection destination.');
			return;
		}

		if (!path.trim()) {
			setError('Select an Agent JSON or YAML declaration file.');
			return;
		}

		setIsPreviewing(true);
		setError('');
		setPreview(null);
		setAcceptedConfirmationCodes(new Set());

		try {
			const value = await agentStoreAPI.previewAgentImport({
				path: path.trim(),
				collection: {
					rootID: selectedDestination.collection.artifact.rootID,
					artifactID: selectedDestination.collection.artifact.id,
				},
				expectedCollectionRevision: selectedDestination.collectionRevision,
			});

			setPreview(value);
		} catch (previewError) {
			setError(getErrorMessage(previewError, 'Agent import preview failed.'));
		} finally {
			setIsPreviewing(false);
		}
	};

	const commitImport = async () => {
		if (!preview?.canImport || !preview.prepared || !preview.preparedFingerprint || isCommitting) {
			return;
		}

		if (!allConfirmationsAccepted) {
			setError('Accept every required confirmation before importing.');
			return;
		}

		setIsCommitting(true);
		setError('');

		try {
			const result = await agentStoreAPI.commitAgentImport({
				prepared: preview.prepared,
				preparedFingerprint: preview.preparedFingerprint,
				acceptedConfirmationCodes: requiredConfirmationCodes,
			});

			await onCommitted(result);
			onClose();
		} catch (commitError) {
			setError(getErrorMessage(commitError, 'Agent import failed. Preview again if the destination changed.'));
		} finally {
			setIsCommitting(false);
		}
	};

	if (!isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose} blockCancel={isPreviewing || isCommitting}>
			<div className="modal-box bg-base-200 max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-5xl overflow-hidden rounded-2xl p-0">
				<div className="app-scrollbar-thin max-h-[calc(100dvh-1rem)] overflow-y-auto p-4 sm:p-6">
					<ModalHeader
						title="Import Managed Agent"
						description="Preview a JSON or YAML Agent declaration, accept required confirmations, then publish it into the selected Collection."
						onClose={onClose}
						closeDisabled={isPreviewing || isCommitting}
					/>

					<div className="space-y-5">
						{error ? (
							<div className="alert alert-error rounded-2xl text-sm">
								<FiAlertCircle size={16} />
								<span>{error}</span>
							</div>
						) : null}

						<ModalSection title="Import destination">
							<ModalField label="Agent Collection" htmlFor="agent-import-destination" required>
								<Dropdown<string>
									dropdownItems={dropdownItems}
									orderedKeys={destinations.map(destination => agentCollectionKey(destination.collection))}
									selectedKey={destinationKey}
									onChange={value => {
										setDestinationKey(value);
										resetPreview();
									}}
									disabled={destinations.length === 0 || isPreviewing || isCommitting}
									placeholderLabel="No editable Agent Collections are available"
									title="Select an Agent Collection"
									getDisplayName={value => {
										const destination = destinationByKey.get(value);

										if (!destination) {
											return 'Select an Agent Collection';
										}

										return `${destination.rootDisplayName || destination.rootID} / ${collectionDisplayName(
											destination.collection
										)}`;
									}}
								/>
							</ModalField>

							{selectedDestination ? (
								<div className="text-base-content/70 text-xs">
									Collection revision: {selectedDestination.collectionRevision}
									{selectedDestination.baseline ? ' · Baseline Collection' : ''}
								</div>
							) : null}
						</ModalSection>

						<ModalSection title="Agent declaration file">
							<ModalField label="Selected file" htmlFor="agent-import-path" required>
								<div className="flex gap-2">
									<input
										id="agent-import-path"
										className="input min-w-0 flex-1 rounded-xl font-mono text-xs"
										value={path}
										readOnly
										placeholder="Select a .json, .yaml, or .yml file"
									/>
									<button
										type="button"
										className="btn btn-ghost rounded-xl"
										onClick={() => {
											void chooseFile();
										}}
										disabled={isPickingFile || isPreviewing || isCommitting}
									>
										<FiUpload size={15} />
										<span>{isPickingFile ? 'Selecting...' : 'Choose File'}</span>
									</button>
								</div>
							</ModalField>

							<button
								type="button"
								className="btn btn-primary rounded-xl"
								disabled={!selectedDestination || !path.trim() || isPreviewing || isCommitting}
								onClick={() => {
									void previewImport();
								}}
							>
								<FiFileText size={15} />
								<span>{isPreviewing ? 'Previewing...' : 'Preview Import'}</span>
							</button>
						</ModalSection>

						{preview ? (
							<>
								<ModalSection title="Preview summary">
									<div className="grid gap-3 text-sm sm:grid-cols-2">
										<div className="border-base-content/10 rounded-2xl border p-3">
											<div className="text-base-content/70 text-xs">Can import</div>
											<div className={preview.canImport ? 'text-success font-medium' : 'text-error font-medium'}>
												{preview.canImport ? 'Yes' : 'No'}
											</div>
										</div>
										<div className="border-base-content/10 rounded-2xl border p-3">
											<div className="text-base-content/70 text-xs">Definition digest</div>
											<div className="truncate font-mono text-xs">{preview.definitionDigest || '—'}</div>
										</div>
										<div className="border-base-content/10 rounded-2xl border p-3">
											<div className="text-base-content/70 text-xs">Projected Artifacts</div>
											<div>{preview.projectedArtifacts?.length ?? 0}</div>
										</div>
										<div className="border-base-content/10 rounded-2xl border p-3">
											<div className="text-base-content/70 text-xs">Prepared payload expires</div>
											<div className="text-xs">{formatDateish(preview.expiresAt)}</div>
										</div>
									</div>
								</ModalSection>

								{preview.issues?.length ? (
									<ModalSection title="Import issues">
										<div className="space-y-2">
											{preview.issues.map(issue => (
												<div
													key={`${issue.severity}:${issue.code}:${issue.path ?? ''}`}
													className={`alert ${getIssueClass(issue.severity)} rounded-2xl text-sm`}
												>
													<FiAlertCircle size={16} />
													<div>
														<div className="font-medium">{issue.code}</div>
														<div>{issue.message}</div>
														{issue.path ? <div className="mt-1 font-mono text-xs">{issue.path}</div> : null}
													</div>
												</div>
											))}
										</div>
									</ModalSection>
								) : null}

								{preview.conflicts?.length ? (
									<ModalSection title="Conflicts">
										<div className="space-y-2">
											{preview.conflicts.map(conflict => (
												<div
													key={`${conflict.code}:${conflict.path ?? ''}`}
													className="alert alert-error rounded-2xl text-sm"
												>
													<FiAlertCircle size={16} />
													<div>
														<div className="font-medium">{conflict.code}</div>
														<div>{conflict.message}</div>
													</div>
												</div>
											))}
										</div>
									</ModalSection>
								) : null}

								{preview.relationships?.length ? (
									<ModalSection title="Managed dependency observations">
										<div className="space-y-2">
											{preview.relationships.map(relationship => (
												<div key={relationship.path} className="border-base-content/10 rounded-2xl border p-3">
													<div className="flex flex-wrap items-center gap-2">
														<span className="font-medium">
															{relationship.type}: {relationship.name}
														</span>
														<span className={`badge badge-sm ${getAgentRelationshipBadgeClass(relationship.status)}`}>
															{relationship.status}
														</span>
													</div>
													<div className="text-base-content/70 mt-1 font-mono text-xs">{relationship.path}</div>
													{relationshipTarget(relationship) ? (
														<div className="text-base-content/70 mt-1 text-xs">
															Target: {relationshipTarget(relationship)}
														</div>
													) : null}
													{relationship.message ? (
														<div className="text-warning mt-1 text-xs">{relationship.message}</div>
													) : null}
												</div>
											))}
										</div>
									</ModalSection>
								) : null}

								{preview.normalizedYAML ? (
									<ModalSection title="Normalized Agent YAML">
										<pre className="bg-base-300 max-h-96 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
											{preview.normalizedYAML}
										</pre>
									</ModalSection>
								) : null}

								{preview.mcpSetupDescriptors?.length ? (
									<ModalSection title="MCP setup after import">
										<div className="text-base-content/70 text-sm">
											{preview.mcpSetupDescriptors.length} MCP declaration
											{preview.mcpSetupDescriptors.length === 1 ? '' : 's'} may require installation setup after the
											Agent is imported.
										</div>
									</ModalSection>
								) : null}

								{requiredConfirmationCodes.length > 0 ? (
									<ModalSection title="Required confirmations">
										<div className="space-y-2">
											{requiredConfirmationCodes.map(code => (
												<label
													key={code}
													className="border-base-content/10 flex cursor-pointer items-center gap-3 rounded-2xl border p-3"
												>
													<input
														type="checkbox"
														className="checkbox checkbox-sm rounded-sm"
														checked={acceptedConfirmationCodes.has(code)}
														disabled={isCommitting}
														onChange={event => {
															setAcceptedConfirmationCodes(previous => {
																const next = new Set(previous);

																if (event.currentTarget.checked) {
																	next.add(code);
																} else {
																	next.delete(code);
																}

																return next;
															});
														}}
													/>
													<span className="text-base-content/70 text-xs">
														This declaration requires explicit acceptance before import.
													</span>
													<div className="font-mono text-sm">{code}</div>
												</label>
											))}
										</div>
									</ModalSection>
								) : null}
							</>
						) : null}

						<ModalActions>
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								disabled={isPreviewing || isCommitting}
								onClick={onClose}
							>
								Cancel
							</button>
							<button
								type="button"
								className="btn btn-primary rounded-xl"
								disabled={
									!preview?.canImport ||
									!preview.prepared ||
									!preview.preparedFingerprint ||
									!allConfirmationsAccepted ||
									isCommitting
								}
								onClick={() => {
									void commitImport();
								}}
							>
								<FiCheck size={15} />
								<span>{isCommitting ? 'Importing...' : 'Import Agent'}</span>
							</button>
						</ModalActions>
					</div>
				</div>
			</div>
		</ModalDialog>
	);
}
