package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version.
type Version struct {
	Major      int64
	Minor      int64
	Patch      int64
	PreRelease string // e.g. "alpha", "beta.1"
	Metadata   string // e.g. "build.123" (ignored in precedence)
	original   string // preserve original input
}

// versionRegex matches core semver strings (with optional leading "v").
var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z\-.]+))?(?:\+([0-9A-Za-z\-.]+))?$`)

// New parses a semantic version string.
func New(s string) *Version {
	s = strings.TrimSpace(s)
	matches := versionRegex.FindStringSubmatch(s)
	if matches == nil {
		return nil
	}

	major, _ := strconv.ParseInt(matches[1], 10, 64)
	minor, _ := strconv.ParseInt(matches[2], 10, 64)
	patch, _ := strconv.ParseInt(matches[3], 10, 64)

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		PreRelease: matches[4],
		Metadata:   matches[5],
		original:   s,
	}
}

// MustNew is like New but panics on invalid input.
func MustNew(s string) *Version {
	v := New(s)
	if v == nil {
		panic(fmt.Sprintf("invalid semver: %s", s))
	}
	return v
}

// Compare compares two versions: -1 (v < other), 0 (v == other), 1 (v > other).
func (v *Version) Compare(other *Version) int {
	if v == nil && other == nil {
		return 0
	}
	if v == nil {
		return -1
	}
	if other == nil {
		return 1
	}

	if v.Major != other.Major {
		return cmpInt64(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return cmpInt64(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return cmpInt64(v.Patch, other.Patch)
	}
	return comparePreRelease(v.PreRelease, other.PreRelease)
}

// LessThan checks v < other.
func (v *Version) LessThan(other *Version) bool {
	return v.Compare(other) < 0
}

// LessThanOrEqual checks v <= other.
func (v *Version) LessThanOrEqual(other *Version) bool {
	return v.Compare(other) <= 0
}

// Equal checks version equality (metadata ignored).
func (v *Version) Equal(other *Version) bool {
	return v.Compare(other) == 0
}

// GreaterThan checks v > other.
func (v *Version) GreaterThan(other *Version) bool {
	return v.Compare(other) > 0
}

// GreaterThanOrEqual checks v >= other.
func (v *Version) GreaterThanOrEqual(other *Version) bool {
	return v.Compare(other) >= 0
}

// String returns "MAJOR.MINOR.PATCH[-PRERELEASE][+METADATA]".
func (v *Version) String() string {
	if v == nil {
		return ""
	}
	if v.original != "" && v.PreRelease == "" && v.Metadata == "" {
		return v.original
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		fmt.Fprintf(&sb, "-%s", v.PreRelease)
	}
	if v.Metadata != "" {
		fmt.Fprintf(&sb, "+%s", v.Metadata)
	}
	return sb.String()
}

func cmpInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// comparePreRelease compares prerelease identifiers split by ".".
func comparePreRelease(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return 1 // release > prerelease
	}
	if b == "" {
		return -1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		if i >= len(aParts) {
			return -1
		}
		if i >= len(bParts) {
			return 1
		}

		aPart, bPart := aParts[i], bParts[i]
		aNum, aIsNum := tryParseInt(aPart)
		bNum, bIsNum := tryParseInt(bPart)

		if aIsNum && bIsNum {
			if aNum != bNum {
				return cmpInt64(aNum, bNum)
			}
			continue
		}
		if aIsNum {
			return -1 // numeric identifiers have lower precedence
		}
		if bIsNum {
			return 1
		}
		if aPart < bPart {
			return -1
		}
		if aPart > bPart {
			return 1
		}
	}
	return 0
}

func tryParseInt(s string) (int64, bool) {
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}
