---
layout: post
author: Clement
title: "Protein: Lexer (Part 3)"
categories: [Protocol Buffers, Go]
---

This article is a small one intended to solve a bug related to token position. As of right now, we only tested that our token got the right literal and the right token kind. In this article, we are going to add the position checking in our tests.

## Position Checking

Adding position checking in our test is pretty trivial since it's 3 ifs checking `Offset`, `Line`, and `Column`. So let's add that:

```go lexer/impl.go
func runChecks(t *testing.T, l Lexer, tests []Check) {
  for i, tt := range tests {
    //...

    if tok.Position.Offset != tt.expectedPosition.Offset {
      t.Fatalf("tests[%d] - offset wrong. expected=%d, got=%d", i, tt.expectedPosition.Offset, tok.Position.Offset)
    }

    if tok.Position.Line != tt.expectedPosition.Line {
      t.Fatalf("tests[%d] - line wrong. expected=%d, got=%d", i, tt.expectedPosition.Line, tok.Position.Line)
    }

    if tok.Position.Column != tt.expectedPosition.Column {
      t.Fatalf("tests[%d] - column wrong. expected=%d, got=%d", i, tt.expectedPosition.Column, tok.Position.Column)
    }
  }
}
```

And now if we run our tests, we should have a lot of errors coming from the fact that Go will initialize `Offset`, `Line`, and `Column` to 0 (default value). An example of error received is:

```shell
$ go test ./..
--- FAIL: TestNextTokenOnSymbols (0.00s)
    lexer_test.go:31: tests[0] - line wrong. expected=0, got=1
FAIL
```

> Before going to the new section, make sure that you update the position objects in your tests. If you are not willing to calculate all of the positions, you can just refer to the [tests in the github repo](https://github.com/Clement-Jean/protein/blob/lexer/lexer/lexer_test.go) where I did it for you.

## A bug ?!

Now that we have all our positions set, we can rerun our tests.

```shell
$ go test ./..
--- FAIL: TestNextTokenOnSpace (0.00s)
    lexer_test.go:35: tests[1] - column wrong. expected=4, got=0
FAIL
```

And yes we have an error. Let's understand it.

The problem here comes from the way we handle newlines in the `emit` function. As of right now, this is done like so:

```go lexer/impl.go
func (l *Impl) emit(tt TokenType) stateFn {
  //...
  if tt == TokenSpace && strings.Contains(t.Literal, "\n") {
    l.startLineOffset = l.start
  }
  //..
}
```

This code is checking for a newline inside the literal and if it finds one, it will just set the index of `\n` in the literal to startLineOffset. The problem here is that we handle all consecutive spaces (the general term) as one token. So when we have `\t\n\v\f\r `, we are effectively saying that the line starts at the beginning our our space token. This is not correct, right? We should be setting `startLineOffset` to 2 (just after `\n`) and then this should affect the `Column` position because of `Column: l.start - l.startLineOffset` in the `Token` instantiation in `emit`.

So how do we solve that? Well, it turns out to be pretty simple. We are going to look for the last instance of `\n` in the literal and this will give us the beginning of the line. After that we are going to take the current position (which is after the token right now) and subtract it with the length of the literal minus the beginning of the line. This gives us the offset at which the line begins. So now we should have this:

```go lexer/impl.go
func (l *Impl) emit(tt TokenType) stateFn {
  //...
  if tt == TokenSpace {
    if lineStart := strings.LastIndex(t.Literal, "\n"); lineStart != -1 {
      l.startLineOffset = l.start - (len(t.Literal) - 1 - lineStart)
    }
  }
  //..
}
```

Note that we are only finding the last index when the token kind is a space. This is important because if we do that for all the tokens we will have performance hits (especially on large tokens).

And now, if we rerun our test:

```shell
$ go test ./...
ok      github.com/Clement-Jean/protein/lexer  0.857s
```

## Conclusion

In this article, we made the final test for our lexer and we solved a critical bug for `Token` positions. We now have a functional lexer and in the next article we are going to start the parser!

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**

<div class="container">
  <div class="row">
    <div class="col text-center">
      <a href="/protein_lexer_part_1" class="btn btn-danger text-center">Previous Article</a>
    </div>
    <!-- <div class="col text-center">
      <a href="/protein_lexer_part_1" class="btn btn-danger text-center">Next Article</a>
    </div> -->
  </div>
</div>