package i18n

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// yamlFile 结构和上面给的示例 YAML 对应
type yamlFile struct {
	Language string            `yaml:"language"`
	Messages map[string]string `yaml:"messages"`
}

// LoadYAMLDir 从目录中加载所有 `.yaml/.yml` 文件
// 例如: ./locales/en.yaml, ./locales/zh-CN.yaml
func (b *Bundle) LoadYAMLDir(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		if err := b.loadYAMLFile(path); err != nil {
			return fmt.Errorf("loadYAMLFile %s: %w", path, err)
		}
		return nil
	})
}

func (b *Bundle) loadYAMLFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var yf yamlFile
	if err := yaml.Unmarshal(data, &yf); err != nil {
		return fmt.Errorf("yaml unmarshal: %w", err)
	}
	if yf.Language == "" {
		return fmt.Errorf("file %s missing 'language' field", path)
	}
	if len(yf.Messages) == 0 {
		return nil
	}
	b.RegisterMessages(yf.Language, yf.Messages)
	return nil
}

// MustLoadYAMLDir 版本，在初始化阶段直接 panic
func (b *Bundle) MustLoadYAMLDir(dir string) {
	if err := b.LoadYAMLDir(dir); err != nil {
		panic(err)
	}
}
