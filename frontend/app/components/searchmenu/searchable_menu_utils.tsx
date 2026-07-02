import type { Dispatch, SetStateAction } from 'react';
import { useEffect, useState } from 'react';

interface SearchableTextField {
	value?: string | string[] | null | undefined;
	weight?: number;
}

interface RankedCandidate<T> {
	item: T;
	score: number;
	bestFieldIndex: number;
	originalIndex: number;
}

interface RankSearchableItemsArgs<T> {
	query: string;
	getKey: (item: T) => string;
	getFields: (item: T) => SearchableTextField[];
	fallbackCompare?: (a: T, b: T) => number;
}

function normalizeSearchText(value: string): string {
	return value
		.normalize('NFKD')
		.replaceAll(/\p{Diacritic}/gu, '')
		.toLowerCase()
		.trim();
}

function splitQueryTokens(query: string): string[] {
	return normalizeSearchText(query)
		.split(/\s+/)
		.map(token => token.trim())
		.filter(Boolean);
}

function fieldValues(field: SearchableTextField): string[] {
	if (Array.isArray(field.value)) {
		return field.value.map(value => value?.trim()).filter((value): value is string => Boolean(value));
	}
	const value = field.value?.trim();
	return value ? [value] : [];
}

function scoreText(rawText: string, normalizedQuery: string, tokens: string[]): number {
	const text = normalizeSearchText(rawText);
	if (!text || !normalizedQuery) {
		return 0;
	}

	if (text === normalizedQuery) {
		return 1000;
	}

	if (text.startsWith(normalizedQuery)) {
		return 850;
	}

	if (text.split(/[\s/_@.:#-]+/).some(part => part.startsWith(normalizedQuery))) {
		return 725;
	}

	if (text.includes(normalizedQuery)) {
		return 575;
	}

	if (tokens.length > 1 && tokens.every(token => text.includes(token))) {
		return 425;
	}

	return 0;
}

const searchableMenuCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

export function isSearchQueryActive(query: string): boolean {
	return normalizeSearchText(query).length > 0;
}

export function rankSearchableItems<T>(
	items: readonly T[],
	{ query, getKey, getFields, fallbackCompare }: RankSearchableItemsArgs<T>
): T[] {
	const normalizedQuery = normalizeSearchText(query);
	if (!normalizedQuery) {
		return [...items];
	}

	const tokens = splitQueryTokens(query);
	const ranked: RankedCandidate<T>[] = [];

	for (const [originalIndex, item] of items.entries()) {
		const fields = getFields(item);
		let bestScore = 0;
		let bestFieldIndex = Number.POSITIVE_INFINITY;

		for (const [fieldIndex, field] of fields.entries()) {
			const weight = field.weight ?? Math.max(1, fields.length - fieldIndex);
			for (const value of fieldValues(field)) {
				const score = scoreText(value, normalizedQuery, tokens) * weight;
				if (score > bestScore) {
					bestScore = score;
					bestFieldIndex = fieldIndex;
				}
			}
		}

		if (bestScore > 0) {
			ranked.push({
				item,
				score: bestScore,
				bestFieldIndex,
				originalIndex,
			});
		}
	}

	return ranked
		.toSorted((a, b) => {
			const scoreCompare = b.score - a.score;
			if (scoreCompare !== 0) {
				return scoreCompare;
			}

			const fieldCompare = a.bestFieldIndex - b.bestFieldIndex;
			if (fieldCompare !== 0) {
				return fieldCompare;
			}

			const fallback =
				fallbackCompare?.(a.item, b.item) ?? searchableMenuCollator.compare(getKey(a.item), getKey(b.item));
			if (fallback !== 0) {
				return fallback;
			}

			return a.originalIndex - b.originalIndex;
		})
		.map(candidate => candidate.item);
}

export function focusFirstSearchableMenuItem(root: HTMLElement | null | undefined): boolean {
	const first = root?.querySelector<HTMLElement>(
		'[data-searchable-menu-item="true"]:not([aria-disabled="true"]):not(:disabled)'
	);
	if (!first) {
		return false;
	}

	first.focus();
	return true;
}

export function useSearchableMenuState(open: boolean): [string, Dispatch<SetStateAction<string>>] {
	const [query, setQuery] = useState('');

	useEffect(() => {
		if (!open) {
			// oxlint-disable-next-line jsreact-hooks/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
			setQuery('');
		}
	}, [open]);

	return [query, setQuery];
}
