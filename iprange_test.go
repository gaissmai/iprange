package iprange_test

import (
	"net/netip"
	"reflect"
	"testing"

	"github.com/gaissmai/iprange"
)

var (
	mustParseAddr   = netip.MustParseAddr
	mustParsePrefix = netip.MustParsePrefix

	mustFromPrefix = func(p netip.Prefix) iprange.IPRange {
		r, err := iprange.FromPrefix(p)
		if err != nil {
			panic(err)
		}
		return r
	}

	mustFromString = func(s string) iprange.IPRange {
		r, err := iprange.FromString(s)
		if err != nil {
			panic(err)
		}
		return r
	}
)

func TestFromAddrs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		first netip.Addr
		last  netip.Addr
		ok    bool
	}{
		{
			first: mustParseAddr("1.2.3.4"),
			last:  mustParseAddr("5.6.7.8"),
			ok:    true,
		},
		{
			first: mustParseAddr("fe80::1"),
			last:  mustParseAddr("fe80::2"),
			ok:    true,
		},
		{
			first: mustParseAddr("5.6.7.8"),
			last:  mustParseAddr("1.2.3.4"),
			ok:    false,
		},
		{
			first: netip.Addr{},
			last:  mustParseAddr("5.6.7.8"),
			ok:    false,
		},
		{
			first: mustParseAddr("5.6.7.8"),
			last:  netip.Addr{},
			ok:    false,
		},
		{
			first: mustParseAddr("fe80::1"),
			last:  mustParseAddr("5.6.7.8"),
			ok:    false,
		},
		{
			first: mustParseAddr("5.6.7.8"),
			last:  mustParseAddr("fe80::1"),
			ok:    false,
		},
		{
			first: mustParseAddr("5.6.7.8"),
			last:  mustParseAddr("fe80::1"),
			ok:    false,
		},
		{
			first: mustParseAddr("fe80::1"),
			last:  mustParseAddr("fe80::2%eth1"),
			ok:    false,
		},
		{
			first: mustParseAddr("fe80::1%eth1"),
			last:  mustParseAddr("fe80::2"),
			ok:    false,
		},
	}

	for _, tt := range tests {
		ok := true
		_, err := iprange.FromAddrs(tt.first, tt.last)
		if err != nil {
			ok = false
		}
		if ok != tt.ok {
			t.Fatalf("FromAddrs(%s, %s), got: %v, want: %v\n", tt.first, tt.last, ok, tt.ok)
		}
	}
}

func TestFromStringInvalid(t *testing.T) {
	t.Parallel()
	tests := []string{
		"::ffff:0.0.0.0-0.0.0.1",
		"0.0.0.0-::ffff:0.0.0.1",
		"1.2.3.4-fe80::1",
		"fe80::1-127.0.0.1",
		"fe80::1-",
		"-fe80::1",
		"fe80::/130",
		"127.0.0.0/35",
		"fe80::1%eth0-fe80::2",
		"fe80::1-fe80::2%eth2",
		"fe80::2-fe80::1",
		"fe80::2%eth2",
		"1.2.3.4.5",
	}

	for _, s := range tests {
		if r, err := iprange.FromString(s); err == nil {
			t.Fatalf("ParseRange(%s); got %q, want err; got %v", s, r, err)
		}
	}
}

func TestFromPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pfx   netip.Prefix
		first netip.Addr
		last  netip.Addr
	}{
		{
			pfx:   mustParsePrefix("0.0.0.0/0"),
			first: mustParseAddr("0.0.0.0"),
			last:  mustParseAddr("255.255.255.255"),
		},
		{
			pfx:   mustParsePrefix("0.0.0.0/32"),
			first: mustParseAddr("0.0.0.0"),
			last:  mustParseAddr("0.0.0.0"),
		},
		{
			pfx:   mustParsePrefix("0.0.0.0/8"),
			first: mustParseAddr("0.0.0.0"),
			last:  mustParseAddr("0.255.255.255"),
		},
		{
			pfx:   mustParsePrefix("::ffff:0.0.0.0/104"),
			first: mustParseAddr("::ffff:0.0.0.0"),
			last:  mustParseAddr("::ffff:0.255.255.255"),
		},
		{
			pfx:   mustParsePrefix("::/0"),
			first: mustParseAddr("::"),
			last:  mustParseAddr("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"),
		},
		{
			pfx:   mustParsePrefix("::/128"),
			first: mustParseAddr("::"),
			last:  mustParseAddr("::"),
		},
	}

	for _, tt := range tests {
		r := mustFromPrefix(tt.pfx)
		first, last := r.Addrs()
		if first != tt.first || last != tt.last {
			t.Fatalf("FromPrefix(%s), want: (%s, %s), got: (%s, %s)", tt.pfx, tt.first, tt.last, first, last)
		}
	}

	// corner case
	r, err := iprange.FromPrefix(netip.Prefix{})
	if r.IsValid() || err == nil {
		t.Fatalf("FomPrefix() of invalid prefix, want: inavlid range and error, got: (%v, %v)", r, err)
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   []iprange.IPRange
		want []iprange.IPRange
	}{
		{
			in:   nil,
			want: nil,
		},
		{
			in:   []iprange.IPRange{{}},
			want: nil,
		},
		{
			in:   []iprange.IPRange{{}, mustFromString("1.2.3.4-5.6.7.8")},
			want: []iprange.IPRange{mustFromString("1.2.3.4-5.6.7.8")},
		},
		{
			in:   []iprange.IPRange{{}, {}, mustFromString("::/64"), {}, mustFromString("1.2.3.4-5.6.7.8")},
			want: []iprange.IPRange{mustFromString("1.2.3.4-5.6.7.8"), mustFromString("::/64")},
		},
		{
			in:   []iprange.IPRange{mustFromString("1.2.3.4-5.6.7.8"), mustFromString("5.6.7.0-10.0.0.0")},
			want: []iprange.IPRange{mustFromString("1.2.3.4-10.0.0.0")},
		},
		{
			in:   []iprange.IPRange{mustFromString("1.2.3.4-5.6.7.8"), mustFromString("5.6.7.9-10.0.0.0")},
			want: []iprange.IPRange{mustFromString("1.2.3.4-10.0.0.0")},
		},
		{
			in:   []iprange.IPRange{mustFromString("2001:db8::4/126"), mustFromString("2001:db8::8/127")},
			want: []iprange.IPRange{mustFromString("2001:db8::4-2001:db8::9")},
		},
	}

	for _, tt := range tests {
		rs := iprange.Merge(tt.in)
		if !reflect.DeepEqual(tt.want, rs) {
			t.Fatalf("Merge(%v): want: %v, got: %v", tt.in, tt.want, rs)
		}
	}
}

func TestMerge2(t *testing.T) {
	t.Parallel()
	rs := []iprange.IPRange{
		mustFromString("0.0.0.0/0"),
		mustFromString("10.0.0.15/32"),
		mustFromString("10.0.0.16/28"),
		mustFromString("10.0.0.32/27"),
		mustFromString("10.0.0.64/26"),
		mustFromString("10.0.0.128/26"),
		mustFromString("10.0.0.192/27"),
		mustFromString("134.60.0.0/16"),
		mustFromString("134.60.0.255/24"),
		mustFromString("193.197.62.192/29"),
		mustFromString("193.197.64.0/22"),
		mustFromString("193.197.228.0/22"),
		mustFromString("::/0"),
		mustFromString("::-::ffff"),
		mustFromString("2001:7c0:900::/48"),
		mustFromString("2001:7c0:900::/49"),
		mustFromString("2001:7c0:900::/52"),
		mustFromString("2001:7c0:900::/53"),
		mustFromString("2001:7c0:900:800::/56"),
		mustFromString("2001:7c0:900:800::/64"),
	}
	got := iprange.Merge(rs)

	want := []iprange.IPRange{
		mustFromString("0.0.0.0/0"),
		mustFromString("::/0"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Merge():\ngot:  %v\nwant: %v", got, want)
	}

	// corner cases
	rs = []iprange.IPRange{} // nil slice
	if got = iprange.Merge(rs); got != nil {
		t.Errorf("Merge() nil slice should return nil, got %v\n", got)
	}

	rs = []iprange.IPRange{mustFromString("0.0.0.0/8")}
	want = []iprange.IPRange{mustFromString("0.0.0.0/8")}
	got = iprange.Merge(rs)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Merge():\ngot:  %v\nwant: %v", got, want)
	}
}

