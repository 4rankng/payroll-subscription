package domain

import "testing"

func TestCompareUsersIncludesMobile(t *testing.T) {
	beforeMobile := "0901234567"
	afterMobile := "0912345678"
	original := &User{Mobile: &beforeMobile}
	updated := *original
	updated.Mobile = &afterMobile

	changes := CompareUsers(original, &updated)
	change, ok := changes["mobile"]
	if !ok {
		t.Fatalf("expected mobile in user audit changes: %#v", changes)
	}
	if change.Before != beforeMobile || change.After != afterMobile {
		t.Fatalf("unexpected mobile audit change: %#v", change)
	}
}
