# Siegfried

[Siegfried](http://www.itforarchivists.com/siegfried) is a signature-based file format identification tool.

Key features are:

  - complete implementation of [PRONOM](http://apps.nationalarchives.gov.uk/pronom) (byte and container signatures)
  - fast matching without limiting the number of bytes scanned
  - detailed information about the basis for format matches
  - simple command line interface with a choice of outputs
  - a [built-in server](https://github.com/richardlehane/siegfried/wiki/Using-the-siegfried-server) for integrating with workflows and language inter-op
  - power options including [debug mode](https://github.com/richardlehane/siegfried/wiki/Inspect-and-Debug), [signature modification](https://github.com/richardlehane/siegfried/wiki/Building-a-signature-file-with-ROY), and [multiple identifiers](https://github.com/richardlehane/siegfried/wiki/Building-a-signature-file-with-ROY#one-signature-file-multiple-identifiers)

## Version

1.1.0

[![Build Status](https://travis-ci.org/richardlehane/siegfried.png?branch=master)](https://travis-ci.org/richardlehane/siegfried) [![GoDoc](https://godoc.org/github.com/richardlehane/siegfried/pkg/core?status.svg)](https://godoc.org/github.com/richardlehane/siegfried/pkg/core)

## Usage

### Command line

    sf file.ext
    sf DIR

#### Options

    sf -csv file.ext | DIR                     // Output CSV rather than YAML
    sf -json file.ext | DIR                    // Output JSON rather than YAML
    sf -                                       // Read list of files piped to stdin
    sf -nr DIR                                 // Don't scan subdirectories
    sf -z file.zip | DIR                       // Decompress and scan zip, tar, gzip
    sf -sig custom.sig file.ext                // Use a custom signature file
    sf -home c:\junk -sig custom.sig file.ext  // Use a custom home directory
    sf -debug file.ext                         // Scan in debug mode
    sf -version                                // Display version information
    sf -serve hostname:port                    // Server mode


![Usage](usage.gif)

By default, siegfried uses the latest PRONOM and container signatures with no buffer limits. You can customise your signature file by using the [roy tool](https://github.com/richardlehane/siegfried/wiki/Building-a-signature-file-with-ROY).

## Install

### With go installed: 

    go get github.com/richardlehane/siegfried/cmd/sf

    sf -update


### Or, without go installed:

For OS X:

    brew install mistydemeo/digipres/siegfried

For Ubuntu/Debian:

    wget -qO - https://bintray.com/user/downloadSubjectPublicKey?username=bintray | sudo apt-key add -
    echo "deb http://dl.bintray.com/siegfried/debian wheezy main" | sudo tee -a /etc/apt/sources.list
    sudo apt-get update && sudo apt-get install siegfried

For Win:

Download a pre-built binary from the [releases page](https://github.com/richardlehane/siegfried/releases). Unzip to a location in your system path. Then run:

	sf -update

## Recent Changes
### Version 1.1.0 (17/5/2015)
- scan within archive formats (zip, tar, gzip) with -z flag
- format sets (e.g. roy build -exclude @pdfa)
- leaner, faster signature format
- support bitmask patterns
- mirror bof patterns as eof patterns where both roy -bof and -eof limits set
- 'sf -' reads files piped to stdin
- bugfix: ([mscfb](https://github.com/richardlehane/mscfb/commit/22552265cefc80b400ff64156155f53a5d5751e6)) reported by [Pascal Aantz](https://github.com/richardlehane/siegfried/issues/32)
- bufix: race condition in scorer (affected tip golang)
- archivematica build: fpr server

### Version 1.0.0 (22/3/2015)
- [user documentation](http://github.com/richardlehane/siegfried/wiki)
- bugfixes (mscfb, match/wac and sf)
- QA using [comparator](http://github.com/richardlehane/comparator)

### Version 0.8.3 (24/2/2015)
- json string escaping

### Version 0.8.2 (22/2/2015)
- json output
- server mode

### Version 0.8.1 (1/2/2015)
- bugfix: single quote YAML output

### Version 0.8.0 (26/1/2015)
- optimisations (mmap, multithread, etc.)

[Full change history](https://github.com/richardlehane/siegfried/wiki/Change-history)

## Rights

Copyright 2015 Richard Lehane 

Licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)

## Contributing

Like Siegfried and want to get involved in its development? That'd be wonderful! There are some notes on the [wiki](https://github.com/richardlehane/siegfried/wiki) to get you started, and please get in touch.

## Thanks

Thanks TNA for http://www.nationalarchives.gov.uk/pronom/ and http://www.nationalarchives.gov.uk/information-management/projects-and-work/droid.htm

Thanks Ross for https://github.com/exponential-decay/skeleton-test-suite-generator and http://exponentialdecay.co.uk/sd/index.htm, both are very handy!

Thanks Misty for the brew and ubuntu packaging
