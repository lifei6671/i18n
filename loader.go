package i18n

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// yamlFile 结构和上面给的示例 YAML 对应
type yamlFile struct {
	Language string            `yaml:"language"`
	Messages map[string]string `yaml:"messages"`
}

// Config 定义 i18n 的基础配置
type Config struct {
	// 默认语言，例如 "en"
	DefaultLang string

	// Fallback 链，比如：
	// "zh-CN": {"zh-CN", "zh", "en"}
	// "zh": {"zh", "en"}
	// 如果 Locale 没有显式传 langs，就使用 DefaultLang + 对应 fallback
	Fallbacks map[string][]string
}

// Bundle 是整个 i18n 的核心对象，负责持有所有语言的数据
type Bundle struct {
	mu       sync.RWMutex
	messages MessageStore
	config   Config
}

// New 创建一个新的 Bundle
func New(cfg Config) *Bundle {
	if cfg.DefaultLang == "" {
		cfg.DefaultLang = "en"
	}
	if cfg.Fallbacks == nil {
		cfg.Fallbacks = make(map[string][]string)
	}
	return &Bundle{
		messages: make(MessageStore),
		config:   cfg,
	}
}

// RegisterMessages 注册某个语言的一批翻译信息
// 通常由 loader.go 调用
func (b *Bundle) RegisterMessages(lang string, msgs map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.messages == nil {
		b.messages = make(MessageStore)
	}
	// 简单做 merge，不做删除
	if _, ok := b.messages[lang]; !ok {
		b.messages[lang] = make(map[string]string)
	}
	for k, v := range msgs {
		b.messages[lang][k] = v
	}
}

// Locale 返回一个 Locale 视图，用于在业务中做翻译
// lang 可以是 "zh-CN" / "en" 等
func (b *Bundle) Locale(lang string) *Locale {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 构造 fallback 链：显式配置 > 默认语言
	var chain []string
	if lang != "" {
		if fb, ok := b.config.Fallbacks[lang]; ok && len(fb) > 0 {
			chain = append(chain, fb...)
		} else {
			// 默认：当前 lang + 默认语言
			chain = append(chain, lang)
			if b.config.DefaultLang != "" && b.config.DefaultLang != lang {
				chain = append(chain, b.config.DefaultLang)
			}
		}
	} else {
		chain = append(chain, b.config.DefaultLang)
	}

	return &Locale{
		bundle: b,
		langs:  chain,
	}
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
