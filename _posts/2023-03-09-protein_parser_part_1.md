---
layout: post
author: Clement
title: "Protein: Parser (Part 1)"
categories: [Protocol Buffers, Go]
---

In this article we are going to finally get to building the Parser. We are going to start parsing syntax, package and import statements, and we are going to see how to represent our serializable AST. Hope you are reasy for this, it's gonna be fun!

## Boilerplate

As always, we need to think a little bit before to actually write the features themselves. The first thing that we can do to get us started is to write the Parser interface.

```go parser/parser.go
package parser

// Parser is protein's parser
type Parser interface {
	// Parse returns ???
	Parse() ???
}
```

This doesn't seem like a fancy interface but we do have a problem. What is our parser returning when finished? Well, it should return an AST, right? But how do we represent this AST. It turns out, we have two good possibilities:

- We roll our own serializable AST where each object is a Protobuf Message.
- We use the descriptor.proto file which defines Messages for describing elements in a Protobuf file.

Both have pros and cons. If we go with the first one we have more control over our serialization. It means that we can optimize some elements' serialized data. However, it also means that we are not compatible with the official way and that's not good.
For the using the official serialization, I think you get the idea. We have the pros being the cons of the other implementation, and the cons being the pros of the other implementation.

In the end, for the sake of compatibility, I will be sacrificing some performance. However, these performance are only saving few bytes and having compatibility with programs serialized by protoc far overweights them.

### Depending on Protobuf

To use the descriptor, we are going to depend on Protobuf's library. To do that we are going to add in our dependency:

```shell
$ go get google.golang.org/protobuf
```

This will let us access `descriptorpb` package, which contains the `FileDescriptorProto` struct. If you look at the definition of that struct, you will see the following comment:

```go
// Describes a complete .proto file.
type FileDescriptorProto struct
```

That's exactly what we are trying to do.

### Back to interface

With that dependency on Protobuf, we can now finish our interface:

```go parser/parser.go
package parser

import pb "google.golang.org/protobuf/types/descriptorpb"

// Parser is protein's parser
type Parser interface {
	// Parse returns the representation of a file in Protobuf Descriptor
	Parse() pb.FileDescriptorProto
}
```

## Implementation

Let's now implement the interface. But by now you know the drill. We are going to create a minimal implementation so that our first test fails. So what we need is an `Impl` struct and we need to implement `Parser` by writing the `Parse` function.

For now, the `Parse` function will simply return an empty `FileDescriptorProto`.

```go parser/impl.go
package parser

import (
	pb "google.golang.org/protobuf/types/descriptorpb"
)

// Impl is the implementation for the Parser interface.
type Impl struct {
	l lexer.Lexer
}

// New creates a new instance of the Parser
func New(l lexer.Lexer) Parser {
	p := &Impl{l: l}
	return p
}

// Parse populates a FileDescriptorProto
func (p *Impl) Parse() pb.FileDescriptorProto {
	d := pb.FileDescriptorProto{}
	return d
}
```

### First test

As our first test we are going to create the test for a syntax statement. This test will take advantage of the fact that `Lexer` is an interface by creating a `FakeLexer`. This fake lexer will simply iterate over an array of tokens and return them one by one.

```go parser/parser_test.go
package parser

import (
	"github.com/Clement-Jean/protein/lexer"
)

type FakeLexer struct {
	i      int
	tokens []lexer.Token
}

func (l *FakeLexer) NextToken() lexer.Token {
	if l.i >= len(l.tokens) {
		return lexer.Token{Type: lexer.EOF, Position: lexer.Position{}}
	}

	token := l.tokens[l.i]
	l.i++
	return token
}
```

This basically means that each time we are running a test in the parser, we are not going to run the lexer code. We are going to simply focus on our current features. So, if we encounter a bug, it means that it is in the parser, not anywhere else.

With that, we can write out first test for `syntax = "proto3";`.

