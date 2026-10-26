import { FiFileText } from 'react-icons/fi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

export interface InstructionSourceDetails {
	displayName: string;
	sourceLabel: string;
	text: string;
}

interface InstructionSourceDetailsModalProps {
	isOpen: boolean;
	source: InstructionSourceDetails | null;
	onClose: () => void;
}

export function InstructionSourceDetailsModal({ isOpen, source, onClose }: InstructionSourceDetailsModalProps) {
	if (!isOpen || !source) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<div className="modal-box bg-base-200 flex max-h-[80vh] w-[calc(100%-1rem)] max-w-2xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader title={source.displayName} description={source.sourceLabel} onClose={onClose} />

				<div className="app-scrollbar-thin min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
					<ModalSection title="Source">
						<details className="border-base-300 bg-base-100 rounded-xl border">
							<summary className="hover:bg-base-200 flex cursor-pointer list-none items-center gap-2 rounded-xl px-3 py-2 text-sm font-medium">
								<FiFileText size={14} />
								View applied instruction text
							</summary>

							<div className="border-base-300 border-t p-3">
								<pre className="app-scrollbar-thin bg-base-300 max-h-96 overflow-auto rounded-xl p-3 text-xs/5 wrap-break-word whitespace-pre-wrap">
									{source.text || '(The selected source is empty.)'}
								</pre>
							</div>
						</details>
					</ModalSection>
				</div>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
						Close
					</button>
				</ModalActions>
			</div>
		</ModalDialog>
	);
}
