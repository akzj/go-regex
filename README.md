# go-regex

A lightweight regular expression engine written in Go, implementing a Thompson NFA-based DFA for efficient pattern matching.

## Installation

```bash
go get github.com/akzj/go-regex
```

## Quick Start

```go
package main

import (
    "fmt"
    regex "github.com/akzj/go-regex/api"
)

func main() {
    // Compile a pattern
    r, err := regex.Compile(`hello`)
    if err != nil {
        panic(err)
    }

    // Match: check if pattern matches entire string
    fmt.Println(r.Match("hello world"))  // false (partial match only)

    // Find: get leftmost match
    fmt.Println(r.Find("say hello world"))  // "hello"

    // Replace: replace all matches
    r2, _ := regex.Compile(`world`)
    fmt.Println(r2.ReplaceAllString("hello world", "go"))  // "hello go"

    // Split: split string by pattern
    r3, _ := regex.Compile(`,`)
    fmt.Println(r3.Split("a,b,c"))  // ["a" "b" "c"]
}
```

## API Reference

### Compile and MustCompile

```go
// Compile returns a Regex or error
r, err := regex.Compile(`\d+`)

// MustCompile panics on error
r := regex.MustCompile(`\d+`)
```

### Match

Reports whether the regex matches the entire input string.

```go
r := regex.MustCompile(`^hello$`)
fmt.Println(r.Match("hello"))  // true
fmt.Println(r.Match("hello world"))  // false
```

### Find

Returns the text of the leftmost match.

```go
r := regex.MustCompile(`\d+`)
fmt.Println(r.Find("abc 123 def"))  // "123"
```

### FindStringSubmatch

Returns the full match and captured groups.

```go
r := regex.MustCompile(`(\w+)@(\w+)`)
matches := r.FindStringSubmatch("email: user@domain")
// matches[0] = "user@domain"
// matches[1] = "user"
// matches[2] = "domain"
```

### ReplaceAllString

Replaces all non-overlapping matches with a replacement string.

```go
r := regex.MustCompile(`cat|dog`)
fmt.Println(r.ReplaceAllString("cats and dogs", "bird"))
// "birds and birds"
```

### Split

Splits the string by matches of the regex.

```go
r := regex.MustCompile(`\s+`)
fmt.Println(r.Split("hello   world"))
// ["hello" "world"]
```

## Supported Syntax

| Feature | Syntax | Example |
|---------|--------|---------|
| Literal characters | `abc` | `hello` matches "hello" |
| Wildcard | `.` | `a.b` matches "acb" |
| Character class | `[abc]` | `[aeiou]` matches vowels |
| Negated class | `[^abc]` | `[^0-9]` matches non-digits |
| Range | `[a-z]` | `[a-zA-Z]` matches letters |
| Alternation | `\|` | `cat\|dog` matches either |
| Grouping | `(expr)` | `(ab)+` matches repeated "ab" |
| Star quantifier | `*` | `a*` matches 0+ 'a' |
| Plus quantifier | `+` | `a+` matches 1+ 'a' |
| Question quantifier | `?` | `a?` matches 0 or 1 'a' |
| Repetition | `{n,m}` | `a{2,4}` matches 2-4 'a's |
| Start anchor | `^` | `^hello` matches at start |
| End anchor | `$` | `hello$` matches at end |
| Escape sequences | `\d`, `\w`, `\s` | Digit, word, space |

### Escape Sequences

- `\d` — Digit character `[0-9]`
- `\w` — Word character `[a-zA-Z0-9_]`
- `\s` — Whitespace character (space, tab, newline)
- `\.`, `\*`, `\?`, etc. — Match literal special characters

## Known Limitations

This is a lightweight regex engine. Some features from full POSIX or PCRE are not supported:

| Feature | Status | Workaround |
|---------|--------|------------|
| Non-capturing groups `(?:...)` | Not supported | Use capturing groups `(...)` |
| Named groups `(?P<name>...)` | Not supported | Use positional groups |
| Lookahead/Lookbehind | Not supported | Preprocess input |
| Backreferences `\1` | Not supported | N/A |
| Unicode categories `\p{L}` | Limited | Use explicit character classes |
| Multi-byte UTF-8 | Partial | Results may vary |
| Greedy vs lazy quantifiers | Greedy only | All quantifiers are greedy |

## Architecture

See [docs/architecture.md](docs/architecture.md) for detailed architecture documentation.

```
lexer → parser → ast → compiler → machine (nfa/dfa) → engine → api
```

## Performance

The engine uses Thompson NFA construction with DFA simulation for efficient matching:

- **Best case**: O(n) where n is input length
- **Worst case**: O(nm) where m is pattern size
- **Memory**: Compact DFA representation

## License

MIT License
