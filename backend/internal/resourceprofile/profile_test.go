package resourceprofile

import "testing"

func TestConfigureDefaults(t *testing.T) {
	t.Cleanup(func() {
		if err := ConfigureDefaults(map[string]Profile{
			Static:      minimumProfiles[Static],
			GoSmall:     minimumProfiles[GoSmall],
			NodePython:  minimumProfiles[NodePython],
			ComposeMain: minimumProfiles[ComposeMain],
		}); err != nil {
			t.Fatalf("restore defaults: %v", err)
		}
	})

	if err := ConfigureDefaults(map[string]Profile{
		Static: {MemoryMB: 128, CPULimit: 0.25},
	}); err != nil {
		t.Fatalf("ConfigureDefaults() error = %v", err)
	}
	_, memory, cpu, err := Resolve(Static, "static", 0, 0)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if memory != 128 || cpu != 0.25 {
		t.Fatalf("Resolve() = %d MB / %.2f CPU, want 128 MB / 0.25 CPU", memory, cpu)
	}
}

func TestConfigureDefaultsRejectsValuesBelowFloor(t *testing.T) {
	if err := ConfigureDefaults(map[string]Profile{
		Static: {MemoryMB: 32, CPULimit: 0.10},
	}); err == nil {
		t.Fatal("ConfigureDefaults() expected an error")
	}
}
