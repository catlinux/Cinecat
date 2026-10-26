package update

import (
	"strconv"
	"strings"
)

// Semver es una versión X.Y.Z.
type Semver struct {
	Major int
	Minor int
	Patch int
}

// ParseSemver acepta X.Y.Z con o sin prefijo v. Para etiquetas de GitHub se
// exige el prefijo mediante ParseTag.
func ParseSemver(value string) (Semver, bool) {
	text := strings.TrimSpace(value)
	text = strings.TrimPrefix(text, "v")
	parts := strings.Split(text, ".")
	if len(parts) != 3 {
		return Semver{}, false
	}
	numbers := [3]int{}
	for index, part := range parts {
		if part == "" || hasNonDigit(part) {
			return Semver{}, false
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return Semver{}, false
		}
		numbers[index] = number
	}
	return Semver{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}, true
}

// ParseTag acepta únicamente vX.Y.Z, el formato de las etiquetas publicadas.
func ParseTag(tag string) (Semver, bool) {
	if !strings.HasPrefix(tag, "v") {
		return Semver{}, false
	}
	return ParseSemver(tag)
}

func hasNonDigit(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return true
		}
	}
	return false
}

// Compare devuelve -1, 0 o 1.
func Compare(left, right Semver) int {
	switch {
	case left.Major != right.Major:
		return compareInt(left.Major, right.Major)
	case left.Minor != right.Minor:
		return compareInt(left.Minor, right.Minor)
	default:
		return compareInt(left.Patch, right.Patch)
	}
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
