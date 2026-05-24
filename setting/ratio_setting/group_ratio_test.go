package ratio_setting

import "testing"

func TestEnsureDefaultGroupRatiosFillsMissingWithoutOverwriting(t *testing.T) {
	original := GroupRatio2JSONString()
	t.Cleanup(func() {
		if err := UpdateGroupRatioByJSONString(original); err != nil {
			t.Fatalf("restore group ratio: %v", err)
		}
	})

	if err := UpdateGroupRatioByJSONString(`{"default":1,"free":2}`); err != nil {
		t.Fatalf("set group ratio: %v", err)
	}

	changed := EnsureDefaultGroupRatios("free", "plus", "pro", "unknown")
	if !changed {
		t.Fatal("expected missing onecard groups to be filled")
	}

	ratios := GetGroupRatioCopy()
	if ratios["free"] != 2 {
		t.Fatalf("expected existing free ratio to stay 2, got %v", ratios["free"])
	}
	if ratios["plus"] != 1.2 {
		t.Fatalf("expected plus default ratio 1.2, got %v", ratios["plus"])
	}
	if ratios["pro"] != 1.5 {
		t.Fatalf("expected pro default ratio 1.5, got %v", ratios["pro"])
	}
	if _, ok := ratios["unknown"]; ok {
		t.Fatal("unexpected unknown group ratio")
	}

	if EnsureDefaultGroupRatios("free", "plus", "pro") {
		t.Fatal("expected no change when all groups already exist")
	}
}
