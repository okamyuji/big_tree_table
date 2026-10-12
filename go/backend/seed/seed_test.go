package main

import "testing"

func TestPlanSeed(t *testing.T) {
	tests := []struct {
		name         string
		count        int64
		reset        bool
		wantSkip     bool
		wantTruncate bool
		wantErr      bool
	}{
		{name: "empty table inserts", count: 0},
		{name: "empty table ignores reset", count: 0, reset: true},
		{name: "complete table skips", count: totalRows, wantSkip: true},
		{name: "larger table skips", count: totalRows + 1, wantSkip: true},
		{name: "partial table without reset refuses to delete", count: 1, wantErr: true},
		{name: "one row short without reset refuses to delete", count: totalRows - 1, wantErr: true},
		{name: "partial table with reset truncates", count: 1, reset: true, wantTruncate: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, truncate, err := planSeed(tt.count, tt.reset)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if skip != tt.wantSkip || truncate != tt.wantTruncate {
				t.Errorf("planSeed(%d, %v) = skip %v, truncate %v; want skip %v, truncate %v",
					tt.count, tt.reset, skip, truncate, tt.wantSkip, tt.wantTruncate)
			}
		})
	}
}
