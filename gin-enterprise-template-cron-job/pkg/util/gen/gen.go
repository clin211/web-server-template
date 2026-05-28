package gen

import (
	"fmt"
	"os"
	"path/filepath"
)

// OutDir 从路径创建绝对路径名并检查路径是否存在。
// 返回包括尾随 '/' 的绝对路径，如果路径不存在则返回错误。
func OutDir(path string) (string, error) {
	outDir, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	stat, err := os.Stat(outDir)
	if err != nil {
		return "", err
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("output directory %s is not a directory", outDir)
	}
	outDir = outDir + "/"
	return outDir, nil
}