```go parser/parser_syntax_test.go
package parser

import (
	"testing"

	"github.com/Clement-Jean/protein/lexer"
)

func TestParseSyntaxProto3(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.TokenIdentifier, Literal: "syntax"},
		{Type: lexer.TokenEqual, Literal: "="},
		{Type: lexer.TokenStr, Literal: "\"proto3\""},
		{Type: lexer.TokenSemicolon, Literal: ";"},
	}
	l := &FakeLexer{tokens: tokens}
	p := New(l)
	d := p.Parse()
	expected := "proto3"

	if syntax := d.GetSyntax(); syntax != expected {
		t.Fatalf("syntax wrong. expected='%s', got='%s'", expected, syntax)
	}
}
```

Obviously:

```shell
$ go test ./...
--- FAIL: TestParseSyntaxProto3 (0.00s)
    parser_syntax_test.go:22: syntax wrong. expected='proto3', got=''
FAIL
```

### Parsing

We should now improve the `Parse` function to consume the `Lexer`'s tokens and do things with that. Here is the pseudo code:

```
while currToken.Type != EOF {
	if currToken.Type == Identifier {
		fn := parseFuncs[curToken.Literal] // find the function depending on keyword
		fn(&descriptor) // populate the descriptor
	}
}
```

You notice that we need a `currToken` representing the current token being parsed. We will also need the peek token for parsing syntax and other statements. This is because we are going to make sure each time that the peek token is correct, otherwise we will return an error. So `Impl` now has a `currToken` and `peekToken`:

```go parser/impl.go
type Impl struct {
	l lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
}
```

Now, we need to populate these tokens before being able to use them. The first time we need to initialize them is in the `New` function.

```go parser/impl.go
func New(l lexer.Lexer) Parser {
	p := &Impl{l: l}
	p.nextToken()
	p.nextToken()
	return p
}
```

But `nextToken` is not the `Lexer.NextToken`, this is a private function in `Parser`. This is a function that looks for the next non-space token.

```go parser/impl.go
func (p *Impl) nextToken() {
	for p.curToken = p.peekToken; p.curToken.Type == lexer.TokenSpace; p.curToken = p.l.NextToken() {
	}
	for p.peekToken = p.l.NextToken(); p.peekToken.Type == lexer.TokenSpace; p.peekToken = p.l.NextToken() {
	}
}
```

With that we can start updating our `Parse` function.

```go parser/impl.go
func (p *Impl) Parse() pb.FileDescriptorProto {
	d := pb.FileDescriptorProto{}

	for p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.TokenIdentifier {
			//Do something with token
		}
		p.nextToken()
	}

	return d
}
```

Finally, we are going to register all the parsing functions that we are gonna write in this and next articles. We are going to have a map mapping "syntax" to parseSyntax, ...

```go parser/impl.go
var parseFuncs = map[string]func(p *Impl, d *pb.FileDescriptorProto){
	"syntax":  func(p *Impl, d *pb.FileDescriptorProto) { d.Syntax = p.parseSyntax() },
}
```

With this we can finalize the `Parse` function by looking at the relevant function for the `currToken.Literal`.

```go parser/impl.go
func (p *Impl) Parse() pb.FileDescriptorProto {
	d := pb.FileDescriptorProto{}

	for p.curToken.Type != lexer.EOF {
		if p.curToken.Type == lexer.TokenIdentifier {
			fn, ok := parseFuncs[p.curToken.Literal]
			if !ok { // keyword not found
				break
			}
			fn(p, &d)
		}
		p.nextToken()
	}

	return d
}
```

### parseSyntax()

Before actually parsing a syntax statement, we need two helper functions: `accept` and `acceptPeek`. `acceptPeek` will just call `accept` with the `peekToken.Type`. `accept` take a `TokenType` and checks if it exists in the following variadic arguments.

