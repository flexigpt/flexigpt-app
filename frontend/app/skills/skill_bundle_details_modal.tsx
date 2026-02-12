import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiX } from 'react-icons/fi';

import type { SkillBundle } from '@/spec/skill';

import { ModalBackdrop } from '@/components/modal_backdrop';

interface SkillBundleDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: SkillBundle | null;
}

export function SkillBundleDetailsModal({ isOpen, onClose, bundle }: SkillBundleDetailsModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);

	useEffect(() => {
		if (!isOpen) return;
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) dialog.showModal();

		return () => {
			if (dialog.open) dialog.close();
		};
	}, [isOpen]);

	const handleDialogClose = () => {
		onClose();
	};

	if (!isOpen || !bundle) return null;

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">Skill Bundle Details</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<div className="space-y-3 text-sm">
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-3 font-semibold">Display Name</div>
							<div className="col-span-9">{bundle.displayName || '-'}</div>
						</div>
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-3 font-semibold">Slug</div>
							<div className="col-span-9">{bundle.slug}</div>
						</div>
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-3 font-semibold">ID</div>
							<div className="col-span-9">{bundle.id}</div>
						</div>
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-3 font-semibold">Built-in</div>
							<div className="col-span-9">{bundle.isBuiltIn ? 'Yes' : 'No'}</div>
						</div>
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-3 font-semibold">Enabled</div>
							<div className="col-span-9">{bundle.isEnabled ? 'Yes' : 'No'}</div>
						</div>
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-3 font-semibold">Description</div>
							<div className="col-span-9 whitespace-pre-wrap">{bundle.description || '-'}</div>
						</div>
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-3 font-semibold">Created</div>
							<div className="col-span-9">{String(bundle.createdAt)}</div>
						</div>
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-3 font-semibold">Modified</div>
							<div className="col-span-9">{String(bundle.modifiedAt)}</div>
						</div>
					</div>

					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
							Close
						</button>
					</div>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
