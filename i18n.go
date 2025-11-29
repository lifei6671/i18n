package i18n

import (
	"sync"
)

// MessageStore lang -> key -> message
type MessageStore map[string]map[string]string

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
			if lang != "" {
				chain = append(chain, lang)
			}
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

// Locale 是绑定了“语言链”的翻译入口
type Locale struct {
	bundle *Bundle
	langs  []string // lang fallback chain
}

// T 翻译函数：T("user.login.success", map[string]any{"name": "Tom"})
func (l *Locale) T(key string, args map[string]any) string {
	if l.bundle == nil {
		return key
	}
	l.bundle.mu.RLock()
	defer l.bundle.mu.RUnlock()

	for _, lang := range l.langs {
		if msgs, ok := l.bundle.messages[lang]; ok {
			if text, ok2 := msgs[key]; ok2 {
				// 使用自定义模板引擎替换 {name} 等占位符
				res, err := RenderTemplate(text, args)
				if err != nil {
					// 模板解析失败时，退化为原文
					return text
				}
				return res
			}
		}
	}
	// 找不到翻译时，直接返回 key（或者返回 key + 提示）
	return key
}
