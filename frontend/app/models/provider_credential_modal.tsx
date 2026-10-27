import type { SubmitEventHandler } from 'react';
import { useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { ModelProviderManagementItem } from '@/apis/model_management';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Dropdown } from '@/components/dropdown';
import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';

type CredentialAction = 'replace' | 'remove';

interface ProviderCredentialModalProps {
	isOpen: boolean;
	provider: ModelProviderManagementItem;
	onClose: () => void;
	onSave: (credential: string) => Promise<void>;
}

function ProviderCredentialModalContent({
	provider,
	onSave,
}: Omit<ProviderCredentialModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose, unmountingRef } = useModalDialogController();
	const [action, setAction] = useState<CredentialAction>('replace');
	const [credential, setCredential] = useState('');
	const [error, setError] = useState('');
	const [saving, setSaving] = useState(false);
	const [confirmRemoval, setConfirmRemoval] = useState(false);

	const actionItems: Record<CredentialAction, { isEnabled: boolean; displayName: string }> = {
		replace: {
			isEnabled: true,
			displayName: provider.apiKey.configured ? 'Replace API key' : 'Set API key',
		},
		remove: {
			isEnabled: provider.apiKey.configured,
			displayName: 'Remove API key',
		},
	};

	const save = async (nextCredential: string) => {
		setSaving(true);
		setError('');

		try {
			await onSave(nextCredential);
			requestClose(true);
		} catch (saveError) {
			if (!unmountingRef.current) {
				setError(
					saveError instanceof Error && saveError.message.trim() ? saveError.message : 'Credential could not be saved.'
				);
			}
		} finally {
			if (!unmountingRef.current) {
				setSaving(false);
			}
		}
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();

		if (action === 'remove') {
			setConfirmRemoval(true);
			return;
		}

		if (!credential.trim()) {
			setError('Enter a credential before saving.');
			return;
		}

		void save(credential.trim());
	};

	return (
		<>
			<ManagementModalPanel width="compact">
				<ModalHeader
					title={`API Key for ${provider.list.displayName}`}
					description="API keys are write-only and are never shown after saving."
					onClose={() => {
						requestClose();
					}}
					closeDisabled={saving}
				/>

				<form noValidate onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col" aria-busy={saving}>
					<ManagementModalBody>
						{error ? (
							<div className="alert alert-error rounded-xl text-sm" role="alert">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={15} />
									<span>{error}</span>
								</div>
							</div>
						) : null}

						{provider.apiKey.configured ? (
							<ModalField label="Action">
								<Dropdown<CredentialAction>
									dropdownItems={actionItems}
									selectedKey={action}
									onChange={setAction}
									filterDisabled={false}
									title="Select credential action"
									getDisplayName={key => actionItems[key].displayName}
									disabled={saving}
								/>
							</ModalField>
						) : null}

						{action === 'replace' ? (
							<ModalField label="Credential" htmlFor="provider-credential" required>
								<input
									id="provider-credential"
									type="password"
									autoComplete="new-password"
									className="input w-full rounded-xl"
									value={credential}
									onChange={event => {
										setCredential(event.target.value);
										setError('');
									}}
									disabled={saving}
									autoFocus
								/>
							</ModalField>
						) : (
							<div className="border-warning/40 bg-warning/10 rounded-xl border p-3 text-sm">
								Requests using this provider will fail until a replacement credential is configured.
							</div>
						)}
					</ManagementModalBody>

					<ModalActions>
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							disabled={saving}
							onClick={() => {
								requestClose();
							}}
						>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={saving}>
							{saving ? 'Saving...' : action === 'remove' ? 'Remove API Key' : 'Save API Key'}
						</button>
					</ModalActions>
				</form>
			</ManagementModalPanel>

			<DeleteConfirmationModal
				isOpen={confirmRemoval}
				onClose={() => {
					setConfirmRemoval(false);
				}}
				onConfirm={async () => {
					setConfirmRemoval(false);
					await save('');
				}}
				confirmButtonText="Remove"
				title="Remove Provider API Key"
				message={`Remove the API key for “${provider.list.displayName}”? Requests using this provider will fail until a new API key is configured.`}
			/>
		</>
	);
}

export function ProviderCredentialModal({ isOpen, provider, onClose, onSave }: ProviderCredentialModalProps) {
	if (!isOpen) {
		return null;
	}

	const key = `${provider.list.ref.rootID}:${provider.list.ref.artifactID}:${provider.apiKey.apiKeyRevision}`;

	return (
		<ManagementModalDialog isOpen={isOpen} onClose={onClose} blockCancel>
			<ProviderCredentialModalContent key={key} provider={provider} onSave={onSave} />
		</ManagementModalDialog>
	);
}
