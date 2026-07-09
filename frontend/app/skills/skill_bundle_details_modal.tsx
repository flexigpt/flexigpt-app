import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiX } from 'react-icons/fi';

import type { Skill, SkillBundle } from '@/spec/skill';

import { ModalBackdrop } from '@/components/modal_backdrop';

import { getSkillInsertCounts, getSkillInsertDescription } from '@/skills/lib/skill_artifact_utils';

interface SkillBundleDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: SkillBundle | null;
	skills: Skill[];
}

export function SkillBundleDetailsModal({ isOpen, onClose, bundle, skills }: SkillBundleDetailsModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const skillCounts = getSkillInsertCounts(skills);

	useEffect(() => {
		if (!isOpen) {
			return;
		}
		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	const handleDialogClose = () => {
		onClose();
	};

	if (!isOpen || !bundle) {
		return null;
	}

	const totalSkills = skills.length;

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

						<div className="divider">Skill summary</div>
						<div className="grid grid-cols-12 gap-2 text-sm">
							<div className="col-span-3 font-semibold">Total skills</div>
							<div className="col-span-9">{totalSkills}</div>

							<div className="col-span-3 font-semibold">Instruction skills</div>
							<div className="col-span-9">
								<div className="flex items-center gap-2">
									<span className="badge badge-info rounded-xl">{skillCounts.instructions}</span>
									<span className="text-base-content/70 text-xs">{getSkillInsertDescription('instructions')}</span>
								</div>
							</div>

							<div className="col-span-3 font-semibold">User-message skills</div>
							<div className="col-span-9">
								<div className="flex items-center gap-2">
									<span className="badge badge-secondary rounded-xl">{skillCounts['user-message']}</span>
									<span className="text-base-content/70 text-xs">{getSkillInsertDescription('user-message')}</span>
								</div>
							</div>

							<div className="col-span-3 font-semibold">Usage note</div>
							<div className="text-base-content/70 col-span-9 text-xs">
								Instruction skills affect session state. User-message skills render into the composer or user message
								body and are not active session skills.
							</div>
							<div className="col-span-3 font-semibold">Prompt migration note</div>
							<div className="text-base-content/70 col-span-9 text-xs">
								A prompt-like template should be a filesystem skill whose <span className="font-mono">SKILL.md</span>{' '}
								frontmatter contains <span className="font-mono">insert: user-message</span>. Its declared arguments
								replace the old prompt variable form.
							</div>
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
