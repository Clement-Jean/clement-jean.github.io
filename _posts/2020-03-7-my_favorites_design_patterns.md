---
layout: post
author: Clement
title: My favorites design patterns - Builder
categories: [Design pattern]
---

Recently, I was trying to handle errors in the programming language I'm developping. Nothing fancy here and the problem was quickly solved with a simple:

```cpp
throw Error(Error::Type::UNEXPECTED_TOKEN, "Expected '(' but got " + token->get_literal());
```

## The problems
- Lack of genericity: Each time I expected or got a different token, I needed to change the text in the message.

```cpp
throw Error(Error::Type::UNEXPECTED_TOKEN, "Expected '(' but got " + token->get_literal());
throw Error(Error::Type::UNEXPECTED_TOKEN, "Expected ')' but got " + token->get_literal());
```

- Lack of testability: I basically wanted to be able to check if the error type and the error message were the same. But then if I have a stupid typo then my test fail.

```cpp
ASSERT_EQ(error, Error::Type::UNEXPECTED_TOKEN, "xpected '(' but got " + token->get_literal());
```

So at that point, I decided to make the solution more generic and more testable. Basically, I wanted something roughly like:

```cpp
Error error = expect("(").got("{")
```

## The solution

And here comes the Builder pattern. The idea is that we could build an object by changing the variables in an expressive way. So finally, I came up with this:

```cpp
class Expect {
private:
    std::string _expected;
    std::string _got;

public:
    static Expect builder() { return Expect(); }

    Expect &expect(const std::string &expected) { _expected = expected; return *this; }
    Expect &got(const std::string &got) { _got = got; return *this; }

    Error build() {
        return Error(Error::Type::UNEXPECTED_TOKEN, "Expected '" + _expected + "' but got '" + _got + "'");
    }
};
```

That you would use like:

```cpp
Expect::builder().expect("(").got("{").build()
```

A much shorter, expressive and typo incensitive solution !