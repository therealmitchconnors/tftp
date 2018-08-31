In-memory TFTP Server
=====================

This is a simple in-memory TFTP server, implemented in Go as a proof of concept.  It is RFC1350-compliant, but only supports "octet" mode, and doesn't implement the additions in later RFCs.  In particular, options are not recognized.

Installation
------------

go get github.com/therealmitchconnors/tftp

Usage
-----
TBD

Testing
-------
**Unit Tests**

go test -coverprofile=test.out -timeout 30s github.com/therealmitchconnors/tftp

**Functional Tests**

TBD

Building
--------

go build github.com/therealmitchconnors/tftp

tftp has no runtime dependencies outside the universe block.  Test does have a dependency on github.com/jordwest/mock-conn, to avoid opening actual UDP ports in a unit test sandbox.  
