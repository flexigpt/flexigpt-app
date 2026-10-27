import type { ArtifactRef } from '@/spec/artifact';

import type { AgentCatalogOption } from '@/apis/agent_management';

import type { DefaultArtifactOption } from '@/components/managementui/default_artifact_control';
import { DefaultArtifactControl } from '@/components/managementui/default_artifact_control';

interface DefaultAgentControlProps {
	agents: AgentCatalogOption[];
	defaultAgent: ArtifactRef | null;
	pending?: boolean;
	onChange: (ref: ArtifactRef | undefined) => void;
}

export function DefaultAgentControl({ agents, defaultAgent, pending = false, onChange }: DefaultAgentControlProps) {
	const options: DefaultArtifactOption[] = agents.map(agent => ({
		ref: agent.ref,
		displayName: agent.displayName,
		isSelectable: agent.isSelectable,
		availabilityReason: agent.availabilityReason,
	}));

	return (
		<DefaultArtifactControl
			label="Default Agent"
			automaticLabel="Use automatic Agent"
			options={options}
			defaultRef={defaultAgent}
			pending={pending}
			onChange={onChange}
		/>
	);
}
