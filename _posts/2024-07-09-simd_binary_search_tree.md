---
layout: post
author: Clement
title: "Binary Search Tree with SIMD"
categories: [Go, SIMD]
---

Recently, I've been looking at cache friendly algorithm for common data structures like trees, tries, ... One such algorithm kept coming up to mind and that's why I decided to implement it in Go. You can find the paper describing the algorithm [here](https://dl.acm.org/doi/10.1145/1807167.1807206).

## The Intuition

Let's assume that we have a binary tree:

```
      ┌────── 41 ──────┐
      │                │
   ┌──23──┐       ┌───61───┐
   │      │       │        │
┌─11─┐  ┌─31─┐  ┌─47─┐  ┌─73─┐
│    │  │    │  │    │  │    │
2   19  29  37  43  53  67  79
```

A normal way of representing a binary tree into an array is by representing it like so:

```
┌────┬────┬────┬────┬────┬────┬────┬───┬────┬────┬────┬────┬────┬────┬────┐
│ 41 │ 23 │ 61 │ 11 │ 31 │ 47 │ 73 │ 2 │ 19 │ 29 │ 37 │ 43 │ 53 │ 67 │ 79 │
└────┴────┴────┴────┴────┴────┴────┴───┴────┴────┴────┴────┴────┴────┴────┘
```

or, in other words, we have the level nodes layed out consecutively.

This approach however, is not that cache friendly. While the locality of the data for a level is good, in a binary search, we actually care more about the parent-children locality. This is because, by keeping the children next to the parent, we wouldn't need to jump far ahead in the array.

The paper mentionned at the beginning, propose to have a binary tree layed out like the following:

```
┌────┬────┬────┬────┬───┬────┬────┬────┬────┬────┬────┬────┬────┬────┬────┐
│ 41 │ 23 │ 61 │ 11 │ 2 │ 19 │ 31 │ 29 │ 37 │ 47 │ 43 │ 53 │ 73 │ 67 │ 79 │
└────┴────┴────┴────┴───┴────┴────┴────┴────┴────┴────┴────┴────┴────┴────┘
```

And if you take time to understand how this maps back to the binary tree, you will notice that we are storing parent-children triangles. With this, we can now apply SIMD operations on both the parent and the children in order to either dtermine if the data we are looking for is in the triangle, or if we should continue our search.

The next important thing to understand is how we do the search of elements.

Let's take an example to make things clearer. Let's say that we are looking for the number 62. We will start by loading 41, 23, 61 into a vector.

```
┌────┬────┬────┐
│ 41 │ 23 │ 61 │
└────┴────┴────┘
```

Then, we will compare (smaller than) each number with the element we are looking for:

```
┌────┬────┬────┐
│ 41 │ 23 │ 61 │
└────┴────┴────┘
        <
┌────┬────┬────┐
│ 62 │ 62 │ 62 │
└────┴────┴────┘
        =
┌────┬────┬────┐
│  1 │  1 │  1 │
└────┴────┴────┘
```

and with the mask we get, we can map to an index in the following subtrees. Here is the full mapping:

```
 0 0 0 -> 0
 0 1 0 -> 1
 1 1 0 -> 2
 1 1 1 -> 3
```

It's actually a popcount.

So, with our `[1, 1, 1]`, we should access the 4th child (mapping is 0 indexed).

If you look at the binary tree, this means that we need to go to the number 73 and we now have the vector:

```
┌────┬────┬────┐
│ 73 │ 67 │ 79 │
└────┴────┴────┘
```

Now, we can obviously repeat the process.

On top of the lookup for index, we also need to be able to check the equality of the search vector and the curr loaded vector. This is as simple as:

```
┌────┬────┬────┐
│ 41 │ 23 │ 61 │
└────┴────┴────┘
       ==
┌────┬────┬────┐
│ 62 │ 62 │ 62 │
└────┴────┴────┘
        =
┌────┬────┬────┐
│  0 │  0 │  0 │
└────┴────┴────┘
```

If we found a 1, it would mean that the element is in the tree. Otherwise, we keep running as long as we are within the boundaries of the array.

Hopefully, this all makes sense. Let us move to the code.

## The Code

As [my proposal for adding SIMD intrinsics](https://github.com/golang/go/issues/67520) is still not evaluated/accepted, we will need to write Go assembly (ARM64) to make use of SIMD instructions. I'll try to be as clear as possible on what each instruction is doing but you should know at least basics of assembly.

Let's start by defining the function definition in our `main.go`:

```go main.go
package main

func binarySearch(arr []uint32, n uint32) bool

func main() {
  //...
}
```

You can see that we are working with uint32s. This is because on ARM64 Neon, we only have 128 bits, and as we need to load at least 3 elements per triangle, uint32 is our best choice.

Next, we will jump to our `main.s` file and start defining our function:

```c main.s
#include "textflag.h"

//func binarySearch(arr []int, n int) bool
TEXT ·binarySearch(SB),NOSPLIT,$0-33
  //...
```

The most important thing here is the `$0-33` part. We are saying that we do not have local variables (0) and that our arguments/return value take 33 bytes (24 for the slice, 8 for the int, and 1 for the bool).

Next, as part of my function, I generally like to define some names for the register. It helps me remember what each register is supposed to contain. This looks like this:

```c main.s
TEXT ·binarySearch(SB),NOSPLIT,$0-33
#define data R0
#define dataLen R1
#define toFind R2
#define curr R3
#define tmp R4
#define child_idx R5
#define nb_subtree R6
#define level R7
#define searchKey V0
#define mask V1
#define idx V2
#define one V3
#define equalMask V4
```

With that, we can initialize the registers and check the base cases:

```c main.s
TEXT ·binarySearch(SB),NOSPLIT,$0-33
//...

  // initialize registers
  MOVD arr+0(FP), data
  MOVD arr_len+8(FP), dataLen
  MOVD n+24(FP), toFind
  MOVD $0, curr
  MOVD $1, level
  MOVD $0, nb_subtree
  VDUP level, one.S4

  // if array len is 0 return false
  CMP $0, dataLen
  BEQ not_found

  // if array len > 1 start the work
  // otherwise check if the first element is equal
  //  to the one we are looking for
  CMP $1, dataLen
  BGT load
  MOVD (data), tmp
  CMP tmp, toFind
  BEQ found
  B not_found

  //...

not_found:
  MOVD $0, R19 // false
  MOVD R19, ret+32(FP)
  RET

found:
  MOVD $1, R19 // true
  MOVD R19, ret+32(FP)
  RET
```

Now, we can start the real work. We will have a simple loop which we load 4 elements at the `curr` position in `data`:

```c main.s
TEXT ·binarySearch(SB),NOSPLIT,$0-33
//...

load:
  VDUP toFind, searchKey.S4

check:
  CMP dataLen, curr
  BGE not_found

loop:
  MOVD $4, R19
  MUL R19, curr
  ADD curr, data, R19
  VLD1 (R19), [mask.S4]

  //TODO update curr

  B check
```

Notice that we are multiplying `curr` by 4. This is because we are working with uint32s (4 bytes) so our index (`curr`) need to be moved by `curr * 4` bytes.

Then, inside the loop, we will check for equality between the four loaded elements and the search vector:

```c main.s
TEXT ·binarySearch(SB),NOSPLIT,$0-33
//...

loop:
  //...

  VCMEQ mask.S4, searchKey.S4, equalMask.S4
  WORD $0x6eb0a893 //umaxv.4s s19, v4
  FMOVS F19, R19
  CMP $4294967295, R19
  BEQ found

  //...
```

You can notice that if the maximum value inside `equalMask` is `math.MaxUint32` (4294967295), it means that we found the element and thus we can return.

After that, we fall into the binary search algorithm. We will first start by looking for the index:

```c main.s
TEXT ·binarySearch(SB),NOSPLIT,$0-33
//...

loop:
  //...

  WORD $0x6ea13401 //cmhi.4s v1, v0, v1
  MOVD $0, R19
  VMOV R19, mask.S[3]
  VAND mask.B16, one.B16, idx.B16
  WORD $0x6eb0384f //uaddlv.4s d15, v2
  FMOVD F15, child_idx

  //...
```

The `cmhi` checks whether the data we are looking for is bigger that the data we loaded. Then, we bitwise AND the loaded vector with ones and finally, we do a basic popcount to determine the `child_idx`.

Finally, we need to update the `curr`, `level`, and the `nb_subtree`. As mentionned, the former is telling us from where to read the data in the array. The two last ones actually help us calculate the `curr` by running the following formula: `curr = nb_subtree * 3 + (3 * (child_idx + 1))`.

```c main.s
TEXT ·binarySearch(SB),NOSPLIT,$0-33
//...

loop:
  //...

  //curr = nb_subtree * 3 + (3 * (child_idx + 1))
  MOVD child_idx, tmp
  ADD $1, tmp
  MOVD $3, R19
  MUL R19, tmp
  MOVD tmp, curr
  MUL R19, nb_subtree, R19
  ADD R19, curr

  //nb_subtree = level << 2
  LSL $2, level, nb_subtree

  //level++
  ADD $1, level

  //...
```

And that actually is all for the binary search algorithm.

## A Demo

We can now go back to our `main.go` and try it.

```go main.go
package main

import "fmt"

func binarySearch(arr []uint32, n uint32) bool

func main() {
  arr := []uint32{41, 23, 61, 11, 2, 19, 31, 29, 37, 47, 43, 53, 73, 67, 79}

  fmt.Printf("%v\n", binarySearch(arr, 19))
  fmt.Printf("%v\n", binarySearch(arr, 100))
}
```

and if we run, we should have:

```sh
$ go run .
true
false
```

## Benchmark

```go main_test.go
package main

import (
  "testing"
  "slices"
)

var ok bool

func BenchmarkBinarySearchSIMD(b *testing.B) {
  nbs := []uint32{41, 23, 61, 11, 2, 19, 31, 29, 37, 47, 43, 53, 73, 67, 79}

  for i := 0; i < b.N; i++ {
    for _, item := range nbs {
      if ok = binarySearch(nbs, item); !ok {
        b.Fail()
      }
    }
  }
}

func BenchmarkBinarySearch(b *testing.B) {
  arr := []uint32{41, 23, 11, 2, 19, 31, 29, 37, 61, 47, 43, 53, 73, 67, 79}
  slices.Sort(arr)

  for i := 0; i < b.N; i++ {
    for _, item := range arr {
      if _, ok = slices.BinarySearch(arr, item); !ok {
        b.Fail()
      }
    }
  }
}
```

```sh result
$ go test -run=Benchmark -bench=. -count=10 .
goos: darwin
goarch: arm64
BenchmarkBinarySearchSIMD-10            29052963                40.82 ns/op
BenchmarkBinarySearchSIMD-10            29059149                40.80 ns/op
BenchmarkBinarySearchSIMD-10            29166654                40.83 ns/op
BenchmarkBinarySearchSIMD-10            29083417                40.83 ns/op
BenchmarkBinarySearchSIMD-10            29022134                40.84 ns/op
BenchmarkBinarySearchSIMD-10            29075196                40.83 ns/op
BenchmarkBinarySearchSIMD-10            28986556                40.81 ns/op
BenchmarkBinarySearchSIMD-10            29005532                40.81 ns/op
BenchmarkBinarySearchSIMD-10            29118674                40.79 ns/op
BenchmarkBinarySearchSIMD-10            28919640                40.79 ns/op
BenchmarkBinarySearch-10                12682536                94.19 ns/op
BenchmarkBinarySearch-10                12656307                94.20 ns/op
BenchmarkBinarySearch-10                12666488                94.19 ns/op
BenchmarkBinarySearch-10                12660024                94.68 ns/op
BenchmarkBinarySearch-10                12671432                94.22 ns/op
BenchmarkBinarySearch-10                12671989                94.25 ns/op
BenchmarkBinarySearch-10                12659746                94.19 ns/op
BenchmarkBinarySearch-10                12688084                94.23 ns/op
BenchmarkBinarySearch-10                12639111                94.21 ns/op
BenchmarkBinarySearch-10                12664016                94.24 ns/op
PASS
```

## Conclusion

In this article we saw a cache friendly version of the binary search using SIMD. While, we focused on the algorithm itself, I worked on this as part of a data structure. It is an AVL Tree that can be frozen at any point, and once frozen, it will let you do the binary search described in this article.

If you would like to have more details on the data structure implemented, or if you have feedback on this article, let me know in the comments section!
