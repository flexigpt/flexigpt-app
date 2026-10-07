import type { Skill, SkillPlugin } from '@/spec/skill';

export interface PluginData {
	plugin: SkillPlugin;
	skills: Skill[];
	runtimeMetadataLoaded?: boolean;
	skillLoadError?: string;
}

export function sortPluginData(pluginData: PluginData[]): PluginData[] {
	return [...pluginData].toSorted((a, b) => {
		if (a.plugin.isBuiltIn !== b.plugin.isBuiltIn) {
			return a.plugin.isBuiltIn ? -1 : 1;
		}

		const aName = (a.plugin.displayName ?? a.plugin.slug).toLowerCase();
		const bName = (b.plugin.displayName ?? b.plugin.slug).toLowerCase();

		return aName.localeCompare(bName);
	});
}
