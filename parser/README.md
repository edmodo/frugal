frugal/parser
=============

The raw parsing API for frugal. The API entry point is CompileContext. From there, you can either call ParseRecursive() to parse a file and all included files, or create a parser via NewParser() to parse a single file.

The parser does not perform any semantic analysis. To do that, use the frugal/sema package.
