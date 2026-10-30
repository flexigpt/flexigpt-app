package workflowv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

var (
	schemaType = reflect.TypeFor[Schema]()
	valueType  = reflect.TypeFor[Value]()
	durationRE = regexp.MustCompile(
		`^P(?:[0-9]+Y)?(?:[0-9]+M)?(?:[0-9]+W)?(?:[0-9]+D)?(?:T(?:[0-9]+H)?(?:[0-9]+M)?(?:[0-9]+(?:\.[0-9]+)?S)?)?$`,
	)
)

func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", storeSpec.ErrInvalid, fmt.Sprintf(format, arguments...))
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateName(label, value string) error {
	return storeSpec.ValidatePortableName(label, value)
}

func validateDuration(value string, positive bool) error {
	if value == "" {
		return nil
	}
	if len(value) > 256 || !durationRE.MatchString(value) ||
		value == "P" || strings.HasSuffix(value, "T") {
		return invalid("%q is not an ISO 8601 duration", value)
	}
	if positive && !strings.ContainsAny(value, "123456789") {
		return invalid("duration %q must be positive", value)
	}
	return nil
}

func validateCalendar(value string) error {
	// "v1" calendar syntax is the conventional five-field numeric cron grammar:
	// minute hour day-of-month month day-of-week.
	fields := strings.Fields(value)
	if len(fields) != 5 {
		return invalid("calendar requires five cron fields")
	}
	minimum := []int{0, 0, 1, 1, 0}
	maximum := []int{59, 23, 31, 12, 7}
	for index, field := range fields {
		for part := range strings.SplitSeq(field, ",") {
			base, stride, hasStride := strings.Cut(part, "/")
			if hasStride {
				number, err := strconv.Atoi(stride)
				if err != nil || number <= 0 || number > maximum[index]-minimum[index]+1 {
					return invalid("invalid cron stride %q", part)
				}
			}
			if base == "*" {
				continue
			}
			left, right, hasRange := strings.Cut(base, "-")
			start, err := strconv.Atoi(left)
			if err != nil || start < minimum[index] || start > maximum[index] {
				return invalid("invalid cron field %q", field)
			}
			if hasRange {
				end, err := strconv.Atoi(right)
				if err != nil || end < start || end > maximum[index] {
					return invalid("invalid cron range %q", part)
				}
			}
		}
	}
	return nil
}

func validateTimeZone(value string) error {
	if value == "" {
		return invalid("calendar schedule requires a time zone")
	}
	if _, err := time.LoadLocation(value); err != nil {
		return invalid("unknown calendar time zone %q", value)
	}
	return nil
}

func (v WorkflowDocument) schemaObject(raw Schema) ([]byte, error) {
	canonical, err := boundedJSON(raw)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(canonical, []byte("true")) || bytes.Equal(canonical, []byte("false")) {
		return json.Marshal(map[string]any{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"allOf":   []json.RawMessage{json.RawMessage(canonical)},
		})
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &object); err != nil || object == nil {
		return nil, invalid("JSON Schema must be an object or boolean")
	}
	if dialect, found := object["$schema"]; found {
		var value string
		if err := json.Unmarshal(dialect, &value); err != nil ||
			value != "https://json-schema.org/draft/2020-12/schema" {
			return nil, invalid("Workflow schemas use JSON Schema Draft 2020-12")
		}
	}
	// Reusable declaration schemas are available under #/$defs/<name>.
	// A schema's own local definitions retain precedence.
	definitions := map[string]json.RawMessage{}
	if local, found := object["$defs"]; found {
		if err := json.Unmarshal(local, &definitions); err != nil {
			return nil, err
		}
		if definitions == nil {
			return nil, invalid("JSON Schema $defs must be an object")
		}
	}
	for name, definition := range v.Schemas {
		if _, present := definitions[name]; !present {
			definitions[name] = json.RawMessage(definition)
		}
	}
	if len(definitions) > 0 {
		encoded, err := json.Marshal(definitions)
		if err != nil {
			return nil, err
		}
		object["$defs"] = encoded
	}
	object["$schema"] = json.RawMessage(`"https://json-schema.org/draft/2020-12/schema"`)
	encoded, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	var tree any
	if err := json.Unmarshal(encoded, &tree); err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	collectSchemaIDs(tree, ids)
	if err := checkSchemaReferences(tree, ids); err != nil {
		return nil, err
	}
	return encoded, nil
}

func collectSchemaIDs(value any, ids map[string]bool) {
	_ = walkSchema(value, func(object map[string]any) error {
		if id, ok := object["$id"].(string); ok && id != "" {
			ids[strings.SplitN(id, "#", 2)[0]] = true
		}
		return nil
	})
}

func checkSchemaReferences(value any, ids map[string]bool) error {
	return walkSchema(value, func(object map[string]any) error {
		for _, keyword := range []string{"$ref", "$dynamicRef"} {
			if reference, ok := object[keyword].(string); ok &&
				!strings.HasPrefix(reference, "#") &&
				!ids[strings.SplitN(reference, "#", 2)[0]] {
				return invalid("schema reference %q has no declaration-local dependency", reference)
			}
		}
		return nil
	})
}

func (v WorkflowDocument) compileSchema(raw Schema) (jsonutil.JSONSchema, error) {
	encoded, err := v.schemaObject(raw)
	if err != nil {
		return nil, err
	}
	compiled, err := jsonutil.CompileJSONSchema(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid Workflow JSON Schema: %w", storeSpec.ErrInvalid, err)
	}
	return compiled, nil
}

