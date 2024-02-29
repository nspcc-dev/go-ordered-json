.NET-compatible JSON library
============================

It's made for 100% compatibility with the JSON variation used by
[Neo blockchain](https://github.com/neo-project/). There are three problems
there:
 * it's ordered (that's why it's a fork of [go-ordered-json](https://github.com/virtuald/go-ordered-json))
 * it has different conventions regarding control and "special" symbols
 * it has different conventions wrt incorrect UTF-8

The primary user of this library is [NeoGo](https://github.com/nspcc-dev/neo-go/),
it has to be 100% compatible with C# implementation to correctly process
transactions, that's why we're maintaining this library and solving any
inconsistencies with .NET libraries if found.

**If you can, you should avoid using this package**. However, if you can't
avoid it, then you are welcome to. Provided under the MIT license, just like
golang.
