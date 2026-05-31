package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/clin211/gin-enterprise-template/pkg/db"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

// 帮助信息文本.
const helpText = `Usage: main [flags] [component...]

This is a code generator for GORM models.

Flags:
`

// Querier 定义了数据库查询接口.
type Querier interface {
	// FilterWithNameAndRole 按名称和角色查询记录
	FilterWithNameAndRole(name string) ([]gen.T, error)
}

// GenerateConfig 保存代码生成的配置.
type GenerateConfig struct {
	ModelPackagePath string
	GenerateFunc    func(g *gen.Generator)
}

// 预定义的生成配置.
var generateConfigs = map[string]GenerateConfig{
	"gin-enterprise-template": {ModelPackagePath: "../../internal/apiserver/model", GenerateFunc: GenerateTemplateModels},
}

// 命令行参数.
var (
	configFile = pflag.StringP("config", "c", "", "Config file path (default: ./configs/gin-enterprise-template-apiserver.yaml)")
	modelPath  = ""
	components = pflag.StringSlice("component", []string{"gin-enterprise-template"}, "Generated model code's for specified component.")
	help      = pflag.BoolP("help", "h", false, "Show this help message.")
)

func main() {
	// 设置自定义的使用说明函数
	pflag.Usage = func() {
		fmt.Printf("%s", helpText)
		pflag.PrintDefaults()
	}
	pflag.Parse()

	// 如果设置了帮助标志，则显示帮助信息并退出
	if *help {
		pflag.Usage()
		return
	}

	// 初始化数据库连接
	dbInstance, err := initializeDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 处理组件并生成代码
	for _, component := range *components {
		processComponent(component, dbInstance)
	}
}

// Config 数据库配置结构
type Config struct {
	PostgreSQL PostgreSQLConfig `mapstructure:"postgresql"`
}

// PostgreSQLConfig PostgreSQL 配置
type PostgreSQLConfig struct {
	Addr     string `mapstructure:"addr"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// loadConfig 从配置文件加载配置
func loadConfig(path string) (*Config, error) {
	// 获取配置文件的绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", absPath)
	}

	v := viper.New()
	v.SetConfigFile(absPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// initializeDatabase 创建并返回一个数据库连接.
func initializeDatabase() (*gorm.DB, error) {
	// 确定配置文件路径
	configPath := *configFile
	if configPath == "" {
		// 默认从项目根目录的 configs 目录查找
		// 获取当前工作目录的父目录的父目录（即项目根目录）
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		// 从 cmd/gen-gorm-model 目录回退到项目根目录
		configPath = filepath.Join(cwd, "..", "..", "configs", "gin-enterprise-template-apiserver.yaml")
	}

	// 加载配置
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	log.Printf("Using database config from: %s", configPath)
	log.Printf("Database: %s@%s/%s", cfg.PostgreSQL.Username, cfg.PostgreSQL.Addr, cfg.PostgreSQL.Database)

	dbOptions := &db.PostgreSQLOptions{
		Addr:     cfg.PostgreSQL.Addr,
		Username: cfg.PostgreSQL.Username,
		Password: cfg.PostgreSQL.Password,
		Database: cfg.PostgreSQL.Database,
	}

	// 创建并返回数据库连接
	return db.NewPostgreSQL(dbOptions)
}

// processComponent 处理单个组件以生成代码.
func processComponent(component string, dbInstance *gorm.DB) {
	config, ok := generateConfigs[component]
	if !ok {
		log.Printf("Component '%s' not found in configuration. Skipping.", component)
		return
	}

	// 解析模型包路径
	modelPkgPath := resolveModelPackagePath(config.ModelPackagePath)

	// 创建生成器实例
	generator := createGenerator(modelPkgPath)
	generator.UseDB(dbInstance)

	// 应用自定义生成器选项
	applyGeneratorOptions(generator)

	// 使用指定的函数生成模型
	config.GenerateFunc(generator)

	// 执行代码生成
	generator.Execute()
}

// resolveModelPackagePath 确定模型生成的包路径.
func resolveModelPackagePath(defaultPath string) string {
	if modelPath != "" {
		return modelPath
	}
	absPath, err := filepath.Abs(defaultPath)
	if err != nil {
		log.Printf("Error resolving path: %v", err)
		return defaultPath
	}
	return absPath
}

// createGenerator 初始化并返回一个新的生成器实例.
func createGenerator(packagePath string) *gen.Generator {
	return gen.NewGenerator(gen.Config{
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
		ModelPkgPath:      packagePath,
		WithUnitTest:      true,
		FieldNullable:     true,  // 对于数据库中可空的字段，使用指针类型。
		FieldSignable:     false, // 禁用无符号属性以提高兼容性。
		FieldWithIndexTag: false, // 不包含 GORM 的索引标签。
		FieldWithTypeTag:  false, // 不包含 GORM 的类型标签。
	})
}

// applyGeneratorOptions 设置自定义生成器选项.
func applyGeneratorOptions(g *gen.Generator) {
	// 为特定字段自定义 GORM 标签
	g.WithOpts(
		// 将下划线连接起来的单词使用小驼峰的形式写入 json tag 中
		gen.FieldJSONTagWithNS(func(columnName string) (tagContent string) {
			// 只对包含下划线的字段名进行转换
			if strings.Contains(columnName, "_") {
				return lo.CamelCase(columnName)
			}
			return columnName
		}),
		// 为时间戳字段设置 PostgreSQL 默认值
		gen.FieldGORMTag("created_at", func(tag field.GormTag) field.GormTag {
			tag.Set("default", "current_timestamp")
			return tag
		}),
		gen.FieldGORMTag("updated_at", func(tag field.GormTag) field.GormTag {
			tag.Set("default", "current_timestamp")
			return tag
		}),
	)
}

// GenerateTemplateModels 为 gin-enterprise-template 组件生成模型。
func GenerateTemplateModels(g *gen.Generator) {
	// 系统核心表
	g.GenerateModelAs("user", "UserM")
	g.GenerateModelAs("user_config", "UserConfigM")
	g.GenerateModelAs("user_login_log", "UserLoginLogM")

	// RBAC 权限控制表
	g.GenerateModelAs("role", "RoleM")
	g.GenerateModelAs("user_role", "UserRoleM")
	g.GenerateModelAs("permission", "PermissionM")
	g.GenerateModelAs("role_permission", "RolePermissionM")
	g.GenerateModelAs("menu", "MenuM")
	g.GenerateModelAs("menu_role", "MenuRoleM")
	g.GenerateModelAs("audit_log", "AuditLogM")

	// 定时任务表
	g.GenerateModelAs("scheduled_task", "ScheduledTaskM")
	g.GenerateModelAs("scheduled_task_execution", "ScheduledTaskExecutionM")

	// 权限控制表
	g.GenerateModelAs("casbin_rule", "CasbinRuleM")
}
