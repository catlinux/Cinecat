package update

import "testing"

func TestComparacionSemver(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{"0.13.1", "0.13.1", 0},
		{"v0.13.1", "0.13.1", 0},
		{"0.9.0", "0.10.0", -1},
		{"0.10.0", "0.9.0", 1},
		{"1.0.0", "0.99.99", 1},
	}
	for _, test := range tests {
		left, ok1 := ParseSemver(test.left)
		right, ok2 := ParseSemver(test.right)
		if !ok1 || !ok2 {
			t.Fatalf("versiones no válidas: %q, %q", test.left, test.right)
		}
		if got := Compare(left, right); got != test.want {
			t.Fatalf("Compare(%q, %q) = %d, se esperaba %d", test.left, test.right, got, test.want)
		}
	}
}

func TestEtiquetasYVersionesInvalidas(t *testing.T) {
	for _, value := range []string{"dev", "", "1.2", "v1.2", "v1.2.3-beta", "v1.2.3.4", "1.2.x"} {
		if _, ok := ParseTag(value); ok {
			t.Fatalf("la etiqueta %q debería ignorarse", value)
		}
	}
	if version, ok := ParseTag("v1.2.3"); !ok || version.Major != 1 || version.Minor != 2 || version.Patch != 3 {
		t.Fatalf("v1.2.3 mal interpretada: %+v, %v", version, ok)
	}
	if _, ok := ParseSemver("dev"); ok {
		t.Fatal("dev no es una versión semver")
	}
}
