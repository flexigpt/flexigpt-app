import type { ErrorInfo, ReactElement } from 'react';
import { Component } from 'react';

interface MdErrProps {
	source: string; // <-- raw markdown
	children: ReactElement;
}
interface MdErrState {
	failedSource: string | null;
}

export class MdErrorBoundary extends Component<MdErrProps, MdErrState> {
	public state: MdErrState = { failedSource: null };

	public static getDerivedStateFromProps(props: MdErrProps, state: MdErrState): Partial<MdErrState> | null {
		if (state.failedSource !== null && state.failedSource !== props.source) {
			return { failedSource: null };
		}
		return null;
	}

	public componentDidCatch(err: Error, info: ErrorInfo) {
		console.error('Markdown render error', err, info);
		this.setState({ failedSource: this.props.source });
	}

	public render() {
		if (this.state.failedSource !== null) {
			return (
				<pre className="bg-base-200 text-error overflow-x-auto rounded-sm p-2">
					{`Failed to render message. Showing raw content below:

${this.props.source}`}
				</pre>
			);
		}
		return this.props.children;
	}
}
