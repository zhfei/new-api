package setting

import "testing"

func TestEnsureAutoGroupsFillsRequiredOrder(t *testing.T) {
	original := AutoGroups2JsonString()
	t.Cleanup(func() {
		if err := UpdateAutoGroupsByJsonString(original); err != nil {
			t.Fatalf("restore auto groups: %v", err)
		}
	})

	if err := UpdateAutoGroupsByJsonString(`["default","vip","svip"]`); err != nil {
		t.Fatalf("set auto groups: %v", err)
	}
	if !EnsureAutoGroups([]string{"free", "plus", "pro"}) {
		t.Fatal("expected auto groups to be changed")
	}
	if !ValidateOneCardAutoGroups(GetAutoGroups()) {
		t.Fatalf("expected valid onecard auto groups, got %v", GetAutoGroups())
	}
	if EnsureAutoGroups([]string{"free", "plus", "pro"}) {
		t.Fatal("expected no change when auto groups already match")
	}
}
