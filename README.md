# i18n-go: A Lightweight, Fast and Extensible Golang i18n Engine

`i18n-go` 是一个 **轻量、可扩展、高性能** 的 Golang 国际化引擎，支持：

* YAML 翻译文件
* 短 key 格式
* 自定义模板语法
* 嵌套字段访问
* 日期 & 数字格式化
* 链式 Formatter
* 条件表达式
* AST 解析与缓存
* 自定义 Formatter 注册
* i18n Lint 工具（检查缺失与冗余 key）

该项目特别适合：

* Web 服务（Gin、Fiber、Echo、gRPC）
* 多语言后端应用
* 企业级 SaaS
* AI / ChatGPT 类 prompt 模板国际化
* 可扩展插件体系的 i18n 服务

---

## Features

### 强大的模板能力（简单优雅）

模板格式：

```
{path}
{user.name}
{order.price | number}
{order.price | number:2 | currency:¥}
{created_at | date:2006-01-02}
{count | eq:0?No items:{count} items}
{name | upper}
```

支持：

| 功能            | 描述                                    |   
| ------------- |---------------------------------------| 
| 嵌套字段          | `{user.name}` / `{order.price}`       |            
| 数字格式化         | `{price \| number:2}`                 |            
| 日期格式化         | `{date \| date:2006-01-02}`           |            
| 货币格式化         | `{price \| currency:¥}`               |            
| 大小写处理         | `upper/lower/title`                   |   
| 条件表达式         | `{count \| eq:0?No:{count}}`          |            
| 链式 Formatter  | `{price \| number:2 \| currency}`     |
| 自定义 Formatter | `RegisterFormatter("slugify", func…)` | 

---

## Performance

* **AST 缓存机制**：模板解析只发生一次
* 渲染阶段直接执行 AST，无需重复解析字符串
* Formatter 插件通过注册表查找，性能等同原生函数调用

非常适合高 QPS 服务。

---

## Installation

```sh
go get github.com/lifei6671/i18n
```

---

## Usage

### 1. 加载 YAML 翻译文件

```yaml
# locales/en.yaml
language: en
messages:
  user.login.success: "Welcome back, {user.name}!"
  order.info: "Price: {order.price | number:2 | currency:$}"
```

```yaml
# locales/zh-CN.yaml
language: zh-CN
messages:
  user.login.success: "欢迎回来，{user.name}！"
  order.info: "价格：{order.price | number:2 | currency:¥}"
```

### 2. 初始化 i18n Bundle

```go
bundle := i18n.New(i18n.Config{
    DefaultLang: "en",
    Fallbacks: map[string][]string{
        "zh-CN": {"zh-CN", "zh", "en"},
        "en":    {"en"},
    },
})

bundle.MustLoadYAMLDir("./locales")

loc := bundle.Locale("zh-CN")
msg := loc.T("user.login.success", map[string]any{
    "user": map[string]any{"name": "咸鱼"},
})
fmt.Println(msg)
```

输出：

```
欢迎回来，咸鱼！
```

---

# Template Syntax

模板使用 `{...}` 占位符，内部为“表达式 + 可选格式化链”。

### 支持的表达式结构

```
{path}
{path | formatter}
{path | formatter:arg}
{path | f1 | f2:arg}
{path | op:val?true:false}
```

### 嵌套字段访问

```
{user.name}
{order.detail.price}
```

支持：

* map[string]any
* struct 字段（大小写不敏感）

---

# Formatters

默认支持以下 Formatter：

| Name       | 示例                       | 说明                |             
| ---------- |--------------------------| ----------------- | 
| number     | `{p \| number}`          | 千分位格式化（无小数） |
| number:2   | `{p \| number:2}`        | 指定小数位       |
| currency   | `{p \| currency}`        | 默认 `$`      |
| currency:¥ | `{p \| currency:¥}`      | 自定义符号       |
| date       | `{t \| date:2006-01-02}` | Go 时间格式化    |
| upper      | `{name \| upper}`        | 全大写         |
| lower      | `{name \| lower}`        | 全小写         |
| title      | `{name \| title}`        | 首字母大写       |

支持链式调用：

```
{price | number:2 | currency:¥}
```

---

# Conditional Expression

支持轻量级条件表达式：

```
{count | eq:0?No items:{count} items}
```

语法：

```
op:value?expr_true:expr_false
```

操作符：

| op | 说明 |
| -- | -- |
| eq | 等于 |
| gt | 大于 |
| lt | 小于 |

---

# Register Custom Formatters

你可以像插件一样注册新的 Formatter。

示例：注册一个 slugify 格式化器

```go
i18n.RegisterFormatter("slugify", func(v any, arg string) (any, error) {
    s := strings.ToLower(fmt.Sprint(v))
    s = strings.ReplaceAll(s, " ", "-")
    return s, nil
})
```

使用：

```go
tpl := "URL: /user/{name | slugify}"
result, _ := i18n.RenderTemplate(tpl, map[string]any{
    "name": "Xian Yu NoFlip",
})
fmt.Println(result)
```

输出：

```
URL: /user/xian-yu-noflip
```

---

# AST Cache

`RenderTemplate` 内部自动维护一个 AST 缓存：

```go
cacheMutex.RLock()
ast, ok := astCache[tpl]
cacheMutex.RUnlock()

if !ok {
    ast = ParseTemplate(tpl)
    cacheMutex.Lock()
    astCache[tpl] = ast
    cacheMutex.Unlock()
}
```

缓存带来：

* 模板解析 只发生 1 次
* 渲染成本只剩 AST Eval
* 性能提升 5x~50x（depending on template complexity）

适合高 QPS 应用或大规模 i18n 替换。

---

# Lint Tool

内置 Lint 工具检查：

* 缺失 key
* 冗余 key
* 多语言文件 key 不对齐

使用方式：

```sh
go build -o i18n-lint ./cmd/i18n-lint
./i18n-lint -d ./locales -fail
```

输出示例：

```
=== I18N CHECK RESULT ===
Languages: [en zh-CN]
Total keys: 5

--- [zh-CN] ---
Missing keys:
  - order.info
```

CI 中可用 `-fail` 让校验失败。

---

# Project Structure Suggestion

推荐目录结构：

```
i18n/
  i18n.go
  loader.go
  template.go
  registry.go
  locales/
    en.yaml
    zh-CN.yaml
  cmd/
    i18n-lint/
      main.go
```

---

# Example: Full Template Rendering

```go
result, _ := i18n.RenderTemplate(
    "User {user.name | upper}, price {order.price | number:2 | currency:¥}, items {count | eq:0?No items:{count} items}, created {order.created_at | date:2006/01/02}",
    map[string]any{
        "user": map[string]any{"name": "咸鱼"},
        "order": map[string]any{
            "price":      12345.678,
            "created_at": time.Now(),
        },
        "count": 5,
    },
)

fmt.Println(result)
```

输出：

```
User 咸鱼, price ¥12,345.68, items 5 items, created 2025/11/29
```

---

# License

MIT License
