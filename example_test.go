package iprange_test

import (
	"fmt"

	"github.com/gaissmai/iprange"
)

func mustParse(s string) iprange.IPRange {
	r, err := iprange.FromString(s)
	if err != nil {
		panic(err)
	}
	return r
}

func isPrefix(p iprange.IPRange) bool {
	_, ok := p.Prefix()
	return ok
}

func ExampleFromString() {
	for _, s := range []string{
		"fe80::1-fe80::2",         // as range
		"10.0.0.0-11.255.255.255", // as range but true CIDR, see output
		"",                        // invalid
	} {
		r, _ := iprange.FromString(s)
		fmt.Printf("%-20s isPrefix: %5v\n", r, isPrefix(r))
	}

	// Output:
	// fe80::1-fe80::2      isPrefix: false
	// 10.0.0.0/7           isPrefix:  true
	// invalid IPRange      isPrefix: false
}

func ExampleIPRange_Addrs() {
	first, last := mustParse("fe80::/10").Addrs()

	fmt.Printf("Addrs() fe80::/10\n")
	fmt.Printf("first:  %s\n", first)
	fmt.Printf("last:   %s\n", last)

	// Output:
	// Addrs() fe80::/10
	// first:  fe80::
	// last:   febf:ffff:ffff:ffff:ffff:ffff:ffff:ffff
}

func ExampleMerge() {
	var rs []iprange.IPRange
	for _, s := range []string{
		"10.0.0.0/32",
		"10.0.0.1/32",
		"10.0.0.4/30",
		"10.0.0.6-10.0.0.99",
		"fe80::/12",
		"fe80:0000:0000:0000:fe2d:5eff:fef0:fc64/128",
		"fe80::/10",
	} {
		r, _ := iprange.FromString(s)
		rs = append(rs, r)
	}

	merged := iprange.Merge(rs)
	fmt.Printf("%v\n", merged)

	// Output:
	// [10.0.0.0/31 10.0.0.4-10.0.0.99 fe80::/10]
}

func ExampleIPRange_Prefixes() {
	r, _ := iprange.FromString("10.0.0.6-10.0.0.99")
	fmt.Printf("%s -> Prefixes:\n", r)
	for _, p := range r.Prefixes() {
		fmt.Println(p)
	}

	fmt.Println()

	r, _ = iprange.FromString("2001:db8::affe-2001:db8::ffff")
	fmt.Printf("%s -> Prefixes:\n", r)
	for _, p := range r.Prefixes() {
		fmt.Println(p)
	}

	// Output:
	// 10.0.0.6-10.0.0.99 -> Prefixes:
	// 10.0.0.6/31
	// 10.0.0.8/29
	// 10.0.0.16/28
	// 10.0.0.32/27
	// 10.0.0.64/27
	// 10.0.0.96/30
	//
	// 2001:db8::affe-2001:db8::ffff -> Prefixes:
	// 2001:db8::affe/127
	// 2001:db8::b000/116
	// 2001:db8::c000/114
}

func ExampleIPRange_Remove_v4() {
	outer, _ := iprange.FromString("192.168.2.0/24")
	inner := []iprange.IPRange{
		mustParse("192.168.2.0/26"),
		mustParse("192.168.2.240-192.168.2.249"),
	}

	fmt.Printf("outer: %v\n", outer)
	fmt.Printf("inner: %v\n", inner)
	fmt.Println("Result:")
	for _, r := range outer.Remove(inner) {
		fmt.Println(r)
	}

	// Output:
	// outer: 192.168.2.0/24
	// inner: [192.168.2.0/26 192.168.2.240-192.168.2.249]
	// Result:
	// 192.168.2.64-192.168.2.239
	// 192.168.2.250-192.168.2.255
}

func ExampleIPRange_Remove_v6() {
	outer, _ := iprange.FromString("2001:db8:de00::/40")
	inner := []iprange.IPRange{mustParse("2001:db8:dea0::/44")}

	fmt.Printf("outer: %v\n", outer)
	fmt.Printf("inner: %v\n", inner)
	fmt.Println("Result:")
	for _, r := range outer.Remove(inner) {
		fmt.Println(r)
	}

	// Output:
	// outer: 2001:db8:de00::/40
	// inner: [2001:db8:dea0::/44]
	// Result:
	// 2001:db8:de00::-2001:db8:de9f:ffff:ffff:ffff:ffff:ffff
	// 2001:db8:deb0::-2001:db8:deff:ffff:ffff:ffff:ffff:ffff
}
