package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// ContractFile returns the example request or response named name from the
// repository's contract directory, which the app's tests check against too.
// It expects to be called from a package directly under internal.
func ContractFile(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "contract", name))
	if err != nil {
		t.Fatalf("failed to read contract file: %v", err)
	}
	return data
}

// AssertJSONEqual fails t unless got and want hold the same JSON value,
// regardless of formatting and key order.
func AssertJSONEqual(t testing.TB, got, want []byte) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("failed to parse JSON %q: %v", got, err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("failed to parse expected JSON %q: %v", want, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("JSON = %s\nwant %s", got, want)
	}
}
