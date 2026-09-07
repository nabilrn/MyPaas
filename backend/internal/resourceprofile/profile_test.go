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
		Static: {MemoryMB: 64, CPULimit: 0.01},
	}); err != nil {
		t.Fatalf("ConfigureDefaults() error = %v", err)
	}
	_, memory, cpu, err := Resolve(Static, "static", 0, 0)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if memory != 64 || cpu != 0 {
		t.Fatalf("Resolve() = %d MB / %.2f CPU, want 64 MB / shared CPU (0)", memory, cpu)
	}
}

func TestResolveIgnoresLegacyCPUCap(t *testing.T) {
	_, memory, cpu, err := Resolve(NodePython, "dockerfile", 384, 0.25)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if memory != 384 {
		t.Fatalf("Resolve() memory = %d, want 384", memory)
	}
	if cpu != 0 {
		t.Fatalf("Resolve() CPU = %.2f, want shared CPU (0)", cpu)
	}
}

func TestConfigureDefaultsRejectsValuesBelowFloor(t *testing.T) {
	if err := ConfigureDefaults(map[string]Profile{
		Static: {MemoryMB: 64, CPULimit: 0.009},
	}); err == nil {
		t.Fatal("ConfigureDefaults() expected an error")
	}
}
