package definition

import (
	"maps"
	"sort"
)

func cloneSelectors(input []Selector) []Selector {
	if input == nil {
		return nil
	}
	output := make([]Selector, len(input))
	for index, value := range input {
		output[index] = value
		output[index].Labels = cloneLabels(value.Labels)
	}
	return output
}

func cloneLabels(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	maps.Copy(output, input)
	return output
}

func sortedLabels(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	output := make(map[string]string, len(input))
	for _, key := range keys {
		output[key] = input[key]
	}
	return output
}
