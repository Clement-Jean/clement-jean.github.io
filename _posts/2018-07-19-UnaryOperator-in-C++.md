---
category: Programming
tags: [C/C++, programming]
---

Sometimes, while writing a class in c++ you just notice little differences between two member functions. While writing my [SparseMatrix class](https://github.com/Clement-Jean/CsJourney/blob/master/CsJourney/SparseMatrix.h) in my [cjl library](https://github.com/Clement-Jean/CsJourney), I noticed that the addition and the subtraction of matrix is really similar (Just change + by -).

After analyzing how the c++ stl is working, I noticed something really great in term of code factorization : UnaryOperators.

``` cpp
template <class InputIterator, class OutputIterator, class UnaryOperator>
OutputIterator transform (InputIterator first1, InputIterator last1, OutputIterator result, UnaryOperator op)
{
  while (first1 != last1) {
    *result = op(*first1);
    ++result; ++first1;
  }
  return result;
}
```

In this example given in the [std::transform page](http://www.cplusplus.com/reference/algorithm/transform/), we can see a template parameter called UnaryOperator and a parameter using this type called ‘op’. **But what’s this ?** Let’s see it by seeing an example of std::transform usage.

``` cpp
std::transform (foo.begin(),foo.end(),bar.begin(),std::plus<int>);
```

In this example, foo and bar are two std::vector<int>. std::transform will then store the transformation’s result in bar. **But what’s the transformation’s result?**

Here the transformation result is the result of std::plus<int> which add two int together. **This is a UnaryOperator!** Now let’s see how to use it for SparseMatrix code factorization.

## The problem

``` cpp
SparseMatrix<T> &Add(SparseMatrix &matrix)
{
   if (matrix._col != this->_col || matrix._row != this->_row)
      throw exception;
   SparseMatrix *newMatrix = new SparseMatrix(this->_row,
                                              this->_col);
   int sum;
   for (int i = 0; i < this->_row; ++i)
   {
      for (int j = 0; j < this->_col; ++j)
      {
         sum = this->Get(i, j) + matrix.Get(i, j));
         newMatrix->Set(i, j, sum);
      }
   }
   
   return *newMatrix;
}
SparseMatrix<T> &Subtract(SparseMatrix &matrix)
{
   if (matrix._col != this->_col || matrix._row != this->_row)
      throw exception;
   SparseMatrix *newMatrix = new SparseMatrix(this->_row,
                                              this->_col);
   int sub;
   for (int i = 0; i < this->_row; ++i)
   {
      for (int j = 0; j < this->_col; ++j)
      {
         sub = this->Get(i, j) - matrix.Get(i, j));
         newMatrix->Set(i, j, sub);
      }
   }
   
   return *newMatrix;
}
```

And this is a disaster because the only concrete changes are the operators.

## The solution

``` cpp
template<class UnaryOperator>
SparseMatrix<T> &Operation(UnaryOperator operation, SparseMatrix &matrix)
{
   if (matrix._col != this->_col || matrix._row != this->_row)
      throw exception;
   SparseMatrix *newMatrix = new SparseMatrix(this->_row, 
                                              this->_col);
   int resultOperation;
   for (int i = 0; i < this->_row; ++i)
   {
      for (int j = 0; j < this->_col; ++j)
      {
         resultOperation = operation(this->Get(i, j),
                                     matrix.Get(i, j));
         newMatrix->Set(i, j, resultOperation);
      }
   }
   return *newMatrix;
}

SparseMatrix<T> &Add(SparseMatrix<T> &matrix)
{
   return Operation(std::plus<T>(), matrix);
}

SparseMatrix<T> &Subtract(SparseMatrix<T> &matrix)
{
   return Operation(std::minus<T>(), matrix);
}
```

What do you think about it? Cleaner right?

## Conclusion

**If you would like to join me in the adventure of developing a little library for c++ which include other data structures than the ones in stl, I’m looking for people who can improve the overall architecture and help me to develop some algorithms.**

⚠️ ⚠️ ⚠️ ⚠️[Come here to join !](https://github.com/Clement-Jean/CsJourney) ⚠️ ⚠️ ⚠️ ⚠️
