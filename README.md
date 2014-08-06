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
 - Although parsing will accept fields and method arguments without explicit ordering ("field keys"), semantic analysis will report an error for anything not explicitly ordered.
 - When creating a constant value with a struct type, if the struct has required fields, those fields must be assigned in the initializer.
 - When assigning default values to optional struct fields, frugal will type-check and evaluate those fields (whereas Apache thrift does not).
