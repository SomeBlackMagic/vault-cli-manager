package vaultsync

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ExpandValue attempts to parse a string as JSON.
// If the string starts with '{' or '[' and is valid JSON, returns the parsed structure.
// Otherwise returns the original string.
func ExpandValue(s string) interface{} {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return s
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return s
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return s
	}
	return parsed
}

// PackValue converts an interface{} value back to a string for Vault storage.
// string → returned as-is
// map/slice/other → json.Marshal into compact JSON string
func PackValue(v interface{}) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal value to JSON: %w", err)
	}
	return string(b), nil
}

// ExpandMap converts map[string]string (from Vault) to map[string]interface{}
// by running ExpandValue on each value.
func ExpandMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = ExpandValue(v)
	}
	return result
}

// PackMap converts map[string]interface{} (from local JSON) to map[string]string
// by running PackValue on each value. Used before writing to Vault.
func PackMap(m map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string, len(m))
	for k, v := range m {
		packed, err := PackValue(v)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		result[k] = packed
	}
	return result, nil
}

// FieldChange represents a single field-level difference within a JSON value.
type FieldChange struct {
	Path     string      // e.g. "db.host" or "tags[0]"
	OldValue interface{} // nil if added
	NewValue interface{} // nil if removed
}

// DeepDiffJSON compares two interface{} values and returns a list of field-level changes.
// Uses dot notation for map paths and bracket notation for array indices.
func DeepDiffJSON(old, new interface{}, prefix string) []FieldChange {
	if reflect.DeepEqual(old, new) {
		return nil
	}

	oldMap, oldIsMap := old.(map[string]interface{})
	newMap, newIsMap := new.(map[string]interface{})
	if oldIsMap && newIsMap {
		return diffMaps(oldMap, newMap, prefix)
	}

	oldSlice, oldIsSlice := old.([]interface{})
	newSlice, newIsSlice := new.([]interface{})
	if oldIsSlice && newIsSlice {
		return diffSlices(oldSlice, newSlice, prefix)
	}

	// Different types or scalar change
	path := prefix
	if path == "" {
		path = "."
	}
	return []FieldChange{{Path: path, OldValue: old, NewValue: new}}
}

func diffMaps(old, new map[string]interface{}, prefix string) []FieldChange {
	var changes []FieldChange

	// Collect all keys
	allKeys := make(map[string]bool)
	for k := range old {
		allKeys[k] = true
	}
	for k := range new {
		allKeys[k] = true
	}

	sorted := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	for _, k := range sorted {
		childPrefix := k
		if prefix != "" {
			childPrefix = prefix + "." + k
		}

		oldVal, oldExists := old[k]
		newVal, newExists := new[k]

		if !oldExists {
			changes = append(changes, FieldChange{Path: childPrefix, OldValue: nil, NewValue: newVal})
		} else if !newExists {
			changes = append(changes, FieldChange{Path: childPrefix, OldValue: oldVal, NewValue: nil})
		} else {
			changes = append(changes, DeepDiffJSON(oldVal, newVal, childPrefix)...)
		}
	}

	return changes
}

func diffSlices(old, new []interface{}, prefix string) []FieldChange {
	var changes []FieldChange

	maxLen := len(old)
	if len(new) > maxLen {
		maxLen = len(new)
	}

	for i := 0; i < maxLen; i++ {
		childPrefix := fmt.Sprintf("%s[%d]", prefix, i)
		if prefix == "" {
			childPrefix = fmt.Sprintf("[%d]", i)
		}

		if i >= len(old) {
			changes = append(changes, FieldChange{Path: childPrefix, OldValue: nil, NewValue: new[i]})
		} else if i >= len(new) {
			changes = append(changes, FieldChange{Path: childPrefix, OldValue: old[i], NewValue: nil})
		} else {
			changes = append(changes, DeepDiffJSON(old[i], new[i], childPrefix)...)
		}
	}

	return changes
}

// ValuesEqual checks if two interface{} values are deeply equal.
func ValuesEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}
