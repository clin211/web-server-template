package app

import (
	"testing"

	cliflag "k8s.io/component-base/cli/flag"
)

func TestNewAppWithoutOptionsRegistersFlags(t *testing.T) {
	application := NewApp("example", "Example application")
	for _, name := range []string{"config", "version"} {
		if application.Command().Flags().Lookup(name) == nil && application.Command().PersistentFlags().Lookup(name) == nil {
			t.Errorf("missing %s flag", name)
		}
	}
}

type namedTestOptions struct{}

func (namedTestOptions) Flags() cliflag.NamedFlagSets { return cliflag.NamedFlagSets{} }
func (namedTestOptions) Complete() error              { return nil }
func (namedTestOptions) Validate() error              { return nil }

func TestNewAppNamedOptionsRegistersGlobalFlags(t *testing.T) {
	application := NewApp("example", "Example application", WithOptions(namedTestOptions{}))
	for _, name := range []string{"config", "version"} {
		if application.Command().Flags().Lookup(name) == nil && application.Command().PersistentFlags().Lookup(name) == nil {
			t.Errorf("missing %s flag", name)
		}
	}
}

func TestConfigFlagKeepsDefaultHelp(t *testing.T) {
	application := NewApp("example", "Example application")
	flag := application.Command().PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("config flag missing")
	}
	if want := "从指定的 `FILE` 读取配置，支持 JSON、TOML、YAML、HCL 或 Java properties 格式。"; flag.Usage != want {
		t.Fatalf("config help = %q, want %q", flag.Usage, want)
	}
}
