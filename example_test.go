package iprange_test

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/gaissmai/iprange"
)

func ExampleFromString() {
	// Parsing standard CIDR prefixes
	r1, err := iprange.FromString("192.168.1.0/24")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s (is CIDR: %t)\n", r1, isPrefix(r1))

	// Parsing explicit address ranges
	r2, err := iprange.FromString("10.0.0.1-10.0.0.5")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s (is CIDR: %t)\n", r2, isPrefix(r2))

	// Parsing single IP addresses
	r3, err := iprange.FromString("8.8.8.8")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s (is CIDR: %t)\n", r3, isPrefix(r3))

	// Parsing IPv6 ranges
	r4, err := iprange.FromString("2001:db8::1-2001:db8::ff")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s (is CIDR: %t)\n", r4, isPrefix(r4))

	// Output:
	// 192.168.1.0/24 (is CIDR: true)
	// 10.0.0.1-10.0.0.5 (is CIDR: false)
	// 8.8.8.8/32 (is CIDR: true)
	// 2001:db8::1-2001:db8::ff (is CIDR: false)
}

func ExampleFromPrefix() {
	prefix := netip.MustParsePrefix("2001:db8::/32")
	r, err := iprange.FromPrefix(prefix)
	if err != nil {
		panic(err)
	}
	fmt.Println(r)

	// Output:
	// 2001:db8::/32
}

func ExampleFromAddrs() {
	first := netip.MustParseAddr("192.168.1.5")
	last := netip.MustParseAddr("192.168.1.10")

	r, err := iprange.FromAddrs(first, last)
	if err != nil {
		panic(err)
	}
	fmt.Println(r)

	// Output:
	// 192.168.1.5-192.168.1.10
}

func ExampleIPRange_Addrs() {
	r, _ := iprange.FromString("10.0.0.0/30")
	first, last := r.Addrs()

	fmt.Printf("first: %s\n", first)
	fmt.Printf("last:  %s\n", last)

	// Output:
	// first: 10.0.0.0
	// last:  10.0.0.3
}

func ExampleIPRange_Prefix() {
	// A range that perfectly represents a CIDR prefix
	r1, _ := iprange.FromString("192.168.1.0-192.168.1.255")
	p1, ok1 := r1.Prefix()
	fmt.Printf("p1: %s (ok: %t)\n", p1, ok1)

	// A range that is not representable by a single prefix
	r2, _ := iprange.FromString("192.168.1.1-192.168.1.255")
	p2, ok2 := r2.Prefix()
	fmt.Printf("p2: %s (ok: %t)\n", p2, ok2)

	// Output:
	// p1: 192.168.1.0/24 (ok: true)
	// p2: invalid Prefix (ok: false)
}

func ExampleIPRange_Prefixes() {
	r, _ := iprange.FromString("10.0.0.6-10.0.0.13")

	// Split the non-CIDR range into the minimal set of CIDR prefixes covering it
	fmt.Println("Prefixes covering 10.0.0.6-10.0.0.13:")
	for prefix := range r.Prefixes() {
		fmt.Println(prefix)
	}

	// Output:
	// Prefixes covering 10.0.0.6-10.0.0.13:
	// 10.0.0.6/31
	// 10.0.0.8/30
	// 10.0.0.12/31
}

func ExampleIPRange_Prefixes_ipv6() {
	r, _ := iprange.FromString("2001:db8::affe-2001:db8::b003")

	fmt.Println("Prefixes covering 2001:db8::affe-2001:db8::b003:")
	for prefix := range r.Prefixes() {
		fmt.Println(prefix)
	}

	// Output:
	// Prefixes covering 2001:db8::affe-2001:db8::b003:
	// 2001:db8::affe/127
	// 2001:db8::b000/126
}

func ExampleMerge() {
	ranges := []iprange.IPRange{
		mustParse("10.0.0.1/32"),
		mustParse("10.0.0.2/32"),
		mustParse("10.0.0.5/32"),
		mustParse("10.0.0.3-10.0.0.4"),
	}

	// Merge adjacent and overlapping ranges
	merged := iprange.Merge(ranges)
	fmt.Println(merged)

	// Output:
	// [10.0.0.1-10.0.0.5]
}

func ExampleIPRange_Remove() {
	outer, _ := iprange.FromString("192.168.1.0/24")
	exclusions := []iprange.IPRange{
		mustParse("192.168.1.0-192.168.1.10"),
		mustParse("192.168.1.200/29"),
	}

	// Exclude the sub-ranges from the outer range
	remaining := outer.Remove(exclusions)
	for _, r := range remaining {
		fmt.Println(r)
	}

	// Output:
	// 192.168.1.11-192.168.1.199
	// 192.168.1.208-192.168.1.255
}

func ExampleIPRange_Remove_ipv6() {
	outer, _ := iprange.FromString("2001:db8:de00::/40")
	exclusions := []iprange.IPRange{
		mustParse("2001:db8:dea0::/44"),
	}

	// Exclude the sub-ranges from the outer range
	remaining := outer.Remove(exclusions)
	for _, r := range remaining {
		fmt.Println(r)
	}

	// Output:
	// 2001:db8:de00::-2001:db8:de9f:ffff:ffff:ffff:ffff:ffff
	// 2001:db8:deb0::-2001:db8:deff:ffff:ffff:ffff:ffff:ffff
}

func ExampleCompare() {
	r1 := mustParse("10.0.0.0-10.0.0.10")
	r2 := mustParse("10.0.0.5-10.0.0.15")

	ll, rr, lr, rl := iprange.Compare(r1, r2)
	fmt.Printf("Compare result: ll=%d, rr=%d, lr=%d, rl=%d\n", ll, rr, lr, rl)

	// Output:
	// Compare result: ll=-1, rr=-1, lr=-1, rl=1
}

func ExampleIPRange_MarshalText() {
	// Struct using IPRange that supports JSON serialization
	type Config struct {
		Allowed iprange.IPRange `json:"allowed"`
	}

	c := Config{Allowed: mustParse("192.168.1.0/24")}
	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	// Output:
	// {"allowed":"192.168.1.0/24"}
}

func ExampleIPRange_UnmarshalText() {
	type Config struct {
		Allowed iprange.IPRange `json:"allowed"`
	}

	jsonData := []byte(`{"allowed":"10.0.0.1-10.0.0.10"}`)
	var c Config
	if err := json.Unmarshal(jsonData, &c); err != nil {
		panic(err)
	}

	fmt.Println(c.Allowed)

	// Output:
	// 10.0.0.1-10.0.0.10
}

func isPrefix(r iprange.IPRange) bool {
	_, ok := r.Prefix()
	return ok
}

func mustParse(s string) iprange.IPRange {
	r, err := iprange.FromString(s)
	if err != nil {
		panic(err)
	}
	return r
}
