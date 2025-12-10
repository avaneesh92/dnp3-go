package types

import (
	"testing"
)

// TestFlags_IndividualBits tests individual flag bit operations
func TestFlags_IndividualBits(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		check func(Flags) bool
		want  bool
	}{
		// Online flag
		{"Online set", FlagOnline, Flags.IsOnline, true},
		{"Online not set", Flags(0x00), Flags.IsOnline, false},

		// Restart flag
		{"Restart set", FlagRestart, Flags.HasRestart, true},
		{"Restart not set", Flags(0x00), Flags.HasRestart, false},

		// CommLost flag
		{"CommLost set", FlagCommLost, Flags.HasCommLost, true},
		{"CommLost not set", Flags(0x00), Flags.HasCommLost, false},

		// RemoteForced flag
		{"RemoteForced set", FlagRemoteForced, Flags.IsRemoteForced, true},
		{"RemoteForced not set", Flags(0x00), Flags.IsRemoteForced, false},

		// LocalForced flag
		{"LocalForced set", FlagLocalForced, Flags.IsLocalForced, true},
		{"LocalForced not set", Flags(0x00), Flags.IsLocalForced, false},

		// OverRange flag
		{"OverRange set", FlagOverRange, Flags.IsOverRange, true},
		{"OverRange not set", Flags(0x00), Flags.IsOverRange, false},

		// ReferenceErr flag
		{"ReferenceErr set", FlagReferenceErr, Flags.HasReferenceErr, true},
		{"ReferenceErr not set", Flags(0x00), Flags.HasReferenceErr, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check(tt.flags)
			if result != tt.want {
				t.Errorf("%s: got %v, want %v (flags=0x%02X)", tt.name, result, tt.want, tt.flags)
			}
		})
	}
}

// TestFlags_CombinedBits tests combinations of flags
func TestFlags_CombinedBits(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
	}{
		{"All flags set", Flags(0xFF)},
		{"Online + Restart", FlagOnline | FlagRestart},
		{"Online + CommLost", FlagOnline | FlagCommLost},
		{"RemoteForced + LocalForced", FlagRemoteForced | FlagLocalForced},
		{"Multiple quality issues", FlagCommLost | FlagOverRange | FlagReferenceErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all set bits are detected
			if tt.flags&FlagOnline != 0 && !tt.flags.IsOnline() {
				t.Errorf("IsOnline() failed for flags 0x%02X", tt.flags)
			}
			if tt.flags&FlagRestart != 0 && !tt.flags.HasRestart() {
				t.Errorf("HasRestart() failed for flags 0x%02X", tt.flags)
			}
			if tt.flags&FlagCommLost != 0 && !tt.flags.HasCommLost() {
				t.Errorf("HasCommLost() failed for flags 0x%02X", tt.flags)
			}
			if tt.flags&FlagRemoteForced != 0 && !tt.flags.IsRemoteForced() {
				t.Errorf("IsRemoteForced() failed for flags 0x%02X", tt.flags)
			}
			if tt.flags&FlagLocalForced != 0 && !tt.flags.IsLocalForced() {
				t.Errorf("IsLocalForced() failed for flags 0x%02X", tt.flags)
			}
			if tt.flags&FlagOverRange != 0 && !tt.flags.IsOverRange() {
				t.Errorf("IsOverRange() failed for flags 0x%02X", tt.flags)
			}
			if tt.flags&FlagReferenceErr != 0 && !tt.flags.HasReferenceErr() {
				t.Errorf("HasReferenceErr() failed for flags 0x%02X", tt.flags)
			}
		})
	}
}

// TestFlags_IsForced tests combined forced flag logic
func TestFlags_IsForced(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		want  bool
	}{
		{"No forced", FlagOnline, false},
		{"Remote forced", FlagRemoteForced, true},
		{"Local forced", FlagLocalForced, true},
		{"Both forced", FlagRemoteForced | FlagLocalForced, true},
		{"Remote forced with other flags", FlagOnline | FlagRemoteForced | FlagRestart, true},
		{"Local forced with other flags", FlagOnline | FlagLocalForced | FlagCommLost, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.flags.IsForced()
			if result != tt.want {
				t.Errorf("IsForced() = %v, want %v (flags=0x%02X)", result, tt.want, tt.flags)
			}
		})
	}
}

// TestFlags_IsGood tests quality goodness check
func TestFlags_IsGood(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		want  bool
	}{
		{"Good quality (online only)", FlagOnline, true},
		{"Good quality (online + restart)", FlagOnline | FlagRestart, true},
		{"Good quality (online + forced)", FlagOnline | FlagRemoteForced, true},
		{"Bad quality (offline)", Flags(0x00), false},
		{"Bad quality (online + comm lost)", FlagOnline | FlagCommLost, false},
		{"Bad quality (online + reference error)", FlagOnline | FlagReferenceErr, false},
		{"Bad quality (online + both errors)", FlagOnline | FlagCommLost | FlagReferenceErr, false},
		{"Bad quality (comm lost only)", FlagCommLost, false},
		{"Bad quality (reference error only)", FlagReferenceErr, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.flags.IsGood()
			if result != tt.want {
				t.Errorf("IsGood() = %v, want %v (flags=0x%02X)", result, tt.want, tt.flags)
			}
		})
	}
}

