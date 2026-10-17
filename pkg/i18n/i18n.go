package i18n

import (
	"embed"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// I18n 用于存储国际化的选项和配置。
type I18n struct {
	ops       Options
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
	lang      language.Tag
}

// New 使用给定选项创建翻译器，语言资源由调用方显式提供。
// 接受可变的函数选项参数，并返回指向 I18n 结构体的指针。
func New(options ...func(*Options)) *I18n {
	ops := getOptionsOrSetDefault(nil)
	for _, f := range options {
		f(ops)
	}
	bundle := i18n.NewBundle(ops.language)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	localizer := i18n.NewLocalizer(bundle, ops.language.String())
	switch ops.format {
	case "toml":
		bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	case "json":
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	default:
		bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	}
	rp := &I18n{
		ops:       *ops,
		bundle:    bundle,
		localizer: localizer,
		lang:      ops.language,
	}
	for _, item := range ops.files {
		rp.Add(item)
	}
	rp.AddFS(ops.fs)
	return rp
}

// Select 可以更改语言。
func (i I18n) Select(lang language.Tag) *I18n {
	if lang.String() == "und" {
		lang = i.ops.language
	}
	return &I18n{
		ops:       i.ops,
		bundle:    i.bundle,
		localizer: i18n.NewLocalizer(i.bundle, lang.String()),
		lang:      lang,
	}
}

// Language 获取当前语言。
func (i I18n) Language() language.Tag {
	return i.lang
}

// DefaultLanguage 获取资源回退时使用的默认语言。
func (i I18n) DefaultLanguage() language.Tag {
	return i.ops.language
}

type localizedError struct{ message string }

func (e *localizedError) Error() string { return e.message }

// Add 添加语言文件或目录，通过文件名获取语言。
// 仅在启动时调用，不能与 LocalizeT 或 Select 并发使用。
func (i I18n) Add(f string) {
	info, err := os.Stat(f)
	if err != nil {
		return
	}
	if !info.IsDir() {
		i.loadFile(f)
		return
	}
	if err := filepath.Walk(f, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !fi.IsDir() {
			i.loadFile(path)
		}
		return nil
	}); err != nil {
		slog.Warn("walk translation directory", "path", f, "error", err)
	}
}

func (i I18n) loadFile(name string) {
	if _, err := i.bundle.LoadMessageFile(name); err != nil {
		slog.Warn("load translation file", "path", name, "error", err)
	}
}

// AddFS 添加语言嵌入文件，仅在启动时调用，不能与翻译并发使用。
func (i I18n) AddFS(fs embed.FS) {
	for _, name := range readFS(fs, ".") {
		if _, err := i.bundle.LoadMessageFileFS(fs, name); err != nil {
			slog.Warn("load embedded translation file", "path", name, "error", err)
		}
	}
}

func readFS(fs embed.FS, dir string) []string {
	files := make([]string, 0)
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return files
	}
	for _, entry := range entries {
		name := entry.Name()
		if dir != "." {
			name = dir + "/" + name
		}
		if entry.IsDir() {
			files = append(files, readFS(fs, name)...)
		} else {
			files = append(files, name)
		}
	}
	return files
}
