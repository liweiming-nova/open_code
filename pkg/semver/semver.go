package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version 表示一个语义化版本
type Version struct {
	Major      int64
	Minor      int64
	Patch      int64
	PreRelease string // 如: "alpha", "beta.1"
	Metadata   string // 如: "build.123" (不参与比较)
	original   string // 保留原始字符串
}

// semver 正则表达式 (简化版，符合 2.0.0 核心规范)
var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z\-.]+))?(?:\+([0-9A-Za-z\-.]+))?$`)

// New 解析版本字符串，失败返回 nil
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

// MustNew 类似 New，但解析失败会 panic
func MustNew(s string) *Version {
	v := New(s)
	if v == nil {
		panic(fmt.Sprintf("invalid semver: %s", s))
	}
	return v
}

// Compare 比较两个版本: -1 (v < other), 0 (相等), 1 (v > other)
// 遵循 semver 2.0.0: 预发布版本 < 正式版本, 元数据不参与比较
func (v *Version) Compare(other *Version) int {
	// 1. 比较主版本号
	if v.Major != other.Major {
		return cmpInt64(v.Major, other.Major)
	}
	// 2. 比较次版本号
	if v.Minor != other.Minor {
		return cmpInt64(v.Minor, other.Minor)
	}
	// 3. 比较修订号
	if v.Patch != other.Patch {
		return cmpInt64(v.Patch, other.Patch)
	}
	// 4. 预发布版本比较 (空 = 正式版本 > 预发布)
	return comparePreRelease(v.PreRelease, other.PreRelease)
}

// LessThan 判断 v < other
func (v *Version) LessThan(other *Version) bool {
	return v.Compare(other) < 0
}

// Equal 判断版本号相等 (忽略元数据)
func (v *Version) Equal(other *Version) bool {
	return v.Compare(other) == 0
}

// String 返回标准格式: "MAJOR.MINOR.PATCH[-PRERELEASE][+METADATA]"
func (v *Version) String() string {
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

// --- 内部辅助函数 ---

func cmpInt64(a, b int64) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// 比较预发布版本: 按 "." 分割标识符，逐个比较 (数字按数值, 其他按 ASCII)
func comparePreRelease(a, b string) int {
	// 规则: 无预发布 > 有预发布 (1.0.0 > 1.0.0-alpha)
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return 1 // 正式版本 > 预发布
	}
	if b == "" {
		return -1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		if i >= len(aParts) {
			return -1 // a 的标识符更少，优先级更低
		}
		if i >= len(bParts) {
			return 1
		}

		aPart, bPart := aParts[i], bParts[i]

		// 尝试按数字比较
		aNum, aIsNum := tryParseInt(aPart)
		bNum, bIsNum := tryParseInt(bPart)

		if aIsNum && bIsNum {
			if aNum != bNum {
				return cmpInt64(aNum, bNum)
			}
		} else if aIsNum {
			return -1 // 数字标识符 < 非数字标识符
		} else if bIsNum {
			return 1
		} else {
			// 都是字符串，按 ASCII 比较
			if aPart != bPart {
				if aPart < bPart {
					return -1
				}
				return 1
			}
		}
	}
	return 0
}

func tryParseInt(s string) (int64, bool) {
	// 前导零的数字标识符按字符串处理 (semver 规范)
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}