func TestRemoveCornerCases(t *testing.T) {
	t.Parallel()
	// nil
	var r iprange.IPRange
	rs := r.Remove(nil)

	if rs != nil {
		t.Errorf("(nil).Remove(nil), got %v, want %v", rs, nil)
	}

	// nil
	r = mustFromString("::/0")
	rs = r.Remove(nil)

	if rs[0] != r {
		t.Errorf("Remove(nil), got %v, want %v", rs, []iprange.IPRange{r})
	}

	// zero value
	r = mustFromString("::/0")
	rs = r.Remove([]iprange.IPRange{{}})

	if rs[0] != r {
		t.Errorf("Remove(nil), got %v, want %v", rs, []iprange.IPRange{r})
	}

	// self
	r = mustFromString("::/0")
	rs = r.Remove([]iprange.IPRange{r})
	if rs != nil {
		t.Errorf("Remove(self), got %v, want nil", rs)
	}

	// disjunct after
	r = mustFromString("10.0.0.0/16")
	rs = r.Remove([]iprange.IPRange{mustFromString("::/0")})
	if rs[0] != r {
		t.Errorf("Remove(disjunct), got %v, want %v", rs, []iprange.IPRange{r})
	}

	// disjunct before
	r = mustFromString("::/0")
	rs = r.Remove([]iprange.IPRange{mustFromString("0.0.0.0/0")})
	if rs[0] != r {
		t.Errorf("Remove(disjunct), got %v, want %v", rs, []iprange.IPRange{r})
	}

	// disjunct in loop
	r = mustFromString("0.0.0.0/0")
	rs = r.Remove([]iprange.IPRange{mustFromString("0.0.0.0/1"), mustFromString("::/0")})
	wantRs := []iprange.IPRange{mustFromString("128.0.0.0/1")}
	if !reflect.DeepEqual(rs, wantRs) {
		t.Errorf("Remove(...), got %v, want %v", rs, wantRs)
	}

	// covers
	r = mustFromString("10.0.0.0/16")
	rs = r.Remove([]iprange.IPRange{mustFromString("10.0.0.0/8")})
	if rs != nil {
		t.Errorf("Remove(coverage), got %v, want nil", rs)
	}

	// overflow
	r = mustFromString("0.0.0.0/0")
	rs = r.Remove([]iprange.IPRange{mustFromString("255.255.255.255")})
	want := mustFromString("0.0.0.0-255.255.255.254")
	if rs[0] != want {
		t.Errorf("Remove(overflow), got %v, want %v", rs, want)
	}

	// base > last
	r = mustFromString("10.0.0.0/8")
	rs = r.Remove([]iprange.IPRange{mustFromString("10.128.0.0/9")})
	want = mustFromString("10.0.0.0/9")
	if rs[0] != want {
		t.Errorf("Remove(base>last), got %v, want %v", rs, want)
	}

	// left overlap v4
	r = mustFromString("10.0.0.5-10.0.0.15")
	rs = r.Remove([]iprange.IPRange{mustFromString("10.0.0.3-10.0.0.10")})
	want = mustFromString("10.0.0.11-10.0.0.15")
	if rs[0] != want {
		t.Errorf("Remove(leftOverlapV4), got %v, want %v", rs, want)
	}

	// right overlap v4
	r = mustFromString("10.0.0.4-10.0.0.15")
	rs = r.Remove([]iprange.IPRange{mustFromString("10.0.0.6-10.0.0.19")})
	want = mustFromString("10.0.0.4-10.0.0.5")
	if rs[0] != want {
		t.Errorf("Remove(leftOverlapV4), got %v, want %v", rs, want)
	}

	// left overlap v6
	r = mustFromString("2001:db8::17-2001:db8::177")
	rs = r.Remove([]iprange.IPRange{mustFromString("2001:db8::14-2001:db8::137")})
	want = mustFromString("2001:db8::138-2001:db8::177")
	if rs[0] != want {
		t.Errorf("Remove(leftOverlapV4), got %v, want %v", rs, want)
	}

	// right overlap v6
	r = mustFromString("2001:db8::17-2001:db8::177")
	rs = r.Remove([]iprange.IPRange{mustFromString("2001:db8::3f-2001:db8::fff")})
	want = mustFromString("2001:db8::17-2001:db8::3e")
	if rs[0] != want {
		t.Errorf("Remove(leftOverlapV4), got %v, want %v", rs, want)
	}
}

