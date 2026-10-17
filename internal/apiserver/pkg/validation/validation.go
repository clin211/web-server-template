package validation

import (
	"regexp"
	"unicode/utf8"

	"github.com/google/wire"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/pkg/i18n"
)

// Validator 是一个实现自定义验证逻辑的结构体。
type Validator struct {
	// 某些复杂的验证逻辑可能需要直接查询数据库。
	// 这只是一个示例。如果验证需要其他依赖项
	// 如客户端、服务、资源等，都可以在这里注入。
	store store.IStore
}

// 比较条件和提示参数共用阈值，调整业务规则时不需要修改语言资源。
const (
	usernameMinLength = 3
	usernameMaxLength = 20
	nicknameMaxLength = 30
	passwordMinLength = 6
	minimumListLimit  = 0
)

// 使用全局预编译的正则表达式，避免重复创建和编译。
var (
	validRegex      = regexp.MustCompile(`^[A-Za-z0-9_]+$`)                                  // 仅包含字母、数字和下划线
	validMenuCodeRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)                                 // 仅包含字母、数字、下划线和连字符
	letterRegex     = regexp.MustCompile(`[A-Za-z]`)                                         // 至少包含一个字母
	numberRegex     = regexp.MustCompile(`\d`)                                               // 至少包含一个数字
	emailRegex      = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`) // 电子邮件格式
	phoneRegex      = regexp.MustCompile(`^1[3-9]\d{9}$`)                                    // 中国手机号
)

// ProviderSet 是 Wire 提供者集，用于声明依赖注入规则。
var ProviderSet = wire.NewSet(New, wire.Bind(new(any), new(*Validator)))

// New 创建 Validator 的新实例。
func New(store store.IStore) *Validator {
	return &Validator{store: store}
}

// isValidUsername 验证用户名是否有效。
func isValidUsername(username string) bool {
	// 验证长度
	if len(username) < usernameMinLength || len(username) > usernameMaxLength {
		return false
	}
	// 验证字符合法性
	if !validRegex.MatchString(username) {
		return false
	}
	return true
}

// isValidMenuCode 验证菜单代码是否有效。
func isValidMenuCode(code string) bool {
	if len(code) == 0 || len(code) > 50 {
		return false
	}
	return validMenuCodeRe.MatchString(code)
}

// isValidPassword 检查密码是否满足复杂性要求。
func isValidPassword(password string) error {
	if err := validatePasswordRequired(password); err != nil {
		return err
	}
	switch {
	// 检查新密码的长度要求
	case utf8.RuneCountInString(password) < passwordMinLength:
		return errno.ErrInvalidArgument.
			WithMessageID("Validation.PasswordTooShort").
			WithMessageParams(i18n.Params{"MinLength": passwordMinLength})
	// 使用正则表达式检查是否至少包含一个字母
	case !letterRegex.MatchString(password):
		return errno.ErrInvalidArgument.WithMessageID("Validation.PasswordLetterRequired")
	// 使用正则表达式检查是否至少包含一个数字
	case !numberRegex.MatchString(password):
		return errno.ErrInvalidArgument.WithMessageID("Validation.PasswordNumberRequired")
	}
	return nil
}

func validatePasswordRequired(password string) error {
	if password == "" {
		return errno.ErrInvalidArgument.WithMessageID("Validation.PasswordRequired")
	}
	return nil
}

// isValidEmail 检查电子邮件是否有效。
func isValidEmail(email string) error {
	// 检查电子邮件是否为空
	if email == "" {
		return errno.ErrInvalidArgument.WithMessageID("Validation.EmailRequired")
	}

	// 使用正则表达式验证电子邮件格式
	if !emailRegex.MatchString(email) {
		return errno.ErrInvalidArgument.WithMessageID("Validation.EmailInvalid")
	}

	return nil
}

// isValidPhone 检查手机号码是否有效。
func isValidPhone(phone string) error {
	// 检查手机号码是否为空
	if phone == "" {
		return errno.ErrInvalidArgument.WithMessageID("Validation.PhoneRequired")
	}

	// 验证手机号码格式（假设为中国手机号，11位数字）
	if !phoneRegex.MatchString(phone) {
		return errno.ErrInvalidArgument.WithMessageID("Validation.PhoneInvalid")
	}

	return nil
}
