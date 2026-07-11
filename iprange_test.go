package iprange_test

import (
	"net/netip"
	"slices"
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

func TestPrefixes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   iprange.IPRange
		want []netip.Prefix
	}{
		{
			name: "zero value returns nil",
			in:   iprange.IPRange{},
			want: nil,
		},
		{
			name: "single IPv4 host",
			in:   mustFromString("10.0.0.1"),
			want: []netip.Prefix{mustParsePrefix("10.0.0.1/32")},
		},
		{
			name: "single IPv6 host",
			in:   mustFromString("2001:db8::1"),
			want: []netip.Prefix{mustParsePrefix("2001:db8::1/128")},
		},
		{
			name: "two adjacent IPv4 addresses spanning a /31",
			in:   mustFromString("10.0.0.0-10.0.0.1"),
			want: []netip.Prefix{mustParsePrefix("10.0.0.0/31")},
		},
		{
			name: "non-aligned 3-address IPv4 range splits into two prefixes",
			in:   mustFromString("10.0.0.1-10.0.0.3"),
			want: []netip.Prefix{
				mustParsePrefix("10.0.0.1/32"),
				mustParsePrefix("10.0.0.2/31"),
			},
		},
		{
			name: "IPv4 range crossing /24 boundary",
			in:   mustFromString("10.0.0.128-10.0.1.127"),
			want: []netip.Prefix{
				mustParsePrefix("10.0.0.128/25"),
				mustParsePrefix("10.0.1.0/25"),
			},
		},
		{
			name: "small IPv6 non-CIDR range",
			in:   mustFromString("2001:db8::1-2001:db8::3"),
			want: []netip.Prefix{
				mustParsePrefix("2001:db8::1/128"),
				mustParsePrefix("2001:db8::2/127"),
			},
		},
		{
			name: "IPv6 range aligned to /48",
			in:   mustFromString("2001:db8::/48"),
			want: []netip.Prefix{mustParsePrefix("2001:db8::/48")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := slices.Collect(tt.in.Prefixes())
			if !slices.Equal(got, tt.want) {
				t.Errorf("Prefixes() for %q\n got:  %v\n want: %v", tt.in, got, tt.want)
			}
		})
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
		if !slices.Equal(tt.want, rs) {
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

	if !slices.Equal(got, want) {
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

	if !slices.Equal(got, want) {
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
	if !slices.Equal(rs, wantRs) {
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

	if !slices.Equal(rs, want) {
		t.Errorf("Remove for IANAv6 blocks, got %v, want %v", rs, want)
	}
}

func TestMarshalUnmarshalBinary(t *testing.T) {
	t.Parallel()

	t.Run("ValidRoundtrips", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name  string
			input iprange.IPRange
		}{
			{"ZeroValue", iprange.IPRange{}},
			{"IPv4SingleIP", mustFromString("1.2.3.4")},
			{"IPv4Prefix", mustFromString("1.2.3.0/24")},
			{"IPv4Range", mustFromString("1.2.3.4-6.7.8.9")},
			{"IPv6Zero", mustFromString("::/0")},
			{"IPv6Local", mustFromString("::1")},
			{"IPv6Prefix", mustFromString("fe80::/10")},
			{"IPv6Range", mustFromString("::1-::ff")},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				data, err := tt.input.MarshalBinary()
				if err != nil {
					t.Fatalf("MarshalBinary failed: %v", err)
				}

				var decoded iprange.IPRange
				if err := decoded.UnmarshalBinary(data); err != nil {
					t.Fatalf("UnmarshalBinary failed: %v", err)
				}

				if decoded != tt.input {
					t.Errorf("decoded range does not match: got %v, want %v", decoded, tt.input)
				}
			})
		}
	})

	t.Run("ExpectedBinaryLength", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name       string
			input      iprange.IPRange
			wantLength int
		}{
			{"ZeroValue", iprange.IPRange{}, 0},
			{"IPv4Range", mustFromString("1.2.3.4-1.2.3.10"), 8}, // 2 * 4 bytes
			{"IPv6Range", mustFromString("::1-::ff"), 32},        // 2 * 16 bytes
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				data, err := tt.input.MarshalBinary()
				if err != nil {
					t.Fatalf("MarshalBinary failed: %v", err)
				}
				if len(data) != tt.wantLength {
					t.Errorf("incorrect byte length: got %d, want %d", len(data), tt.wantLength)
				}
			})
		}
	})

	t.Run("UnmarshalErrors", func(t *testing.T) {
		t.Parallel()

		t.Run("NilReceiver", func(t *testing.T) {
			t.Parallel()
			var r *iprange.IPRange
			if err := r.UnmarshalBinary([]byte{1, 2, 3, 4, 5, 6, 7, 8}); err == nil {
				t.Error("expected error when UnmarshalBinary on nil receiver, got nil")
			}
		})

		t.Run("NonZeroReceiver", func(t *testing.T) {
			t.Parallel()
			r := mustFromString("1.2.3.4")
			if err := r.UnmarshalBinary([]byte{1, 2, 3, 4, 5, 6, 7, 8}); err == nil {
				t.Error("expected error when UnmarshalBinary into non-zero receiver, got nil")
			}
		})

		t.Run("EmptyAndNilBinary", func(t *testing.T) {
			t.Parallel()
			var r1 iprange.IPRange
			if err := r1.UnmarshalBinary(nil); err != nil {
				t.Errorf("unexpected error on UnmarshalBinary(nil): %v", err)
			}
			if r1 != (iprange.IPRange{}) {
				t.Errorf("expected zero value after UnmarshalBinary(nil), got %v", r1)
			}

			var r2 iprange.IPRange
			if err := r2.UnmarshalBinary([]byte{}); err != nil {
				t.Errorf("unexpected error on UnmarshalBinary([]byte{}): %v", err)
			}
			if r2 != (iprange.IPRange{}) {
				t.Errorf("expected zero value after UnmarshalBinary([]byte{}), got %v", r2)
			}
		})

		t.Run("InvalidBufferLength", func(t *testing.T) {
			t.Parallel()
			var buf [100]byte
			for length := 1; length <= len(buf); length++ {
				if length == 8 || length == 32 {
					continue // these are valid lengths
				}
				var r iprange.IPRange
				if err := r.UnmarshalBinary(buf[:length]); err == nil {
					t.Errorf("expected error for invalid buffer length %d, got nil", length)
				}
			}
		})

		t.Run("InvalidIPOrder", func(t *testing.T) {
			t.Parallel()
			badBinary := [][]byte{
				{0, 0, 0, 1, 0, 0, 0, 0}, // IPv4: 0.0.0.1-0.0.0.0
			}

			badIPv6 := make([]byte, 32)
			badIPv6[15] = 1 // first = ::1, last = ::
			badBinary = append(badBinary, badIPv6)

			for _, data := range badBinary {
				var r iprange.IPRange
				if err := r.UnmarshalBinary(data); err == nil {
					t.Errorf("expected error when last address is less than first address, got nil (data: %v)", data)
				}
			}
		})
	})
}

