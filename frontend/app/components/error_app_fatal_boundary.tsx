import type { ErrorInfo, ReactNode } from 'react';
import { Component } from 'react';

import { recordFrontendCrash, reportFrontendError } from '@/lib/frontend_error_reporter';

import { ErrorRecoveryScreen } from '@/components/error_recovery_screen';

interface AppFatalBoundaryProps {
	children: ReactNode;
}

interface AppFatalBoundaryState {
	error: Error | null;
	crashCount: number;
}

export class AppFatalBoundary extends Component<AppFatalBoundaryProps, AppFatalBoundaryState> {
	state: AppFatalBoundaryState = {
		error: null,
		crashCount: 0,
	};

	static getDerivedStateFromError(error: Error): Partial<AppFatalBoundaryState> {
		return { error };
	}

	componentDidCatch(error: Error, info: ErrorInfo) {
		const crashCount = recordFrontendCrash('react.app_fatal_boundary');

		this.setState({ crashCount });

		reportFrontendError(error, {
			phase: 'react.app_fatal_boundary',
			componentStack: info.componentStack as string | undefined,
		});
	}

	private handleTryAgain = () => {
		this.setState({
			error: null,
			crashCount: 0,
		});
	};

	render() {
		if (!this.state.error) {
			return this.props.children;
		}

		return (
			<ErrorRecoveryScreen
				fullDocument
				title="FlexiGPT interface crashed"
				message="The interface reached an unrecoverable state. You can go home or reload the UI."
				technicalDetails={this.state.error.stack || this.state.error.message}
				showResetLocalState={this.state.crashCount >= 2}
				onTryAgain={this.handleTryAgain}
			/>
		);
	}
}
