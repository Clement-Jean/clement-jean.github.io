---
layout: post
author: Clement
title: "Range Testing in Strings"
categories: [Go, Protobuf]
---

Recently, I've been working on adding support for `SourceCodeInfo` into [Protein](https://github.com/Clement-Jean/protein). This required checking a lot of Column/Line ranges in string. An example of this is the following. Given a oneof like this:

```proto
oneof Test {
  int32 id = 1;
}
```

we should come up with the following ranges:

```
[
  // line,column
  {0, 0, 2, 1} // oneof - from 0,0 to 2, 1
  {1, 2, 15}   // oneof field - from 1,2 to 1,15 (same lines get ommited)
  {1, 2, 7}    // oneof field type - from 1,2 to 1,7
  {1, 8, 10}   // oneof field name - from 1,8 to 1,10
  {1, 13, 14}  // oneof field tage - from 1,13 to 1,14
]
```

This might seem like a daunting and it was until I found out how to test ranges correctly for these kind of situations.

## SourceCodeInfo

Before starting with the whole testing thing. It is important to get a sense of what a Protobuf's `SourceCodeInfo` is. As its name suggests this is information about the source code. This information is basically lines and columns for tokens (called spans) and some tags sequence starting from `FileDescriptorProto` (called path). This info is mostly important for tools like what Protein will be: linters, LSPs, ... It gives us a way to find elements both in terms of position (line 1, column 10) in code and in terms of context (a oneof inside a message).

While the second part is pretty interesting, we are not going to cover that. We will focus on testing the spans correctly. However, if you are interested in learning more about paths, I'd be happy to write an article on it. Leave a comment if you are.

## Naive Testing

Now that we know what are `SourceCodeInfo` we can start with the testing. A naive and rather manual solution to solve this is probably writing every span by hand. This is pretty much what I did in the introduction of this article. This could mean something like this:

```go
func runSourceCodeInfoCheck(
  t *testing.T,
  info []*descriptorpb.SourceCodeInfo_Location,
  expectedSpans [][]int32,
) {
  if len(info) == 0 {
    t.Fatal("expected info")
  }

  if len(info) != len(expectedSpans) {
    t.Fatalf("expected %v, got: %v", expectedSpans, info)
  }

  for i, expectedSpan := range expectedSpans {
    if slices.Compare(info[i].Span, expectedSpan) != 0 {
      t.Fatalf("path %d wrong. expected %v, got: %v", i, expectedSpan, info[i].Span)
    }
  }
}

func TestOneofSourceCodeInfo(t *testing.T) {
  // Arrange
  l := lexer.New("oneof Test { int32 id = 1; string uuid = 2; }")
  p := New(l, "")

  // Act
  _, info := augmentParse(p.(*Impl).parseSyntax, p.(*Impl), nil)

  // Assert
  runSourceCodeInfoCheck(
    t,
    info,
    [][]int32{
      {0, 0, 45},  // oneof - from 0,0 to 0, 45
      {0, 13, 26}, // oneof field - from 0,13 to 0,26
      {0, 13, 18}, // oneof field type - from 0,13 to 0,18
      {0, 19, 21}, // oneof field name - from 0,19 to 0,21
      {0, 24, 25}, // oneof field tage - from 0,24 to 0,25
      // etc...
    },
  )
}
```

This looks rather simple and if we stick to testing small pieces of code, it is feasible to get our way through. However, as you might expect, this is tiring and very repetitive work. Imagine doing that for every single concept in Protobuf...

## A Better Way

For full disclosure, this idea for testing ranges in strings is not my idea. This is an idea I discovered after reading Protobuf documentation and unit tests. An example of this is the documentation for `SourceCodeInfo` in the descriptor.proto file:

```go
// Let's look at just the field definition:
//   optional string foo = 1;
//   ^       ^^     ^^  ^  ^^^
//   a       bc     de  f  ghi
// We have the following locations:
//   span   path               represents
//   [a,i)  [ 4, 0, 2, 0 ]     The whole field definition.
//   [a,b)  [ 4, 0, 2, 0, 4 ]  The label (optional).
//   [c,d)  [ 4, 0, 2, 0, 5 ]  The type (string).
//   [e,f)  [ 4, 0, 2, 0, 1 ]  The name (foo).
//   [g,h)  [ 4, 0, 2, 0, 3 ]  The number (1).
```

We can just focus on the span and how they mark the beginning and end of them with letters. `optional` as a span of [a, b) (from a to b non-inclusive). Meaning that we go from column 0 to column 8 (length of the work optional) but you can see that `b` is marking the space character so we do not include that.