```go parser/impl.go
func (p *Impl) accept(original lexer.TokenType, expected ...lexer.TokenType) bool {
	if !slices.Contains(expected, original) {
		// TODO: add error
		return false
	}

	p.nextToken()
	return true
}

// acceptPeek returns true and advance token
// if tt contains the peekToken.Type
// else it returns false
func (p *Impl) acceptPeek(tt ...lexer.TokenType) bool {
	return p.accept(p.peekToken.Type, tt...)
}
```

And now we ready for our `parseSyntax` function. We are first going to check that we have an `=` after syntax. Then we check that we have a String, if its the case we are going to take the value inside the quotes. And finally, we are going to check that there is a semicolon at the end of the statement.

```go parser/syntax.go
package parser

import (
	"github.com/Clement-Jean/protein/lexer"
)

func (p *Impl) parseSyntax() *string {
	if !p.acceptPeek(lexer.TokenEqual) {
		return nil
	}
	if !p.acceptPeek(lexer.TokenStr) {
		return nil
	}

	s := destringify(p.curToken.Literal)

	if !p.acceptPeek(lexer.TokenSemicolon) {
		return nil
	}

	return &s
}
```

The `destringify` function looks like the following:

```go parser/utils.go
package parser

import "strings"

func destringify(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return r == '\'' || r == '"'
	})
}
```

As mentionned, it takes the values between the quotes.

With our `parseSyntax` finished, we can rerun the test and:

```shell
$ go test ./...
ok      github.com/Clement-Jean/protein/parser  1.361s
```

### parseImport()

`parseImport` is really similar to `parseSyntax`. However, with imports, we get introduced to optional keywords. An import can be written in 3 ways:

- `import "my.proto";`
- `import public "my.proto";`
- `import weak "my.proto";`

Let's write the tests:

```go parser/parser_import_test.go
package parser

import (
	"testing"

	"github.com/Clement-Jean/protein/lexer"
)

func TestParseImport(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.TokenIdentifier, Literal: "import"},
		{Type: lexer.TokenStr, Literal: "\"google/protobuf/empty.proto\""},
		{Type: lexer.TokenSemicolon, Literal: ";"},
	}
	l := &FakeLexer{tokens: tokens}
	p := New(l)
	d := p.Parse()
	expected := []string{"google/protobuf/empty.proto"}
	public := []int32{}
	weak := []int32{}

	if imp := d.GetDependency(); slices.Compare(imp, expected) != 0 {
		t.Fatalf("import wrong. expected='%v', got='%v'", expected, imp)
	}

	if p := d.GetPublicDependency(); slices.Compare(p, public) != 0 {
		t.Fatalf("public import wrong. expected='%v', got='%v'", public, p)
	}

	if w := d.GetWeakDependency(); slices.Compare(w, weak) != 0 {
		t.Fatalf("weak import wrong. expected='%v', got='%v'", weak, w)
	}
}

func TestParsePublicImport(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.TokenIdentifier, Literal: "import"},
		{Type: lexer.TokenIdentifier, Literal: "public"},
		{Type: lexer.TokenStr, Literal: "\"google/protobuf/empty.proto\""},
		{Type: lexer.TokenSemicolon, Literal: ";"},
	}
	l := &FakeLexer{tokens: tokens}
	p := New(l)
	d := p.Parse()
	expected := []string{"google/protobuf/empty.proto"}
	public := []int32{0}
	weak := []int32{}

	if imp := d.GetDependency(); slices.Compare(imp, expected) != 0 {
		t.Fatalf("import wrong. expected='%v', got='%v'", expected, imp)
	}

	if p := d.GetPublicDependency(); slices.Compare(p, public) != 0 {
		t.Fatalf("public import wrong. expected='%v', got='%v'", public, p)
	}

	if w := d.GetWeakDependency(); slices.Compare(w, weak) != 0 {
		t.Fatalf("weak import wrong. expected='%v', got='%v'", weak, w)
	}
}

func TestParseWeakImport(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.TokenIdentifier, Literal: "import"},
		{Type: lexer.TokenIdentifier, Literal: "weak"},
		{Type: lexer.TokenStr, Literal: "\"google/protobuf/empty.proto\""},
		{Type: lexer.TokenSemicolon, Literal: ";"},
	}
	l := &FakeLexer{tokens: tokens}
	p := New(l)
	d := p.Parse()
	expected := []string{"google/protobuf/empty.proto"}
	public := []int32{}
	weak := []int32{0}

	if imp := d.GetDependency(); slices.Compare(imp, expected) != 0 {
		t.Fatalf("import wrong. expected='%v', got='%v'", expected, imp)
	}

	if p := d.GetPublicDependency(); slices.Compare(p, public) != 0 {
		t.Fatalf("public import wrong. expected='%v', got='%v'", public, p)
	}

	if w := d.GetWeakDependency(); slices.Compare(w, weak) != 0 {
		t.Fatalf("weak import wrong. expected='%v', got='%v'", weak, w)
	}
}
```

