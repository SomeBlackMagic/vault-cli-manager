package vaultsync

import (
	"fmt"
	"sort"
	"strings"
)

const (
	ansiGray  = "\033[90m" // dark gray (bright black)
	ansiReset = "\033[0m"
)

// gray wraps text in dark gray ANSI escape codes.
// Raw ANSI codes pass through go-ansi's Fprintf unmodified.
func gray(s string) string {
	return ansiGray + s + ansiReset
}

// ComputeChanges compares local state vs remote state and returns a ChangeSet.
func ComputeChanges(local []LocalSecret, remote map[string]map[string]interface{}) ChangeSet {
	var changes []Change

	localMap := make(map[string]map[string]interface{}, len(local))
	for _, ls := range local {
		localMap[ls.Path] = ls.Data
	}

	// Collect all paths
	allPaths := make(map[string]bool)
	for _, ls := range local {
		allPaths[ls.Path] = true
	}
	for p := range remote {
		allPaths[p] = true
	}

	sorted := make([]string, 0, len(allPaths))
	for p := range allPaths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)

	for _, path := range sorted {
		localData, localExists := localMap[path]
		remoteData, remoteExists := remote[path]

		switch {
		case localExists && !remoteExists:
			changes = append(changes, Change{
				Type:      ChangeAdd,
				Path:      path,
				LocalData: localData,
			})
		case !localExists && remoteExists:
			changes = append(changes, Change{
				Type:       ChangeDelete,
				Path:       path,
				RemoteData: remoteData,
			})
		case localExists && remoteExists:
			if mapsEqual(localData, remoteData) {
				changes = append(changes, Change{
					Type:       ChangeNone,
					Path:       path,
					LocalData:  localData,
					RemoteData: remoteData,
				})
			} else {
				changes = append(changes, Change{
					Type:       ChangeModify,
					Path:       path,
					LocalData:  localData,
					RemoteData: remoteData,
				})
			}
		}
	}

	return ChangeSet{Changes: changes}
}

// mapsEqual compares two map[string]interface{} values deeply.
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || !ValuesEqual(va, vb) {
			return false
		}
	}
	return true
}

// FormatDiff returns a colored string showing key-level diff for a single Change.
// Includes type labels ([create], [update], [destroy], [no changes]),
// gray coloring for unchanged secrets, and blank line separators.
func FormatDiff(c Change) string {
	var sb strings.Builder

	switch c.Type {
	case ChangeAdd:
		sb.WriteString(fmt.Sprintf("  @G{+ %s}  @G{[create]}\n", c.Path))
		for _, k := range sortedKeys(c.LocalData) {
			sb.WriteString(fmt.Sprintf("      @G{+ %s}: %s\n", k, formatValue(c.LocalData[k])))
		}

	case ChangeDelete:
		sb.WriteString(fmt.Sprintf("  @R{- %s}  @R{[destroy]}\n", c.Path))
		for _, k := range sortedKeys(c.RemoteData) {
			sb.WriteString(fmt.Sprintf("      @R{- %s}: %s\n", k, formatValue(c.RemoteData[k])))
		}

	case ChangeModify:
		sb.WriteString(fmt.Sprintf("  @Y{~ %s}  @Y{[update]}\n", c.Path))
		for _, k := range mergedKeys(c.LocalData, c.RemoteData) {
			localVal, localHas := c.LocalData[k]
			remoteVal, remoteHas := c.RemoteData[k]

			if !remoteHas {
				sb.WriteString(fmt.Sprintf("      @G{+ %s}: %s\n", k, formatValue(localVal)))
			} else if !localHas {
				sb.WriteString(fmt.Sprintf("      @R{- %s}: %s\n", k, formatValue(remoteVal)))
			} else if !ValuesEqual(localVal, remoteVal) {
				sb.WriteString(formatKeyDiff(k, remoteVal, localVal))
			}
		}

	case ChangeNone:
		sb.WriteString(gray(fmt.Sprintf("    %s  [no changes]", c.Path)) + "\n")
	}

	sb.WriteString("\n") // blank line between blocks
	return sb.String()
}

