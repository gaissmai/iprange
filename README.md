# package iprange
[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/iprange.svg)](https://pkg.go.dev/github.com/gaissmai/iprange#section-documentation)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/iprange)
[![CI](https://github.com/gaissmai/iprange/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/iprange/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/iprange/badge.svg?branch=master)](https://coveralls.io/github/gaissmai/iprange?branch=master)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


`package iprange` is an extension to net/netip

An additional type IPRange is defined and the most useful methods for it. Not all IP address ranges in the wild are CIDRs,
very often you have to deal with ranges not representable as a prefix. This library handels IP ranges and CIDRs transparently. 

## API

```go
import "github.com/gaissmai/iprange"

type IPRange struct{ ... }

  func FromString(s string) (IPRange, error)
  func FromAddrs(first, last netip.Addr) (IPRange, error)
  func FromPrefix(p netip.Prefix) (IPRange, error)

  func (r IPRange) Addrs() (first, last netip.Addr)
  func (r IPRange) String() string
  func (r IPRange) IsValid() bool

  func Merge(in []IPRange) (out []IPRange)
  func (r IPRange) Remove(in []IPRange) (out []IPRange)

  func (r IPRange) Prefix() (prefix netip.Prefix, ok bool)
  func (r IPRange) Prefixes() []netip.Prefix
  func (r IPRange) PrefixesAppend(dst []netip.Prefix) []netip.Prefix

  func (r IPRange) MarshalBinary() ([]byte, error)
  func (r IPRange) MarshalText() ([]byte, error)

  func (r *IPRange) UnmarshalBinary(data []byte) error
  func (r *IPRange) UnmarshalText(text []byte) error
```

## Advanced features
For more advanced functionality IPRange implements the `interval.Interface` for fast lookups.

see also: https://github.com/gaissmai/interval
