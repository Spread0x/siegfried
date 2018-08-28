# Siegfried

[Siegfried](http://www.itforarchivists.com/siegfried) is a signature-based file format identification tool, implementing:

  - the National Archives UK's [PRONOM](http://www.nationalarchives.gov.uk/pronom) file format signatures
  - freedesktop.org's [MIME-info](https://freedesktop.org/wiki/Software/shared-mime-info/) file format signatures
  - the Library of Congress's [FDD](http://www.digitalpreservation.gov/formats/fdd/descriptions.shtml) file format signatures (*beta*).

### Version

1.7.9

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
### v1.7.8 (2017-12-02)
### Changed
- update LOC signatures to 2017-09-28
- update PRONOM signatures to v93

### v1.7.7 (2017-11-30)
### Added
- version information for MIME-info signatures (freedesktop.org and tika-mimetypes) now recorded in mime-info.json file and presented in results
- new sets file for PRONOM extensions. This creates sets like @.doc and @.txt (i.e. all PUIDs with those extensions). Allows you to do commands like `roy build -limit @.doc,@.docx`, `roy inspect @.txt` and `sf -log @.pdf,o DIR`

### Changed
- update freedesktop.org signatures to v1.9

### Fixed
- out of memory error when using `sf -z` on compressed files that contain very large files; reported by [Terry Jolliffe](https://github.com/richardlehane/siegfried/issues/109)
- report errors that occur during file decompression. Previously, only fatal errors encountered when a compressed file is first opened were reported. Now errors that are encountered while attempting to walk the contents of a compressed file are also reported. 
- report errors for 'roy inspect' when roy can't find anything to inspect; reported by [Ross Spencer](https://github.com/richardlehane/siegfried/issues/108)

### v1.7.6 (2017-10-04)
### Added
- continue on error flag (-coe) can now be used to continue scans despite fatal file errors that would normally cause scanning to halt. This may be useful e.g. for big directory scans over unreliable networks. Usage: `sf -coe DIR`.

### Changed
- update PRONOM signatures to v92

### Fixed
- file scanning is now restricted to regular files (i.e. not symlinks, sockets, devices etc.). Reported by [Henk Vanstappen](https://github.com/richardlehane/siegfried/issues/107).
- windows longpath fix now works for paths that appear short

### v1.7.5 (2017-08-12)
### Added
- `sf -update` flag can now be used to download/update non-PRONOM signatures. Options are "loc", "tika", "freedesktop", "pronom-tika-loc", "deluxe" and "archivematica". To update a non-PRONOM signature, include the signature name as an argument after the flags e.g. `sf -update freedesktop`. This command will overwrite 'default.sig' (the default signature file that sf loads). You can preserve your default signature file by providing an alternative `-sig` target e.g. `sf -sig notdefault.sig -update loc`. If you use one of the signature options as a filename (with or without a .sig extension), you can omit the signature argument i.e. `sf -update -sig loc.sig` is equivalent to `sf -sig loc.sig -update loc`. Feature requested by [Ross Spencer](https://github.com/richardlehane/siegfried/issues/103).
- `sf -update` now does SHA-256 hash verification of updates and communication with the update server is via HTTPS.

### Changed
- update PRONOM signatures to v91

### Fixed
- fixes to config package where global variables are polluted with subsquent calls to the Add(Identifier) function
- fix to reader package where panic triggered by illegal slice access in some cases

### v1.7.4 (2017-07-14)
### Added
- `roy build` and `roy add` now take a `-nobyte` flag to omit byte signatures from the identifier; requested by [Nick Krabbenhoeft](https://github.com/richardlehane/siegfried/issues/102) 

### Changed
- update Tika MIMEInfo signatures to 1.16
- update LOC to 2017-06-10

### v1.7.3 (2017-05-20)
### Added
- sf now accepts multiple files or directories as input e.g. `sf myfile1.doc mydir myfile3.txt`
- LOC signature update

### Changed
- code re-organisation to export reader and writer packages
- `sf -replay` can now take lists of results files with `-f` flag e.g. `sf -replay -f list-of-results.txt`

### Fixed
- the command `sf -replay -` now works on Windows as expected e.g. `sf myfiles | sf -replay -json -`
- text matcher not allocating hits to correct identifiers; fixes [#101](https://github.com/richardlehane/siegfried/issues/101)
- unescaped YAML field contains quote; reported by [Ross Spencer](https://github.com/richardlehane/siegfried/issues/100)

### v1.7.2 (2017-04-4)
### Added
- PRONOM v90 update

### Fixed
- the -home flag was being overriden for roy subcommands due to interaction other flags

### v1.7.1 (2017-03-12)
### Added
- signature updates for PRONOM, LOC and tika-mimetypes

### Changed
- `roy inspect` accepts space as well as comma-separated lists of formats e.g. `roy inspect fmt/1 fmt/2`

### v1.7.0 (2017-02-17)
### Added
- log files that match particular formats with `-log fmt/1,@set2` (comma separated list of format IDs/format sets). These can be mixed with regular log options e.g. `-log unknown,fmt/1,chart`
- generate a summary view of formats matched during a scan with `-log chart` (or just `-log c`)
- replay scans from results files with `sf -replay`: load one or more results files to replay logging or to convert to a different output format e.g. `sf -replay -csv results.yaml` or `sf -replay -log unknown,chart,stdout results1.yaml results2.csv`
- compare results with `roy compare` subcommand: view the difference between two or more results e.g. `roy compare results1.yaml results2.csv droid.csv ...`
- `roy sets` subcommand: `roy sets` creates pronom-all.json, pronom-families.json, and pronom-types.json sets files;
`roy sets -changes` creates a pronom-changes.json sets file from a PRONOM release-notes.xml file; `roy sets -list @set1,@set2` lists contents of a comma-separated list of format sets
- `roy inspect releases` provides a summary view of a PRONOM release-notes.xml file

### Changed
- the `sf -` command now scans stdin e.g. `cat mypdf.pdf | sf -`. You can pass a filename in to supplement the analysis with the `-name` flag e.g. `cat myfile.pdf | sf -name myfile.pdf -`. In previous versions of sf, the dash argument signified treating stdin as a newline separated list of filenames for scanning. Use the new `-f` flag for this e.g. `sf -f myfiles.txt` or `cat myfiles.txt | sf -f -`; change requested by [pm64](https://github.com/richardlehane/siegfried/issues/96)

### Fixed
- some files cause endless scanning due to large numbers of signature hits; reported by [workflowsguy](https://github.com/richardlehane/siegfried/issues/94)
- null bytes can be written to output due to bad zip filename decoding; reported by [Tim Walsh](https://github.com/richardlehane/siegfried/issues/95)

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