func TestRemoveIANAv6(t *testing.T) {
	t.Parallel()
	b, _ := iprange.FromString("::/0")

	var inner []iprange.IPRange
	for _, s := range []string{
		"0000::/8",
		"0100::/8",
		"0200::/7",
		"0400::/6",
		"0800::/5",
		"1000::/4",
		"2000::/3",
		"4000::/3",
		// "6000::/3",
		"8000::/3",
		"a000::/3",
		"c000::/3",
		"e000::/4",
		"f000::/5",
		"f800::/6",
		// "fc00::/7",
		"fe00::/9",
		"fe80::/10",
		"fec0::/10",
		"ff00::/8",
	} {
		inner = append(inner, mustFromString(s))
	}

	var want []iprange.IPRange
	for _, s := range []string{
		"6000::/3",
		"fc00::/7",
	} {
		want = append(want, mustFromString(s))
	}

	rs := b.Remove(inner)

	if !reflect.DeepEqual(rs, want) {
		t.Errorf("Remove for IANAv6 blocks, got %v, want %v", rs, want)
	}
}

func TestPrefixes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   iprange.IPRange
		want []netip.Prefix
	}{
		{
			in:   mustFromString("::/0"),
			want: []netip.Prefix{mustParsePrefix("::/0")},
		},
		{
			in:   mustFromString("0.0.0.0/0"),
			want: []netip.Prefix{mustParsePrefix("0.0.0.0/0")},
		},
		{
			in:   mustFromString("::ffff:0.0.0.0/96"),
			want: []netip.Prefix{mustParsePrefix("::ffff:0.0.0.0/96")},
		},
		{
			in:   mustFromString("2001:db8::/128"),
			want: []netip.Prefix{mustParsePrefix("2001:db8::/128")},
		},
		{
			in:   mustFromString("::ffff:0.0.0.0/128"),
			want: []netip.Prefix{mustParsePrefix("::ffff:0.0.0.0/128")},
		},
		{
			in:   mustFromString("0.0.0.0/32"),
			want: []netip.Prefix{mustParsePrefix("0.0.0.0/32")},
		},
		{
			in:   mustFromString("0.0.0.0-255.255.255.255"),
			want: []netip.Prefix{mustParsePrefix("0.0.0.0/0")},
		},
		{
			in: mustFromString("1.2.3.5-5.6.7.8"),
			want: []netip.Prefix{
				mustParsePrefix("1.2.3.5/32"),
				mustParsePrefix("1.2.3.6/31"),
				mustParsePrefix("1.2.3.8/29"),
				mustParsePrefix("1.2.3.16/28"),
				mustParsePrefix("1.2.3.32/27"),
				mustParsePrefix("1.2.3.64/26"),
				mustParsePrefix("1.2.3.128/25"),
				mustParsePrefix("1.2.4.0/22"),
				mustParsePrefix("1.2.8.0/21"),
				mustParsePrefix("1.2.16.0/20"),
				mustParsePrefix("1.2.32.0/19"),
				mustParsePrefix("1.2.64.0/18"),
				mustParsePrefix("1.2.128.0/17"),
				mustParsePrefix("1.3.0.0/16"),
				mustParsePrefix("1.4.0.0/14"),
				mustParsePrefix("1.8.0.0/13"),
				mustParsePrefix("1.16.0.0/12"),
				mustParsePrefix("1.32.0.0/11"),
				mustParsePrefix("1.64.0.0/10"),
				mustParsePrefix("1.128.0.0/9"),
				mustParsePrefix("2.0.0.0/7"),
				mustParsePrefix("4.0.0.0/8"),
				mustParsePrefix("5.0.0.0/14"),
				mustParsePrefix("5.4.0.0/15"),
				mustParsePrefix("5.6.0.0/22"),
				mustParsePrefix("5.6.4.0/23"),
				mustParsePrefix("5.6.6.0/24"),
				mustParsePrefix("5.6.7.0/29"),
				mustParsePrefix("5.6.7.8/32"),
			},
		},
		{
			in: mustFromString("0.0.0.0-255.255.255.254"),
			want: []netip.Prefix{
				mustParsePrefix("0.0.0.0/1"),
				mustParsePrefix("128.0.0.0/2"),
				mustParsePrefix("192.0.0.0/3"),
				mustParsePrefix("224.0.0.0/4"),
				mustParsePrefix("240.0.0.0/5"),
				mustParsePrefix("248.0.0.0/6"),
				mustParsePrefix("252.0.0.0/7"),
				mustParsePrefix("254.0.0.0/8"),
				mustParsePrefix("255.0.0.0/9"),
				mustParsePrefix("255.128.0.0/10"),
				mustParsePrefix("255.192.0.0/11"),
				mustParsePrefix("255.224.0.0/12"),
				mustParsePrefix("255.240.0.0/13"),
				mustParsePrefix("255.248.0.0/14"),
				mustParsePrefix("255.252.0.0/15"),
				mustParsePrefix("255.254.0.0/16"),
				mustParsePrefix("255.255.0.0/17"),
				mustParsePrefix("255.255.128.0/18"),
				mustParsePrefix("255.255.192.0/19"),
				mustParsePrefix("255.255.224.0/20"),
				mustParsePrefix("255.255.240.0/21"),
				mustParsePrefix("255.255.248.0/22"),
				mustParsePrefix("255.255.252.0/23"),
				mustParsePrefix("255.255.254.0/24"),
				mustParsePrefix("255.255.255.0/25"),
				mustParsePrefix("255.255.255.128/26"),
				mustParsePrefix("255.255.255.192/27"),
				mustParsePrefix("255.255.255.224/28"),
				mustParsePrefix("255.255.255.240/29"),
				mustParsePrefix("255.255.255.248/30"),
				mustParsePrefix("255.255.255.252/31"),
				mustParsePrefix("255.255.255.254/32"),
			},
		},
		{
			in:   mustFromString("::-ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"),
			want: []netip.Prefix{mustParsePrefix("::/0")},
		},
		{
			in: mustFromString("::-ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe"),
			want: []netip.Prefix{
				mustParsePrefix("::/1"),
				mustParsePrefix("8000::/2"),
				mustParsePrefix("c000::/3"),
				mustParsePrefix("e000::/4"),
				mustParsePrefix("f000::/5"),
				mustParsePrefix("f800::/6"),
				mustParsePrefix("fc00::/7"),
				mustParsePrefix("fe00::/8"),
				mustParsePrefix("ff00::/9"),
				mustParsePrefix("ff80::/10"),
				mustParsePrefix("ffc0::/11"),
				mustParsePrefix("ffe0::/12"),
				mustParsePrefix("fff0::/13"),
				mustParsePrefix("fff8::/14"),
				mustParsePrefix("fffc::/15"),
				mustParsePrefix("fffe::/16"),
				mustParsePrefix("ffff::/17"),
				mustParsePrefix("ffff:8000::/18"),
				mustParsePrefix("ffff:c000::/19"),
				mustParsePrefix("ffff:e000::/20"),
				mustParsePrefix("ffff:f000::/21"),
				mustParsePrefix("ffff:f800::/22"),
				mustParsePrefix("ffff:fc00::/23"),
				mustParsePrefix("ffff:fe00::/24"),
				mustParsePrefix("ffff:ff00::/25"),
				mustParsePrefix("ffff:ff80::/26"),
				mustParsePrefix("ffff:ffc0::/27"),
				mustParsePrefix("ffff:ffe0::/28"),
				mustParsePrefix("ffff:fff0::/29"),
				mustParsePrefix("ffff:fff8::/30"),
				mustParsePrefix("ffff:fffc::/31"),
				mustParsePrefix("ffff:fffe::/32"),
				mustParsePrefix("ffff:ffff::/33"),
				mustParsePrefix("ffff:ffff:8000::/34"),
				mustParsePrefix("ffff:ffff:c000::/35"),
				mustParsePrefix("ffff:ffff:e000::/36"),
				mustParsePrefix("ffff:ffff:f000::/37"),
				mustParsePrefix("ffff:ffff:f800::/38"),
				mustParsePrefix("ffff:ffff:fc00::/39"),
				mustParsePrefix("ffff:ffff:fe00::/40"),
				mustParsePrefix("ffff:ffff:ff00::/41"),
				mustParsePrefix("ffff:ffff:ff80::/42"),
				mustParsePrefix("ffff:ffff:ffc0::/43"),
				mustParsePrefix("ffff:ffff:ffe0::/44"),
				mustParsePrefix("ffff:ffff:fff0::/45"),
				mustParsePrefix("ffff:ffff:fff8::/46"),
				mustParsePrefix("ffff:ffff:fffc::/47"),
				mustParsePrefix("ffff:ffff:fffe::/48"),
				mustParsePrefix("ffff:ffff:ffff::/49"),
				mustParsePrefix("ffff:ffff:ffff:8000::/50"),
				mustParsePrefix("ffff:ffff:ffff:c000::/51"),
				mustParsePrefix("ffff:ffff:ffff:e000::/52"),
				mustParsePrefix("ffff:ffff:ffff:f000::/53"),
				mustParsePrefix("ffff:ffff:ffff:f800::/54"),
				mustParsePrefix("ffff:ffff:ffff:fc00::/55"),
				mustParsePrefix("ffff:ffff:ffff:fe00::/56"),
				mustParsePrefix("ffff:ffff:ffff:ff00::/57"),
				mustParsePrefix("ffff:ffff:ffff:ff80::/58"),
				mustParsePrefix("ffff:ffff:ffff:ffc0::/59"),
				mustParsePrefix("ffff:ffff:ffff:ffe0::/60"),
				mustParsePrefix("ffff:ffff:ffff:fff0::/61"),
				mustParsePrefix("ffff:ffff:ffff:fff8::/62"),
				mustParsePrefix("ffff:ffff:ffff:fffc::/63"),
				mustParsePrefix("ffff:ffff:ffff:fffe::/64"),
				mustParsePrefix("ffff:ffff:ffff:ffff::/65"),
				mustParsePrefix("ffff:ffff:ffff:ffff:8000::/66"),
				mustParsePrefix("ffff:ffff:ffff:ffff:c000::/67"),
				mustParsePrefix("ffff:ffff:ffff:ffff:e000::/68"),
				mustParsePrefix("ffff:ffff:ffff:ffff:f000::/69"),
				mustParsePrefix("ffff:ffff:ffff:ffff:f800::/70"),
				mustParsePrefix("ffff:ffff:ffff:ffff:fc00::/71"),
				mustParsePrefix("ffff:ffff:ffff:ffff:fe00::/72"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ff00::/73"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ff80::/74"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffc0::/75"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffe0::/76"),
				mustParsePrefix("ffff:ffff:ffff:ffff:fff0::/77"),
				mustParsePrefix("ffff:ffff:ffff:ffff:fff8::/78"),
				mustParsePrefix("ffff:ffff:ffff:ffff:fffc::/79"),
				mustParsePrefix("ffff:ffff:ffff:ffff:fffe::/80"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff::/81"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:8000::/82"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:c000::/83"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:e000::/84"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:f000::/85"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:f800::/86"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:fc00::/87"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:fe00::/88"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ff00::/89"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ff80::/90"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffc0::/91"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffe0::/92"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:fff0::/93"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:fff8::/94"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:fffc::/95"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:fffe::/96"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff::/97"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:8000:0/98"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:c000:0/99"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:e000:0/100"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:f000:0/101"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:f800:0/102"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:fc00:0/103"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:fe00:0/104"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ff00:0/105"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ff80:0/106"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffc0:0/107"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffe0:0/108"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:fff0:0/109"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:fff8:0/110"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:fffc:0/111"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:fffe:0/112"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:0/113"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:8000/114"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:c000/115"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:e000/116"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:f000/117"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:f800/118"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fc00/119"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fe00/120"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ff00/121"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ff80/122"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffc0/123"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffe0/124"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fff0/125"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fff8/126"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffc/127"),
				mustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/128"),
			},
		},
	}

	for _, tt := range tests {
		got := tt.in.Prefixes()
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("iprange.Prefixes(), for '%s', want %v, got %v", tt.in, tt.want, got)
		}
	}

	for _, tt := range tests {
		got := []netip.Prefix{}
		got = tt.in.PrefixesAppend(got)
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("iprange.Prefixes(), for '%s', want %v, got %v", tt.in, tt.want, got)
		}
	}
}