// formatKeyDiff formats a single key diff, with nested JSON support.
func formatKeyDiff(key string, oldVal, newVal interface{}) string {
	var sb strings.Builder

	_, oldIsMap := oldVal.(map[string]interface{})
	_, newIsMap := newVal.(map[string]interface{})
	_, oldIsSlice := oldVal.([]interface{})
	_, newIsSlice := newVal.([]interface{})

	if (oldIsMap && newIsMap) || (oldIsSlice && newIsSlice) {
		sb.WriteString(fmt.Sprintf("      @Y{~ %s}:\n", key))
		for _, fc := range DeepDiffJSON(oldVal, newVal, "") {
			if fc.OldValue == nil {
				sb.WriteString(fmt.Sprintf("          @G{+ %s}: %s\n", fc.Path, formatValue(fc.NewValue)))
			} else if fc.NewValue == nil {
				sb.WriteString(fmt.Sprintf("          @R{- %s}: %s\n", fc.Path, formatValue(fc.OldValue)))
			} else {
				sb.WriteString(formatScalarChange(fc.Path, fc.OldValue, fc.NewValue, 10))
			}
		}
	} else {
		sb.WriteString(formatScalarChange(key, oldVal, newVal, 6))
	}

	return sb.String()
}

// formatScalarChange formats a scalar value change.
// For short values (combined length <= 80): single-line "key: old => new".
// For long strings: two-line format with ^ markers showing the changed portion.
// indent = number of leading spaces for the key line.
func formatScalarChange(key string, oldVal, newVal interface{}, indent int) string {
	var sb strings.Builder
	pad := strings.Repeat(" ", indent)

	oldStr := formatValue(oldVal)
	newStr := formatValue(newVal)

	// For short values, use compact single-line format
	if len(oldStr)+len(newStr) <= 80 {
		sb.WriteString(fmt.Sprintf("%s@Y{~ %s}: %s => %s\n", pad, key, oldStr, newStr))
		return sb.String()
	}

	// Long values: two-line format with diff markers
	sb.WriteString(fmt.Sprintf("%s@Y{~ %s}:\n", pad, key))
	innerPad := pad + "    "

	prefix, oldMid, newMid, suffix := SplitDiff(oldStr, newStr)

	sb.WriteString(fmt.Sprintf("%s@R{-} %s@R{%s}%s\n", innerPad, prefix, oldMid, suffix))
	sb.WriteString(fmt.Sprintf("%s@G{+} %s@G{%s}%s\n", innerPad, prefix, newMid, suffix))

	// Show ^ markers under the changed region
	// markerOffset = innerPad + "- " + prefix length
	markerOffset := len(innerPad) + 2 + len(prefix)
	markerLen := len(oldMid)
	if len(newMid) > markerLen {
		markerLen = len(newMid)
	}
	if markerLen > 0 {
		sb.WriteString(strings.Repeat(" ", markerOffset) + strings.Repeat("^", markerLen) + "\n")
	}

	return sb.String()
}

// SplitDiff finds the common prefix, differing middle parts, and common suffix
// between two strings. Used to highlight only the changed characters.
//
// Example: SplitDiff("abcXXXdef", "abcYYdef") → ("abc", "XXX", "YY", "def")
func SplitDiff(a, b string) (prefix, aMid, bMid, suffix string) {
	// Find longest common prefix
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	prefix = a[:i]

	// Find longest common suffix (after prefix)
	aRest := a[i:]
	bRest := b[i:]
	j := 0
	for j < len(aRest) && j < len(bRest) && aRest[len(aRest)-1-j] == bRest[len(bRest)-1-j] {
		j++
	}

	aMid = aRest[:len(aRest)-j]
	bMid = bRest[:len(bRest)-j]
	if j > 0 {
		suffix = aRest[len(aRest)-j:]
	}

	return
}

// FormatChangeSummary returns "Plan: X to create, Y to update, Z to destroy."
func FormatChangeSummary(cs ChangeSet) string {
	adds, modifies, deletes := cs.Counts()
	return fmt.Sprintf("Plan: @G{%d} to create, @Y{%d} to update, @R{%d} to destroy.", adds, modifies, deletes)
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func mergedKeys(a, b map[string]interface{}) []string {
	all := make(map[string]bool)
	for k := range a {
		all[k] = true
	}
	for k := range b {
		all[k] = true
	}
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case nil:
		return "<nil>"
	default:
		return fmt.Sprintf("%v", val)
	}
}
