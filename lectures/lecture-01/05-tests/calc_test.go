package calc

import "testing"

func TestAdd(t *testing.T) {
	cases := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive", 2, 3, 5},
		{"negative", -2, -3, -5},
		{"zero", 0, 5, 5},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Add(c.a, c.b)
			if got != c.expected {
				t.Errorf("Add(%d, %d) = %d, want %d", c.a, c.b, got, c.expected)
			}
		})
	}
}

func TestMax(t *testing.T) {
	got := Max(5, 3)
	if got != 5 {
		t.Errorf("Max(5, 3) = %d, want 5", got)
	}
}

func TestDivide(t *testing.T) {
	cases := []struct {
		name    string
		a, b    int
		want    int
		wantErr bool
	}{
		{"exact", 10, 2, 5, false},
		{"by zero", 10, 0, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Divide(c.a, c.b)
			if c.wantErr {
				if err == nil {
					t.Fatalf("Divide(%d, %d) expected error, got nil", c.a, c.b)
				}
				return
			}
			if err != nil {
				t.Fatalf("Divide(%d, %d) unexpected error: %v", c.a, c.b, err)
			}
			if got != c.want {
				t.Errorf("Divide(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
			}
		})
	}
}