func TestMarshalUnmarshalBinary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		iprange iprange.IPRange
		wantLen int
	}{
		{mustFromString("1.2.3.4"), 2 * 4},
		{mustFromString("1.2.3.4/24"), 2 * 4},
		{mustFromString("1.2.3.4-6.7.8.9"), 2 * 4},
		{mustFromString("::/0"), 2 * 16},
		{mustFromString("::"), 2 * 16},
		{mustFromString("fe80::ff05:834f:41ff:5de9/10"), 2 * 16},
		{mustFromString("::1-::ff"), 2 * 16},
		{mustFromString("::ffff:1.2.3.4/120"), 2 * 16},
		{iprange.IPRange{}, 0},
	}

	for _, tt := range tests {
		r := tt.iprange
		b, err := r.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if len(b) != tt.wantLen {
			t.Fatalf("%q encoded to size %d; want %d", tt.iprange, len(b), tt.wantLen)
		}
		var r2 iprange.IPRange
		if err := r2.UnmarshalBinary(b); err != nil {
			t.Fatal(err)
		}
		if r != r2 {
			t.Fatalf("got %v; want %v", r2, r)
		}
	}

	// ###
	// test slize size
	var buf [100]byte

	for i := 0; i < len(buf); i++ {
		// base,last: IPv4: 2x4=8 bytes, IPv6: 2x16=32 bytes
		if i == 0 || i == 8 || i == 32 {
			continue
		}

		b := buf[:i]
		var r iprange.IPRange
		err := r.UnmarshalBinary(b)
		if err == nil {
			t.Fatalf("%q decoded from byte slize, len %d; want err, got %v", r, len(b), err)
		}
	}

	// ###
	// last is less than base
	badBinary := [][]byte{
		{3: 1, 7: 0},   // 0.0.0.1-0.0.0.0
		{15: 1, 31: 0}, //::1-::
	}

	for _, data := range badBinary {
		r := iprange.IPRange{}
		if err := r.UnmarshalBinary(data); err == nil {
			t.Fatalf("%q decoded from byte slize %v; want err, got %v", r, data, err)
		}
	}

	// ###
	// only unmarshal into zero Range
	r := mustFromString("10.0.0.0/24")
	if err := r.UnmarshalBinary([]byte{1, 2, 3, 0, 1, 2, 3, 255}); err == nil {
		t.Fatalf("%q decoded from byte slize into non zero range; want err, got %v", r, err)
	}
}

