# 正则表达式引擎架构设计

**项目**: github.com/akzj/go-regex  
**版本**: Go 1.26.1  
**状态**: 设计中

---

## 1. 模块划分

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Layer (api/)                               │
│  Compile(pattern) → *Regex  |  Match/Find/Replace methods                  │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                 ▼
┌──────────────────────────┐ ┌──────────────────┐ ┌──────────────────────────┐
│     Lexer (lexer/)       │ │   Parser (parser/)│ │   Compiler (compiler/)   │
│   Tokenize pattern       │ │  AST from tokens  │ │  NFA/DFA from AST        │
└──────────────────────────┘ └──────────────────┘ └──────────────────────────┘
                                      │                       │
                                      ▼                       │
                    ┌─────────────────────────────────────────┐              │
                    │              AST (ast/)                 │              │
                    │     Node types & traversal interfaces   │              │
                    └─────────────────────────────────────────┘              │
                                      │                                        │
                                      ▼                                        │
                    ┌─────────────────────────────────────────┐              │
                    │           Machine (machine/)            │◄─────────────┘
                    │      NFA → DFA conversion                │
                    └─────────────────────────────────────────┘
                                      │
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │          Engine (engine/)               │
                    │      DFA execution, backtracking         │
                    └─────────────────────────────────────────┘
```

---

## 2. 依赖关系图 (ASCII)

```
        ┌──────────────────────────────────────────────────────────────────┐
        │                                                                  │
        │   ┌─────────┐     ┌─────────┐     ┌─────────────┐               │
        │   │  lexer  │────▶│  parser │────▶│     ast     │               │
        │   └─────────┘     └─────────┘     └─────────────┘               │
        │        │              │                  │                       │
        │        │              │                  ▼                       │
        │        │              │           ┌─────────────┐               │
        │        │              └──────────▶│  compiler   │               │
        │        │                        └──────┬──────┘               │
        │        │                               │                        │
        │        │                               ▼                        │
        │        │                        ┌─────────────┐                 │
        │        └────────────────────────▶│   machine   │                 │
        │                                 └──────┬──────┘                 │
        │                                        │                         │
        │                                        ▼                         │
        │   ┌─────────┐                    ┌─────────────┐                 │
        │   │   api   │───────────────────▶│   engine    │                 │
        │   └─────────┘                    └─────────────┘                 │
        │                                                                  │
        └──────────────────────────────────────────────────────────────────┘

依赖规则：
- api → engine, compiler, ast
- compiler → ast, machine
- parser → lexer, ast
- machine → ast (仅为类型标注，无循环依赖)
```

---

## 3. 数据结构定义

### 3.1 AST 节点 (ast/node.go)

```go
package ast

// NodeType 表示 AST 节点类型
type NodeType int

const (
    NodeChar       NodeType = iota // 单字符
    NodeConcat                      // 连接: AB
    NodeAlt                         // 选择: A|B
    NodeStar                        // 闭包: A*
    NodePlus                        // 正闭包: A+
    NodeQuest                       // 可选: A?
    NodeGroup                       // 分组: (A)
    NodeCapture                     // 捕获组: (A)
    NodeClass                       // 字符类: [abc]
    NodeNegClass                    // 否定字符类: [^abc]
    NodeBegin                       // 开始锚定: ^
    NodeEnd                         // 结束锚定: $
    NodeAny                         // 任意字符: .
    NodeRep                         // 重复: A{2,3}
    NodeEmpty                       // 空表达式
)

// Node 是所有 AST 节点的公共接口
type Node interface {
    Type() NodeType
    String() string
    // Children 返回子节点
    Children() []Node
}

// CharNode 单字符节点
type CharNode struct {
    Ch rune
}

func (n *CharNode) Type() NodeType   { return NodeChar }
func (n *CharNode) String() string   { return string(n.Ch) }
func (n *CharNode) Children() []Node { return nil }

// ClassNode 字符类节点
// 不变量: Ranges 必须是规范化的（无重叠，按起始排序）
type ClassNode struct {
    Ranges []CharRange // 字符范围: 'a'-'z'
}

func (n *ClassNode) Type() NodeType   { return NodeClass }
func (n *ClassNode) Children() []Node { return nil }

// CharRange 字符范围
type CharRange struct {
    Lo, Hi rune // 范围: [Lo, Hi] 包含两端
}

