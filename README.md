frugal
======

A reimplementation of thrift written in Go. Currently, this is just a parsing and semantic analysis library for custom generators and tools.

Packages:
 - `parser` - The parsing library.
 - `sema` - The semantic analysis library.
 - `gen` - A helper library for writing generators.
 - `lib/frugal` - API extensions to Thrift's Go API.

Unimplemented Features
----------------------
These features are not yet implemented yet.
 - `byte` type
 - `binary` type
 - `double` literals (the type is supported)
 - `set` container type
 - `cpp_type` specifier
 - `cpp_include` directive
 - field attribute strings (called "XsdFieldOptions" in thrift IDL)
 - `senum` and `slist` types (these are deprecated in thrift).
 - `union` types

Changes from Apache Thrift
--------------------------
 - Although parsing will accept fields and method arguments without explicit ordering ("field keys"), semantic analysis will report an error for anything not explicitly ordered.
 - When creating a constant value with a struct type, if the struct has required fields, those fields must be assigned in the initializer.
 - When assigning default values to optional struct fields, frugal will type-check and evaluate those fields (whereas Apache thrift does not).
 - Unused includes are an error.
