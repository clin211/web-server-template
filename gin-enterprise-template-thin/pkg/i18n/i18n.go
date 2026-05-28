package i18n

import (
	"embed"
	"encoding/json"
	"errors"
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

// New 使用给定选项创建 I18n 结构体实例。
// 接受可变的函数选项参数，并返回指向 I18n 结构体的指针。
func New(options ...func(*Options)) (rp *I18n) {
	ops := getOptionsOrSetDefault(nil)
	for _, f := range options {
		f(ops)
	}
	bundle := i18n.NewBundle(ops.language)
	localizer := i18n.NewLocalizer(bundle, ops.language.String())
	switch ops.format {
	case "toml":
		bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	case "json":
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	default:
		bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	}
	rp = &I18n{
		ops:       *ops,
		bundle:    bundle,
		localizer: localizer,
		lang:      ops.language,
	}
	for _, item := range ops.files {
		rp.Add(item)
	}
	rp.AddFS(ops.fs)
	return
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

// LocalizeT 本地化给定消息并返回本地化字符串。
// 如果无法翻译，则返回消息 ID 作为默认消息。
func (i I18n) LocalizeT(message *i18n.Message) (rp string) {
	if message == nil {
		return ""
	}

	var err error
	rp, err = i.localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: message,
	})
	if err != nil {
		// 无法翻译时使用 id 作为默认消息
		rp = message.ID
	}
	return
}

// LocalizeE 是 LocalizeT 方法的包装器，将本地化字符串转换为错误类型并返回。
func (i I18n) LocalizeE(message *i18n.Message) error {
	return errors.New(i.LocalizeT(message))
}

// T 本地化给定 ID 的消息并返回本地化字符串。
// 使用 LocalizeT 方法执行翻译。
func (i I18n) T(id string) (rp string) {
	return i.LocalizeT(&i18n.Message{ID: id})
}

// E 是 T 的包装器，将本地化字符串转换为错误类型并返回。
func (i I18n) E(id string) error {
	return errors.New(i.T(id))
}

// Add 添加语言文件或目录(通过文件名自动获取语言)。
func (i *I18n) Add(f string) {
	info, err := os.Stat(f)
	if err != nil {
		return
	}
	if info.IsDir() {
		filepath.Walk(f, func(path string, fi os.FileInfo, errBack error) (err error) {
			if !fi.IsDir() {
				i.bundle.LoadMessageFile(path)
			}
			return
		})
	} else {
		i.bundle.LoadMessageFile(f)
	}
}

// AddFS 添加语言嵌入文件。
func (i *I18n) AddFS(fs embed.FS) {
	files := readFS(fs, ".")
	for _, name := range files {
		i.bundle.LoadMessageFileFS(fs, name)
	}
}

func readFS(fs embed.FS, dir string) (rp []string) {
	rp = make([]string, 0)
	dirs, err := fs.ReadDir(dir)
	if err != nil {
		return
	}
	for _, item := range dirs {
		name := dir + string(os.PathSeparator) + item.Name()
		if dir == "." {
			name = item.Name()
		}
		if item.IsDir() {
			rp = append(rp, readFS(fs, name)...)
		} else {
			rp = append(rp, name)
		}
	}
	return
}
