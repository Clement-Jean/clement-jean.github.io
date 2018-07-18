---
title: Improving search in double linked lists — C/C++
category: Programming
tags: [C/C++, programming]
---

![chain](https://images.pexels.com/photos/145683/pexels-photo-145683.jpeg?auto=compress&cs=tinysrgb&dpr=2&h=750&w=1260)

**Disclaimer**: I consider you know what are [linked lists](https://en.wikipedia.org/wiki/Linked_list) and [binary search trees](https://en.wikipedia.org/wiki/Binary_search_tree)

After repeating and repeating some code, we finish by losing creativity and write the same code without really thinking about it. Are there some improvements I can do? Are there possible ways to make the code clearer? We do not focus on that but we focus more on the result.

## Read some code

While writing the [RedBlackTree class](https://github.com/Clement-Jean/CsJourney/blob/master/CsJourney/RedBlackTree.h) in the [cjl library](https://github.com/Clement-Jean/CsJourney) and after reading some code on the web and especially [this article](http://eternallyconfuzzled.com/tuts/datastructures/jsw_tut_rbtree.aspx), I discovered a new way to think about the prev and next pointers.

In his article, the writer of this blog, gives us the definition of the structure he is using all along the tutorial. And here it is:

``` cpp
struct jsw_node
{
    int red;
    int data;
    struct jsw_node *link[2];
};

struct jsw_tree
{
    struct jsw_node *root;
};
```

What’s important here is the line "struct jsw_node *link[2];", link[0] will be the left node and link[1] will be the right one. He gives this definition instead of:

``` cpp
struct jsw_node
{
    int red;
    int data;
    struct jsw_node *left;
    struct jsw_node *right;
};

struct jsw_tree
{
    struct jsw_node *root;
};
```

There is no mistake, it’s left and right because we are in a red black tree (binary search tree). The first definition helps him a lot in the iterations through the tree. Here is his method:

``` cpp
int dir = node->data < data;

node->link[dir] = ...
```

As all the left elements are smaller and the right elements are greater than the current node, the initialization of ‘dir’ will then give either 0 or 1 (remember our left and right node).

## Be creative

For some of you, you understood the link with the linked lists and you are already thinking about how to improve yours, right? But let’s continue for the ones who can’t see clearly what is my point.

As explained, now for my linked lists, instead of doing:

``` cpp
struct node
{
    int data;
    struct node *prev;
    struct node *next;
};

struct list
{
    struct node *root;
};
```

I can do:

``` cpp
struct node
{
    int data;
    struct node *link[2];
};

struct list
{
    struct node *root;
};
```

By itself it is not amazing, I totally agree with you but the usage of it makes the code cleaner. Let’s see how.

First, I redefine my list struct:

``` cpp
struct list
{
   struct node *root;
   struct node *end;
   int size;
};
```

I’m using this structure because, first I’m developing a little library so I need to be able to do “list.Size()” without iterating through all the list each time, then I can do iterators (like: list.Begin() and list.End()), and finally because I can eliminate few step in the access of a node at position P (list[4]).

As the 2 firsts steps are common sense, we will only talk about the last one. As you already know linked lists are not the best data structure in terms of search ( Θ(n)). However, even if it will keep its Θ(n), we can improve them a little bit by removing some unnecessary steps.

If the element is nearer from the end pointer than from the root one, why would we start by root and then iterate? We shouldn’t ! However, with the use of next and prev pointer you would use if and else clauses, right? With the new structure it is not necessary. Look at that:

``` cpp
int distanceFromEnd = this->size - position - 1;
int min = std::min(position, distanceFromEnd);
int dir = position < distanceFromEnd;

auto current = dir == 1 ? this->root : this->end;

for (int i = 0; i < min; ++i)
{
   current = current->_link[dir];
}
```

We first calculate the difference between the end and the position (distanceFromEnd), then we define which one between the distance from the begin (position) and distanceFromEnd is smaller, and finally we define the starting point (root or end) and if we need to iterate with the equivalent of prev or the next pointer. It gives us an implementation without extra if/else clause.

## Further reading

- [Red black trees](http://eternallyconfuzzled.com/tuts/datastructures/jsw_tut_rbtree.aspx)
- [UnaryOperator in C++](https://clement-jean.github.io/UnaryOperator-in-C++/)

## Conclusion

**If you would like to join me in the adventure of developing a little library for c++ which include other data structures than the ones in stl, I’m looking for people who can improve the overall architecture and help me to develop some algorithms.**

⚠️ ⚠️ ⚠️ ⚠️[Come here to join !](https://github.com/Clement-Jean/CsJourney) ⚠️ ⚠️ ⚠️ ⚠️
