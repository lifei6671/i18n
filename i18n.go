package i18n

// MessageStore lang -> key -> message
type MessageStore map[string]map[string]string

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
