package domain

import "testing"

func TestNextCursorPointsAtLastItemInPage(t *testing.T) {
	items := []Event{{ID: "e1"}, {ID: "e2"}, {ID: "e3"}}
	if got := NextCursor(items, 2); got != "e2" { t.Fatalf("cursor=%q", got) }
}