func TestMarshalUnmarshalText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		r          iprange.IPRange
		wantString string
	}{
		{mustFromString("1.2.3.4"), "1.2.3.4/32"},
		{mustFromString("1.2.3.4/24"), "1.2.3.0/24"},
		{mustFromString("1.2.3.4-6.7.8.9"), "1.2.3.4-6.7.8.9"},
		{mustFromString("::/0"), "::/0"},
		{mustFromString("::"), "::/128"},
		{mustFromString("fe80::ff05:834f:41ff:5de9/10"), "fe80::/10"},
		{mustFromString("::-::ff"), "::/120"},
		{mustFromString("::ffff:1.2.3.4/112"), "::ffff:1.2.0.0/112"},
		{iprange.IPRange{}, ""},
	}

	for _, tt := range tests {
		r := tt.r
		b, err := r.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != tt.wantString {
			t.Fatalf("%q encoded to '%s'; want %s", tt.r, b, tt.wantString)
		}
		var r2 iprange.IPRange
		if err := r2.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}
		if r != r2 {
			t.Fatalf("got %v; want %v", r2, r)
		}
	}

	// ###
	// only unmarshal into zero Range
	r := mustFromString("10.0.0.0/24")
	if err := r.UnmarshalText([]byte{1, 2, 3, 0, 1, 2, 3, 255}); err == nil {
		t.Fatalf("%q decoded from byte slize into non zero range; want err, got %v", r, err)
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()
	tests := []struct {
		r1             iprange.IPRange
		r2             iprange.IPRange
		ll, rr, lr, rl int
	}{
		{
			r1: mustFromString("1.2.3.4-1.2.3.5"),
			r2: mustFromString("1.2.3.4-1.2.3.5"),
			ll: 0, rr: 0, lr: -1, rl: +1,
		},
		{
			r1: mustFromString("1.2.3.3-1.2.3.7"),
			r2: mustFromString("1.2.3.4-1.2.3.8"),
			ll: -1, rr: -1, lr: -1, rl: +1,
		},
		{
			r1: mustFromString("1.2.3.4-1.2.3.8"),
			r2: mustFromString("1.2.3.3-1.2.3.7"),
			ll: +1, rr: +1, lr: -1, rl: +1,
		},
		{
			r1: mustFromString("2001:db8::1"),
			r2: mustFromString("fe80::/10"),
			ll: -1, rr: -1, lr: -1, rl: -1,
		},
		{
			r1: mustFromString("fe80::/10"),
			r2: mustFromString("2001:db8::1"),
			ll: 1, rr: 1, lr: 1, rl: 1,
		},
		{
			r1: mustFromString("::1"),
			r2: mustFromString("::1"),
			ll: 0, rr: 0, lr: 0, rl: 0,
		},
	}

	for _, tt := range tests {
		ll, rr, lr, rl := iprange.Compare(tt.r1, tt.r2)
		if !(ll == tt.ll && rr == tt.rr && lr == tt.lr && rl == tt.rl) {
			t.Fatalf("Compare(%s, %s), want: (%v, %v, %v, %v), got: (%v, %v, %v, %v) \n",
				tt.r1, tt.r2, tt.ll, tt.rr, tt.lr, tt.rl, ll, rr, lr, rl)
		}
	}
}
