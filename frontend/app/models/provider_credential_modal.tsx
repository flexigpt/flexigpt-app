import type { SubmitEventHandler } from 'react';
import { useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { ModelProviderManagementItem } from '@/apis/model_management';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Dropdown } from '@/components/dropdown';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
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
			displayName: provider.list.credentialConfigured ? 'Replace credential' : 'Set credential',
		},
		remove: {
			isEnabled: provider.list.credentialConfigured ?? false,
			displayName: 'Remove credential',
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
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
				<ModalHeader
					title={`Credential for ${provider.list.displayName}`}
					description="Credentials are write-only and are never shown after saving."
					onClose={() => {
						requestClose();
					}}
					closeDisabled={saving}
				/>

				<form noValidate onSubmit={handleSubmit} className="space-y-4 p-6" aria-busy={saving}>
					{error ? (
						<div className="alert alert-error rounded-xl text-sm" role="alert">
							<div className="flex items-center gap-2">
								<FiAlertCircle size={15} />
								<span>{error}</span>
							</div>
						</div>
					) : null}

					{provider.list.credentialConfigured ? (
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

					<ModalActions className="-mx-6 mt-6 -mb-6">
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
							{saving ? 'Saving...' : action === 'remove' ? 'Remove Credential' : 'Save Credential'}
						</button>
					</ModalActions>
				</form>
			</div>

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
				title="Remove Provider Credential"
				message={`Remove the credential for “${provider.list.displayName}”? Requests using this provider will fail until a new credential is configured.`}
			/>
		</>
	);
}

export function ProviderCredentialModal({ isOpen, provider, onClose, onSave }: ProviderCredentialModalProps) {
	if (!isOpen) {
		return null;
	}

	const key = `${provider.list.ref.rootID}:${provider.list.ref.artifactID}:${provider.list.runtimeOverlayRevision ?? 0}`;

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose} blockCancel>
			<ProviderCredentialModalContent key={key} provider={provider} onSave={onSave} />
		</ModalDialog>
	);
}
