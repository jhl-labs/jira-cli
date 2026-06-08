package main

import (
	"strings"
	"testing"
)

func TestBuildJQL(t *testing.T) {
	tests := []struct {
		name, project, text, status, want string
	}{
		{"project only", "PROJ", "", "", `project = "PROJ" ORDER BY updated DESC`},
		{"project+status", "PROJ", "", "In Progress", `project = "PROJ" AND status = "In Progress" ORDER BY updated DESC`},
		{"text", "", "login", "", `text ~ "login" ORDER BY updated DESC`},
		{"empty", "", "", "", ""},
		{"quote escaping", "", `say "hi"`, "", `text ~ "say \"hi\"" ORDER BY updated DESC`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildJQL(tt.project, tt.text, tt.status); got != tt.want {
				t.Errorf("buildJQL(%q,%q,%q) = %q, want %q", tt.project, tt.text, tt.status, got, tt.want)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" a, b ,, c ")
	want := []string{"a", "b", "c"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("splitCSV = %v, want %v", got, want)
	}
}
