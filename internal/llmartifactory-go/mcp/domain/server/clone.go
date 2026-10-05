package server

import (
	"maps"
	"slices"
)

func cloneConfiguration(input ServerConfiguration) ServerConfiguration {
	output := input
	output.Install.Inputs = maps.Clone(input.Install.Inputs)
	for name, value := range output.Install.Inputs {
		value.Default = cloneStringPointer(value.Default)
		output.Install.Inputs[name] = value
	}
	output.Install.AllowEnvironment = slices.Clone(input.Install.AllowEnvironment)
	output.ConnectionProfiles = maps.Clone(input.ConnectionProfiles)
	for name, profile := range output.ConnectionProfiles {
		profile.Platforms = slices.Clone(profile.Platforms)
		if profile.Stdio != nil {
			value := *profile.Stdio
			value.Command = cloneStringPointer(value.Command)
			value.Env = maps.Clone(value.Env)
			value.RemoveEnv = slices.Clone(value.RemoveEnv)
			if value.Args != nil {
				args := slices.Clone(*value.Args)
				value.Args = &args
			}
			profile.Stdio = &value
		}
		if profile.HTTP != nil {
			value := *profile.HTTP
			value.URL = cloneStringPointer(value.URL)
			value.Headers = maps.Clone(value.Headers)
			value.RemoveHeaders = slices.Clone(value.RemoveHeaders)
			profile.HTTP = &value
		}
		output.ConnectionProfiles[name] = profile
	}
	if input.Policy != nil {
		value := *input.Policy
		output.Policy = &value
	}
	return output
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}
