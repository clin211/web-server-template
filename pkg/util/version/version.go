package version

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Version 是版本号的不透明表示
type Version struct {
	components    []uint
	semver        bool
	preRelease    string
	buildMetadata string
}

var (
	// versionMatchRE 将版本字符串拆分为数字和"额外"部分
	versionMatchRE = regexp.MustCompile(`^\s*v?([0-9]+(?:\.[0-9]+)*)(.*)*$`)
	// extraMatchRE 将 versionMatchRE 的"额外"部分拆分为 semver 预发布和构建元数据；它不验证预发布的"无前导零"约束
	extraMatchRE = regexp.MustCompile(`^(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?\s*$`)
)

var (
	errInvalidVersion     = errors.New("invalid version")
	errNoSupportedVersion = errors.New("no supported v1.x version")
)

func parse(str string, semver bool) (*Version, error) {
	parts := versionMatchRE.FindStringSubmatch(str)
	if parts == nil {
		return nil, fmt.Errorf("%w: could not parse %q", errInvalidVersion, str)
	}
	components, err := parseComponents(parts[1], str, semver)
	if err != nil {
		return nil, err
	}
	v := &Version{components: components, semver: semver}
	if semver && parts[2] != "" {
		if err := v.parseExtra(parts[2], str); err != nil {
			return nil, err
		}
	}
	return v, nil
}

func parseComponents(numbers, str string, semver bool) ([]uint, error) {
	parts := strings.Split(numbers, ".")
	if (semver && len(parts) != 3) || (!semver && len(parts) < 2) {
		return nil, fmt.Errorf("%w: illegal version string %q", errInvalidVersion, str)
	}
	components := make([]uint, len(parts))
	for i, component := range parts {
		number, err := parseComponent(component, i == 0 || semver)
		if err != nil {
			return nil, fmt.Errorf("version %q: %w", str, err)
		}
		components[i] = number
	}
	return components, nil
}

func parseComponent(component string, rejectZeroPrefix bool) (uint, error) {
	if rejectZeroPrefix && zeroPrefixed(component) {
		return 0, fmt.Errorf("%w: zero-prefixed component %q", errInvalidVersion, component)
	}
	number, err := strconv.ParseUint(component, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("%w: component %q: %w", errInvalidVersion, component, err)
	}
	return uint(number), nil
}

func zeroPrefixed(component string) bool {
	return strings.HasPrefix(component, "0") && component != "0"
}

func (v *Version) parseExtra(extra, str string) error {
	parts := extraMatchRE.FindStringSubmatch(extra)
	if parts == nil {
		return fmt.Errorf("%w: pre-release/metadata %q in %q", errInvalidVersion, extra, str)
	}
	v.preRelease, v.buildMetadata = parts[1], parts[2]
	for component := range strings.SplitSeq(v.preRelease, ".") {
		if _, err := strconv.ParseUint(component, 10, 0); err == nil && zeroPrefixed(component) {
			return fmt.Errorf("%w: zero-prefixed component %q in %q", errInvalidVersion, component, str)
		}
	}
	return nil
}

// HighestSupportedVersion 返回支持的最高版本。
// 此函数假设支持的最高版本必须是 v1.x。
func HighestSupportedVersion(versions []string) (*Version, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("%w: empty array", errNoSupportedVersion)
	}
	highest, err := highestSupportedVersion(versions)
	if err != nil {
		return nil, err
	}
	if highest.Major() != 1 {
		return nil, fmt.Errorf("%w: highest version is %v", errNoSupportedVersion, highest)
	}
	return highest, nil
}

func highestSupportedVersion(versions []string) (*Version, error) {
	var highest *Version
	var parseErr error
	for _, version := range slices.Backward(versions) {
		current, err := ParseGeneric(version)
		if err != nil {
			parseErr = err
			continue
		}
		if current.Major() > 1 {
			continue
		}
		if highest == nil || highest.LessThan(current) {
			highest = current
		}
	}
	if highest == nil {
		return nil, fmt.Errorf("versions %v: %w", versions, errors.Join(errNoSupportedVersion, parseErr))
	}
	return highest, nil
}

// ParseGeneric 解析"通用"版本字符串。版本字符串必须由两个
// 或多个点分隔的数字字段组成（其中第一个不能有前导零），
// 后跟任意未解释的数据（不一定需要通过标点符号与最终的数字字段分隔）。
// 为了方便，前导和尾随空格将被忽略，版本前面可以加字母"v"。另请参见 ParseSemantic。
func ParseGeneric(str string) (*Version, error) {
	return parse(str, false)
}

// MustParseGeneric 类似于 ParseGeneric，但在出错时会 panic
func MustParseGeneric(str string) *Version {
	v, err := ParseGeneric(str)
	if err != nil {
		panic(err)
	}
	return v
}

// ParseSemantic 解析完全遵守"语义版本"规范 (http://semver.org/) 的语法和语义的版本字符串
// （尽管它忽略前导和尾随空格，并允许版本前面加"v"）。
// 对于不保证遵守语义版本语法的版本字符串，请使用 ParseGeneric。
func ParseSemantic(str string) (*Version, error) {
	return parse(str, true)
}

// MustParseSemantic 类似于 ParseSemantic，但在出错时会 panic
func MustParseSemantic(str string) *Version {
	v, err := ParseSemantic(str)
	if err != nil {
		panic(err)
	}
	return v
}

// MajorMinor 返回具有提供的主版本和次版本的版本。
func MajorMinor(major, minor uint) *Version {
	return &Version{components: []uint{major, minor}}
}

