// Package iprange is an extension to net/netip.
//
// It defines the IPRange type, representing inclusive IP address ranges,
// and provides utility methods for manipulation, comparison, and merging.
//
// For advanced lookup functionality, IPRange is designed to integrate
// with the interval tree package at https://github.com/gaissmai/interval.
package iprange

import (
	"errors"
	"fmt"
	"iter"
	"net/netip"
	"sort"
	"strings"

	"github.com/gaissmai/extnetip"
)

// IPRange represents an inclusive range of IP addresses from the same address family.
//
// Examples of valid ranges:
//
//	10.0.0.3-10.0.17.134        // Arbitrary range
//	2001:db8::1-2001:db8::f6    // IPv6 range
//	192.168.0.1/24              // CIDR prefix
//	::1/128                     // Host prefix
//
// Unlike standard CIDR prefixes, IPRange handles arbitrary IP bounds transparently.
type IPRange struct {
	first netip.Addr
	last  netip.Addr
}

var zeroValue IPRange

// FromString parses the input string s and returns an IPRange.
// It returns an error if the input format is invalid.
//
// Valid input formats:
//   - CIDR Prefix: "192.168.0.0/24", "2001:db8::/32"
//   - Explicit Range: "192.168.2.3-192.168.7.255"
//   - Single IP address: "4.4.4.4", "::0" (converted to /32 or /128 single-host ranges)
func FromString(s string) (IPRange, error) {
	if s == "" {
		return zeroValue, errors.New("empty string")
	}

	// Parse as a CIDR prefix if a slash is present.
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return zeroValue, err
		}
		return FromPrefix(p)
	}

	// Parse as a hyphen-separated explicit address range.
	ip, ip2, found := strings.Cut(s, "-")
	if found {
		first, err := netip.ParseAddr(ip)
		if err != nil {
			return zeroValue, err
		}

		last, err := netip.ParseAddr(ip2)
		if err != nil {
			return zeroValue, err
		}

		return FromAddrs(first, last)
	}

	// Parse as a single IP address.
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return zeroValue, err
	}
	return FromAddrs(addr, addr)
}

// FromPrefix returns an IPRange representation of the provided netip.Prefix.
// It returns an error if the prefix is invalid.
func FromPrefix(p netip.Prefix) (IPRange, error) {
	if !p.IsValid() {
		return zeroValue, errors.New("netip.Prefix is invalid")
	}
	first, last := extnetip.Range(p)
	return IPRange{first, last}, nil
}

// FromAddrs returns an IPRange from the provided first and last IP addresses.
// Both addresses must be of the same family (both IPv4 or both IPv6),
// must not contain zones, and last must not be less than first.
// Otherwise, it returns an error.
func FromAddrs(first, last netip.Addr) (IPRange, error) {
	//nolint:staticcheck // De Morgan conversion reduces readability here
	if !((first.Is4() && last.Is4()) || (first.Is6() && last.Is6())) {
		return zeroValue, errors.New("invalid or different IP versions")
	}
	if first.Zone() != "" || last.Zone() != "" {
		return zeroValue, errors.New("ip address MUST NOT have a zone")
	}
	if last.Less(first) {
		return zeroValue, errors.New("last address is less than first address")
	}

	return IPRange{first, last}, nil
}

// IsValid reports whether r is a valid, initialized IPRange.
func (r IPRange) IsValid() bool {
	return r != zeroValue
}

// Addrs returns the inclusive boundary IP addresses (first and last) of the IPRange.
func (r IPRange) Addrs() (first, last netip.Addr) {
	return r.first, r.last
}

// Prefix returns r as a netip.Prefix if it can be represented exactly as a single CIDR block.
// If r is invalid or cannot be represented by a single prefix, it returns a zero netip.Prefix and false.
func (r IPRange) Prefix() (prefix netip.Prefix, ok bool) {
	return extnetip.Prefix(r.first, r.last)
}

