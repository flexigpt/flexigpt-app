import type { TableHTMLAttributes } from 'react';
import type { ExtraProps } from 'react-markdown';
import { useContext, useId, useState } from 'react';
import { FiZoomIn } from 'react-icons/fi';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { MarkdownCodeRendererContext } from '@/components/markdown/markdown_code_renderer_context';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

type TableProps = TableHTMLAttributes<HTMLTableElement>;

interface TableSurfaceProps extends TableProps {
	onZoom?: () => void;
}

function TableSurface({ children, className, style, onZoom, ...props }: TableSurfaceProps) {
	const wrapID = useId();
	const [wrap, setWrap] = useState(true);

	return (
		<div className="relative min-w-0">
			<div className="border-base-300 bg-base-200/40 absolute inset-x-0 top-0 z-10 flex h-10 items-center justify-end gap-1 border-b px-2">
				<button
					type="button"
					className="btn btn-ghost btn-xs"
					aria-pressed={wrap}
					title={wrap ? 'Keep table cells on one line' : 'Wrap long table cell content'}
					onClick={() => {
						setWrap(current => !current);
					}}
				>
					<label htmlFor={wrapID} className="sr-only">
						Wrap table cells
					</label>
					<span id={wrapID}>{wrap ? 'Unwrap' : 'Wrap'}</span>
				</button>

				{onZoom ? (
					<button
						type="button"
						className="btn btn-ghost btn-xs"
						aria-label="Enlarge table"
						title="Enlarge table"
						onClick={onZoom}
					>
						<FiZoomIn size={16} />
					</button>
				) : null}
			</div>

			<div className="max-w-full overflow-x-auto pt-10">
				<table
					{...props}
					className={`[&_code]:[white-space:inherit] [&_td]:wrap-break-word [&_th]:wrap-break-word ${className ?? ''}`}
					style={{
						...style,
						width: wrap ? '100%' : 'max-content',
						minWidth: '100%',
						tableLayout: wrap ? 'fixed' : 'auto',
						whiteSpace: wrap ? 'normal' : 'nowrap',
						overflowWrap: wrap ? 'anywhere' : 'normal',
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
			<div className="modal-box bg-base-100 flex h-[90vh] w-11/12 max-w-[90vw] flex-col overflow-hidden rounded-2xl p-0 shadow-2xl">
				<ModalHeader
					title="Table"
					onClose={() => {
						requestClose();
					}}
				/>

				<div className="min-h-0 flex-1 overflow-auto p-4">
					<TableSurface {...tableProps} />
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</>
	);
}

export function MarkdownTable({ node: _node, ...tableProps }: TableProps & ExtraProps) {
	const { isBusy } = useContext(MarkdownCodeRendererContext);
	const [isZoomOpen, setIsZoomOpen] = useState(false);

	return (
		<div className="border-base-300 relative my-3 min-w-0 overflow-hidden rounded-lg border">
			<TableSurface
				{...tableProps}
				onZoom={
					isBusy
						? undefined
						: () => {
								setIsZoomOpen(true);
							}
				}
			/>

			{isZoomOpen ? (
				<ModalDialog
					isOpen={true}
					onClose={() => {
						setIsZoomOpen(false);
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
