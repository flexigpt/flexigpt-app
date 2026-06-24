export const MessageEnterValidURL = 'Please enter a valid URL, for example: https://example.com';

const isValidIPv4 = (hostname: string): boolean => {
	// Basic IPv4: 4 dot-separated decimal octets, each 0–255
	if (!/^\d{1,3}(\.\d{1,3}){3}$/.test(hostname)) return false;

	return hostname.split('.').every(part => {
		const n = Number(part);
		return n >= 0 && n <= 255;
	});
};

const isValidDomainName = (hostname: string): boolean => {
	// Disallow trailing dot: "example.com."
	if (hostname.endsWith('.')) return false;

	const labels = hostname.split('.');

	// Require at least two labels: "example.com"
	if (labels.length < 2) return false;

	// Max total length per RFC
	if (hostname.length > 253) return false;

	for (const label of labels) {
		// Reject empty labels → catches "..", leading ".", trailing "."
		if (!label.length) return false;

		// Allowed chars: A-Z, a-z, 0-9, hyphen (punycode: xn--... is fine)
		if (!/^[a-zA-Z0-9-]+$/.test(label)) return false;

		// Cannot start/end with hyphen
		if (label.startsWith('-') || label.endsWith('-')) return false;

		// Per RFC 1035: label length 1–63
		if (label.length > 63) return false;
	}

	return true;
};

/**
 * @public
 */
export const normalizeUrl = (value: string): string | null => {
	const raw = value.trim();
	if (!raw) return null;

	if (/\s/.test(raw)) {
		throw new Error('Only one URL is allowed at a time.');
	}

	// If user didn't specify a scheme, default to https
	const hasAnyScheme = /^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(raw);
	const candidate = hasAnyScheme ? raw : `https://${raw}`;

	let parsed: URL;
	try {
		parsed = new URL(candidate);
	} catch {
		throw new Error(MessageEnterValidURL);
	}

	// Only allow HTTP / HTTPS
	if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
		throw new Error('Only HTTP and HTTPS URLs are supported.');
	}

	const hostname = parsed.hostname;
	if (!hostname) {
		throw new Error(MessageEnterValidURL);
	}

	// Optional: validate port (if present)
	if (parsed.port) {
		const portNum = Number(parsed.port);
		if (!Number.isInteger(portNum) || portNum < 1 || portNum > 65535) {
			throw new Error('Please enter a valid port number (1–65535).');
		}
	}

	const lowerHost = hostname.toLowerCase();
	const isLocalhost = lowerHost === 'localhost';
	const isIPv4 = isValidIPv4(hostname);

	// For http/https, if URL gave us a colon in hostname, it's an IPv6 literal.
	// (Port is not part of `hostname`.)
	const isIPv6 = hostname.includes(':');

	const isDomain = isValidDomainName(hostname);

	// Accept only: localhost, IPv4, IPv6 literal, or well-formed domain
	if (!isLocalhost && !isIPv4 && !isIPv6 && !isDomain) {
		throw new Error('Please enter a full domain (for example, example.com).');
	}

	// At this point the URL is syntactically valid, scheme is http/https,
	// hostname is acceptable, and optional port is sane.
	return parsed.href;
};

interface UrlValidationResult {
	normalized: string | null;
	error: string | null;
}

interface ValidateUrlOptions {
	/**
	 * If true, an empty/whitespace-only value is treated as an error.
	 * Default: false (empty is allowed, no error).
	 */
	required?: boolean;

	/**
	 * Optional custom message when required and value is empty.
	 */
	requiredMessage?: string;
}

export const validateUrlForInput = (
	value: string,
	input?: HTMLInputElement | null,
	options?: ValidateUrlOptions
): UrlValidationResult => {
	const trimmed = value.trim();

	// Handle empty value
	if (!trimmed) {
		if (options?.required) {
			return {
				normalized: null,
				error: options.requiredMessage ?? 'Please enter a URL.',
			};
		}
		// Not required → no error, no normalized value
		return { normalized: null, error: null };
	}

	// For explicit http(s) URLs, let the browser validate first.
	if (/^https?:\/\//i.test(trimmed) && input && !input.validity.valid) {
		return {
			normalized: null,
			error: MessageEnterValidURL,
		};
	}

	try {
		const normalized = normalizeUrl(trimmed);
		return { normalized, error: null };
	} catch (err) {
		return {
			normalized: null,
			error: (err as Error).message,
		};
	}
};

export type FieldErrorState<FormState> = {
	[K in keyof FormState]?: string;
};

export function createUrlFieldChangeHandler<FormState extends Record<string, unknown>>(
	fieldName: Extract<keyof FormState, string>,
	setFormData: React.Dispatch<React.SetStateAction<FormState>>,
	setErrors: React.Dispatch<React.SetStateAction<FieldErrorState<FormState>>>,
	options?: ValidateUrlOptions
) {
	return (e: React.ChangeEvent<HTMLInputElement>) => {
		const value = e.target.value;

		setFormData(prev => ({
			...prev,
			[fieldName]: value as FormState[keyof FormState],
		}));

		setErrors(prev => {
			const { error } = validateUrlForInput(value, e.target, options);
			const next: FieldErrorState<FormState> = { ...prev };

			if (error) {
				next[fieldName] = error;
			} else {
				// oxlint-disable-next-line @typescript-eslint/no-dynamic-delete
				delete next[fieldName];
			}

			return next;
		});
	};
}