// String returns the string representation of the IPRange.
// If the range aligns perfectly with a single CIDR prefix, it returns its CIDR notation.
// Otherwise, it returns the range formatted as "first-last".
// If the range is invalid, it returns "invalid IPRange".
func (r IPRange) String() string {
	if r == zeroValue {
		return "invalid IPRange"
	}

	pfx, ok := r.Prefix()
	if !ok {
		return fmt.Sprintf("%s-%s", r.first, r.last)
	}

	return pfx.String()
}

// Prefixes returns a standard iterator yielding the minimal set of netip.Prefix values
// that fully cover the IPRange r.
func (r IPRange) Prefixes() iter.Seq[netip.Prefix] {
	return extnetip.All(r.Addrs())
}

// Merge combines adjacent and overlapping IPRanges in the input slice.
// It filters out duplicates, subsets, and invalid ranges, returning a new
// slice of merged, non-overlapping IPRanges sorted in ascending order.
func Merge(in []IPRange) (out []IPRange) {
	if len(in) == 0 {
		return nil
	}

	// Copy the input slice to avoid mutating it, and sort the ranges.
	rs := make([]IPRange, len(in))
	copy(rs, in)
	sortRanges(rs)

	for _, r := range rs {
		if r == zeroValue {
			continue
		}

		// Initialize the output slice with the first valid range.
		if out == nil {
			out = append(out, r)
			continue
		}

		// Compare the last merged range in the output with the current range.
		topic := &out[len(out)-1]

		switch {
		case topic.last.Next() == r.first:
			// Ranges are adjacent (e.g., [1.1.1.1-1.1.1.2] and [1.1.1.3-1.1.1.4]).
			topic.last = r.last
		case topic.isDisjunctLeft(r):
			// Ranges are disjoint (e.g., [1.1.1.1-1.1.1.2] and [1.1.1.4-1.1.1.5]).
			out = append(out, r)
		case topic.covers(r):
			// Current range is a subset of the last merged range (no-op).
			continue
		case topic.last.Less(r.last):
			// Ranges partially overlap; extend the last merged range's upper bound.
			topic.last = r.last
		default:
			panic("unreachable")
		}
	}

	return
}

// Remove subtracts the slice of exclusion ranges in from the IPRange r.
// It returns the remaining segments of r as a slice of non-overlapping
// IPRanges sorted in ascending order.
func (r IPRange) Remove(in []IPRange) (out []IPRange) {
	if r == zeroValue {
		return nil
	}

	// Merge the exclusion slice to get clean, sorted, non-overlapping segments.
	merged := Merge(in)

	// Quick exit checks if there are no exclusions or no overlap.
	if len(merged) == 0 {
		return []IPRange{r}
	}
	if r.isDisjunctLeft(merged[0]) {
		return []IPRange{r}
	}
	if r.isDisjunctRight(merged[len(merged)-1]) {
		return []IPRange{r}
	}

	for _, m := range merged {
		switch {
		case m.isDisjunct(r):
			// No overlap with the current exclusion segment; continue.
			continue
		case m.covers(r):
			// The exclusion fully covers the remaining range; nothing is left.
			return out
		case m.first.Compare(r.first) <= 0:
			// Exclusion overlaps on the left; advance r's lower bound past the exclusion.
			r.first = m.last.Next()
		case m.first.Compare(r.first) > 0:
			// Exclusion overlaps on the right; output the segment before the exclusion starts,
			// then advance r's lower bound past the exclusion.
			out = append(out, IPRange{r.first, m.first.Prev()})
			r.first = m.last.Next()
		default:
			panic("unreachable")
		}

		// Prevent infinite loops or invalid states when r's lower bound overflows.
		if !r.first.IsValid() {
			return out
		}
		// Terminate early if the advanced lower bound surpasses the upper bound.
		if r.last.Less(r.first) {
			return out
		}
	}

	// Append any remaining portion of the range.
	out = append(out, r)

	return out
}

