// const isExternalFileLikeDrop = (e: DragEvent) => {
// 	const dt = e.dataTransfer;
// 	if (!dt) return false;
// 	const types = Array.from(dt.types || []);
// 	return types.includes('text/uri-list') || types.includes('Files');
// };

const onDragEnter = (e: DragEvent) => {
	e.preventDefault();
	if (e.dataTransfer) {
		e.dataTransfer.dropEffect = 'copy';
	}
};

const onDragOver = (e: DragEvent) => {
	e.preventDefault();
	if (e.dataTransfer) {
		e.dataTransfer.dropEffect = 'copy';
	}
};

const onDrop = (e: DragEvent) => {
	e.preventDefault();
	// console.log('DROP fired', e.dataTransfer?.files, e.dataTransfer?.types);
};

export function installDropGuard(el: HTMLElement) {
	el.addEventListener('dragenter', onDragEnter);
	el.addEventListener('dragover', onDragOver);
	el.addEventListener('drop', onDrop);
	window.addEventListener('dragenter', onDragEnter);
	window.addEventListener('dragover', onDragOver);
	window.addEventListener('drop', onDrop);
	return () => {
		el.removeEventListener('dragenter', onDragEnter);
		el.removeEventListener('dragover', onDragOver);
		el.removeEventListener('drop', onDrop);
		window.removeEventListener('dragenter', onDragEnter);
		window.removeEventListener('dragover', onDragOver);
		window.removeEventListener('drop', onDrop);
	};
}
