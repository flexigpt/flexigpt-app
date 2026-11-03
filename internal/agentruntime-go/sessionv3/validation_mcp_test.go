package sessionv3

import "testing"

func TestValidateDefaultsAcceptsSelectedMCPToolConstraints(
	t *testing.T,
) {
	t.Parallel()

	approval := "ask"
	execution := "manual"
	defaults := SessionDefaults{
		Skills: []ArtifactSelection{{
			RootID:     "root",
			ArtifactID: "skill",
		}},
		ActiveSkills: []ArtifactSelection{{
			RootID:     "root",
			ArtifactID: "skill",
		}},
		MCP: []MCPServerSelection{{
			Server: ArtifactSelection{
				RootID:     "root",
				ArtifactID: "mcp",
			},
			ToolExposure: MCPToolExposureSelected,
			SelectedTools: []MCPToolSelection{{
				ToolName:       "lookup",
				ExpectedDigest: "sha256:test",
				ApprovalRule:   &approval,
				ExecutionMode:  &execution,
			}},
		}},
	}

	if err := validateDefaults(defaults); err != nil {
		t.Fatalf("validateDefaults: %v", err)
	}
}

func TestValidateDefaultsRejectsActiveSkillOutsideEnabledSelection(
	t *testing.T,
) {
	t.Parallel()

	err := validateDefaults(SessionDefaults{
		Skills: []ArtifactSelection{{
			RootID:     "root",
			ArtifactID: "enabled",
		}},
		ActiveSkills: []ArtifactSelection{{
			RootID:     "root",
			ArtifactID: "not-enabled",
		}},
	})
	if err == nil {
		t.Fatal("validateDefaults accepted an active Skill outside enabled Skills")
	}
}
