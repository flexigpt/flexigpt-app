package sessionv3

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"cel.dev/cel-go/common/types/ref"
	"cel.dev/cel-go/common/types/traits"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// workflowActivationValue is the JSON-to-CEL boundary. It preserves integers
// where CEL has an exact integer representation.
func workflowActivationValue(value CanonicalValue) (any, error) {
	if len(value) == 0 {
		return nil, nil
	}
	if !value.Valid() {
		return nil, errors.New("workflow activation value is not canonical json")
	}
	decoded, err := jsonutil.DecodeJSONRawNumbers[any](json.RawMessage(value))
	if err != nil {
		return nil, err
	}
	return workflowActivationNode(decoded)
}

func workflowActivationNode(value any) (any, error) {
	switch typed := value.(type) {
	case json.Number:
		if integer, err := typed.Int64(); err == nil {
			return integer, nil
		}
		if integer, err := strconv.ParseUint(string(typed), 10, 64); err == nil {
			return integer, nil
		}
		number, err := typed.Float64()
		if err != nil || math.IsInf(number, 0) || math.IsNaN(number) {
			return nil, errors.New("workflow expression number exceeds the supported numeric range")
		}
		return number, nil

	case map[string]any:
		output := make(map[string]any, len(typed))
		for key, child := range typed {
			converted, err := workflowActivationNode(child)
			if err != nil {
				return nil, err
			}
			output[key] = converted
		}
		return output, nil

	case []any:
		output := make([]any, len(typed))
		for index, child := range typed {
			converted, err := workflowActivationNode(child)
			if err != nil {
				return nil, err
			}
			output[index] = converted
		}
		return output, nil

	default:
		return value, nil
	}
}

// canonicalWorkflowCELValue converts CEL collections through their typed
// iteration contracts. Runtime never persists CEL values or compiled programs.
func canonicalWorkflowCELValue(value ref.Val) (CanonicalValue, error) {
	if value == nil {
		return nil, errors.New("workflow expression returned no value")
	}

	if object, ok := value.(traits.Mapper); ok {
		output := make(map[string]CanonicalValue)
		iterator := object.Iterator()
		for iterator.HasNext().Value() == true {
			key := iterator.Next()
			name, ok := key.Value().(string)
			if !ok {
				return nil, errors.New("workflow expression object keys must be strings")
			}
			child, err := canonicalWorkflowCELValue(object.Get(key))
			if err != nil {
				return nil, err
			}
			output[name] = child
		}
		return canonicalWorkflowValue(output)
	}

	if list, ok := value.(traits.Lister); ok {
		output := make([]CanonicalValue, 0)
		iterator := list.Iterator()
		for iterator.HasNext().Value() == true {
			child, err := canonicalWorkflowCELValue(iterator.Next())
			if err != nil {
				return nil, err
			}
			output = append(output, child)
		}
		return canonicalWorkflowValue(output)
	}

	switch native := value.Value().(type) {
	case time.Duration:
		return canonicalWorkflowValue(native.String())
	case error:
		return nil, fmt.Errorf("workflow expression result: %w", native)
	default:
		return canonicalWorkflowValue(native)
	}
}

func validateWorkflowCallReceipt(
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	output *WorkItemOutput,
) error {
	if step.Call == nil {
		return nil
	}
	var receipt *ExternalReceipt
	if output != nil {
		receipt = output.Receipt
	}
	return validateWorkflowCallReceiptForTarget(
		program,
		state.PendingTarget,
		step,
		receipt,
	)
}

func validateWorkflowCallReceiptForTarget(
	program WorkflowProgram,
	targetName string,
	step WorkflowStep,
	receipt *ExternalReceipt,
) error {
	target, found := program.Targets[targetName]
	if !found {
		return errors.New("workflow call has no persisted selected target")
	}

	contracts := []*WorkflowCallableInterface{
		target.Contract,
		step.Call.Contract,
		step.Call.SetContract,
	}
	if effective := workflowTargetSetContract(program, *step.Call); effective != nil {
		contracts[2] = effective
	}
	for _, contract := range contracts {
		if contract == nil || len(contract.ReceiptSchema) == 0 {
			continue
		}
		if receipt == nil {
			return errors.New("workflow call requires a receipt")
		}
		value, err := canonicalWorkflowValue(receipt)
		if err != nil {
			return err
		}
		if err := validateWorkflowSchema(contract.ReceiptSchema, value); err != nil {
			return fmt.Errorf("validate workflow receipt: %w", err)
		}
	}
	return nil
}