Obviously:

```shell
$ go test ./...
--- FAIL: TestParseImport (0.00s)
    parser_import_test.go:25: import wrong. expected='[google/protobuf/empty.proto]', got='[]'
--- FAIL: TestParsePublicImport (0.00s)
    parser_import_test.go:52: import wrong. expected='[google/protobuf/empty.proto]', got='[]'
--- FAIL: TestParseWeakImport (0.00s)
    parser_import_test.go:79: import wrong. expected='[google/protobuf/empty.proto]', got='[]'
FAIL
```

Even though the 2nd and 3rd one are rarely used, we still need to support them. To do so, we are going to need to create an enum called `DependencyType` which will have the variants: None, Public, and Weak. After that, we are going to check if we have an identifier and depending on the `Literal`, we are going to return the type.

```go parser/import.go
package parser

import (
	"fmt"

	"github.com/Clement-Jean/protein/lexer"
)

type DependencyType int

const (
	None DependencyType = iota
	Public
	Weak
)

func (p *Impl) parseImport() (string, DependencyType) {
	if !p.acceptPeek(lexer.TokenStr, lexer.TokenIdentifier) {
		return "", None
	}

	depType := None

	if p.curToken.Type == lexer.TokenIdentifier {
		switch p.curToken.Literal {
		case "public":
			depType = Public
		case "weak":
			depType = Weak
		default:
			return "", None
		}

		if !p.acceptPeek(lexer.TokenStr) {
			// TODO: add error
			return "", None
		}
	}

	s := destringify(p.curToken.Literal)

	if !p.acceptPeek(lexer.TokenSemicolon) {
		return "", None
	}

	return s, depType
}
```

And the last thing we need to do is register that to the `parseFuncs`.

```go parser/impl.go
var parseFuncs = map[string]func(p *Impl, d *pb.FileDescriptorProto){
	"syntax":  func(p *Impl, d *pb.FileDescriptorProto) { d.Syntax = p.parseSyntax() },
	"import": func(p *Impl, d *pb.FileDescriptorProto) {
		dep, t := p.parseImport()
		if len(dep) != 0 {
			i := int32(len(d.Dependency))
			d.Dependency = append(d.Dependency, dep)
			switch t {
			case Public:
				d.PublicDependency = append(d.PublicDependency, i)
			case Weak:
				d.WeakDependency = append(d.WeakDependency, i)
			}
		}
	},
}
```

We basically append the dependency and if we have a public or weak dependency we add its index into `PublicDependency` and `WeakDependency` respectively.

We rerun our test:

```shell
$ go test ./...
ok      github.com/Clement-Jean/protein/parser  0.450s
```

### parsePackage()

Once again this function is pretty similar. The main difference is that we are goin to look for identifiers and fully qualified names (identifiers sperated by dots).

Let's write some tests.