// RepNode 重复节点
// 不变量: Min <= Max，Max == -1 表示无上限
type RepNode struct {
    Child Node
    Min   int
    Max   int // -1 表示无穷
}

func (n *RepNode) Type() NodeType   { return NodeRep }
func (n *RepNode) Children() []Node { return []Node{n.Child} }
```

### 3.2 Token 定义 (lexer/token.go)

```go
package lexer

// TokenType 表示词法单元类型
type TokenType int

const (
    TokenChar    TokenType = iota // 普通字符
    TokenDot                      // '.'
    TokenStar                     // '*'
    TokenPlus                     // '+'
    TokenQuest                    // '?'
    TokenBar                      // '|'
    TokenLParen                   // '('
    TokenRParen                   // ')'
    TokenLBracket                 // '['
    TokenRBracket                 // ']'
    TokenLBrace                   // '{'
    TokenRBrace                   // '}'
    TokenCaret                    // '^'
    TokenDollar                   // '$'
    TokenDash                    // '-' (在字符类中)
    TokenEOF
    TokenError
)

// Token 词法单元
type Token struct {
    Type  TokenType
    Pos   int    // 原始位置
    Val   rune   // 单字符值
    Class []rune // 字符类内容
}

// IsQuantifier 检查 token 是否为量词
func (t *Token) IsQuantifier() bool {
    return t.Type == TokenStar || t.Type == TokenPlus || t.Type == TokenQuest
}
```

### 3.3 NFA 结构 (machine/nfa.go)

```go
package machine

// NFAState NFA 状态
// 不变量: Epsilon 转换永远优先于字符转换
type NFAState struct {
    ID       int
    Label    string // 调试标签
    Trans    []NFAEdge // 所有转换（按优先级排序）
    IsAccept bool
}

// NFAEdge NFA 边
type NFAEdge struct {
    Kind  EdgeKind // epsilon, literal, any
    Char  rune     // literal 边的字符
    Next  *NFAState
    Min, Max       // 用于 character class ranges
}

type EdgeKind int

const (
    EdgeEpsilon EdgeKind = iota
    EdgeLiteral
    EdgeAny
    EdgeClass
)

// Fragment NFA 片段（用于 Thompson 构造）
type Fragment struct {
    Start *NFAState
    End   *NFAState
}
```

### 3.4 DFA 结构 (machine/dfa.go)

```go
package machine

// DFAState DFA 状态
// 不变量: 每个状态有且只有一个接受状态
type DFAState struct {
    ID       int
    NFASet   map[*NFAState]struct{} // 对应的 NFA 状态集合
    Trans    []DFAEdge
    IsAccept bool
    AcceptInfos []AcceptInfo // 捕获组信息
}

// AcceptInfo 接受状态信息
type AcceptInfo struct {
    GroupIndex int
    Captures   []string
}

// DFA 确定性有限自动机
type DFA struct {
    Start    *DFAState
    Alphabet []rune // DFA 字母表（优化用）
}

// DFAEdge DFA 边
type DFAEdge struct {
    Lo, Hi rune // 字符范围 [Lo, Hi]
    Next   *DFAState
}
```

---

## 4. 接口签名

### 4.1 Lexer 接口 (lexer/lexer.go)

```go
package lexer

// Lexer 词法分析器接口
type Lexer interface {
    // Next 返回下一个 token
    Next() Token
    // Peek 返回下一个 token 但不消耗
    Peek() Token
    // Reset 重置 lexer 状态
    Reset(input string)
}

// New 创建标准 lexer
func New() Lexer

// Tokenize 便捷函数，一次性返回所有 token
func Tokenize(input string) ([]Token, error)
```

### 4.2 Parser 接口 (parser/parser.go)

```go
package parser

import "github.com/akzj/go-regex/lexer"

// Parser 语法分析器接口
type Parser interface {
    // Parse 解析正则表达式
    Parse(l lexer.Lexer) (ast.Node, error)
    // ParseString 便捷方法
    ParseString(pattern string) (ast.Node, error)
}

// New 创建标准 parser
func New() Parser
```

### 4.3 Compiler 接口 (compiler/compiler.go)

```go
package compiler

import (
    "github.com/akzj/go-regex/ast"
    "github.com/akzj/go-regex/machine"
)

