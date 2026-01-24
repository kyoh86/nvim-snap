package snapshots

import "testing"

func TestNormalizeSorts(t *testing.T) {
	s := Snapshot{
		HLGroups: []HLGroup{
			{Name: "Z", HlID: 2},
			{Name: "A", HlID: 1},
		},
		HLAttrs: []HLAttr{
			{ID: 10},
			{ID: 2},
		},
		Grids: []Grid{
			{ID: 2},
			{ID: 1},
		},
	}
	out := Normalize(s)
	if out.Grids[0].ID != 1 || out.Grids[1].ID != 2 {
		t.Fatalf("Grids order = %v", []int{out.Grids[0].ID, out.Grids[1].ID})
	}
	if out.HLAttrs[0].ID != 2 || out.HLAttrs[1].ID != 10 {
		t.Fatalf("HLAttrs order = %v", []int{out.HLAttrs[0].ID, out.HLAttrs[1].ID})
	}
	if out.HLGroups[0].Name != "A" || out.HLGroups[1].Name != "Z" {
		t.Fatalf("HLGroups order = %v", []string{out.HLGroups[0].Name, out.HLGroups[1].Name})
	}
}
