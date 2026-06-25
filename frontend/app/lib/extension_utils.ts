type LanguageMap = Record<string, { extension: string; mimeType: string }>;

export const ProgrammingLanguages: LanguageMap = {
	javascript: { extension: '.js', mimeType: 'text/javascript' },
	jsx: { extension: '.jsx', mimeType: 'text/javascript' },
	python: { extension: '.py', mimeType: 'text/x-python' },
	java: { extension: '.java', mimeType: 'text/x-java-source' },
	c: { extension: '.c', mimeType: 'text/x-c' },
	cpp: { extension: '.cpp', mimeType: 'text/x-c++' },
	csharp: { extension: '.cs', mimeType: 'text/x-csharp' },
	ruby: { extension: '.rb', mimeType: 'text/x-ruby' },
	php: { extension: '.php', mimeType: 'application/php' },
	swift: { extension: '.swift', mimeType: 'text/x-swift' },
	'objective-c': { extension: '.m', mimeType: 'text/x-objectivec' },
	kotlin: { extension: '.kt', mimeType: 'text/x-kotlin' },
	typescript: { extension: '.ts', mimeType: 'application/typescript' },
	tsx: { extension: '.tsx', mimeType: 'application/typescript' },
	go: { extension: '.go', mimeType: 'text/x-go' },
	perl: { extension: '.pl', mimeType: 'text/x-perl' },
	rust: { extension: '.rs', mimeType: 'text/x-rustsrc' },
	scala: { extension: '.scala', mimeType: 'text/x-scala' },
	haskell: { extension: '.hs', mimeType: 'text/x-haskell' },
	lua: { extension: '.lua', mimeType: 'text/x-lua' },
	shell: { extension: '.sh', mimeType: 'application/x-sh' },
	bash: { extension: '.sh', mimeType: 'text/x-bash' },
	sql: { extension: '.sql', mimeType: 'application/sql' },
	html: { extension: '.html', mimeType: 'text/html' },
	css: { extension: '.css', mimeType: 'text/css' },
	json: { extension: '.json', mimeType: 'application/json' },
	toml: { extension: '.toml', mimeType: 'application/toml' },
	yaml: { extension: '.yaml', mimeType: 'application/yaml' },
	dart: { extension: '.dart', mimeType: 'application/dart' },
};

// Determine the file extension from the MIME type
export const MimeTypeMap: Record<string, string> = {
	'image/png': '.png',
	'image/jpeg': '.jpg',
	'image/gif': '.gif',
	'application/pdf': '.pdf',
	// Add more MIME types and extensions as needed
};