// TestFlags_WithOnline tests setting/clearing online flag
func TestFlags_WithOnline(t *testing.T) {
	tests := []struct {
		name     string
		initial  Flags
		setOnline bool
		want     Flags
	}{
		{"Set online on empty", Flags(0x00), true, FlagOnline},
		{"Clear online when set", FlagOnline, false, Flags(0x00)},
		{"Set online when already set", FlagOnline, true, FlagOnline},
		{"Clear online when already clear", Flags(0x00), false, Flags(0x00)},
		{"Set online preserves other flags", FlagRestart | FlagCommLost, true, FlagOnline | FlagRestart | FlagCommLost},
		{"Clear online preserves other flags", FlagOnline | FlagRestart | FlagCommLost, false, FlagRestart | FlagCommLost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.WithOnline(tt.setOnline)
			if result != tt.want {
				t.Errorf("WithOnline(%v) = 0x%02X, want 0x%02X (initial=0x%02X)",
					tt.setOnline, result, tt.want, tt.initial)
			}

			// Verify original is not modified
			if tt.initial.IsOnline() != (tt.initial&FlagOnline != 0) {
				t.Errorf("Original flags were modified")
			}
		})
	}
}

// TestFlags_WithRestart tests setting/clearing restart flag
func TestFlags_WithRestart(t *testing.T) {
	tests := []struct {
		name       string
		initial    Flags
		setRestart bool
		want       Flags
	}{
		{"Set restart on empty", Flags(0x00), true, FlagRestart},
		{"Clear restart when set", FlagRestart, false, Flags(0x00)},
		{"Set restart when already set", FlagRestart, true, FlagRestart},
		{"Clear restart when already clear", Flags(0x00), false, Flags(0x00)},
		{"Set restart preserves other flags", FlagOnline | FlagCommLost, true, FlagOnline | FlagRestart | FlagCommLost},
		{"Clear restart preserves other flags", FlagOnline | FlagRestart | FlagCommLost, false, FlagOnline | FlagCommLost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.WithRestart(tt.setRestart)
			if result != tt.want {
				t.Errorf("WithRestart(%v) = 0x%02X, want 0x%02X (initial=0x%02X)",
					tt.setRestart, result, tt.want, tt.initial)
			}
		})
	}
}

// TestFlags_AllBitsUnique verifies all flag constants are unique
func TestFlags_AllBitsUnique(t *testing.T) {
	flags := []Flags{
		FlagOnline,
		FlagRestart,
		FlagCommLost,
		FlagRemoteForced,
		FlagLocalForced,
		FlagOverRange,
		FlagReferenceErr,
		FlagReserved,
	}

	names := []string{
		"FlagOnline",
		"FlagRestart",
		"FlagCommLost",
		"FlagRemoteForced",
		"FlagLocalForced",
		"FlagOverRange",
		"FlagReferenceErr",
		"FlagReserved",
	}

	// Check each flag is unique
	for i := 0; i < len(flags); i++ {
		for j := i + 1; j < len(flags); j++ {
			if flags[i] == flags[j] {
				t.Errorf("Flags %s and %s have same value: 0x%02X", names[i], names[j], flags[i])
			}
		}
	}

	// Check each flag is a single bit
	for i, flag := range flags {
		// Count bits set
		bits := 0
		for b := uint8(0); b < 8; b++ {
			if uint8(flag)&(1<<b) != 0 {
				bits++
			}
		}
		if bits != 1 {
			t.Errorf("Flag %s has %d bits set, expected 1 (value=0x%02X)", names[i], bits, flag)
		}
	}
}

// TestFlags_ConstantValues verifies flag constant values match DNP3 spec
func TestFlags_ConstantValues(t *testing.T) {
	tests := []struct {
		name     string
		flag     Flags
		expected uint8
	}{
		{"FlagOnline", FlagOnline, 0x01},
		{"FlagRestart", FlagRestart, 0x02},
		{"FlagCommLost", FlagCommLost, 0x04},
		{"FlagRemoteForced", FlagRemoteForced, 0x08},
		{"FlagLocalForced", FlagLocalForced, 0x10},
		{"FlagOverRange", FlagOverRange, 0x20},
		{"FlagReferenceErr", FlagReferenceErr, 0x40},
		{"FlagReserved", FlagReserved, 0x80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint8(tt.flag) != tt.expected {
				t.Errorf("%s = 0x%02X, expected 0x%02X", tt.name, uint8(tt.flag), tt.expected)
			}
		})
	}
}

