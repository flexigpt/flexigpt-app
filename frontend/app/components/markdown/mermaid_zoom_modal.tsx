import type { CSSProperties } from 'react';
import { useEffect, useRef } from 'react';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';

interface MermaidZoomModalProps {
	isOpen: boolean;
	onClose: () => void;
	svgNode: SVGSVGElement | null;
	surfaceStyle?: CSSProperties;
}

function MermaidZoomModalContent({ svgNode, surfaceStyle }: Omit<MermaidZoomModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose } = useModalDialogController();
	const modalRef = useRef<HTMLDivElement | null>(null);

	// Inject the SVG into the modal container whenever we have one and the modal is open
	useEffect(() => {
		if (!modalRef.current) {
			return;
		}

		const container = modalRef.current;
		container.innerHTML = '';

		if (!svgNode) {
			return;
		}

		const newNode = svgNode.cloneNode(true) as SVGSVGElement;
		newNode.style.display = 'block';
		newNode.style.margin = 'auto';
		newNode.style.width = 'auto';
		newNode.style.height = 'auto';
		newNode.style.maxWidth = '90vw';
		newNode.style.maxHeight = '80vh';
		newNode.style.backgroundColor = 'transparent';
		const bg = newNode.querySelector('rect.background');
		if (bg) {
			bg.setAttribute('fill', 'transparent');
		}
		container.append(newNode);
	}, [svgNode]);

	return (
		<>
			<div
				className="modal-box app-bg-mermaid flex h-[90vh] max-w-[90vw] cursor-zoom-out items-center justify-center"
				style={surfaceStyle}
				onClick={() => {
					requestClose();
				}}
			>
				{/* enlarged diagram; pointer events disabled so clicks bubble to the container */}
				<div ref={modalRef} className="w-full overflow-auto" style={{ pointerEvents: 'none' }} />
			</div>

			<ModalBackdrop enabled={true} />
		</>
	);
}

export function MermaidZoomModal(props: MermaidZoomModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} aria-label="Enlarged Mermaid diagram">
			<MermaidZoomModalContent svgNode={props.svgNode} surfaceStyle={props.surfaceStyle} />
		</ModalDialog>
	);
}
