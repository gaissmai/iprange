# package iprange
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/iprange.svg)](https://pkg.go.dev/github.com/gaissmai/iprange#section-documentation)
[![CI](https://github.com/gaissmai/iprange/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/iprange/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/iprange/badge.svg?branch=master)](https://coveralls.io/github/gaissmai/iprange?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gaissmai/iprange)](https://goreportcard.com/report/github.com/gaissmai/iprange)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


`package iprange` is an extension to net/netip

An additional type IPRange is defined and the most useful methods for it. Not all IP address ranges in the wild are CIDRs,
very often you have to deal with ranges not representable as a prefix. This library handels IP ranges and CIDRs transparently. 

## API

```go
import "github.com/gaissmai/iprange"

type IPRange

func Parse(s string) (IPRange, error)
func FromNetipAddrs(first, last netip.Addr) (IPRange, error)
func FromNetipPrefix(p netip.Prefix) (IPRange error)

func (r IPRange) String() string
func (r IPRange) Addrs() (first, last netip.Addr)

func Merge(in []IPRange) []IPRange
func (r IPRange) Remove(in []IPRange) []IPRange

func (r IPRange) Prefix() (prefix netip.Prefix, ok bool)
func (r IPRange) Prefixes() []netip.Prefix
func (r IPRange) PrefixesAppend(dst []netip.Prefix) []netip.Prefix

func (r IPRange) CompareLower(r2 IPRange) int
func (r IPRange) CompareUpper(r2 IPRange) int

func (r IPRange) MarshalText() ([]byte, error)
func (r IPRange) MarshalBinary() ([]byte, error)

func (r *IPRange) UnmarshalText(text []byte) error
func (r *IPRange) UnmarshalBinary(data []byte) error
```

## Advanced features
For more advanced functionality IPRange implements the `interval.Interface` for fast lookups.

see also: https://github.com/gaissmai/interval
