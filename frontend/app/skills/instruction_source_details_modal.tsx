import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';
import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
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
		<ManagementModalDialog isOpen={isOpen} onClose={onClose}>
			<ManagementModalPanel width="compact">
				<ModalHeader title={source.displayName} description={source.sourceLabel} onClose={onClose} />

				<ManagementModalBody>
					<ModalSection title="Source">
						<ManagementArtifactSource
							label="applied instruction text"
							prefetched={{
								data: source,
								text: source.text || '(The selected source is empty.)',
								fileName: 'instructions.md',
								extension: 'md',
							}}
							load={async () => ({
								data: source,
								text: source.text || '(The selected source is empty.)',
								fileName: 'instructions.md',
								extension: 'md',
							})}
						/>
					</ModalSection>
				</ManagementModalBody>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
						Close
					</button>
				</ModalActions>
			</ManagementModalPanel>
		</ManagementModalDialog>
	);
}
