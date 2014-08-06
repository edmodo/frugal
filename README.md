frugal
======

A reimplementation of thrift written in Go.

Not supported yet:
 - byte type
 - cpp\_type
 - field attribute strings

Changes over thrift:
 - Lack of order fields is a type error (but not a parsing error).
 - Required fields in structs must be initialized in literals.
 - Optional fields with default values are checked.
