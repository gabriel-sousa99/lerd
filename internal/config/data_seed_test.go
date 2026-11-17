package config

import "testing"

func TestOracleFaststartRequiresImageDataSeed(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	p, err := LoadPreset("oracle-xe")
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"21-slim-faststart", "21-slim", "21-full"} {
		for _, pinned := range []bool{false, true} {
			var svc *CustomService
			if pinned {
				svc, err = p.ResolvePinned(tag)
			} else {
				svc, err = p.Resolve(tag)
			}
			if err != nil {
				t.Fatal(err)
			}
			if got, want := len(svc.DataSeed) > 0, tag == "21-slim-faststart"; got != want {
				t.Errorf("%s pinned=%v: data seed=%v, want %v", tag, pinned, got, want)
			}
		}
	}
}

func TestValidateDataSeedRejectsPathsOutsideDataDir(t *testing.T) {
	for _, name := range []string{"../control", "/etc/passwd", "db\ncontrol", "."} {
		svc := &CustomService{Name: "seed-db", Image: "db:1", DataDir: "/data", DataSeed: []string{name}}
		if err := ValidateCustomService(svc); err == nil {
			t.Errorf("accepted seed path %q", name)
		}
	}
}