// TestFlags_BitOperations tests basic bit operations work correctly
func TestFlags_BitOperations(t *testing.T) {
	// OR operation - combining flags
	combined := FlagOnline | FlagRestart
	if !combined.IsOnline() {
		t.Errorf("OR operation failed: online bit not set")
	}
	if !combined.HasRestart() {
		t.Errorf("OR operation failed: restart bit not set")
	}

	// AND operation - testing for flags
	if (combined & FlagOnline) == 0 {
		t.Errorf("AND operation failed: online bit not detected")
	}

	// AND NOT operation - clearing flags
	cleared := combined &^ FlagOnline
	if cleared.IsOnline() {
		t.Errorf("AND NOT operation failed: online bit still set")
	}
	if !cleared.HasRestart() {
		t.Errorf("AND NOT operation failed: restart bit was cleared")
	}

	// XOR operation - toggling flags
	toggled := Flags(0x00) ^ FlagOnline
	if !toggled.IsOnline() {
		t.Errorf("XOR operation failed: online bit not set")
	}
	toggled = toggled ^ FlagOnline
	if toggled.IsOnline() {
		t.Errorf("XOR operation failed: online bit still set after second toggle")
	}
}

// TestFlags_ComplexScenarios tests real-world flag combinations
func TestFlags_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name     string
		flags    Flags
		scenario string
		checks   map[string]bool
	}{
		{
			name:     "Normal operation",
			flags:    FlagOnline,
			scenario: "Point is online and operating normally",
			checks: map[string]bool{
				"IsOnline": true,
				"IsGood":   true,
			},
		},
		{
			name:     "Device restart",
			flags:    FlagOnline | FlagRestart,
			scenario: "Device restarted but still online",
			checks: map[string]bool{
				"IsOnline":   true,
				"HasRestart": true,
				"IsGood":     true, // Restart doesn't affect quality
			},
		},
		{
			name:     "Communication failure",
			flags:    FlagCommLost,
			scenario: "Lost communication with device",
			checks: map[string]bool{
				"IsOnline":    false,
				"HasCommLost": true,
				"IsGood":      false,
			},
		},
		{
			name:     "Manual override",
			flags:    FlagOnline | FlagLocalForced,
			scenario: "Value manually forced by operator",
			checks: map[string]bool{
				"IsOnline":      true,
				"IsLocalForced": true,
				"IsForced":      true,
				"IsGood":        true, // Forced doesn't affect quality
			},
		},
		{
			name:     "SCADA override",
			flags:    FlagOnline | FlagRemoteForced,
			scenario: "Value forced by SCADA system",
			checks: map[string]bool{
				"IsOnline":       true,
				"IsRemoteForced": true,
				"IsForced":       true,
				"IsGood":         true,
			},
		},
		{
			name:     "Sensor failure",
			flags:    FlagOnline | FlagReferenceErr,
			scenario: "Sensor online but has reference error",
			checks: map[string]bool{
				"IsOnline":       true,
				"HasReferenceErr": true,
				"IsGood":         false,
			},
		},
		{
			name:     "Out of range",
			flags:    FlagOnline | FlagOverRange,
			scenario: "Measurement exceeds sensor range",
			checks: map[string]bool{
				"IsOnline":    true,
				"IsOverRange": true,
				"IsGood":      true, // OverRange doesn't affect IsGood()
			},
		},
		{
			name:     "Multiple issues",
			flags:    FlagOnline | FlagCommLost | FlagReferenceErr | FlagOverRange,
			scenario: "Online but with multiple quality issues",
			checks: map[string]bool{
				"IsOnline":       true,
				"HasCommLost":    true,
				"HasReferenceErr": true,
				"IsOverRange":    true,
				"IsGood":         false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for checkName, expected := range tt.checks {
				var result bool
				switch checkName {
				case "IsOnline":
					result = tt.flags.IsOnline()
				case "HasRestart":
					result = tt.flags.HasRestart()
				case "HasCommLost":
					result = tt.flags.HasCommLost()
				case "IsRemoteForced":
					result = tt.flags.IsRemoteForced()
				case "IsLocalForced":
					result = tt.flags.IsLocalForced()
				case "IsForced":
					result = tt.flags.IsForced()
				case "IsOverRange":
					result = tt.flags.IsOverRange()
				case "HasReferenceErr":
					result = tt.flags.HasReferenceErr()
				case "IsGood":
					result = tt.flags.IsGood()
				}

				if result != expected {
					t.Errorf("%s.%s() = %v, want %v (flags=0x%02X, scenario=%s)",
						tt.name, checkName, result, expected, tt.flags, tt.scenario)
				}
			}
		})
	}
}

// BenchmarkFlags_Operations benchmarks flag operations
func BenchmarkFlags_IsOnline(b *testing.B) {
	flags := FlagOnline | FlagRestart
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = flags.IsOnline()
	}
}

func BenchmarkFlags_IsGood(b *testing.B) {
	flags := FlagOnline | FlagRestart
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = flags.IsGood()
	}
}

func BenchmarkFlags_IsForced(b *testing.B) {
	flags := FlagOnline | FlagLocalForced
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = flags.IsForced()
	}
}

func BenchmarkFlags_WithOnline(b *testing.B) {
	flags := FlagRestart | FlagCommLost
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = flags.WithOnline(true)
	}
}
