// Package iprange is an extension to net/netip.
//
// An additional type IPRange is defined and the most useful methods for it.
//
// For more advanced functionality IPRange implements the interval.Interface for fast lookups.
//
// see also: https://github.com/gaissmai/interval
package iprange

import (
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/gaissmai/extnetip"
)

// IPRange represents an inclusive range of IP addresses from the same address family.
//
//	10.0.0.3-10.0.17.134        // range
//	2001:db8::1-2001:db8::f6    // range
//	192.168.0.1/24              // Prefix aka CIDR
//	::1/128                     // Prefix aka CIDR
//
// Not all IP address ranges in the wild are CIDRs, very often you have to deal
// with ranges not representable as a prefix.
//
// This library handels IP ranges and CIDRs transparently.
type IPRange struct {
	first netip.Addr
	last  netip.Addr
}

var (
	zeroValue  IPRange
	invalidStr = "invalid IPRange"
)

// Parse returns the input string as type IPRange.
//
// Returns an error on invalid input.
//
// Valid strings are of the form:
//
//	192.168.2.3-192.168.7.255
//	2001:db8::1-2001:db8::ff00:35
//
//	2001:db8:dead::/38
//	10.0.0.0/8
//
//	4.4.4.4
//	::0
//
// IP addresses as input are converted to /32 or /128 ranges.
//
// The hard part is done by netip.ParseAddr and netip.ParsePrefix from the stdlib.
func Parse(s string) (IPRange, error) {
	if s == "" {
		return zeroValue, errors.New("empty string")
	}

	// addr/bits
	i := strings.IndexByte(s, '/')
	if i >= 0 {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return zeroValue, err
		}
		return FromNetipPrefix(p)
	}

	// addr-addr
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

		if first.Zone() != "" || last.Zone() != "" {
			return zeroValue, errors.New("ip address MUST NOT have a zone")
		}

		if first.Is4() && !last.Is4() || first.Is6() && !last.Is6() {
			return zeroValue, errors.New("first and last address have different IP versions")
		}
		if last.Less(first) {
			return zeroValue, errors.New("last address is less than first address")
		}
		return IPRange{first, last}, nil
	}

	// an addr, or maybe just rubbish
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return zeroValue, err
	}
	if addr.Zone() != "" {
		return zeroValue, errors.New("ip address MUST NOT have a zone")
	}
	return IPRange{addr, addr}, nil
}

// FromNetipPrefix returns an IPRange from the standard library's netip.Prefix type.
func FromNetipPrefix(p netip.Prefix) (IPRange, error) {
	if !p.IsValid() {
		return zeroValue, errors.New("netip.Prefix is invalid")
	}
	first, last := extnetip.Range(p)
	return IPRange{first, last}, nil
}

