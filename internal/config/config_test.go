package config

import (
	"os"
	"strings"
	"testing"
)

const baselinePath = "../../scenarios/baseline.yaml"

func TestLoadBaseline(t *testing.T) {
	s, err := Load(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "baseline" || s.Population != 10000 || len(s.Work.Departments) != 11 {
		t.Errorf("baseline loaded wrong: name=%q population=%d departments=%d",
			s.Name, s.Population, len(s.Work.Departments))
	}
}

func TestTypoFailsWithKeyName(t *testing.T) {
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ key, typo string }{
		{"warmupYears:", "warmupYear:"}, // top level
		{"gompertzB:", "gompertzb:"},    // nested
	}
	for _, c := range cases {
		bad := strings.Replace(string(data), c.key, c.typo, 1)
		_, err := Parse([]byte(bad))
		want := strings.TrimSuffix(c.typo, ":")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("typo %q: got error %v, want one naming %q", want, err, want)
		}
	}
}
