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

## ATTENTION: API change

`Prefixes` now returns an iterator and `PrefixesAppend` is removed.
Also `CompareLower` and `CompareUpper` are removed, use `Compare` instead.

## API

```go
package iprange // import "github.com/gaissmai/iprange"

type IPRange struct {
	// Has unexported fields.
}
    IPRange represents an inclusive range of IP addresses from the same address
    family.

        10.0.0.3-10.0.17.134        // range
        2001:db8::1-2001:db8::f6    // range
        192.168.0.1/24              // Prefix aka CIDR
        ::1/128                     // Prefix aka CIDR

    Not all IP address ranges in the wild are CIDRs, very often you have to deal
    with ranges not representable as a prefix.

    This library handles IP ranges and CIDRs transparently.

func FromAddrs(first, last netip.Addr) (IPRange, error)
func FromPrefix(p netip.Prefix) (IPRange, error)
func FromString(s string) (IPRange, error)

func Merge(in []IPRange) (out []IPRange)

func (r IPRange) IsValid() bool
func (r IPRange) Addrs() (first, last netip.Addr)
func (r IPRange) Prefix() (prefix netip.Prefix, ok bool)
func (r IPRange) Prefixes() iter.Seq[netip.Prefix]

func (r IPRange) Remove(in []IPRange) (out []IPRange)

func (r IPRange)  String() string
func (r IPRange)  MarshalBinary() ([]byte, error)
func (r IPRange)  MarshalText() ([]byte, error)
func (r *IPRange) UnmarshalBinary(data []byte) error
func (r *IPRange) UnmarshalText(text []byte) error

func Compare(a, b IPRange) (ll int, rr int, lr int, rl int)
```

## Advanced features
For fast lookups use the `Compare` function together with the [interval package] from the same author.

[interval package]: https://github.com/gaissmai/interval
