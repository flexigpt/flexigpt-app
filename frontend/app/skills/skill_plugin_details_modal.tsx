import { useEffect, useState } from 'react';

import type { PluginListItem, PluginView } from '@/spec/plugin';
import type { StoreSkillListItem } from '@/spec/skill';

import { skillManagementAPI } from '@/apis/baseapi';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { ModalSection } from '@/components/modal/modal_section';

import { canManageSkillPlugin } from '@/skills/lib/skill_management_utils';

interface SkillPluginDetailsModalProps {
	isOpen: boolean;
	plugin: PluginListItem | null;
	onClose: () => void;
	onChanged?: () => Promise<void>;
}

export function SkillPluginDetailsModal({ isOpen, plugin, onClose, onChanged }: SkillPluginDetailsModalProps) {
	const [view, setView] = useState<PluginView | null>(null);
	const [error, setError] = useState('');
	const [resolvedSkills, setResolvedSkills] = useState<StoreSkillListItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [memberToRemove, setMemberToRemove] = useState<{ index: number; revision: number; label: string } | null>(null);
	const [removing, setRemoving] = useState(false);

	useEffect(() => {
		if (!isOpen || !plugin) {
			return;
		}

		let cancelled = false;

		void skillManagementAPI
			.getSkillPluginSkills(plugin.ref)
			.then(next => {
				if (!cancelled) {
					setView(next.plugin);
					setResolvedSkills(next.skills);
					if (next.diagnostics.length > 0) {
						setError(next.diagnostics.join('\n'));
					}
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
					{resolvedSkills.length} Skill Artifact{resolvedSkills.length === 1 ? '' : 's'} loaded for this Plugin.
				</div>

				<div className="space-y-2">
					{resolvedSkills.map(skill => (
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

					{!loading && !error && resolvedSkills.length === 0 ? (
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
								{member.locator?.path ? ` / path: ${member.locator.path}` : ''}
								{member.locator?.url ? ` / URL: ${member.locator.url}` : ''}
								{canManageSkillPlugin(plugin) ? (
									<button
										type="button"
										className="btn btn-xs btn-ghost ml-2 rounded-lg"
										disabled={removing}
										onClick={() => {
											setMemberToRemove({
												index,
												revision: view.artifact.revision,
												label: `${member.type}/${member.name || '(unnamed declaration)'}`,
											});
										}}
									>
										Remove declaration
									</button>
								) : null}
							</div>
						))}

						{view.members.length === 0 ? (
							<div className="text-base-content/60 px-1 text-sm">No declared members.</div>
						) : null}
					</div>
				</ModalSection>
			) : null}
			<DeleteConfirmationModal
				isOpen={memberToRemove !== null}
				onClose={() => {
					if (!removing) {
						setMemberToRemove(null);
					}
				}}
				onConfirm={async () => {
					if (!memberToRemove || removing) {
						return;
					}
					setRemoving(true);
					try {
						await skillManagementAPI.removeSkillPluginMember({
							plugin: plugin.ref,
							expectedRevision: memberToRemove.revision,
							index: memberToRemove.index,
						});
						setMemberToRemove(null);
						try {
							const fresh = await skillManagementAPI.getSkillPluginSkills(plugin.ref);
							setView(fresh.plugin);
							setResolvedSkills(fresh.skills);
							setError(fresh.diagnostics.join('\n'));
							await onChanged?.();
						} catch (cause) {
							setError(
								`Declaration removed, but refresh failed: ${cause instanceof Error ? cause.message : 'unknown error'}`
							);
						}
					} catch (cause) {
						setError(cause instanceof Error ? cause.message : 'Declaration could not be removed.');
					} finally {
						setRemoving(false);
					}
				}}
				title="Remove Plugin Declaration"
				message={`Remove "${memberToRemove?.label || ''}"? A selector declaration can affect several resolved Skills. Artifacts and source files are retained.`}
				confirmButtonText="Remove"
			/>
		</ManagementDetailsModal>
	);
}