func TestMarshalUnmarshalText(t *testing.T) {
	t.Parallel()

	t.Run("ValidRoundtrips", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name  string
			input iprange.IPRange
			want  string
		}{
			{"ZeroValue", iprange.IPRange{}, ""},
			{"IPv4SingleIP", mustFromString("1.2.3.4"), "1.2.3.4/32"},
			{"IPv4Prefix", mustFromString("1.2.3.0/24"), "1.2.3.0/24"},
			{"IPv4Range", mustFromString("1.2.3.4-6.7.8.9"), "1.2.3.4-6.7.8.9"},
			{"IPv6Zero", mustFromString("::/0"), "::/0"},
			{"IPv6Local", mustFromString("::1"), "::1/128"},
			{"IPv6Prefix", mustFromString("fe80::/10"), "fe80::/10"},
			{"IPv6Range", mustFromString("::1-::ff"), "::1-::ff"},
			{"IPv6PrefixRange", mustFromString("::-::ff"), "::/120"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				data, err := tt.input.MarshalText()
				if err != nil {
					t.Fatalf("MarshalText failed: %v", err)
				}

				if string(data) != tt.want {
					t.Errorf("MarshalText output mismatch: got %q, want %q", string(data), tt.want)
				}

				var decoded iprange.IPRange
				if err := decoded.UnmarshalText(data); err != nil {
					t.Fatalf("UnmarshalText failed: %v", err)
				}

				if decoded != tt.input {
					t.Errorf("decoded range does not match: got %v, want %v", decoded, tt.input)
				}
			})
		}
	})

	t.Run("UnmarshalErrors", func(t *testing.T) {
		t.Parallel()

		t.Run("NilReceiver", func(t *testing.T) {
			t.Parallel()
			var r *iprange.IPRange
			if err := r.UnmarshalText([]byte("1.2.3.4")); err == nil {
				t.Error("expected error when UnmarshalText on nil receiver, got nil")
			}
		})

		t.Run("NonZeroReceiver", func(t *testing.T) {
			t.Parallel()
			r := mustFromString("1.2.3.4")
			if err := r.UnmarshalText([]byte("1.2.3.4")); err == nil {
				t.Error("expected error when UnmarshalText into non-zero receiver, got nil")
			}
		})

		t.Run("EmptyAndNilText", func(t *testing.T) {
			t.Parallel()
			var r1 iprange.IPRange
			if err := r1.UnmarshalText(nil); err != nil {
				t.Errorf("unexpected error on UnmarshalText(nil): %v", err)
			}
			if r1 != (iprange.IPRange{}) {
				t.Errorf("expected zero value after UnmarshalText(nil), got %v", r1)
			}

			var r2 iprange.IPRange
			if err := r2.UnmarshalText([]byte("")); err != nil {
				t.Errorf("unexpected error on UnmarshalText([]byte(\"\")): %v", err)
			}
			if r2 != (iprange.IPRange{}) {
				t.Errorf("expected zero value after UnmarshalText([]byte(\"\")), got %v", r2)
			}
		})

		t.Run("InvalidText", func(t *testing.T) {
			t.Parallel()
			invalidInputs := []string{
				"invalid-ip-range",
				"1.2.3.4-",
				"1.2.3.4-5.6.7.8-9.10.11.12",
				"1.2.3.4/999",
				"1.2.3.4-5.6.7",
				"1.2.3.4-::1",
				"fe80::1%eth0-fe80::2",
			}

			for _, input := range invalidInputs {
				t.Run(input, func(t *testing.T) {
					t.Parallel()
					var r iprange.IPRange
					if err := r.UnmarshalText([]byte(input)); err == nil {
						t.Errorf("expected error when UnmarshalText with input %q, got nil", input)
					}
				})
			}
		})
	})
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
		//nolint:staticcheck // De Morgan conversion reduces readability here
		if !(ll == tt.ll && rr == tt.rr && lr == tt.lr && rl == tt.rl) {
			t.Fatalf("Compare(%s, %s), want: (%v, %v, %v, %v), got: (%v, %v, %v, %v) \n",
				tt.r1, tt.r2, tt.ll, tt.rr, tt.lr, tt.rl, ll, rr, lr, rl)
		}
	}
}
