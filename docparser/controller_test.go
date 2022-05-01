package docparser

import "testing"

func Test_parseGroupName(t *testing.T) {
	tests := map[string]struct {
		groupName  string
		wantName   string
		wantWeight int
	}{
		"no weight": {
			groupName:  "Group",
			wantName:   "Group",
			wantWeight: -1,
		},
		"with weight": {
			groupName:  "Group [1]",
			wantName:   "Group",
			wantWeight: 1,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotName, gotWeight := parseGroupName(tt.groupName)
			if gotName != tt.wantName {
				t.Errorf("parseGroupName() gotName = %v, want %v", gotName, tt.wantName)
			}
			if gotWeight != tt.wantWeight {
				t.Errorf("parseGroupName() gotWeight = %v, want %v", gotWeight, tt.wantWeight)
			}
		})
	}
}
