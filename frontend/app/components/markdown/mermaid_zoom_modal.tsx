import { type CSSProperties, useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { ModalBackdrop } from '@/components/modal_backdrop';

type MermaidZoomModalProps = {
	isOpen: boolean;
	onClose: () => void;
	svgNode: SVGSVGElement | null;
	surfaceStyle?: CSSProperties;
};

export function MermaidZoomModal({ isOpen, onClose, svgNode, surfaceStyle }: MermaidZoomModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const modalRef = useRef<HTMLDivElement | null>(null);

	// Open the dialog natively when isOpen becomes true
	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			// If the component unmounts while the dialog is still open, close it.
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	// Sync parent state whenever the dialog is closed (Esc, backdrop, or dialog.close()).
	const handleDialogClose = () => {
		onClose();
	};

	// Inject the SVG into the modal container whenever we have one and the modal is open
	useEffect(() => {
		if (!isOpen || !modalRef.current) return;

		const container = modalRef.current;
		container.innerHTML = '';

		if (!svgNode) return;

		const newNode = svgNode.cloneNode(true) as SVGSVGElement;
		newNode.style.display = 'block';
		newNode.style.margin = 'auto';
		newNode.style.width = 'auto';
		newNode.style.height = 'auto';
		newNode.style.maxWidth = '90vw';
		newNode.style.maxHeight = '80vh';
		newNode.style.backgroundColor = 'transparent';
		const bg = newNode.querySelector('rect.background');
		if (bg) bg.setAttribute('fill', 'transparent');
		container.appendChild(newNode);
	}, [isOpen, svgNode]);

	if (!isOpen) return null;

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose} aria-label="Enlarged Mermaid diagram">
			<div
				className="modal-box bg-mermaid flex h-[90vh] max-w-[90vw] cursor-zoom-out items-center justify-center"
				style={surfaceStyle}
				onClick={() => {
					// Close via native dialog API; this will trigger handleDialogClose -> parent onClose()
					dialogRef.current?.close();
				}}
			>
				{/* enlarged diagram; pointer events disabled so clicks bubble to the container */}
				<div ref={modalRef} className="w-full overflow-auto" style={{ pointerEvents: 'none' }} />
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
