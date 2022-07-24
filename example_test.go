package iprange_test

import (
	"fmt"

	"github.com/gaissmai/iprange"
)

func mustParse(s string) iprange.IPRange {
	b, err := iprange.Parse(s)
	if err != nil {
		panic(err)
	}
	return b
}

func isPrefix(p iprange.IPRange) bool {
	_, ok := p.Prefix()
	return ok
}

func ExampleParse() {
	for _, s := range []string{
		"fe80::1-fe80::2",         // as range
		"10.0.0.0-11.255.255.255", // as range but true CIDR, see output
		"",                        // invalid
	} {
		r, _ := iprange.Parse(s)
		fmt.Printf("%-20s isPrefix: %5v\n", r, isPrefix(r))
	}

	// Output:
	// fe80::1-fe80::2      isPrefix: false
	// 10.0.0.0/7           isPrefix:  true
	// invalid IPRange      isPrefix: false
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
		b, _ := iprange.Parse(s)
		rs = append(rs, b)
	}

	merged := iprange.Merge(rs)
	fmt.Printf("%v\n", merged)

	// Output:
	// [10.0.0.0/31 10.0.0.4-10.0.0.99 fe80::/10]
}

func ExampleIPRange_Prefixes() {
	r, _ := iprange.Parse("10.0.0.6-10.0.0.99")
	fmt.Printf("%v\n", r.Prefixes())

	r, _ = iprange.Parse("2001:db8::affe-2001:db8::ffff")
	fmt.Printf("%v\n", r.Prefixes())

	// Output:
	// [10.0.0.6/31 10.0.0.8/29 10.0.0.16/28 10.0.0.32/27 10.0.0.64/27 10.0.0.96/30]
	// [2001:db8::affe/127 2001:db8::b000/116 2001:db8::c000/114]
}

func ExampleIPRange_Remove_v4() {
	outer, _ := iprange.Parse("192.168.2.0/24")
	inner := []iprange.IPRange{
		mustParse("192.168.2.0/26"),
		mustParse("192.168.2.240-192.168.2.249"),
	}

	fmt.Printf("%v - %v\ndiff: %v\n", outer, inner, outer.Remove(inner))

	// Output:
	// 192.168.2.0/24 - [192.168.2.0/26 192.168.2.240-192.168.2.249]
	// diff: [192.168.2.64-192.168.2.239 192.168.2.250-192.168.2.255]
}

func ExampleIPRange_Remove_v6() {
	outer, _ := iprange.Parse("2001:db8:de00::/40")
	inner := []iprange.IPRange{mustParse("2001:db8:dea0::/44")}

	fmt.Printf("%v - %v\ndiff: %v\n", outer, inner, outer.Remove(inner))

	// Output:
	// 2001:db8:de00::/40 - [2001:db8:dea0::/44]
	// diff: [2001:db8:de00::-2001:db8:de9f:ffff:ffff:ffff:ffff:ffff 2001:db8:deb0::-2001:db8:deff:ffff:ffff:ffff:ffff:ffff]
}
