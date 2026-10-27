import { useEffect, useState } from 'react';
import { FiEye } from 'react-icons/fi';

import type { PluginListItem } from '@/spec/plugin';
import type { StoreSkillListItem } from '@/spec/skill';

import { skillManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { ManagementPluginSource } from '@/components/managementui/management_plugin_source';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalSection } from '@/components/modal/modal_section';

import { SkillDetailsModal, skillDetailsSubjectFromStoreSkill } from '@/skills/skill_details_modal';

interface SkillPluginDetailsModalProps {
	isOpen: boolean;
	plugin: PluginListItem | null;
	onClose: () => void;
}

export function SkillPluginDetailsModal({ isOpen, plugin, onClose }: SkillPluginDetailsModalProps) {
	const [error, setError] = useState('');
	const [resolvedSkills, setResolvedSkills] = useState<StoreSkillListItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [skillToView, setSkillToView] = useState<StoreSkillListItem | null>(null);

	useEffect(() => {
		if (!isOpen || !plugin) {
			return;
		}

		let cancelled = false;
		// oxlint-disable-next-line react/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setLoading(true);
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setError('');
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setResolvedSkills([]);

		void skillManagementAPI
			.getSkillPluginSkills(plugin.ref)
			.then(next => {
				if (!cancelled) {
					setResolvedSkills(next.skills);
					setError(next.diagnostics.join('\n'));
				}
			})
			.catch((cause: unknown) => {
				if (!cancelled) {
					setError(cause instanceof Error ? cause.message : 'Skill Plugin details could not be loaded.');
				}
			})
			.finally(() => {
				if (!cancelled) {
					setLoading(false);
				}
			});

		return () => {
			cancelled = true;
		};
	}, [isOpen, plugin]);

	if (!isOpen || !plugin) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen
			onClose={onClose}
			title="Skill Plugin Details"
			modalKey={`skill-plugin:${plugin.ref.rootID}:${plugin.ref.artifactID}:${plugin.revision}`}
		>
			{loading ? <Loader text="Loading Skill Plugin details..." /> : null}
			{error ? <div className="alert alert-warning rounded-2xl text-sm">{error}</div> : null}

			<ModalSection title="Plugin">
				<ManagementInfoGrid>
					<ManagementInfoRow label="Display name">{plugin.displayName || plugin.name}</ManagementInfoRow>
					<ManagementInfoRow label="State">{plugin.state}</ManagementInfoRow>
					<ManagementInfoRow label="Enabled">{plugin.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Built-in">{plugin.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Baseline">{plugin.baseline ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Description">{plugin.description || '—'}</ManagementInfoRow>
				</ManagementInfoGrid>
			</ModalSection>

			<ManagementPluginSource
				label="skill plugin YAML"
				pluginName={plugin.name}
				load={() => skillManagementAPI.getSkillPlugin(plugin.ref)}
			/>

			<ModalSection title="Skills">
				<div className="mb-3 text-sm">
					{resolvedSkills.length} Skill{resolvedSkills.length === 1 ? '' : 's'} available through this Plugin.
				</div>

				<div className="space-y-2">
					{resolvedSkills.map(skill => (
						<ManagementItemCard
							key={`${skill.ref.rootID}:${skill.ref.artifactID}`}
							title={skill.displayName || skill.name}
							description={skill.description}
							status={
								<>
									<StatusBadge tone={skill.enabled ? 'success' : 'neutral'}>
										{skill.enabled ? 'Enabled' : 'Disabled'}
									</StatusBadge>
									{skill.managed ? <StatusBadge>Managed</StatusBadge> : null}
									{skill.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
								</>
							}
							actions={
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										setSkillToView(skill);
									}}
								>
									<FiEye size={15} />
									View
								</button>
							}
						/>
					))}

					{!loading && !error && resolvedSkills.length === 0 ? (
						<div className="text-base-content/60 rounded-2xl border border-dashed p-3 text-sm">
							No Skill currently resolves through this Plugin.
						</div>
					) : null}
				</div>
			</ModalSection>

			<SkillDetailsModal
				isOpen={skillToView !== null}
				skill={
					skillToView
						? skillDetailsSubjectFromStoreSkill(skillToView, {
								sourceLabel: plugin.displayName || plugin.name,
							})
						: null
				}
				onClose={() => {
					setSkillToView(null);
				}}
				showPreview
			/>
		</ManagementDetailsModal>
	);
}
