package pagination

// PageSizeConfig 定义分页大小配置
type PageSizeConfig struct {
	Default int64 // 默认每页条数
	Max     int64 // 最大每页条数
}

// DefaultPageSizeConfig 返回默认的分页配置
func DefaultPageSizeConfig() PageSizeConfig {
	return PageSizeConfig{
		Default: 20,
		Max:     100,
	}
}

// NormalizePageSize 规范化分页大小，应用默认值和最大值限制
func NormalizePageSize(pageSize int64) int {
	return NormalizePageSizeWithConfig(pageSize, DefaultPageSizeConfig())
}

// NormalizePageSizeWithConfig 使用自定义配置规范化分页大小
func NormalizePageSizeWithConfig(pageSize int64, config PageSizeConfig) int {
	if pageSize <= 0 {
		return int(config.Default)
	}
	if pageSize > config.Max {
		return int(config.Max)
	}
	return int(pageSize)
}