// FromNetipAddrs returns an IPRange from the provided IP addresses.
//
// IP addresses with zones are not allowed.
func FromNetipAddrs(first, last netip.Addr) (IPRange, error) {
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

// Addrs returns the first and last IP address of the IPRange.
func (r IPRange) Addrs() (first, last netip.Addr) {
	return r.first, r.last
}

// Prefix returns r as a netip.Prefix, if it can be presented exactly as such.
// If r is not valid or is not exactly equal to one prefix, ok is false.
func (r IPRange) Prefix() (prefix netip.Prefix, ok bool) {
	return extnetip.Prefix(r.first, r.last)
}

// Prefixes returns the slice of netip.Prefix entries that covers r.
//
// If r is invalid Prefixes returns nil.
//
// Prefixes necessarily allocates. See PrefixesAppend for a version that
// uses memory you provide.
func (r IPRange) Prefixes() []netip.Prefix {
	return extnetip.Prefixes(r.first, r.last)
}

// PrefixesAppend is the append version of Prefixes.
//
// It appends to dst the netip.Prefix entries that covers r.
func (r IPRange) PrefixesAppend(dst []netip.Prefix) []netip.Prefix {
	return extnetip.AppendPrefixes(dst, r.first, r.last)
}

// String returns the string form of the IPRange.
//
//	"127.0.0.1-127.0.0.19"
//	"2001:db8::/32"
func (r IPRange) String() string {
	if r == zeroValue {
		return invalidStr
	}
	p, ok := r.Prefix()
	if !ok {
		return fmt.Sprintf("%s-%s", r.first, r.last)
	}
	return p.String()
}

// #########################################################################################
// more complex functions

// Merge adjacent and overlapping IPRanges.
//
// Remove dups and subsets and invalid ranges, returns the remaining IPRanges sorted.
func Merge(in []IPRange) []IPRange {
	if len(in) == 0 {
		return nil
	}

	// copy and sort
	rs := make([]IPRange, len(in))
	copy(rs, in)
	sortRanges(rs)

	var result []IPRange

	for _, this := range rs {
		if this == zeroValue {
			continue
		}

		// starting point
		if result == nil {
			result = append(result, this)
			continue
		}

		// take ptr to last result item
		topic := &result[len(result)-1]

		// compare topic with this range
		// case order is VERY important!
		switch {
		case topic.last.Next() == this.first:
			// ranges are adjacent [f...l][f...l]
			topic.last = this.last
		case topic.last.Less(this.first):
			// ranges are disjoint [f...l]....[f...l]
			result = append(result, this)
		case topic.last.Less(this.last):
			// partial overlap [f......l]
			//                       [f....l]
			topic.last = this.last
		default:
			// no-op: true covers or equal
		}
	}

	return result
}

// Remove the slice of IPRanges from receiver, returns the remaining IPRanges.
func (r IPRange) Remove(in []IPRange) []IPRange {
	if r == zeroValue {
		return nil
	}

	// copy and sort
	rs := make([]IPRange, len(in))
	copy(rs, in)
	sortRanges(rs)

	var result []IPRange
	for _, d := range rs {
		switch {
		case d == zeroValue:
			// no-op
			continue
		case r.last.Less(d.first) || d.last.Less(r.first):
			// disjunct, no-op
			continue
		case d.first.Compare(r.first) <= 0 && d.last.Compare(r.last) >= 0:
			// d covers|equal r, mask rest
			return result
		case d.first.Compare(r.first) <= 0:
			// move cursor forward
			r.first = d.last.Next()
		case d.first.Compare(r.first) > 0:
			// save [r.first, d.first-1)
			result = append(result, IPRange{r.first, d.first.Prev()})
			// new r, (d.last, r.last]
			r.first = d.last.Next()
		default:
			panic("logic error")
		}
		// overflow from d.last.Next()
		if !r.first.IsValid() {
			return result
		}
		// cursor moved behind r.last
		if r.last.Less(r.first) {
			return result
		}
	}
	// save the rest
	result = append(result, r)

	return result
}

// #########################################################################################
// implement the interval.Interface

// CompareLower implements the interval.Interface from the
// package https://github.com/gaissmai/interval for fast lookups.
//
// Returns an integer comparing the two first IPs.
func (r IPRange) CompareLower(r2 IPRange) int {
	return r.first.Compare(r2.first)
}

// CompareUpper implements the interval.Interface from the
// package https://github.com/gaissmai/interval for fast lookups.
//
// Returns an integer comparing the two last IPs.
func (r IPRange) CompareUpper(r2 IPRange) int {
	return r.last.Compare(r2.last)
}

// #####################################################################################
// MARSHALING

// MarshalText implements the encoding.TextMarshaler interface,
// The encoding is the same as returned by String, with one exception:
// If r is the zero IPRange, the encoding is the empty string.
func (r IPRange) MarshalText() ([]byte, error) {
	if !r.first.IsValid() {
		return []byte(""), nil
	}
	return []byte(r.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// The IPRange is expected in a form accepted by Parse.
//
// If text is empty, UnmarshalText sets *r to the zero IPRange and
// returns no error.
func (r *IPRange) UnmarshalText(text []byte) error {
	if *r != zeroValue {
		return errors.New("refusing to Unmarshal into non-zero IPRange")
	}

	if len(text) == 0 {
		return nil
	}

	var err error
	*r, err = Parse(string(text))
	return err
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (r IPRange) MarshalBinary() ([]byte, error) {
	return append(r.first.AsSlice(), r.last.AsSlice()...), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
// It expects data in the form generated by MarshalBinary.
func (r *IPRange) UnmarshalBinary(data []byte) error {
	if *r != zeroValue {
		return errors.New("refusing to Unmarshal into non-zero IPRange")
	}

	n := len(data)
	if n == 0 {
		return nil
	}

	// first,last: IPv4: 2x4=8 bytes, IPv6: 2x16=32 bytes
	if n != 8 && n != 32 {
		return errors.New("unexpected slice size")
	}

	first, _ := netip.AddrFromSlice(data[:n/2])
	last, _ := netip.AddrFromSlice(data[n/2:])

	if last.Less(first) {
		return errors.New("last address is less than first address")
	}

	(*r).first = first
	(*r).last = last
	return nil
}

// ##################################################################
// mothers little helpers

// cmpRange, by first points, supersets to the left as tiebreaker
func cmpRange(a, b IPRange) int {
	if a == b {
		return 0
	}

	// cmp first
	if cmp := a.first.Compare(b.first); cmp != 0 {
		return cmp
	}

	// first is equal, sort supersets to the left
	return -(a.last.Compare(b.last))
}

// sortRanges in place in default sort order,
// first points ascending, supersets to the left.
func sortRanges(rs []IPRange) {
	sort.Slice(rs, func(i, j int) bool { return cmpRange(rs[i], rs[j]) < 0 })
}
