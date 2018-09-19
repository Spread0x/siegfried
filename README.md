# Siegfried

[Siegfried](http://www.itforarchivists.com/siegfried) is a signature-based file format identification tool, implementing:

  - the National Archives UK's [PRONOM](http://www.nationalarchives.gov.uk/pronom) file format signatures
  - freedesktop.org's [MIME-info](https://freedesktop.org/wiki/Software/shared-mime-info/) file format signatures
  - the Library of Congress's [FDD](http://www.digitalpreservation.gov/formats/fdd/descriptions.shtml) file format signatures (*beta*).

### Version

1.7.10

[![Build Status](https://travis-ci.org/richardlehane/siegfried.png?branch=master)](https://travis-ci.org/richardlehane/siegfried) [![GoDoc](https://godoc.org/github.com/richardlehane/siegfried?status.svg)](https://godoc.org/github.com/richardlehane/siegfried) [![Go Report Card](https://goreportcard.com/badge/github.com/richardlehane/siegfried)](https://goreportcard.com/report/github.com/richardlehane/siegfried)

## Usage

### Command line

    sf file.ext
    sf DIR

#### Options

    sf -csv file.ext | DIR                     // Output CSV rather than YAML
    sf -json file.ext | DIR                    // Output JSON rather than YAML
    sf -droid file.ext | DIR                   // Output DROID CSV rather than YAML
    sf -nr DIR                                 // Don't scan subdirectories
    sf -z file.zip | DIR                       // Decompress and scan zip, tar, gzip, warc, arc
    sf -hash md5 file.ext | DIR                // Calculate md5, sha1, sha256, sha512, or crc hash
    sf -sig custom.sig file.ext                // Use a custom signature file
    sf -                                       // Scan stream piped to stdin
    sf -name file.ext -                        // Provide filename when scanning stream 
    sf -f myfiles.txt                          // Scan list of files
    sf -version                                // Display version information
    sf -home c:\junk -sig custom.sig file.ext  // Use a custom home directory
    sf -serve hostname:port                    // Server mode
    sf -throttle 10ms DIR                      // Pause for duration (e.g. 1s) between file scans
    sf -multi 256 DIR                          // Scan multiple (e.g. 256) files in parallel 
    sf -log [comma-sep opts] file.ext | DIR    // Log errors etc. to stderr (default) or stdout
    sf -log e,w file.ext | DIR                 // Log errors and warnings to stderr
    sf -log u,o file.ext | DIR                 // Log unknowns to stdout
    sf -log d,s file.ext | DIR                 // Log debugging and slow messages to stderr
    sf -log p,t DIR > results.yaml             // Log progress and time while redirecting results
    sf -log fmt/1,c DIR > results.yaml         // Log instances of fmt/1 and chart results
    sf -replay -log u -csv results.yaml        // Replay results file, convert to csv, log unknowns
    sf -setconf -multi 32 -hash sha1           // Save flag defaults in a config file
    sf -setconf -serve :5138 -conf srv.conf    // Save/load named config file with '-conf filename' 

#### Example

[![asciicast](https://asciinema.org/a/ernm49loq5ofuj48ywlvg7xq6.png)](https://asciinema.org/a/ernm49loq5ofuj48ywlvg7xq6)

#### Signature files

By default, siegfried uses the latest PRONOM signatures without buffer limits (i.e. it may do full file scans). To use MIME-info or LOC signatures, or to add buffer limits or other customisations, use the [roy tool](https://github.com/richardlehane/siegfried/wiki/Building-a-signature-file-with-ROY) to build your own signature file.

## Install
### With go installed: 

    go get github.com/richardlehane/siegfried/cmd/sf

    sf -update


### Or, without go installed:
#### Win:

Download a pre-built binary from the [releases page](https://github.com/richardlehane/siegfried/releases). Unzip to a location in your system path. Then run:

    sf -update

#### Mac Homebrew (or [Linuxbrew](http://brew.sh/linuxbrew/)):

    brew install mistydemeo/digipres/siegfried

Or, for the most recent updates, you can install from this fork:

    brew install richardlehane/digipres/siegfried

#### Ubuntu/Debian (64 bit):

    wget -qO - https://bintray.com/user/downloadSubjectPublicKey?username=bintray | sudo apt-key add -
    echo "deb http://dl.bintray.com/siegfried/debian wheezy main" | sudo tee -a /etc/apt/sources.list
    sudo apt-get update && sudo apt-get install siegfried

#### FreeBSD:

    pkg install siegfried

#### Arch Linux: 

    git clone https://aur.archlinux.org/siegfried.git
    cd siegfried
    makepkg -si

## Changes
### v1.7.10 (2018-09-19)
### Added
- print configuration defaults with `sf -version`

### Changed
- update PRONOM to v94

### Fixed
- LOC identifier fixed after regression in v1.7.9
- remove skeleton-suite files triggering malware warnings by adding to .gitignore; reported by [Dave Rice](https://github.com/richardlehane/siegfried/issues/118)
- release built with Go version 11, which includes a fix for a CIFS error that caused files to be skipped during file walk; reported by [Maarten Savels](https://github.com/richardlehane/siegfried/issues/115)

### v1.7.9 (2018-08-30)
### Added
- save defaults in a configuration file: use the -setconf flag to record any other flags used into a config file. These defaults will be loaded each time you run sf. E.g. `sf -multi 16 -setconf` then `sf DIR` (loads the new multi default)
- use `-conf filename` to save or load from a named config file. E.g. `sf -multi 16 -serve :5138 -conf srv.conf -setconf` and then `sf -conf srv.conf` 
- added `-yaml` flag so, if you set json/csv in default config :(, you can override with YAML instead. Choose the YAML!

### Changed
- the `roy compare -join` options that join on filepath now work better when comparing results with mixed windows and unix paths
- exported decompress package to give more functionality for users of the golang API; requested by [Byron Ruth](https://github.com/richardlehane/siegfried/issues/119)
- update LOC signatures to 2018-06-14
- update freedesktop.org signatures to v1.10
- update tika-mimetype signatures to v1.18

### Fixed
- misidentifications of some files e.g. ODF presentation due to sf quitting early on strong matches. Have adjusted this algorithm to make sf wait longer if there is evidence (e.g. from filename) that the file might be something else. Reported by [Jean-Séverin Lair](https://github.com/richardlehane/siegfried/issues/112)
- read and other file errors caused sf to hang; reports by [Greg Lepore and Andy Foster](https://github.com/richardlehane/siegfried/issues/113); fix contributed by [Ross Spencer](https://github.com/richardlehane/siegfried/commit/ea5300d3639d741a451522958e8b99912f7d639d)
- bug reading streams where EOF returned for reads exactly adjacent the end of file
- bug in mscfb library ([race condition for concurrent access to a global variable](https://github.com/richardlehane/siegfried/issues/117))
- some matches result in extremely verbose basis fields; reported by [Nick Krabbenhoeft](https://github.com/richardlehane/siegfried/issues/111). Partly fixed: basis field now reports a single basis for a match but work remains to speed up matching for these cases.

See the [CHANGELOG](CHANGELOG.md) for the full history.

## Rights

Copyright 2017 Richard Lehane 

Licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)

## Announcements

Join the [Google Group](https://groups.google.com/d/forum/sf-roy) for updates, signature releases, and help.

## Contributing

Like siegfried and want to get involved in its development? That'd be wonderful! There are some notes on the [wiki](https://github.com/richardlehane/siegfried/wiki) to get you started, and please get in touch.

## Thanks

Thanks TNA for http://www.nationalarchives.gov.uk/pronom/ and http://www.nationalarchives.gov.uk/information-management/projects-and-work/droid.htm

Thanks Ross for https://github.com/exponential-decay/skeleton-test-suite-generator and http://exponentialdecay.co.uk/sd/index.htm, both are very handy!

Thanks Misty for the brew and ubuntu packaging

Thanks Steffen for the FreeBSD and Arch Linux packaging
