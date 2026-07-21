package definition

import (
	"maps"
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
