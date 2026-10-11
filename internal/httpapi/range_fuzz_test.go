package httpapi

import "testing"

func FuzzByteRangeBounds(f *testing.F) {
	for _, value := range []string{"", "bytes=0-", "bytes=-1", "bytes=3-9223372036854775807", "bytes=0-1,3-5"} {
		f.Add(value, uint64(20))
	}
	f.Fuzz(func(t *testing.T, value string, size uint64) {
		// The payload contract supports up to 1 TiB.
		n := int64(size % (1 << 40))
		start, end, status, err := byteRange(value, n)
		if err == nil && (start < 0 || end < start || end > n || (status != 200 && status != 206) || (status == 206 && start == end)) {
			t.Fatalf("invalid accepted range %d %d %d / %d", start, end, status, n)
		}
	})
}
