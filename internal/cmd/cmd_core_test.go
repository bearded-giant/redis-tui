package cmd

import "testing"

func TestScanSize(t *testing.T) {
	orig := GetScanSize()
	t.Cleanup(func() { SetScanSize(orig) })

	SetScanSize(500)
	if got := GetScanSize(); got != 500 {
		t.Errorf("GetScanSize() = %d, want 500", got)
	}

	SetScanSize(2000)
	if got := GetScanSize(); got != 2000 {
		t.Errorf("GetScanSize() = %d, want 2000", got)
	}
}

func TestIncludeTypes(t *testing.T) {
	orig := GetIncludeTypes()
	t.Cleanup(func() { SetIncludeTypes(orig) })

	SetIncludeTypes(false)
	if GetIncludeTypes() {
		t.Error("GetIncludeTypes() = true, want false")
	}

	SetIncludeTypes(true)
	if !GetIncludeTypes() {
		t.Error("GetIncludeTypes() = false, want true")
	}
}