// Compiler AST 到机器指令的编译器
type Compiler interface {
    // Compile 编译 AST 为 NFA
    Compile(node ast.Node) (*machine.NFA, error)
    // CompileDFA 编译为 DFA（优化执行）
    CompileDFA(node ast.Node) (*machine.DFA, error)
}

// New 创建编译器
func New() Compiler
```

### 4.4 Engine 接口 (engine/engine.go)

```go
package engine

import "github.com/akzj/go-regex/machine"

// Matcher 匹配器接口
type Matcher interface {
    // Match 检查输入是否完全匹配
    Match(input string) bool
    // Find 查找第一个匹配
    Find(input string) (start, end int)
    // FindAll 查找所有匹配
    FindAll(input string) [][]int
    // Replace 替换匹配
    Replace(src, repl string) string
}

// Engine 执行引擎
type Engine struct {
    dfa *machine.DFA
}

// New 创建执行引擎
func New(dfa *machine.DFA) *Engine

// Match 实现 Matcher 接口
func (e *Engine) Match(input string) bool

// Find 实现 Matcher 接口
func (e *Engine) Find(input string) (start, end int)

// FindAll 实现 Matcher 接口
func (e *Engine) FindAll(input string) [][]int
```

### 4.5 API 层 (api/api.go)

```go
package api

import "github.com/akzj/go-regex/engine"

// Regex 编译后的正则表达式
type Regex struct {
    engine *engine.Engine
    expr   string
}

// Compile 编译正则表达式
// 错误条件:
//   - 无效的正则语法
//   - 不支持的特性
//   - 空模式
func Compile(pattern string) (*Regex, error)

// MustCompile 编译正则，错误时 panic
func MustCompile(pattern string) *Regex

// Match 检查是否完全匹配
func (r *Regex) Match(s string) bool

// Find 查找第一个匹配，返回匹配字符串
func (r *Regex) Find(s string) string

// FindStringSubmatch 查找并返回捕获组
func (r *Regex) FindStringSubmatch(s string) []string

// ReplaceAllString 替换所有匹配
func (r *Regex) ReplaceAllString(s, repl string) string

