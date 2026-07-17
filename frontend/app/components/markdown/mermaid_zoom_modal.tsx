import type { CSSProperties } from 'react';
import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { ModalBackdrop } from '@/components/modal/modal_backdrop';

interface MermaidZoomModalProps {
	isOpen: boolean;
	onClose: () => void;
	svgNode: SVGSVGElement | null;
	surfaceStyle?: CSSProperties;
}

function MermaidZoomModalContent({ onClose, svgNode, surfaceStyle }: Omit<MermaidZoomModalProps, 'isOpen'>) {
	const { dialogRef, requestClose, handleClose, handleCancel } = useDialogController({ onClose });
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
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleClose}
			onCancel={handleCancel}
			aria-label="Enlarged Mermaid diagram"
		>
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
		</dialog>
	);
}

export function MermaidZoomModal(props: MermaidZoomModalProps) {
	if (!props.isOpen || typeof document === 'undefined' || !document.body) {
		return null;
	}

	return createPortal(
		<MermaidZoomModalContent onClose={props.onClose} svgNode={props.svgNode} surfaceStyle={props.surfaceStyle} />,
		document.body
	);
}
