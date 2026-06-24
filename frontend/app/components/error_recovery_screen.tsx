import { reloadFrontend, resetLocalFrontendStateAndReload } from '@/lib/frontend_error_reporter';

type ErrorRecoveryScreenProps = {
	title?: string;
	message?: string;
	technicalDetails?: string;
	showResetLocalState?: boolean;
	showReload?: boolean;
	showHome?: boolean;
	homeHref?: string;
	fullDocument?: boolean;
	onTryAgain?: () => void;
	onGoHome?: () => void;
};

function ErrorRecoveryContent({
	title = 'Something went wrong',
	message = 'The interface hit an unexpected error.',
	technicalDetails,
	showResetLocalState = false,
	showReload = true,
	showHome = true,
	homeHref = '/',
	onTryAgain,
	onGoHome,
}: ErrorRecoveryScreenProps) {
	const handleGoHome = () => {
		try {
			if (onGoHome) {
				onGoHome();
				return;
			}
		} catch {
			// Fall back to hard navigation below.
		}

		window.location.assign(homeHref);
	};

	return (
		<main className="flex min-h-screen w-full items-center justify-center p-6">
			<section className="border-base-300 bg-base-100 w-full max-w-3xl rounded-lg border p-6 shadow-sm">
				<h1 className="mb-3 text-2xl font-semibold">{title}</h1>

				<p className="text-base-content/80 mb-6">{message}</p>

				<div className="flex flex-wrap gap-2">
					{onTryAgain && (
						<button type="button" className="btn btn-primary" onClick={onTryAgain}>
							Try again
						</button>
					)}

					{showHome && (
						<button type="button" className="btn" onClick={handleGoHome}>
							Go home
						</button>
					)}

					{showReload && (
						<button type="button" className="btn" onClick={reloadFrontend}>
							Reload UI
						</button>
					)}

					{showResetLocalState && (
						<button type="button" className="btn btn-error" onClick={resetLocalFrontendStateAndReload}>
							Reset local UI state
						</button>
					)}
				</div>

				{technicalDetails && (
					<details className="mt-6">
						<summary className="cursor-pointer">Technical details</summary>
						<pre className="bg-base-200 mt-3 max-h-96 w-full overflow-auto rounded-sm p-4 text-xs">
							<code>{technicalDetails}</code>
						</pre>
					</details>
				)}
			</section>
		</main>
	);
}

export function ErrorRecoveryScreen(props: ErrorRecoveryScreenProps) {
	const content = <ErrorRecoveryContent {...props} />;

	if (!props.fullDocument) {
		return content;
	}

	return (
		<html lang="en" suppressHydrationWarning>
			<head>
				<meta charSet="utf-8" />
				<meta name="viewport" content="width=device-width, initial-scale=1" />
				<title>FlexiGPT error</title>
			</head>
			<body className="m-0 h-full overflow-hidden p-0 antialiased">{content}</body>
		</html>
	);
}
