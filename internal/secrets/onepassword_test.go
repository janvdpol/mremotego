package secrets

import (
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"testing"
)

func newTestProvider(run commandRunner) *OnePasswordProvider {
	return &OnePasswordProvider{enabled: true, run: run}
}

// flagValue returns the value of a space-separated flag pair (e.g. --format json → "json").
func flagValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// argValue returns the right-hand side of an equals-style flag (e.g. --vault=foo → "foo").
func argValue(args []string, prefix string) string {
	for _, a := range args {
		if v, ok := strings.CutPrefix(a, prefix+"="); ok {
			return v
		}
	}
	return ""
}

func containsArg(args []string, needle string) bool {
	return slices.Contains(args, needle)
}

type capturedCall struct {
	name string
	args []string
}

// TestResolveMultiple_SameItemBatched verifies that two references for the same
// vault+item produce exactly one op call carrying both field labels.
func TestResolveMultiple_SameItemBatched(t *testing.T) {
	var calls []capturedCall

	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, capturedCall{name, args})
		out, _ := json.Marshal([]opField{
			{Label: "username", Value: "testuser"},
			{Label: "password", Value: "s3cr3t"},
		})
		return out, nil
	}

	p := newTestProvider(run)
	refs := []string{
		"op://Projects Wij Techniek/DBSQLASRV01 VM/username",
		"op://Projects Wij Techniek/DBSQLASRV01 VM/password",
	}

	result, err := p.ResolveMultiple(refs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 op call for same vault+item, got %d", len(calls))
	}

	c := calls[0]
	args := c.args

	quotedArgs := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsRune(a, ' ') {
			quotedArgs[i] = `"` + a + `"`
		} else {
			quotedArgs[i] = a
		}
	}

	if c.name != "op" {
		t.Errorf("command: want %q, got %q", "op", c.name)
	}

	// Positional args: item get <item-name>
	if len(args) < 3 || args[0] != "item" || args[1] != "get" || args[2] != "DBSQLASRV01 VM" {
		t.Errorf("positional args: want [item get DBSQLASRV01 VM], got %v", args[:min(3, len(args))])
	}

	// --vault=<vault>
	if got := argValue(args, "--vault"); got != "Projects Wij Techniek" {
		t.Errorf("--vault: want %q, got %q", "Projects Wij Techniek", got)
	}

	// --fields must contain both labels (map iteration order is non-deterministic, so sort)
	rawFields := strings.Split(flagValue(args, "--fields"), ",")
	sort.Strings(rawFields)
	if strings.Join(rawFields, ",") != "label=password,label=username" {
		t.Errorf("--fields: want label=password,label=username, got %q", rawFields)
	}

	// --format json
	if got := flagValue(args, "--format"); got != "json" {
		t.Errorf("--format: want %q, got %q", "json", got)
	}

	// --reveal
	if !containsArg(args, "--reveal") {
		t.Errorf("--reveal not found in args %v", args)
	}

	// Results must map back to the original references
	if result[refs[0]] != "testuser" {
		t.Errorf("username result: want %q, got %q", "testuser", result[refs[0]])
	}
	if result[refs[1]] != "s3cr3t" {
		t.Errorf("password result: want %q, got %q", "s3cr3t", result[refs[1]])
	}
}

// TestResolveMultiple_DifferentItemsTwoCalls verifies that refs from two distinct
// items produce two separate op calls.
func TestResolveMultiple_DifferentItemsTwoCalls(t *testing.T) {
	callCount := 0
	run := func(name string, args ...string) ([]byte, error) {
		callCount++
		out, _ := json.Marshal([]opField{{Label: "password", Value: "pw"}})
		return out, nil
	}

	p := newTestProvider(run)
	refs := []string{
		"op://MyVault/ItemA/password",
		"op://MyVault/ItemB/password",
	}

	if _, err := p.ResolveMultiple(refs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 op calls for different items, got %d", callCount)
	}
}

// TestResolveMultiple_NonReferencesPassthrough verifies that plain strings are
// returned unchanged and no op call is made.
func TestResolveMultiple_NonReferencesPassthrough(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		t.Fatal("op must not be called for non-references")
		return nil, nil
	}

	p := newTestProvider(run)
	refs := []string{"plaintext", "also-plain"}

	result, err := p.ResolveMultiple(refs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, ref := range refs {
		if result[ref] != ref {
			t.Errorf("%q: want passthrough, got %q", ref, result[ref])
		}
	}
}

// TestResolveMultiple_DuplicateRefSingleCall verifies that the same reference
// appearing twice results in only one op call.
func TestResolveMultiple_DuplicateRefSingleCall(t *testing.T) {
	callCount := 0
	run := func(name string, args ...string) ([]byte, error) {
		callCount++
		out, _ := json.Marshal([]opField{{Label: "password", Value: "secret"}})
		return out, nil
	}

	p := newTestProvider(run)
	ref := "op://Vault/Item/password"

	if _, err := p.ResolveMultiple([]string{ref, ref}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 op call for duplicate refs, got %d", callCount)
	}
}
