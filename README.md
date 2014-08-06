frugal
======

A reimplementation of thrift written in Go. Currently, this is just a parsing and semantic analysis library for custom generators and tools.

Packages:
 - `parser` - The parsing library.
 - `sema` - The semantic analysis library.

Not supported yet:
 - byte type
 - cpp\_type
 - field attribute strings

Changes over thrift:
 - Lack of order fields is a type error (but not a parsing error).
 - Required fields in structs must be initialized in literals.
 - Optional fields with default values are checked.