// validateSchemaFields visits typed contract fields only. It does not treat
// arbitrary business values or metadata fields named "schema" as schemas.
func (v WorkflowDocument) validateSchemaFields(value reflect.Value, path string) error {
	if !value.IsValid() {
		return nil
	}
	if value.Type() == valueType {
		return nil
	}
	if value.Type() == schemaType {
		raw, ok := reflect.TypeAssert[Schema](value)
		if !ok || len(raw) == 0 {
			return nil
		}
		if _, err := v.compileSchema(raw); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return nil
	}
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !value.IsNil() {
			return v.validateSchemaFields(value.Elem(), path)
		}
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			if field.PkgPath != "" {
				continue
			}
			if err := v.validateSchemaFields(value.Field(index), path+"/"+field.Name); err != nil {
				return err
			}
		}
	case reflect.Map:
		keys := value.MapKeys()
		sort.Slice(keys, func(a, b int) bool { return fmt.Sprint(keys[a]) < fmt.Sprint(keys[b]) })
		for _, key := range keys {
			if err := v.validateSchemaFields(value.MapIndex(key), path+"/"+fmt.Sprint(key)); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := v.validateSchemaFields(value.Index(index), fmt.Sprintf("%s/%d", path, index)); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

func literalValue(value Value) (literal any, known bool, err error) {
	if len(value) == 0 {
		return nil, false, nil
	}
	decoded, err := decodeValueJSON(value)
	if err != nil {
		return nil, false, err
	}
	return constructedLiteral(decoded)
}

func constructedLiteral(value any) (literal any, known bool, err error) {
	switch typed := value.(type) {
	case map[string]any:
		if _, expression := typed["$expr"]; expression {
			return nil, false, nil
		}
		if literal, found := typed["$literal"]; found && len(typed) == 1 {
			return literal, true, nil
		}
		output := make(map[string]any, len(typed))
		for name, child := range typed {
			literal, known, err := constructedLiteral(child)
			if err != nil || !known {
				return nil, known, err
			}
			output[name] = literal
		}
		return output, true, nil
	case []any:
		output := make([]any, len(typed))
		for index, child := range typed {
			literal, known, err := constructedLiteral(child)
			if err != nil || !known {
				return nil, known, err
			}
			output[index] = literal
		}
		return output, true, nil
	default:
		return value, true, nil
	}
}

func (v WorkflowDocument) validateLiteral(value Value, schema Schema) error {
	if len(schema) == 0 {
		return nil
	}
	literal, known, err := literalValue(value)
	if err != nil || !known {
		return err
	}
	compiled, err := v.compileSchema(schema)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(literal)
	if err != nil {
		return err
	}
	return jsonutil.ValidateJSONSchema(compiled, json.RawMessage(raw), storeSpec.MaxDefinitionBodyBytes)
}

// rejectDisjointSchemas establishes only provable incompatibility. General
// JSON Schema implication is not assumed to be decidable by this validator.
// Full value validation still occurs at each future execution boundary.
func rejectDisjointSchemas(left, right Schema) error {
	if bytes.Equal(bytes.TrimSpace(left), []byte("false")) ||
		bytes.Equal(bytes.TrimSpace(right), []byte("false")) {
		return invalid("a connected interface is unsatisfiable")
	}
	types := func(raw Schema) map[string]bool {
		var object map[string]json.RawMessage
		_ = json.Unmarshal(raw, &object)
		result := map[string]bool{}
		var one string
		if json.Unmarshal(object["type"], &one) == nil && one != "" {
			result[one] = true
		}
		var many []string
		if json.Unmarshal(object["type"], &many) == nil {
			for _, name := range many {
				result[name] = true
			}
		}
		if result["number"] {
			result["integer"] = true
		}
		return result
	}
	a, b := types(left), types(right)
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	for name := range a {
		if b[name] {
			return nil
		}
	}
	return invalid("connected schemas have disjoint JSON types")
}

func validateLimits(child, parent *Limits) error {
	if child == nil {
		return nil
	}
	for _, duration := range []string{child.Duration, child.CleanupDuration} {
		if err := validateDuration(duration, true); err != nil {
			return err
		}
	}
	if parent == nil {
		return nil
	}
	a, b := reflect.ValueOf(*child), reflect.ValueOf(*parent)
	for index := 0; index < a.NumField(); index++ {
		if a.Field(index).Type() != reflect.TypeFor[*int]() ||
			a.Field(index).IsNil() || b.Field(index).IsNil() {
			continue
		}
		if a.Field(index).Elem().Int() > b.Field(index).Elem().Int() {
			return invalid("child limit %s exceeds its inherited limit", a.Type().Field(index).Name)
		}
	}
	if child.Cost != nil && parent.Cost != nil &&
		(child.Cost.Currency != parent.Cost.Currency || child.Cost.Amount > parent.Cost.Amount) {
		return invalid("child monetary limit exceeds or changes inherited currency")
	}
	return nil
}

func inheritLimits(parent, child *Limits) *Limits {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}
	output := *parent
	a, b := reflect.ValueOf(&output).Elem(), reflect.ValueOf(*child)
	for index := 0; index < a.NumField(); index++ {
		switch b.Field(index).Kind() {
		case reflect.Pointer:
			if !b.Field(index).IsNil() {
				a.Field(index).Set(b.Field(index))
			}
		case reflect.String:
			if b.Field(index).String() != "" {
				a.Field(index).Set(b.Field(index))
			}
		default:
		}
	}
	return &output
}