// Compare returns four integers comparing the boundary endpoints of two IP ranges.
// It implements the comparison function required by the interval tree package
// at https://github.com/gaissmai/interval.
//
// The return values represent the comparisons:
//   - ll: a.first vs b.first
//   - rr: a.last  vs b.last
//   - lr: a.first vs b.last
//   - rl: a.last  vs b.first
func Compare(a, b IPRange) (ll int, rr int, lr int, rl int) {
	ll = a.first.Compare(b.first)
	rr = a.last.Compare(b.last)
	lr = a.first.Compare(b.last)
	rl = a.last.Compare(b.first)
	return
}

// MarshalText implements encoding.TextMarshaler.
// It returns the text representation of the range using String().
// If the range is invalid or uninitialized, it returns nil.
func (r IPRange) MarshalText() ([]byte, error) {
	if !r.IsValid() {
		return nil, nil
	}
	return []byte(r.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
// It parses the text representation using FromString.
// If text is empty, it leaves the receiver as the zero value.
// It returns an error if the receiver is nil or is not the zero value.
func (r *IPRange) UnmarshalText(text []byte) error {
	if r == nil {
		return errors.New("UnmarshalText on nil receiver")
	}

	if *r != zeroValue {
		return errors.New("refusing to Unmarshal into non-zero IPRange")
	}

	if len(text) == 0 {
		return nil
	}

	res, err := FromString(string(text))
	if err != nil {
		return err
	}

	*r = res
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
// It encodes the boundary addresses consecutively as raw bytes.
// (8 bytes for IPv4, 32 bytes for IPv6). It returns nil if the range is invalid.
func (r IPRange) MarshalBinary() ([]byte, error) {
	if !r.IsValid() {
		return nil, nil
	}

	size := 8
	if r.first.Is6() {
		size = 32
	}

	b := make([]byte, 0, size)
	b = append(b, r.first.AsSlice()...)
	b = append(b, r.last.AsSlice()...)

	return b, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
// It reconstructs the IPRange from bytes generated by MarshalBinary.
// It returns an error if the receiver is nil, not a zero value,
// if the byte slice length is not 8 or 32, or if the decoded last IP
// address is less than the first IP address.
func (r *IPRange) UnmarshalBinary(data []byte) error {
	if r == nil {
		return errors.New("UnmarshalBinary on nil receiver")
	}

	if *r != zeroValue {
		return errors.New("refusing to Unmarshal into non-zero IPRange")
	}

	n := len(data)
	if n == 0 {
		return nil
	}

	// Must be exactly 8 bytes (two 4-byte IPv4 addresses) or 32 bytes (two 16-byte IPv6 addresses).
	if n != 8 && n != 32 {
		return errors.New("unexpected slice size")
	}

	first, _ := netip.AddrFromSlice(data[:n/2])
	last, _ := netip.AddrFromSlice(data[n/2:])

	if last.Less(first) {
		return errors.New("last address is less than first address")
	}

	*r = IPRange{first, last}
	return nil
}

// Helper methods

func (a IPRange) isDisjunctLeft(b IPRange) bool {
	return a.last.Less(b.first)
}

func (a IPRange) isDisjunctRight(b IPRange) bool {
	return b.last.Less(a.first)
}

func (a IPRange) isDisjunct(b IPRange) bool {
	return a.last.Less(b.first) || b.last.Less(a.first)
}

func (a IPRange) covers(b IPRange) bool {
	return a.first.Compare(b.first) <= 0 && a.last.Compare(b.last) >= 0
}

// cmpRange compares two IPRanges. It orders them ascending by their first address.
// If the first addresses are equal, it orders the larger range (the superset) first.
func cmpRange(a, b IPRange) int {
	if a == b {
		return 0
	}

	if cmp := a.first.Compare(b.first); cmp != 0 {
		return cmp
	}

	return -(a.last.Compare(b.last))
}

// sortRanges sorts the slice of IPRanges in-place in ascending order.
func sortRanges(rs []IPRange) {
	sort.Slice(rs, func(i, j int) bool { return cmpRange(rs[i], rs[j]) < 0 })
}
