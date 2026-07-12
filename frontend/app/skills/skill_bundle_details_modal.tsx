import type { Skill, SkillBundle } from '@/spec/skill';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { ModalSection } from '@/components/modal/modal_section';

import { getSkillInsertCounts, getSkillInsertDescription, skillHasResources } from '@/skills/lib/skill_artifact_utils';

interface SkillBundleDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: SkillBundle | null;
	skills: Skill[];
}

export function SkillBundleDetailsModal({ isOpen, onClose, bundle, skills }: SkillBundleDetailsModalProps) {
	const skillCounts = getSkillInsertCounts(skills);

	const totalSkills = skills.length;
	const resourceSkillCount = skills.filter(s => {
		return skillHasResources(s);
	}).length;
	if (!isOpen || !bundle) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="Skill Bundle Details"
			modalKey={`skill-bundle:${bundle.id}:${String(bundle.modifiedAt)}`}
		>
			<ModalSection title="Bundle metadata">
				<ManagementInfoGrid>
					<ManagementInfoRow label="Display Name">{bundle.displayName || '—'}</ManagementInfoRow>
					<ManagementInfoRow label="Slug" mono>
						{bundle.slug}
					</ManagementInfoRow>
					<ManagementInfoRow label="ID" mono>
						{bundle.id}
					</ManagementInfoRow>
					<ManagementInfoRow label="Built-in">{bundle.isBuiltIn ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Enabled">{bundle.isEnabled ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Description">
						<span className="whitespace-pre-wrap">{bundle.description || '—'}</span>
					</ManagementInfoRow>
					<ManagementInfoRow label="Created">{String(bundle.createdAt)}</ManagementInfoRow>
					<ManagementInfoRow label="Modified">{String(bundle.modifiedAt)}</ManagementInfoRow>
				</ManagementInfoGrid>
			</ModalSection>

			<ModalSection title="Skill summary">
				<ManagementInfoGrid>
					<ManagementInfoRow label="Total skills">{totalSkills}</ManagementInfoRow>
					<ManagementInfoRow label="Instruction skills">
						<div className="flex flex-wrap items-center gap-2">
							<MetadataPill label="Count">{skillCounts.instructions}</MetadataPill>
							<span className="text-base-content/70 text-xs">{getSkillInsertDescription('instructions')}</span>
						</div>
					</ManagementInfoRow>
					<ManagementInfoRow label="User-message skills">
						<div className="flex flex-wrap items-center gap-2">
							<MetadataPill label="Count">{skillCounts['user-message']}</MetadataPill>
							<span className="text-base-content/70 text-xs">{getSkillInsertDescription('user-message')}</span>
						</div>
					</ManagementInfoRow>
					<ManagementInfoRow label="Skills with resources">
						<div className="flex flex-wrap items-center gap-2">
							<MetadataPill label="Count">{resourceSkillCount}</MetadataPill>
							<span className="text-base-content/70 text-xs">
								Resources are regular files under each skill directory.
							</span>
						</div>
					</ManagementInfoRow>
					<ManagementInfoRow label="Usage note">
						Instruction skills affect session state. User-message skills render into the composer or user message body
						and are not active session skills.
					</ManagementInfoRow>
					<ManagementInfoRow label="Prompt migration note">
						A prompt-like template should be a filesystem skill whose <span className="font-mono">SKILL.md</span>
						frontmatter contains <span className="font-mono">insert: user-message</span>. Its declared arguments replace
						the old prompt variable form.
					</ManagementInfoRow>
					<ManagementInfoRow label="Resource note">
						Managed creation writes only SKILL.md. Add extra resources to the skill folder, then re-enable the skill or
						restart the app to refresh runtime metadata.
					</ManagementInfoRow>
				</ManagementInfoGrid>
			</ModalSection>
		</ManagementDetailsModal>
	);
}
