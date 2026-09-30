import type { TableHTMLAttributes } from 'react';
import type { ExtraProps } from 'react-markdown';
import { useId, useState } from 'react';
import { FiMaximize2 } from 'react-icons/fi';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

type TableProps = TableHTMLAttributes<HTMLTableElement>;

function TableSurface({ children, className, style, ...props }: TableProps) {
	const wrapID = useId();
	const zoomID = useId();
	const [wrap, setWrap] = useState(true);
	const [zoom, setZoom] = useState(100);

	return (
		<div className="min-w-0">
			<div className="flex flex-wrap items-center gap-3 p-2 pr-12 text-xs">
				<label htmlFor={wrapID} className="flex items-center gap-2">
					<input
						id={wrapID}
						type="checkbox"
						className="checkbox checkbox-xs"
						checked={wrap}
						onChange={event => {
							setWrap(event.target.checked);
						}}
					/>
					Wrap cells
				</label>
				<label htmlFor={zoomID}>Text size</label>
				<input
					id={zoomID}
					type="range"
					min={50}
					max={150}
					step={10}
					value={zoom}
					className="range range-xs w-24"
					onChange={event => {
						setZoom(event.target.valueAsNumber);
					}}
				/>
				<output htmlFor={zoomID}>{zoom}%</output>
				<button
					type="button"
					className="btn btn-ghost btn-xs"
					onClick={() => {
						setZoom(100);
					}}
				>
					Reset
				</button>
			</div>
			<div className="max-w-full overflow-x-auto">
				<table
					{...props}
					className={`[&_code]:[white-space:inherit] ${className ?? ''}`}
					style={{
						...style,
						width: wrap ? '100%' : 'max-content',
						minWidth: '100%',
						tableLayout: wrap ? 'fixed' : 'auto',
						whiteSpace: wrap ? 'normal' : 'nowrap',
						overflowWrap: wrap ? 'anywhere' : 'normal',
						fontSize: `${(14 * zoom) / 100}px`,
					}}
				>
					{children}
				</table>
			</div>
		</div>
	);
}

function TableZoomContent({ tableProps }: { tableProps: TableProps }) {
	const { requestClose } = useModalDialogController();

	return (
		<>
			<div
				className="modal-box bg-base-100 flex flex-col overflow-hidden rounded-none p-0"
				style={{ width: '100vw', maxWidth: 'none', height: '100dvh', maxHeight: 'none' }}
			>
				<ModalHeader
					title="Table"
					onClose={() => {
						requestClose();
					}}
				/>
				<div className="min-h-0 flex-1 overflow-auto p-4">
					<TableSurface {...tableProps} id={undefined} />
				</div>
			</div>
			<ModalBackdrop enabled={true} />
		</>
	);
}

export function MarkdownTable({ node: _node, ...tableProps }: TableProps & ExtraProps) {
	const [open, setOpen] = useState(false);

	return (
		<div className="group border-base-300 relative my-3 min-w-0 rounded-lg border">
			<TableSurface {...tableProps} />
			<button
				type="button"
				className="btn btn-ghost btn-xs absolute top-2 right-2 opacity-100 group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:hover)]:opacity-0"
				aria-label="Enlarge table"
				title="Enlarge table"
				onClick={() => {
					setOpen(true);
				}}
			>
				<FiMaximize2 size={16} />
			</button>
			{open ? (
				<ModalDialog
					isOpen={true}
					onClose={() => {
						setOpen(false);
					}}
					aria-label="Enlarged table"
					data-disable-chat-shortcuts="true"
				>
					<TableZoomContent tableProps={tableProps} />
				</ModalDialog>
			) : null}
		</div>
	);
}
