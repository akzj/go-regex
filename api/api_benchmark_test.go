package api

import (
	"testing"
)

// BenchmarkCompile measures the cost of compiling various patterns
func BenchmarkCompileSimpleLiteral(b *testing.B) {
	pattern := "hello world"
	for i := 0; i < b.N; i++ {
		_, _ = Compile(pattern)
	}
}

func BenchmarkCompileCharacterClass(b *testing.B) {
	pattern := "[a-zA-Z0-9_]+"
	for i := 0; i < b.N; i++ {
		_, _ = Compile(pattern)
	}
}

func BenchmarkCompileQuantifiers(b *testing.B) {
	pattern := `a+b*c?d{1,3}`
	for i := 0; i < b.N; i++ {
		_, _ = Compile(pattern)
	}
}

func BenchmarkCompileAlternation(b *testing.B) {
	pattern := `foo|bar|baz|qux|quux`
	for i := 0; i < b.N; i++ {
		_, _ = Compile(pattern)
	}
}

func BenchmarkCompileComplexPattern(b *testing.B) {
	pattern := `^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$`
	for i := 0; i < b.N; i++ {
		_, _ = Compile(pattern)
	}
}

// BenchmarkMatch measures matching performance
func BenchmarkMatchLiteral(b *testing.B) {
	r, _ := Compile("hello")
	input := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(input)
	}
}

func BenchmarkMatchAnchored(b *testing.B) {
	r, _ := Compile("^hello")
	input := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(input)
	}
}

func BenchmarkMatchNoMatch(b *testing.B) {
	r, _ := Compile("hello")
	input := "world goodbye"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(input)
	}
}

func BenchmarkMatchCharacterClass(b *testing.B) {
	r, _ := Compile("[a-zA-Z]+")
	input := "hello world 123"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(input)
	}
}

func BenchmarkMatchDigits(b *testing.B) {
	r, _ := Compile(`\d+\.\d+`)
	input := "3.14159"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(input)
	}
}

// BenchmarkFind measures find operation performance
func BenchmarkFindSimple(b *testing.B) {
	r, _ := Compile("world")
	input := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find(input)
	}
}

func BenchmarkFindAtStart(b *testing.B) {
	r, _ := Compile(`\w+`)
	input := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find(input)
	}
}

func BenchmarkFindNoMatch(b *testing.B) {
	r, _ := Compile("xyz")
	input := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find(input)
	}
}

func BenchmarkFindDigits(b *testing.B) {
	r, _ := Compile(`\d+`)
	input := "abc 123 def 456"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find(input)
	}
}

// BenchmarkReplaceAllString measures replace operation performance
func BenchmarkReplaceAllString(b *testing.B) {
	r, _ := Compile("cat|dog")
	input := "cats and dogs"
	replacement := "bird"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ReplaceAllString(input, replacement)
	}
}

func BenchmarkReplaceDigits(b *testing.B) {
	r, _ := Compile(`\d+`)
	input := "item1 price2 total3"
	replacement := "X"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ReplaceAllString(input, replacement)
	}
}

func BenchmarkReplaceNoMatch(b *testing.B) {
	r, _ := Compile("xyz")
	input := "hello world"
	replacement := "replaced"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ReplaceAllString(input, replacement)
	}
}

// BenchmarkSplit measures split operation performance
func BenchmarkSplitComma(b *testing.B) {
	r, _ := Compile(",")
	input := "a,b,c,d,e,f,g,h"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Split(input)
	}
}

func BenchmarkSplitWhitespace(b *testing.B) {
	r, _ := Compile(`\s+`)
	input := "hello   world    test"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Split(input)
	}
}

func BenchmarkSplitNoMatch(b *testing.B) {
	r, _ := Compile("xyz")
	input := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Split(input)
	}
}

// BenchmarkLongInput measures performance with longer inputs
func BenchmarkMatchLongInput(b *testing.B) {
	r, _ := Compile(`hello`)
	longStr := make([]byte, 10000)
	copy(longStr, "xxxxxxxxxxxxhello")
	input := string(longStr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(input)
	}
}

func BenchmarkFindLongInput(b *testing.B) {
	r, _ := Compile(`hello`)
	longStr := make([]byte, 10000)
	copy(longStr, "xxxxxxxxxxxxhello")
	input := string(longStr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find(input)
	}
}

func BenchmarkMatchAllDigits(b *testing.B) {
	r, _ := Compile(`\d+`)
	input := "12345678901234567890"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(input)
	}
}