```go parser/parser_package_test.go
package parser

import (
	"testing"

	"github.com/Clement-Jean/protein/lexer"
)

func TestParsePackage(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.TokenIdentifier, Literal: "package", Position: lexer.Position{}},
		{Type: lexer.TokenIdentifier, Literal: "google", Position: lexer.Position{}},
		{Type: lexer.TokenSemicolon, Literal: ";", Position: lexer.Position{}},
	}
	l := &FakeLexer{tokens: tokens}
	p := New(l)
	d := p.Parse()
	expected := "google"

	if pkg := d.GetPackage(); pkg != expected {
		t.Fatalf("package wrong. expected='%s', got='%s'", expected, pkg)
	}
}

func TestParsePackageFullIdentifier(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.TokenIdentifier, Literal: "package", Position: lexer.Position{}},
		{Type: lexer.TokenIdentifier, Literal: "google", Position: lexer.Position{}},
		{Type: lexer.TokenDot, Literal: ".", Position: lexer.Position{}},
		{Type: lexer.TokenIdentifier, Literal: "protobuf", Position: lexer.Position{}},
		{Type: lexer.TokenSemicolon, Literal: ";", Position: lexer.Position{}},
	}
	l := &FakeLexer{tokens: tokens}
	p := New(l)
	d := p.Parse()
	expected := "google.protobuf"

	if pkg := d.GetPackage(); pkg != expected {
		t.Fatalf("package wrong. expected='%s', got='%s'", expected, pkg)
	}
}
```

Let's fail our tests:

```shell
$ go test ./...
--- FAIL: TestParsePackage (0.00s)
    parser_package_test.go:21: package wrong. expected='google', got=''
--- FAIL: TestParsePackageFullIdentifier (0.00s)
    parser_package_test.go:39: package wrong. expected='google.protobuf', got=''
FAIL
```

And now we can implement the `parsePackage` function. We are going to check that we have at least one identifier, and then if we have a dot we are going to make sure that we have another identifier after. Finally, we will be looking for the semicolon.

```go parser/package.go
package parser

import (
	"strings"

	"github.com/Clement-Jean/protein/lexer"
)

func (p *Impl) parsePackage() *string {
	if !p.acceptPeek(lexer.TokenIdentifier) {
		return nil
	}

	var parts []string

	for p.curToken.Type == lexer.TokenIdentifier {
		parts = append(parts, p.curToken.Literal)
		if p.peekToken.Type != lexer.TokenDot {
			break
		}

		p.nextToken()

		if !p.acceptPeek(lexer.TokenIdentifier) {
			return nil
		}
	}
	s := strings.Join(parts, ".")

	if !p.acceptPeek(lexer.TokenSemicolon) {
		return nil
	}

	return &s
}
```

The last thing to do is to register this function in our `parseFuncs`.

```go parser/impl.go
var parseFuncs = map[string]func(p *Impl, d *pb.FileDescriptorProto){
	"syntax":  func(p *Impl, d *pb.FileDescriptorProto) { d.Syntax = p.parseSyntax() },
	"package": func(p *Impl, d *pb.FileDescriptorProto) { d.Package = p.parsePackage() },
	"import": func(p *Impl, d *pb.FileDescriptorProto) {
		dep, t := p.parseImport()
		if len(dep) != 0 {
			i := int32(len(d.Dependency))
			d.Dependency = append(d.Dependency, dep)
			switch t {
			case Public:
				d.PublicDependency = append(d.PublicDependency, i)
			case Weak:
				d.WeakDependency = append(d.WeakDependency, i)
			}
		}
	},
}
```

and we rerun our tests.

```shell
$ go test ./...
ok      github.com/Clement-Jean/protein/parser  0.847s
```

# Conclusion

## Conclusion



**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**

<div class="container">
  <div class="row">
    <div class="col text-center">
      <a href="/protein_lexer_part_3" class="btn btn-danger text-center">Previous Article</a>
    </div>
    <!-- <div class="col text-center">
      <a href="/protein_lexer_part_1" class="btn btn-danger text-center">Next Article</a>
    </div> -->
  </div>
</div>