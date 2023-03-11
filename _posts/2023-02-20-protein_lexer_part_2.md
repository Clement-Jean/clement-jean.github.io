---
layout: post
author: Clement
title: "Protein: Lexer (Part 2)"
categories: [Protocol Buffers, Go]
---

In this article we are going to delve into the second part of the lexing which is tokenizing more advanced part of the input. More precisely, we are going to lex spaces (whitespaces, new lines, ...), comments, Identifiers, Numbers (Int and Float), and Strings. At the end of this article, we will have a fully functioning lexer that can tokenize the `descriptor.proto` which is the longest proto file in the protobuf repo. Let's get started.

> Note: While this article is designed in a way that shows the evolution of the Lexer, you might still want to look at [the commits](https://github.com/Clement-Jean/protein/commits/lexer) in order to see where any piece of code went.

## Spaces

Up until now, we assumed that all the characters that we would encounter were symbols. We are now going to add whitespaces, new lines, ... on top of that. Now, one thing to note here is that in traditional lexer, spaces will be discarded and are not counted as tokens. In our case, since the [Protobuf Style Guide](https://protobuf.dev/programming-guides/style/) mentions that we should use an indent of 2 spaces, we are going to need that information. Before doing tokenizing spaces, we are going to need to more helper functions:

- `backup`: Goes back by one utf8 character.
- `peek`: Check the next utf8 character without advancing the reading position.

`peek` is very similar to next because we are checking that we are still within the limits of our input and we read the character at `lexer.pos`. However, because we are just looking ahead and not advancing in the input, we are not going to update the `lexer.pos`.

```go lexer/impl.go
//...
func (l *Impl) peek() rune {
  if int(l.pos) >= len(l.src) {
    return rune(EOF)
  }

  r, _ := utf8.DecodeRuneInString(l.src[l.pos:])
  return r
}
```

and `backup` is pretty simple. We are going to read the last character before `lexer.pos` and update the reading position by the size of that utf8 character. Furthermore, since in the `next` function we added `l.line++` when we have a new line, we are going to need backing that up too. So we are going to decrease `lexer.line` when we encounter a newline.

```go lexer/impl.go
//...
func (l *Impl) backup() {
  if !l.atEOF && l.pos > 0 {
    r, w := utf8.DecodeLastRuneInString(l.src[:l.pos])
    l.pos -= w

    if r == '\n' {
      l.line--
    }
  }
}
```

As always, we want to add a test for our function that is skipping spaces. Now, it is important to understand that we are not only interested of whitespaces. We also want to 'skip' other non-printable characters such as '\t', '\r', '\n', ... For that we are going to use the `unicode.IsSpace` from the Go standard library. So our test, should contain these different characters.

```go lexer/lexer_test.go
//...
func TestNextTokenOnSpace(t *testing.T) {
  runChecks(t, New("\t\n\v\f\r "), []Check{
    {TokenSpace, "\t\n\v\f\r ", Position{}},
    {EOF, "", Position{}},
  })
}
```

Obviously, if we run this we will get the following:

```shell
$ go test ./...
--- FAIL: TestNextTokenOnSpace (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Space", got="Illegal"
FAIL
```

Now that we have our test, `backup` and `peek`, we can write our `lexSpaces` function. This is a function that loops until we find a non-space character as peek and return a Token for that space.

```go lexer/impl.go
//...
func lexSpaces(l *Impl) stateFn {
  var r rune

  for {
    r = l.peek()
    if !unicode.IsSpace(r) {
      break
    }

    l.next()
  }
  return l.emit(TokenSpace)
}
```

This `lexSpaces` function will now need to be placed in our `lexProto` switch in order to be taken into consideration.

```go lexer/impl.go
func lexProto(l *Impl) stateFn {
  switch r := l.next(); {
  //...
  case unicode.IsSpace(r):
    l.backup() // go back by one character
    return lexSpaces
  }
  //...
}
```

and that's basically it! We run our test and we get:

```shell
$ go test ./...
ok      github.com/Clement-Jean/protein/lexer  0.235s
```

## Comments

It turns out that we can lex comments with a similar technique as the one used for spaces. The only difference here is that we have two type of comment:

- Line comment: Starts with `//` and finishes at the end of the line.
- Multiline comment: Starts with `/*` and finishes with `*/`.

### Line comment

We are going to start with the line comment. This is very similar to `lexSpaces`. The major difference is that we are going to check for '\n' or EOF for finishing the comment. Finally, we are going to return a Token.

But before all that, let's write two tests. One that checks that we are skipping until '\n' and the other that checks that we are skipping until EOF.

```go lexer/lexer_test.go
//...
func TestNextTokenOnLineCommentWithEOF(t *testing.T) {
  runChecks(t, New("//this is a comment"), []Check{
    {TokenComment, "//this is a comment", Position{}},
    {EOF, "", Position{}},
  })
}

func TestNextTokenOnLineCommentWithNewLine(t *testing.T) {
  runChecks(t, New("//this is a comment\n_"), []Check{
    {TokenComment, "//this is a comment", Position{0, 1, 1}},
    {TokenSpace, "\n", Position{19, 1, 19}},
    {TokenUnderscore, "_", Position{20, 1, 20}},
    {EOF, "", Position{21, 1, 21}},
  })
}
```

We fail to pass, once again:

```shell
$ go test ./...
--- FAIL: TestNextTokenOnLineCommentWithEOF (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Comment", got="Illegal"
--- FAIL: TestNextTokenOnLineCommentWithNewLine (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Comment", got="Illegal"
FAIL
```

Let's now write our function.

```go lexer/impl.go
//...
func lexLineComment(l *Impl) stateFn {
  var r rune

  for {
    r = l.peek()
    if r == '\n' || r == rune(EOF) {
      break
    }

    l.next()
  }
  return l.emit(TokenComment)
}
```

and run our tests:

```shell
$ go test ./...
ok      github.com/Clement-Jean/protein/lexer  0.351s
```

We pass the Line Comment tests, let's go to the multiline comment.

### Multiline Comment

These comments are again pretty similar, however, like any other object that doesn't end with EOF we can have errors. The main error here is an unterminated comment. As an example, writing something like:

```go
/*this is a comment
```

should result in an error from the lexer.

To handle such errors, we are going to create a function that will basically stop the lexing process by emitting an Error Token and return nil as state.

```go lexer/impl.go
//...
func (l *Impl) errorf(format string, args ...any) stateFn {
  l.token = Token{TokenError, fmt.Sprintf(format, args...), Position{
    Offset: l.start,
    Line:   l.startLine,
    Column: l.start - l.startLineOffset,
  }}
  l.start = 0
  l.pos = 0
  l.src = l.src[:0]
  return nil
}
```

This is similar to what we did in emit, the only difference is that here the literal of a token is the error message and we reset the state of our lexer.

Now that we know the requirements for our lexer concerning comments, we can write some tests. Before that, though, as a matter of convenience, we are going to define constants for our error messages. This is done to avoid typos when writing tests:

```go lexer/errors.go
package lexer

const (
  errorUnterminatedMultilineComment = "unterminated multiline comment"
)
```

and once this is done we can now have our tests:

```go lexer/lexer_test.go
func TestNextTokenOnMultilineComment(t *testing.T) {
  runChecks(t, New("/*this is a comment*/_"), []Check{
    {TokenComment, "/*this is a comment*/", Position{0, 1, 0}},
    {TokenUnderscore, "_", Position{21, 1, 21}},
    {EOF, "", Position{22, 1, 22}},
  })
}

func TestNextTokenOnUnterminatedMultilineComment(t *testing.T) {
  runChecks(t, New("/*this is a comment"), []Check{
    {TokenError, errorUnterminatedMultilineComment, Position{0, 1, 0}},
  })
}
```

we run:

```shell
$ go test ./...
--- FAIL: TestNextTokenOnMultilineComment (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Comment", got="Illegal"
--- FAIL: TestNextTokenOnUnterminatedMultilineComment (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Error", got="Illegal"
FAIL
```

and we fail (we are used to it).

`lexMultilineComment` is a little bit longer than the other functions we wrote for lexing. This is mostly due to the fact that we are checking for unterminated comment but also because we need to check that the current character is '/' and that the previous character was '*'. So we keep a reference to the previous character and check that for stopping the reading loop.

```go lexer/impl.go
func lexMultilineComment(l *Impl) stateFn {
  var p rune
  var r rune

  for {
    p = r
    if r == rune(EOF) {
      return l.errorf(errorUnterminatedMultilineComment)
    }

    r = l.peek()
    if p == '*' && r == '/' {
      l.next()
      break
    }

    l.next()
  }
  return l.emit(TokenComment)
}
```

Once again this is very similar to `lexSpaces` and `lexLineComment`, isn't it?

Let's now place that in our `lexProto` function:

```go lexer/impl.go
func lexProto(l *Impl) stateFn {
  switch r := l.next(); {
  //...
  case r == '/' && l.peek() == '*':
    l.backup()
    return lexMultilineComment
  }
  //...
}
```

and our tests?

```shell
$ go test ./...
ok      github.com/Clement-Jean/protein/lexer  0.741s
```

## Identifiers

For identifiers, we need a way to keep going while we have a letter (capitalized or not), a number or an underscore. We are going to create a function called `acceptWhile` that does just that. We want to pass the set of possible characters to it and while the set contains the current value it will advance. Once we are done, we are going to use `backup` to make sure that the `lexer.pos` is just after the last character of the identifier.

Before we do that, though, it's testing time. We simply want to test that when we pass some text starting with a letter, `lexProto` will create an Identifier token.

```go lexer/lexer_test.go
func TestNextTokenOnIdentifier(t *testing.T) {
  runChecks(t, New("hello_world2023 HelloWorld2023"), []Check{
    {TokenIdentifier, "hello_world2023", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenIdentifier, "HelloWorld2023", Position{}},
    {EOF, "", Position{}},
  })
}
```

We obviously fail the test:

```shell
$ go test ./...
--- FAIL: TestNextTokenOnIdentifier (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Identifier", got="Illegal"
FAIL
```

and now we can start our `acceptWhile` function:

```go lexer/impl.go
//...

func (l *Impl) acceptWhile(valid string) {
  for strings.ContainsRune(valid, l.next()) {
  }
  l.backup()
}
```

That's it. Nothing more! A few lines of code and we can now simply write our `lexIdentifier`.

```go lexer/impl.go
//...

func lexIdentifier(l *Impl) stateFn {
  l.acceptWhile("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")
  return l.emit(TokenIdentifier)
}
```

and add that to the `lexProto`:

```go lexer/impl.go
func lexProto(l *Impl) stateFn {
  switch r := l.next(); {
  //...
  case unicode.IsLetter(r):
    l.backup()
    return lexIdentifier
  }
  //...
}
```

We rerun our test:

```shell
$ go test ./...
ok      github.com/Clement-Jean/protein/lexer  0.767s
```

## Strings

Before going the numbers, let's continue with something easy. Lexing strings is similar to what we did with multiline comments. We have a beginning and an end delimited by a certain character. In Protobuf, for strings, we can use single and double quotes. However, we cannot match a double quote with a single one. This means that having something like:

```go
"test'
```

will result in a unterminated string.

Now that we know the requirements, let's write out tests.

```go lexer/lexer_test.go
func TestNextTokenOnString(t *testing.T) {
  runChecks(t, New("'test' \"test\""), []Check{
    {TokenStr, "'test'", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenStr, "\"test\"", Position{}},
    {EOF, "", Position{}},
  })
}

func TestNextTokenOnUnterminatedString(t *testing.T) {
  runChecks(t, New("'test"), []Check{
    {TokenError, errorUnterminatedQuotedString, Position{}},
  })
}

func TestNextTokenOnMismatchedQuotesString(t *testing.T) {
  runChecks(t, New("\"test'"), []Check{
    {TokenError, errorUnterminatedQuotedString, Position{}},
  })
}
```

you notice the constant named `errorUnterminatedQuotedString`. It is similar to the error message we added for the multiline comment.

```go lexer/errors.go
package lexer

const (
  errorUnterminatedMultilineComment = "unterminated multiline comment"
  errorUnterminatedQuotedString     = "unterminated quoted string"
)
```

We obviously fail the test:

```shell
$ go test ./...
--- FAIL: TestNextTokenOnString (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="String", got="Illegal"
--- FAIL: TestNextTokenOnUnterminatedString (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Error", got="Illegal"
--- FAIL: TestNextTokenOnMismatchedQuotesString (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Error", got="Illegal"
FAIL
```

Ok, now we can write our `lexString`. We are going to write a function that first take notice of the character it currently is on. This character will be a single or double quote and we are going to need it to determine the end of our string. After that we are going to loop over the input and we are going to check for EOF (unterminated string) and the character at the beginning (end of string). When we encounter the end of the string, we can just break out of the loop and return a String Token.

```go lexer/impl.go
//...

func lexString(l *Impl) stateFn {
  open := l.src[l.pos]
  l.next()
Loop:
  for {
    switch l.next() {
    case rune(EOF):
      return l.errorf(errorUnterminatedQuotedString)
    case rune(open):
      break Loop
    }
  }
  return l.emit(TokenStr)
}
```

and after that, you know the trick, we add that to `lexProto`:

```go lexer/impl.go
func lexProto(l *Impl) stateFn {
  switch r := l.next(); {
  //...
  case r == '"' || r == '\'':
    l.backup()
    return lexString
  }
  //...
}
```

and the tests?

```shell
go test ./...
ok      github.com/Clement-Jean/protein/lexer  0.374s
```

## Numbers

The real challenge comes with numbers. If we take a look at the [Protobuf language specification](https://protobuf.dev/reference/protobuf/proto3-spec/), we need to accept Decimal, Octal and Hexadecimal for integers and exponents for floats. On top of that we need to be able to put a sign before the number to be able to have -5 for example.

In our tests we are going to try listing all the possible kinds of numbers (if you spot something missing, let me know):

```go lexer/lexer_test.go
func TestNextTokenOnIntDecimal(t *testing.T) {
  runChecks(t, New("5 0 -5 +5"), []Check{
    {TokenInt, "5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenInt, "0", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenInt, "-5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenInt, "+5", Position{}},
    {EOF, "", Position{}},
  })
}

func TestNextTokenOnIntHex(t *testing.T) {
  runChecks(t, New("0xff 0XFF"), []Check{
    {TokenInt, "0xff", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenInt, "0XFF", Position{}},
    {EOF, "", Position{}},
  })
}

func TestNextTokenOnIntOctal(t *testing.T) {
  runChecks(t, New("056"), []Check{
    {TokenInt, "056", Position{}},
    {EOF, "", Position{}},
  })
}

func TestNextTokenOnFloat(t *testing.T) {
  runChecks(t, New("-0.5 +0.5 -.5 +.5 .5 .5e5 .5e+5 .5e-5 5e5"), []Check{
    {TokenFloat, "-0.5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenFloat, "+0.5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenFloat, "-.5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenFloat, "+.5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenFloat, ".5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenFloat, ".5e5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenFloat, ".5e+5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenFloat, ".5e-5", Position{}},
    {TokenSpace, " ", Position{}},
    {TokenFloat, "5e5", Position{}},
    {EOF, "", Position{}},
  })
}
```

Tests fail:

```shell
$ go test ./...
--- FAIL: TestNextTokenOnIntDecimal (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Integer", got="Illegal"
--- FAIL: TestNextTokenOnIntHex (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Integer", got="Illegal"
--- FAIL: TestNextTokenOnIntOctal (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Integer", got="Illegal"
--- FAIL: TestNextTokenOnFloat (0.00s)
    lexer_test.go:19: tests[0] - tokentype wrong. expected="Float", got="Illegal"
FAIL
```

Now, before writing our `lexNumber` we want to have an `accept` function which does something similar to `acceptWhile` but only one time instead of in a loop. This will help us to check if our number, as an example, is starting by 0 in which case it might be a hexadecimal number.

```go lexer/impl.go
//...

func (l *Impl) accept(valid string) bool {
  if strings.ContainsRune(valid, l.next()) {
    return true
  }
  l.backup()
  return false
}
```

Now we can write our `lexNumber` function. We are going to start by assuming that our set of possible characters are from 0 to 9. Then we check if the number starts with the character 0. If it's the case, it will be an Hexadecimal or an Octal. We update the set of possible characters based on that.

Now that we know the possible set of characters, we can do an `acceptWhile` to read the digits. After the number we might see a dot for floating-point numbers. There we are going to do another `acceptWhile` to read all the digits. And finally, after all of this, we can still have an exponent followed by a sign and digits.

```go lexer/impl.go
//...

func lexNumber(l *Impl) stateFn {
  var t TokenType = TokenInt

  l.accept("+-")

  digits := "0123456789" // decimal

  if l.accept("0") { // starts with 0
    if l.accept("xX") {
      digits = "0123456789abcdefABCDEF" // hexadecimal
    } else {
      digits = "01234567" // octal
    }
  }

  l.acceptWhile(digits)

  if l.accept(".") {
    t = TokenFloat
    l.acceptWhile("0123456789")
  }

  if l.accept("eE") { // exponent
    t = TokenFloat
    l.accept("+-")
    l.acceptWhile("0123456789")
  }

  return l.emit(t)
}
```

We add that to `lexProto`:

```go lexer/impl.go
func lexProto(l *Impl) stateFn {
  switch r := l.next(); {
  //...
  case r == '+' || r == '-' || r == '.' || ('0' <= r && r <= '9'):
    l.backup()
    return lexNumber
  }
  //...
}
```

We run our tests again:

```shell
$ go test ./...
--- FAIL: TestNextTokenOnFloat (0.00s)
    lexer_test.go:19: tests[8] - tokentype wrong. expected="Float", got="."
FAIL
```

and we still have an error. This is due to the fact that, in part 1, when we were lexing symbols, we added this case statement:

```go lexer/impl.go
//...
case r == '.':
  return l.emit(TokenDot)
//...
```

In Protobuf, numbers can start directly with a dot and thus our lexer will just read Dot and then an Integer. So we need to skip the lexing of a dot if it's followed by a number.

```go lexer/impl.go
//...
case r == '.' && !unicode.IsNumber(l.peek()):
  return l.emit(TokenDot)
//...
```

And our tests pass:

```shell
go test ./...
ok      github.com/Clement-Jean/protein/lexer  0.763s
```

> Note: I am aware that some invalid numbers can pass through this lexing function. For example, the invalid number `0XFF.5` will return you a float. However, this is not the lexer that should handle the verification of number, the parser will. The lexer's job is to return tokens.

## We can Lex!

As promised in the beginning of the article, we are able to lex a file called `descriptor.proto`. This file can be found [here](https://github.com/protocolbuffers/protobuf/blob/main/src/google/protobuf/descriptor.proto). Just copy its content to a file.

Now, we need to write some main function to run our lexer. It will read the first argument from the command line (no error handling because this is just a test), read the file to a string, initialize a lexer and will repeatedly call the `NextToken` until EOF.

```go main.go
package main

import (
  "log"
  "os"

  "github.com/Clement-Jean/protein/lexer"
)

func main() {
  args := os.Args[1:]
  content, err := os.ReadFile(args[0])
  if err != nil {
    log.Fatal(err)
  }
  l := lexer.New(string(content))

  for {
    token := l.NextToken()

    log.Println(token)
    if token.Type == lexer.EOF {
      break
    }
  }
}
```

we run:

```shell
$ go run main.go descriptor.proto
```

and we should have output similar to:

```shell
...
2023/02/21 18:01:58 {} } {38493 920 0}}
2023/02/21 18:01:58 {Space 
 {38494 920 1}}
2023/02/21 18:01:58 {} } {38495 921 0}}
2023/02/21 18:01:58 {Space 
 {38496 921 1}}
2023/02/21 18:01:58 {EOF  {38497 922 0}}
```

## Conclusion

In this article, we tokenized all the elements that we need to get started with our parser. We are even able to lex proto files in the protobuf library! In the next episode, before going to the parser, we are going to make sure that our token positions are correct because up until now we didn't test that.

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**

<div class="container">
  <div class="row">
    <div class="col text-center">
      <a href="/protein_lexer_part_1" class="btn btn-danger text-center">Previous Article</a>
    </div>
    <div class="col text-center">
      <a href="/protein_lexer_part_3" class="btn btn-danger text-center">Next Article</a>
    </div>
  </div>
</div>