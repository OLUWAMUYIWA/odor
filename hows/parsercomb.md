# Functional Programming in Go; How I created my own mini parsec


#### Why make a function take another function as argument?
We do this because some computations are more general than others, and encapsulating another computation that is useful in these general computations makes them quite open to sever use cases. So, I think about it like this:
This function does  something. But what it does is similar to what another function does, but for a few things, say the way a partocular set of local variables are created, or a predicate is satisfied. In such a case, e.g. I want to walk a directory.
But what do i do while working through the directory? That shouldnt matter to my Walk function. All it neds to know is how to walk a directory. It can then take another argument that specifies what exactly any special `Walk` case would like to do while walking. This wakes the general `Walk` function more useful, whether we just want to use perform the specific computation for its side effects (e.g. when we just print the contents of the file system), or when we actually need a specific kind of results. The trouble is that, just like evry function bounded by their argument types, function argumemts of higher-order functions too must have specified arguments and return values. This is really not a limitation, as a function ought to be specified. In languesges, such as go and rust, where there are interfaces and traits, it is possible to make parameters and return types more general. 
So, in essence, functional argumemts to functions are a means to make computation more general and expressive.You want to encapsulate some computation that is defined, and drop it in another computation, to become a part of its process. This functional argument might be a generator of values, a predicate, a function used for its side effects, or any other function for that matter.

##### Example: `fs.WalkDir(fsys FS, root string, fn WalkDirFunc) error`:
It takes an `fs.FS`, which is a file system interface, a root string to start from, and a `WalkDirFunc` function. The interesting part is the `WalkDirFunc` funcction argument. 

### Why make a function return another function?
We return functions from functions, IMO, because we wish to do some pre-processing or manipulate the arguments before actually returning a function that satisfies some specification.
I see it used in go to create middlewares. Here, you basically define a function of any signature you want, with the only constraint that you have to return the kinf of function that youre interested in modifying your function into. You then do some computation before returning the computation you want done as a function. You must realise that when your higher function si called, it executes all other parts of its body except the body of the function you returned. 
Thsi function that is returned can be assigned a value. It can be assigned severally. It does not need to know anyting about the pre-computation that bore it. It can make use of the state of the mother function. 
More importantly, in Parser combinators, we use higher order functions that return functions because they allow us to chain functions. The moment were able to specify a General type of function  (say our parser), this type can be the return type of any custom function we write. Chaining then becomes easy. We do not need to chain different function siginatures (or types). We only need to chain the specific function we specified generally. 

### What are parser combinators?
I first saw parser combinators when i was trying to write a TCP in rust. `nom` is a popular parser combinator library within the rust community.
Parser combinators basically take small functions and compose them gradually into groups, stages, and processes that can fully parse grammers. The reason its cool is that it that you can test the hec out of every sinle bit of it. And if you have a library, all you need to do is carefully compose the set of parsers and their combainators that you need to parse your grammar. 
The downside it has against some other types of parsing libraries and tools is that parsing is done at runtime, which  may nake things a little slower than when parsing is not done on the fly.

WE can also use it to manipulate the way a function is called. Imagine that what we really wnat is a function with a specific signature, we could have any function of any signature return it as a result. The higher order function that returns our needed function can take any form we want, providing our useful function values to work  with as we would wish. This trick is used in currying. There are functional languages which do not allow functions to take more than one parameter. In such a case if you need to provide two arguments, you must do some currying. The second argunent becomes the argument of the second function
e.g.
```
func multiply(a int) func (int) int {
    return func(b int) int {
        return a * b
    }
}

//now in a calling function, we can do the following:
a,b := 5,10
res := multiply(a)(b)
// our multiply function successfully stiks to the rule of not having more than one input parameter. But it can do a whole lot, we can keep currying for as long as we want.
//for fun, let's curry twice

func volume(l int) func(int) func(int) int {
    return func(b int) func (int) int {
        return func (h int) int {
            return l * b * h
        }
    }
}
// in the calling function, we can have:
l,b,h := 5,10,15
vol := volume(l)(b)(h)
```

### A set of examples using the ParserCombinator library I worte to parse bencode(torrent) files


Essentially, `Parsec` is a named type in the library. Its definition is simple:
```go
type Parsec func (in ParserInput) PResult
```
Where we define ParseInput as an interface type that possesses three methods, thus:
```go
type ParserInput interface {
    Car() rune //when it is called, it returns the current rune without advancing the index
    Cdr() ParserInput //returns the remainder of the input after the first one has been removed
    Empty() bool
}
```
`ParserInput` uses these three methods to track and update state. `Car()` is like its `lisp` counterpart. It returns the first rune in the stream. `Cdr()` returns what is left. While `Empty()` is obviously what you think it is. There's a limitation here: any input must be able to churn out `rune`s. Our library does not understand how to parse any other type of stream.  But that is okay for my use case, plus I'm not trying to develop a public library.

The result type returned by the parsers is:
```go
type PResult struct {
    result interface{}
    rem    ParserInput
    err error
}
```
It has three fields; here's the catch: `result`, the first field could be anything in `go`. That means we have to use type assertion to take out what stored inside the inteface value. This also means we must understand a little bit of the inner working of the library, at least wht kind of `result` each parser returns. Luckily, there are mostly just three types; either a `rune`, a `string`, or  a `list of runes`.
