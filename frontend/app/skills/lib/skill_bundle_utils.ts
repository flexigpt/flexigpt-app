import type { Skill, SkillBundle } from '@/spec/skill';

export interface BundleData {
	bundle: SkillBundle;
	skills: Skill[];
}

export function sortBundleData(bundleData: BundleData[]): BundleData[] {
	return [...bundleData].sort((a, b) => {
		if (a.bundle.isBuiltIn !== b.bundle.isBuiltIn) {
			return a.bundle.isBuiltIn ? -1 : 1;
		}

		const aName = (a.bundle.displayName ?? a.bundle.slug).toLowerCase();
		const bName = (b.bundle.displayName ?? b.bundle.slug).toLowerCase();

		return aName.localeCompare(bName);
	});
}