// Major 返回主版本号
func (v *Version) Major() uint {
	return v.components[0]
}

// Minor 返回次版本号
func (v *Version) Minor() uint {
	return v.components[1]
}

// Patch 如果 v 是语义版本，则返回补丁版本号，否则返回 0
func (v *Version) Patch() uint {
	if len(v.components) < 3 {
		return 0
	}
	return v.components[2]
}

// BuildMetadata 返回构建元数据（如果 v 是语义版本），否则返回 ""
func (v *Version) BuildMetadata() string {
	return v.buildMetadata
}

// PreRelease 返回预发布元数据（如果 v 是语义版本），否则返回 ""
func (v *Version) PreRelease() string {
	return v.preRelease
}

// Components 返回版本号组件
func (v *Version) Components() []uint {
	return v.components
}

// WithMajor 返回具有请求的主版本号的版本对象副本
func (v *Version) WithMajor(major uint) *Version {
	result := *v
	result.components = []uint{major, v.Minor(), v.Patch()}
	return &result
}

// WithMinor 返回具有请求的次版本号的版本对象副本
func (v *Version) WithMinor(minor uint) *Version {
	result := *v
	result.components = []uint{v.Major(), minor, v.Patch()}
	return &result
}

// WithPatch 返回具有请求的补丁版本号的版本对象副本
func (v *Version) WithPatch(patch uint) *Version {
	result := *v
	result.components = []uint{v.Major(), v.Minor(), patch}
	return &result
}

// WithPreRelease 返回具有请求的预发布的版本对象副本
func (v *Version) WithPreRelease(preRelease string) *Version {
	result := *v
	result.components = []uint{v.Major(), v.Minor(), v.Patch()}
	result.preRelease = preRelease
	return &result
}

// WithBuildMetadata 返回具有请求的构建元数据的版本对象副本
func (v *Version) WithBuildMetadata(buildMetadata string) *Version {
	result := *v
	result.components = []uint{v.Major(), v.Minor(), v.Patch()}
	result.buildMetadata = buildMetadata
	return &result
}

// String 将版本转换回字符串；请注意，对于使用 ParseGeneric 解析的版本，
// 这将不包括版本号的尾随未解释部分。
func (v *Version) String() string {
	if v == nil {
		return "<nil>"
	}
	var buffer bytes.Buffer

	for i, comp := range v.components {
		if i > 0 {
			buffer.WriteString(".")
		}
		fmt.Fprintf(&buffer, "%d", comp)
	}
	if v.preRelease != "" {
		buffer.WriteString("-")
		buffer.WriteString(v.preRelease)
	}
	if v.buildMetadata != "" {
		buffer.WriteString("+")
		buffer.WriteString(v.buildMetadata)
	}

	return buffer.String()
}

// compareInternal 如果 v 小于 other 返回 -1，如果大于 other 返回 1，如果相等返回 0
func (v *Version) compareInternal(other *Version) int {
	if order := compareComponents(v.components, other.components); order != 0 {
		return order
	}
	if !v.semver || !other.semver {
		return 0
	}
	return comparePreRelease(v.preRelease, other.preRelease)
}

func compareComponents(left, right []uint) int {
	for i := range min(len(left), len(right)) {
		if order := cmp.Compare(left[i], right[i]); order != 0 {
			return order
		}
	}
	switch {
	case len(left) > len(right) && !onlyZeros(left[len(right):]):
		return 1
	case len(right) > len(left) && !onlyZeros(right[len(left):]):
		return -1
	default:
		return 0
	}
}

func comparePreRelease(left, right string) int {
	switch {
	case left == right:
		return 0
	case left == "":
		return 1
	case right == "":
		return -1
	}
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	for i := range min(len(leftParts), len(rightParts)) {
		if order := comparePreReleaseComponent(leftParts[i], rightParts[i]); order != 0 {
			return order
		}
	}
	return cmp.Compare(len(leftParts), len(rightParts))
}

func comparePreReleaseComponent(left, right string) int {
	leftNumber, leftErr := strconv.ParseUint(left, 10, 0)
	rightNumber, rightErr := strconv.ParseUint(right, 10, 0)
	if leftErr == nil && rightErr == nil {
		return cmp.Compare(leftNumber, rightNumber)
	}
	return strings.Compare(left, right)
}

// 如果数组包含任何非零元素，则返回 false
func onlyZeros(array []uint) bool {
	for _, num := range array {
		if num != 0 {
			return false
		}
	}
	return true
}

// AtLeast 测试版本是否至少等于给定的最小版本。如果两个版本都是语义版本，
// 这将使用语义版本比较算法。否则，它将仅比较数字组件，
// 不存在的组件被认为是"0"（即，"1.4" 等于 "1.4.0"）。
func (v *Version) AtLeast(minimum *Version) bool {
	return v.compareInternal(minimum) != -1
}

// LessThan 测试版本是否小于给定版本。（它与 AtLeast 完全相反，
// 用于问"v 是否太旧？"比问"v 是否足够新？"更有意义的情况。）
func (v *Version) LessThan(other *Version) bool {
	return v.compareInternal(other) == -1
}

// Compare 将 v 与版本字符串进行比较（将根据 v 解析为语义或非语义版本）。
// 成功时，如果 v 小于 other 返回 -1，如果大于 other 返回 1，如果相等返回 0。
func (v *Version) Compare(other string) (int, error) {
	ov, err := parse(other, v.semver)
	if err != nil {
		return 0, err
	}
	return v.compareInternal(ov), nil
}
