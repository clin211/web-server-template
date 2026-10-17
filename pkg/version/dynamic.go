package version

import (
	"errors"
	"fmt"
	"sync/atomic"

	utilversion "github.com/clin211/gin-enterprise-template/pkg/util/version"
)

var dynamicGitVersion atomic.Value

var (
	errEmptyVersion    = errors.New("version must not be empty")
	errVersionMismatch = errors.New("version must match major/minor/patch of default version")
)

// SetDynamicVersion 覆盖从 Get() 返回的 GitVersion。
// 指定的版本必须非空、是有效的语义化版本，并且必须
// 与默认 gitVersion 的主版本号/次版本号/补丁版本号匹配。
func SetDynamicVersion(dynamicVersion string) error {
	if err := ValidateDynamicVersion(dynamicVersion); err != nil {
		return err
	}
	dynamicGitVersion.Store(dynamicVersion)
	return nil
}

// ValidateDynamicVersion 确保给定的版本非空、是有效的语义化版本，
// 并且与默认 gitVersion 的主版本号/次版本号/补丁版本号匹配。
func ValidateDynamicVersion(dynamicVersion string) error {
	return validateDynamicVersion(dynamicVersion, gitVersion)
}

func validateDynamicVersion(dynamicVersion, defaultVersion string) error {
	if len(dynamicVersion) == 0 {
		return errEmptyVersion
	}
	if dynamicVersion == defaultVersion {
		// 允许无操作
		return nil
	}
	vRuntime, err := utilversion.ParseSemantic(dynamicVersion)
	if err != nil {
		return err
	}
	// 必须与默认版本的主版本号/次版本号/补丁版本号匹配
	var vDefault *utilversion.Version
	if defaultVersion == "v0.0.0-master+$Format:%H$" {
		// 特殊处理无法解析为语义化版本的占位符值
		vDefault, err = utilversion.ParseSemantic("v0.0.0-master")
	} else {
		vDefault, err = utilversion.ParseSemantic(defaultVersion)
	}
	if err != nil {
		return err
	}
	if vRuntime.Major() != vDefault.Major() || vRuntime.Minor() != vDefault.Minor() || vRuntime.Patch() != vDefault.Patch() {
		return fmt.Errorf("%w: %q and %q", errVersionMismatch, dynamicVersion, defaultVersion)
	}
	return nil
}