// Split 按正则分割
func (r *Regex) Split(s string) []string
```

---

## 5. 支持的正则特性清单

| 特性 | 语法 | 状态 | 说明 |
|------|------|------|------|
| 任意字符 | `.` | ✅ | 匹配除换行外的任意字符 |
| 字面字符 | `a`, `\` | ✅ | 转义特殊字符 |
| 字符类 | `[abc]`, `[a-z]` | ✅ | 支持范围和转义 |
| 否定字符类 | `[^abc]` | ✅ | 不在类中的字符 |
| 量词 | `*`, `+`, `?` | ✅ | 贪婪匹配 |
| 精确重复 | `{n}` | ✅ | 恰好 n 次 |
| 范围重复 | `{n,m}` | ✅ | n 到 m 次 |
| 开头锚定 | `^` | ✅ | 行首 |
| 结尾锚定 | `$` | ✅ | 行尾 |
| 选择 | `\|` | ✅ | 或运算 |
| 分组 | `(...)` | ✅ | 普通分组 |
| 捕获组 | `(expr)` | ✅ | 编号捕获 |
| 非捕获组 | `(?:expr)` | 🔜 | 不创建捕获 |
| 预查 | `(?=...)` | 🔜 | 零宽断言 |
| 转义序列 | `\d`, `\w`, `\s` | 🔜 | 字符类简写 |
| 忽略大小写 | `(?i)` | 🔜 | 大小写不敏感 |
| 多行模式 | `(?m)` | 🔜 | `^`/`$` 匹配行边界 |

**状态图例**:
- ✅ 已计划
- 🔜 后续迭代

---

## 6. 权衡记录

### 6.1 引擎类型选择: NFA → DFA 混合

**决策**: 使用 Thompson 构造生成 NFA，再通过子集构造转换为 DFA。

**理由**:
| 方案 | 优势 | 劣势 |
|------|------|------|
| 纯 NFA | 简单，支持所有特性 | 最坏情况指数级回溯 |
| 纯 DFA | 线性时间，最坏情况保证 | 构造复杂，内存占用高 |
| **混合 (NFA→DFA)** | **平衡表达力和性能** | **实现复杂度中等** |

**替代方案考虑**:
- *为什么不直接使用 NFA 执行？* 存在最坏情况指数级问题（如 `(a|ab)*` 在 `aaaa...` 上）。
- *为什么不使用 Go 标准库的方式？* Go 使用基于 NFA 的回溯引擎，但会在特定情况下退化为指数级。我们选择 DFA 以保证最坏情况线性。

### 6.2 AST 表示: 结构体 vs 接口

**决策**: 使用 `interface{}` 接口表示 AST 节点。

**理由**:
```go
type Node interface {
    Type() NodeType
    String() string
    Children() []Node
}
```

| 方案 | 优势 | 劣势 |
|------|------|------|
| 纯接口 | 灵活，可扩展 | 每个节点有 vtable 开销 |
| 结构体枚举 | 无虚表，性能好 | 难以添加新节点类型 |
| **带类型的结构体** | **类型明确，有虚表但可控** | **轻微性能开销** |

**为什么不是空接口 `interface{}`?** 
需要 `Type()` 方法来区分节点类型，便于 switch 和 visitor 模式。

### 6.3 字符表示: rune vs byte

**决策**: 使用 `rune` 表示字符。

**理由**:
- Go 字符串是 UTF-8 编码，`byte` 无法正确处理多字节字符
- Unicode 范围需要 rune 支持（如 `\uFFFF`）
- 性能影响可接受（rune 是 int32）

### 6.4 贪婪 vs 非贪婪量词

**决策**: 默认贪婪，量词后加 `?` 变为非贪婪。

**理由**:
- POSIX/Perl 兼容的标准行为
- 实现简单：贪婪从右向左匹配，非贪婪从左向右
- 用户可预测

### 6.5 DFA 字母表优化

**决策**: 为 DFA 构建压缩字母表（字符范围合并）。

**理由**:
- 完整 Unicode 表有 65536+ 状态
- 实际使用的字符远少于这个数
- 使用范围区间压缩可大幅减少转换表大小

**实现**: 在 DFA 构造后执行字母表压缩步骤。

### 6.6 错误处理策略

**决策**: 立即返回错误，不使用 panic。

**理由**:
- 正则编译是可恢复的错误（用户输入）
- 便于上层 API 优雅处理
- 不使用 `recover()` 污染调用栈

---

## 7. 实现顺序

### Phase 1: 核心引擎
1. `lexer/` - 词法分析
2. `ast/` - AST 节点定义
3. `parser/` - 语法分析
4. `machine/nfa.go` - NFA 结构
5. `compiler/` - NFA 编译
6. `machine/dfa.go` - DFA 转换
7. `engine/` - 执行引擎
8. `api/` - API 层

### Phase 2: 扩展功能
1. 捕获组支持
2. 非捕获组
3. 零宽断言

### Phase 3: 性能优化
1. DFA 缓存
2. 字母表压缩
3. 增量编译

---

## 8. 文件结构

```
/home/ubuntu/workspace/go-regex/
├── go.mod
├── docs/
│   └── architecture.md          # 本文档
├── lexer/
│   ├── lexer.go                 # Lexer 接口
│   ├── token.go                 # Token 定义
│   └── lexer_test.go
├── ast/
│   ├── node.go                  # 节点定义
│   ├── node_test.go
│   └── visitor.go               # Visitor 模式（可选）
├── parser/
│   ├── parser.go                # Parser 接口
│   ├── parser_test.go
│   └── precedence.go            # 运算符优先级
├── machine/
│   ├── nfa.go                   # NFA 结构
│   ├── dfa.go                   # DFA 结构
│   └── subset.go                # 子集构造法
├── compiler/
│   ├── compiler.go              # 编译器接口
│   ├── ast2nfa.go               # AST → NFA
│   └── nfa2dfa.go               # NFA → DFA
├── engine/
│   ├── engine.go               # 执行引擎
│   ├── executor.go             # DFA 执行
│   └── backtrack.go            # 回溯（未来）
└── api/
    ├── api.go                  # 公共 API
    └── api_test.go
```

---

## 9. 已知风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| DFA 状态爆炸 | 复杂模式导致状态数指数增长 | 字母表压缩，状态合并 |
| 字符类性能 | Unicode 范围查询 O(n) | 使用二分查找或区间树 |
| 递归解析 | 深嵌套正则栈溢出 | 改用迭代解析器 |
| 内存占用 | DFA 转换表过大 | 延迟构造，按需生成 |

---

*文档版本: 1.0*  
*最后更新: 2024*
