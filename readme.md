# Distra
An application to make it easier to build Go executables of multiple platforms for use in GitHub releases or similar

## Installation

You can install Distra with either `go install github.com/voidwyrm-2/distra@latest` or from the [releases](<https://github.com/voidwyrm-2/distra/releases/latest>)

## Usage

Distra requires [Go](<https://go.dev>) to be installed to be used

> **Note:** on Windows, you have to use Subsystem Linux

**Flags**<br>
* `listos`: lists all available operating systems that can be built for
    > example: `distra --listos`
* `listarch`: lists all available architectures that can be built with for the given operating systems
    > example: `distra --listarch linux --listarch windows`
* OS flags: auto-generated flags which correspond to each operating systems that can be built for
    > example: `distra --windows amd64 --windows arm --linux arm --js wasm`
