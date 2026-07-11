# package iprange

[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/iprange.svg)](https://pkg.go.dev/github.com/gaissmai/iprange#section-documentation)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/iprange)](https://github.com/gaissmai/iprange/releases)
[![CI](https://github.com/gaissmai/iprange/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/iprange/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/iprange/badge.svg?branch=master)](https://coveralls.io/github/gaissmai/iprange?branch=master)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`package iprange` is a lightweight, high-performance Go extension to the standard library's `net/netip`.

It introduces the `IPRange` type to represent inclusive ranges of IP addresses (both IPv4 and IPv6) of the same family. Not all IP address ranges in the wild are CIDRs—very often you have to deal with arbitrary ranges not representable as a single prefix (e.g. `10.0.0.3-10.0.17.134`). This library handles arbitrary ranges and CIDRs transparently and efficiently.

---

## Features
- **Flexible parsing**: Parse standard CIDR notation, explicit hyphenated ranges, or single IPs.
- **Merge operations**: Efficiently combine adjacent, overlapping, or subset IP ranges.
- **Subtraction**: Exclude lists of IP ranges from a target range.
- **Prefix Decomposition**: Split arbitrary IP ranges into the minimal set of standard CIDR prefixes.
- **Fast Lookups**: Integrate with interval tree structures via a custom `Compare` function.
- **Zero Allocations & Value Semantics**: Designed to stay on the stack with clean value semantics.

---

## Installation

```bash
go get github.com/gaissmai/iprange
```

---

## Quick Start

### 1. Parsing Ranges
```go
package main

import (
	"fmt"
	"github.com/gaissmai/iprange"
)

func main() {
	// Parse a CIDR prefix
	r1, _ := iprange.FromString("192.168.1.0/24")
	fmt.Println(r1) // Output: 192.168.1.0/24

	// Parse an arbitrary hyphenated range (IPv6)
	r2, _ := iprange.FromString("2001:db8::1-2001:db8::f6")
	fmt.Println(r2) // Output: 2001:db8::1-2001:db8::f6
}
```

### 2. Merging Overlapping & Adjacent Ranges
```go
package main

import (
	"fmt"
	"github.com/gaissmai/iprange"
)

func main() {
	mustParse := func(s string) iprange.IPRange {
		r, err := iprange.FromString(s)
		if err != nil {
			panic(err)
		}
		return r
	}

	ranges := []iprange.IPRange{
		mustParse("10.0.0.1/32"),
		mustParse("10.0.0.2/32"),
		mustParse("10.0.0.5/32"),
		mustParse("10.0.0.3-10.0.0.4"),
	}

	merged := iprange.Merge(ranges)
	fmt.Println(merged) // Output: [10.0.0.1-10.0.0.5]
}
```

### 3. Subtracting Ranges (IPv6)
```go
package main

import (
	"fmt"
	"github.com/gaissmai/iprange"
)

func main() {
	mustParse := func(s string) iprange.IPRange {
		r, err := iprange.FromString(s)
		if err != nil {
			panic(err)
		}
		return r
	}

	outer, _ := iprange.FromString("2001:db8:de00::/40")
	exclusions := []iprange.IPRange{
		mustParse("2001:db8:dea0::/44"),
	}

	remaining := outer.Remove(exclusions)
	for _, r := range remaining {
		fmt.Println(r)
	}
}
// Output:
// 2001:db8:de00::-2001:db8:de9f:ffff:ffff:ffff:ffff:ffff
// 2001:db8:deb0::-2001:db8:deff:ffff:ffff:ffff:ffff:ffff
```

### 4. Splitting Ranges into Minimal Prefixes (IPv6)
```go
package main

import (
	"fmt"
	"github.com/gaissmai/iprange"
)

func main() {
	r, _ := iprange.FromString("2001:db8::affe-2001:db8::b003")
	for prefix := range r.Prefixes() {
		fmt.Println(prefix)
	}
}
// Output:
// 2001:db8::affe/127
// 2001:db8::b000/126
```

---

## API Summary

```go
type IPRange struct { /* unexported fields */ }

// Constructors
func FromString(s string) (IPRange, error)
func FromPrefix(p netip.Prefix) (IPRange, error)
func FromAddrs(first, last netip.Addr) (IPRange, error)

// Core Operations
func Merge(in []IPRange) (out []IPRange)
func (r IPRange) Remove(in []IPRange) (out []IPRange)

// Inspection & Conversion
func (r IPRange) IsValid() bool
func (r IPRange) Addrs() (first, last netip.Addr)
func (r IPRange) Prefix() (prefix netip.Prefix, ok bool)
func (r IPRange) Prefixes() iter.Seq[netip.Prefix]
func (r IPRange) String() string

// Endpoints Comparison
func Compare(a, b IPRange) (ll, rr, lr, rl int)
```

---

## Advanced Feature: Fast Lookups

For high-performance range lookups (e.g. routing tables, ACL matching), the `Compare` function implements the comparison signature required by the author's [interval tree package](https://github.com/gaissmai/interval).

```go
// Compare returns four integers comparing the boundary endpoints of two IP ranges.
func Compare(a, b IPRange) (ll, rr, lr, rl int)
```

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
