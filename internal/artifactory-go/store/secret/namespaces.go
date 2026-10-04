package secret

import (
	"fmt"

	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func namespaceSet(values []overlayModel.Namespace) (map[overlayModel.Namespace]struct{}, error) {
	output := make(map[overlayModel.Namespace]struct{}, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("secret namespace %d: %w", index, err)
		}
		if _, duplicate := output[value]; duplicate {
			return nil, fmt.Errorf("%w: duplicate secret namespace %q", spec.ErrConflict, value)
		}
		output[value] = struct{}{}
	}
	return output, nil
}

func requireNamespace(namespaces map[overlayModel.Namespace]struct{}, value overlayModel.Namespace) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if _, found := namespaces[value]; !found {
		return fmt.Errorf("%w: secret namespace %q is not registered", spec.ErrUnsupported, value)
	}
	return nil
}