Now, even I didn't get the original idea, I believe that implementing it in Go (original in C++) and adding line support is quite interesting. Let us start by the v1 which didn't support multiline code.

The idea is that we are going to have function calculating indices from a string full of separator characters and letters. For example, if we say that the separator is '-', we could have a string like this:

```
a------------b----cd-e--fghi-----jk---l--mno-p
```

that would match a oneof like this one:

```proto
oneof Test { int32 id = 1; string uuid = 2; }
```

To better see it we will have a function that takes both the original Protobuf code and the reference string (that is what I called the separator-full string) as parameters:

```go
referenceString(
  "oneof Test { int32 id = 1; string uuid = 2; }",
  "a------------b----cd-e--fghi-----jk---l--mno-p",
)
```

This nicely matches and it is easier to visually see where the span starts and ends when we know the letter. If I told you there should be a span [b, h), you can clearly understand that these references to the id field definition.

Now, how should we represent all of this in terms of data structure? The naive approach is to create a `map[rune]int32`. The `rune` will be the letterm and we are return `int32` instead of `int` simply because `SourceCodeInfo` is expecting `int32`s. Then, when we will want to check the value of `a`, we can simply do:

```go
a := refs['a']
```

This doesn't seem that bad right? Well, what if you need to access letters a to z? You basically have 26 of these variables around. Feasible but not that ergonomic.

## An Even Better Way

My second thought on how to improve this comes from my early interest in reflection. I find it amazing that we take a look at the guts of our program and manipulate it programmatically. An example of that is listing all the fields in a struct and/or set values to them. I don't know what you think but for me this is just so powerful (and dangerous!).

Enough about my geekiness on reflection. What if we could simply have an object into which we will set the values of our spans. This would let us write something like following for accessing values:

```go
ref.A
```

How nice would that be? We would only have one variable (ref) and we could access the fields.

It turns out that we can do it pretty easily. Think about a `struct` like the following:

```go
type Ref struct {
  A, B, C, D, E, F, G, H, I, J, K, L, M, N, O, P, Q, R, S, T, U, V, W, X, Y, Z int32
}
```

I agree that this definition is not that beautiful but it will make our test code easier to read.

With that `Ref`, we will now use reflection to set `A` (uppercase because reflection require exported fields) when we see a `a` in the string. This will look like this:

```go
// referenceString returns the original string and the newly created Ref
// the sep argument is the separator we skip (e.g `-`)
func referenceString(src string, indices string, sep rune) (string, Ref) {
  // indices should always be longer than src by 1 rune
  if len(indices) != len(src)+1 {
    panic("wrong indices")
  }

  ref := Ref{}

  for i, index := range indices {
    if index != sep {
      // checks valid characters (lowercase letter)
      if !unicode.IsLetter(index) && !unicode.IsLower(index) {
        panic(fmt.Sprintf("%v is not a lowercase letter", index))
      }

      // this is the index of the letter in our Ref struct!
      // e.g A is at index 0 and Z is at index 25
      idx := int(byte(index) - 'a') // ASCII trick to get index of letter in alphabet

      // set the value of i to the field at index idx
      reflect.ValueOf(&ref).Elem().Field(idx).SetInt(int64(i))
    }
  }

  return src, ref
}
```

with that we can simply write the following:

```go
_, ref := referenceString(
  "oneof Test { int32 id = 1; string uuid = 2; }",
  "a------------b----cd-e--fghi-----jk---l--mno-p",
)
```

and if we print `ref` we get:

```go
Ref {A: 0, B: 13, C: 18, D: 19, E: 21, F: 24, G: 25, H: 26, I: 27, J: 33, K: 34, L: 38, M: 41, N: 42, O: 43, P: 45, Q: 0, R: 0, S: 0, T: 0, U: 0, V: 0, W: 0, X: 0, Y: 0, Z: 0}
```

If we check at the span [b, h), we can see that we have [13, 26). This is quite powerful and way more readable. If we rewrite the `TestOneofSourceCodeInfo` function with the use of `Ref`, we have:

