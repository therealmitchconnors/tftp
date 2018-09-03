In-memory TFTP Server
=====================

[![License Apache 2](https://img.shields.io/badge/License-Apache2-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ftherealmitchconnors%2Ftftp.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Ftherealmitchconnors%2Ftftp?ref=badge_shield)
[![Go Report Card](https://goreportcard.com/badge/github.com/therealmitchconnors/tftp)](https://goreportcard.com/report/github.com/therealmitchconnors/tftp) [![Build Status](https://travis-ci.com/therealmitchconnors/tftp.svg?branch=master)](http://travis-ci.com/therealmitchconnors/tftp) [![GoDoc](https://godoc.org/github.com/therealmitchconnors/tftp?status.svg)](http://godoc.org/github.com/therealmitchconnors/tftp) [![Coverage Status](https://coveralls.io/repos/therealmitchconnors/tftp/badge.svg?branch=master)](https://coveralls.io/r/therealmitchconnors/tftp?branch=master)

This is a simple in-memory TFTP server, implemented in Go as a proof of concept.  It is RFC1350-compliant, but only supports "octet" mode, and doesn't implement the additions in later RFCs.  In particular, options are not recognized.

Installation
------------

go get github.com/therealmitchconnors/tftp

Usage
-----
tftpd [options]

  -max-packet-size value
        The max transmission unit for UDP reads.  Larger packets will truncate, smaller values are more efficient. (default 2048)
  -oplog string
        The destination for operation logs (default "./operations.log")
  -port value
        The port tftpd will listen on (default 69)

Testing
-------
**Unit Tests**

go test -coverprofile=test.out -timeout 30s github.com/therealmitchconnors/tftp

**Functional Tests**

Ideally, this will get incorporated into a dockerfile in the future
1. Install tftp (ie `yum install tftp`)
2. Build tftpd
3. Run tftpd
4. **In another shell** Create test file some.txt
5. `tftp -m octet -v localhost 69 -c put some.txt`
6. `mv some.txt expected.txt`
7. `tftp -m octet -v localhost 69 -c get some.txt`
8. `cmp some.txt expected.txt`

Building
--------

go build github.com/therealmitchconnors/tftp

tftp has no runtime dependencies outside the universe block.  Test does have a dependency on github.com/jordwest/mock-conn, to avoid opening actual UDP ports in a unit test sandbox.  


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ftherealmitchconnors%2Ftftp.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Ftherealmitchconnors%2Ftftp?ref=badge_large)
