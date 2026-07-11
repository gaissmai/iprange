# Codebase Summary: package iprange

This document serves as a persistent reference for future development sessions to understand the codebase structure, APIs, and rules without needing to re-analyze all files.

---

## 1. Overview & Purpose
`package iprange` is a Go library that extends the standard library's `net/netip` package. It introduces the `IPRange` type, which represents an inclusive range of IP addresses (IPv4 or IPv6) from the same address family. 
Unlike CIDR prefixes, `IPRange` supports arbitrary bounds (e.g., `10.0.0.3-10.0.17.134`), handling ranges and CIDRs transparently.

---

## 2. File Structure
*   **[iprange.go](file:///home/gaissmai/project/iprange/iprange.go)**: The main source file containing the `IPRange` struct definition, initialization functions, comparison operations, formatting, marshaling logic, and main algorithms (`Merge`, `Remove`).
*   **[iprange_test.go](file:///home/gaissmai/project/iprange/iprange_test.go)**: Test suite covering string parsing, marshaling, comparison, merging, removal, and edge cases.
*   **[example_test.go](file:///home/gaissmai/project/iprange/example_test.go)**: Executable examples for Go doc/documentation, showing how to parse, merge, and remove ranges.
*   **[go.mod](file:///home/gaissmai/project/iprange/go.mod)** / **[go.sum](file:///home/gaissmai/project/iprange/go.sum)**: Module definition. Includes a dependency on `github.com/gaissmai/extnetip` for helper utilities (e.g., iterator conversion, prefix extraction).
*   **[.greptile/rules.md](file:///home/gaissmai/project/iprange/.greptile/rules.md)**: AI assistant guardrails (English communication, ASCII-only comments/variables).

---

## 3. Core Types & Structure

```go
type IPRange struct {
	first netip.Addr
	last  netip.Addr
}
```

*   `first` and `last` are inclusive boundary IP addresses.
*   The zero-value is `zeroValue` (uninitialized/invalid range).
*   Valid ranges must have matching IP versions (both IPv4 or both IPv6), no zones, and `first <= last`.

---

## 4. Key APIs & Functions

### Initialization
*   `FromString(s string) (IPRange, error)`: Parses string representations of ranges. Valid formats include:
    *   CIDR Prefix: `192.168.0.0/24` (uses `netip.ParsePrefix` and `FromPrefix`)
    *   Explicit range: `10.0.0.3-10.0.17.134` (uses `strings.Cut` and `FromAddrs`)
    *   Single IP: `4.4.4.4` (maps to `4.4.4.4-4.4.4.4`)
*   `FromPrefix(p netip.Prefix) (IPRange, error)`: Converts a `netip.Prefix` to an `IPRange` using `extnetip.Range`.
*   `FromAddrs(first, last netip.Addr) (IPRange, error)`: Validates and returns an `IPRange` from boundary addresses.

### Methods & Utilities
*   `IsValid() bool`: Checks if the range is initialized.
*   `Addrs() (first, last netip.Addr)`: Returns the first and last IP address of the range.
*   `Prefix() (prefix netip.Prefix, ok bool)`: Converts the range back to a standard `netip.Prefix` if it aligns perfectly with a single CIDR block.
*   `String() string`: Stringifies the range. Formats as CIDR (e.g. `2001:db8::/32`) if possible, otherwise `first-last` (e.g. `127.0.0.1-127.0.0.19`).
*   `Prefixes() iter.Seq[netip.Prefix]`: Returns a standard Go iterator (`iter.Seq`) over all minimal `netip.Prefix` blocks required to fully cover the `IPRange`.

### Serialization
*   `MarshalText() ([]byte, error)` / `UnmarshalText(text []byte) error`: Implements `encoding.TextMarshaler` / `encoding.TextUnmarshaler` using string representations.
*   `MarshalBinary() ([]byte, error)` / `UnmarshalBinary(data []byte) error`: Implements binary marshaling by writing the raw bytes of `first` and `last` consecutively (8 bytes for IPv4, 32 bytes for IPv6).

---

## 5. Main Algorithms

### Merge Algorithm
*   **Function**: `Merge(in []IPRange) (out []IPRange)`
*   **Logic**:
    1.  Copies and sorts the inputs using `cmpRange` (primary sort by `first` address ascending; secondary tiebreaker sorts larger supersets to the left).
    2.  Iterates through sorted ranges, merging adjacent ranges (`topic.last.Next() == r.first`), overlapping ranges, or subset ranges (`topic.covers(r)`) in a single pass.

### Remove Algorithm
*   **Function**: `(r IPRange) Remove(in []IPRange) (out []IPRange)`
*   **Logic**:
    1.  Merges the exclusion ranges `in`.
    2.  Iterates through merged exclusions. If an exclusion overlaps `r`, it slices/shrinks the current range `r`.
    3.  Appends non-overlapping segments to the output, updating the cursor (`r.first`) as it goes.

### Comparison Interface
*   **Function**: `Compare(a, b IPRange) (ll, rr, lr, rl int)`
*   **Usage**: Designed to implement `cmp` for [github.com/gaissmai/interval](https://github.com/gaissmai/interval) (an interval tree package) to enable fast tree-based range lookups. Compares all four boundary endpoints.

---

## 6. Assistant Guidelines for this Codebase
When writing code or modifications here, always remember:
1.  **Language**: Always communicate in English.
2.  **Identifiers & Code**: Code comments, documentation, and all code identifiers (variables, functions, structs, fields) must be written in English.
3.  **Strict ASCII**: Do not use emojis, typographic quotes, or any non-ASCII Unicode characters in source files.
