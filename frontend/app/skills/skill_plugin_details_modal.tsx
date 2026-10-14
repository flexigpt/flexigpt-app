import { useEffect, useState } from 'react';

import type { PluginListItem, PluginView } from '@/spec/plugin';
import type { StoreSkillListItem } from '@/spec/skill';

import { skillManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { ModalSection } from '@/components/modal/modal_section';

interface SkillPluginDetailsModalProps {
	isOpen: boolean;
	plugin: PluginListItem | null;
	skills: StoreSkillListItem[];
	onClose: () => void;
}

export function SkillPluginDetailsModal({ isOpen, plugin, skills, onClose }: SkillPluginDetailsModalProps) {
	const [view, setView] = useState<PluginView | null>(null);
	const [error, setError] = useState('');

	useEffect(() => {
		if (!isOpen || !plugin) {
			return;
		}

		let cancelled = false;

		void skillManagementAPI
			.getSkillPlugin(plugin.ref)
			.then(next => {
				if (!cancelled) {
					// oxlint-disable-next-line promise/no-callback-in-promise
					setView(next);
				}
			})
			.catch((cause: unknown) => {
				if (!cancelled) {
					setError(cause instanceof Error ? cause.message : 'Skill Plugin details could not be loaded.');
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
			{!view && !error ? <Loader text="Loading Skill Plugin details..." /> : null}
			{error ? <div className="alert alert-warning rounded-2xl text-sm">{error}</div> : null}

			<ModalSection title="Plugin">
				<ManagementInfoGrid>
					<ManagementInfoRow label="Display name">{plugin.displayName || plugin.name}</ManagementInfoRow>
					<ManagementInfoRow label="Logical name" mono>
						{plugin.name}
					</ManagementInfoRow>
					<ManagementInfoRow label="Root" mono>
						{plugin.ref.rootID}
					</ManagementInfoRow>
					<ManagementInfoRow label="Source" mono>
						{plugin.sourceID}
					</ManagementInfoRow>
					<ManagementInfoRow label="State">{plugin.state}</ManagementInfoRow>
					<ManagementInfoRow label="Enabled">{plugin.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Built-in">{plugin.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Baseline">{plugin.baseline ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Revision">{plugin.revision}</ManagementInfoRow>
					<ManagementInfoRow label="Description">{plugin.description || '—'}</ManagementInfoRow>
				</ManagementInfoGrid>
			</ModalSection>

			<ModalSection title="Resolved Skills">
				<div className="mb-3 text-sm">
					{skills.length} Skill Artifact{skills.length === 1 ? '' : 's'} currently resolve through this Plugin.
				</div>

				<div className="space-y-2">
					{skills.map(skill => (
						<div
							key={`${skill.ref.rootID}:${skill.ref.artifactID}`}
							className="border-base-content/10 rounded-2xl border p-3"
						>
							<div className="flex flex-wrap items-center gap-2">
								<span className="font-medium">{skill.displayName || skill.name}</span>
								<MetadataPill label="State">{skill.state}</MetadataPill>
								<MetadataPill label="Revision">{skill.revision}</MetadataPill>
								{skill.managed ? <MetadataPill>Managed</MetadataPill> : null}
								{skill.builtIn ? <MetadataPill>Built-in</MetadataPill> : null}
							</div>
							<div className="text-base-content/60 mt-1 font-mono text-xs">
								{skill.ref.rootID}/{skill.ref.artifactID}
							</div>
						</div>
					))}

					{skills.length === 0 ? (
						<div className="text-base-content/60 rounded-2xl border border-dashed p-3 text-sm">
							No Skill Artifact currently resolves through this Plugin.
						</div>
					) : null}
				</div>
			</ModalSection>

			{view ? (
				<ModalSection title="Declared membership">
					<div className="text-base-content/70 mb-2 text-sm">
						{view.members.length} declared member{view.members.length === 1 ? '' : 's'}.
					</div>

					<div className="space-y-1">
						{view.members.map((member, index) => (
							<div key={`${member.type}:${member.name}:${index}`} className="bg-base-100 rounded-xl px-3 py-2 text-xs">
								<span className="font-mono">{member.type}</span>
								{member.name ? ` / ${member.name}` : ''}
								{member.scope ? ` / scope: ${member.scope}` : ''}
							</div>
						))}

						{view.members.length === 0 ? (
							<div className="text-base-content/60 px-1 text-sm">No declared members.</div>
						) : null}
					</div>
				</ModalSection>
			) : null}
		</ManagementDetailsModal>
	);
}