```go
func TestOneofSourceCodeInfo(t *testing.T) {
  // Arrange
  pb, ref := referenceString(
    "oneof Test { int32 id = 1; string uuid = 2; }",
    "a------------b----cd-e--fghi-----jk---l--mno-p",
    '-',
  )

  l := lexer.New(pb)
  p := New(l, "")

  // Act
  _, info := augmentParse(p.(*Impl).parseSyntax, p.(*Impl), nil)

  // Assert
  runSourceCodeInfoCheck(
    t,
    info,
    [][]int32{
      {0, ref.A, ref.P},  // oneof - from 0,0 to 0, 45
      {0, ref.B, ref.H}, // oneof field - from 0,13 to 0,26
      {0, ref.B, ref.C}, // oneof field type - from 0,13 to 0,18
      {0, ref.D, ref.E}, // oneof field name - from 0,19 to 0,21
      {0, ref.F, ref.G}, // oneof field tage - from 0,24 to 0,25
      // etc...
    },
  )
}
```

This now look a little bit less magic than before with all these numbers everywhere.

## Supporting lines

As you can see, we still have these 0s for each line. They actually represent lines. Could we also support multiline code? This would let us write something like:

```go
pb, ref := referenceString(
  `oneof Test {
int32 id = 1;
string uuid = 2;
}`,
  `a------------
b----cd-e--fgh
i-----jk---l--mno
-p`,
  '-',
)
```

Without indentation that looks a little bit weird but this is already letting us testing a little bit more in depth.

The first thing that we are going to do is adding fields in `Ref` for lines. This looks like:

```go
type Ref struct {
  //...
  LA, LB, LC, LD, LE, LF, LG, LH, LI, LJ, LK, LL, LM, LN, LO, LP, LQ, LR, LS, LT, LU, LV, LW, LX, LY, LZ int32
}
```

Cringing a little? It's fine! Keep in mind that this is for the sake of having more expressive tests.

Now, in referenceString we will keep track of columns and lines and, for `a`, we are going to set `A` to the column and `LA` to the line. We now have:

```go
func referenceString(src string, indices string, sep rune) (string, Ref) {
  if len(strings.ReplaceAll(indices, "\n", "")) != len(src)+1 {
    panic("wrong indices")
  }

  ref := Ref{}
  line := int32(0) // the line
  column := int32(0) // the column - do not use i anymore

  for _, index := range indices {
    if index != sep && index != '\n' { // also check '\n'
      if !unicode.IsLetter(index) && !unicode.IsLower(index) {
        panic(fmt.Sprintf("%v is not a lowercase letter", index))
      }

      idx := int(byte(index) - 'a')

      // set the column
      reflect.ValueOf(&ref).Elem().Field(idx).SetInt(int64(column))

      // set the line
      reflect.ValueOf(&ref).Elem().Field(idx + 26).SetInt(int64(line))
    }

    column += 1

    // on newline reset column and increase line
    if index == '\n' {
      line++
      column = 0
    }
  }

  return src, ref
}
```

With that we can now write a test for multiline like this:

```go
func TestOneofMultilineSourceCodeInfo(t *testing.T) {
  pb, ref := referenceString(
    `oneof Test {
int32 id = 1;
string uuid = 2;
}`,
    `a------------
b----cd-e--fgh
i-----jk---l--mno
-p`,
    '-',
  )

  l := lexer.New(pb)
  p := New(l, "")
  ctx := &oneofContext{}

  // Act
  _, info := augmentParse(p.(*Impl).parseOneof, p.(*Impl), ctx)

  // Assert
  runSourceCodeInfoCheck(
    t,
    info,
    [][]int32{
      {ref.LA, ref.A, ref.LP, ref.P},
      {ref.LB, ref.B, ref.H}, {ref.LB, ref.B, ref.C}, {ref.LD, ref.D, ref.E}, {ref.LF, ref.F, ref.G},
      {ref.LI, ref.I, ref.O}, {ref.LI, ref.I, ref.J}, {ref.LK, ref.K, ref.L}, {ref.LM, ref.M, ref.N},
    },
  )
}
```

Take your time to wrap your mind around it. We just made the all things look a little bit more verbose but less magical and frightening.

## Advantages

- New developers looking at these tests will probably be less afraid of writing a new test.
- Fewer places where we can make typos. Most typos will be in the reference string.
- Changing the spans or separators requires us to only update the reference strings, not all the numbers in int32 arrays.
- Reflection is letting us create a map out of a struct and have fewer variables.

## Disadvantages

- More verbose.
- Reflection is kind of magical too. However, magic is only happening in `referenceString`. Not everywhere like before.

## Conclusion

I hope this makes you as interested as I am on how to improve testing code. I already loved creating readable and deterministic tests but now with this other tool in my tool belt, I'm interested in thinking more about readability and developer onboarding.

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**