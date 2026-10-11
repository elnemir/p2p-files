// Package bundle produces a deterministic, uncompressed ZIP64 without temporary plaintext.
package bundle

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"filex.local/filex/internal/domain"
	"hash/crc32"
	"io"
	"strings"
)

type Entry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	CRC32  uint32 `json:"crc32"`
	FileID string `json:"fileId,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	Data   []byte `json:"data,omitempty"`
}
type Bundle struct {
	Entries []Entry `json:"entries"`
	KeyID   string  `json:"keyId"`
}

func New(entries []Entry, seed []byte, keyID string) (Bundle, error) {
	if len(seed) != ed25519.SeedSize || len(entries) == 0 || len(entries) > 100000 {
		return Bundle{}, errors.New("invalid bundle")
	}
	var sums strings.Builder
	for _, e := range entries {
		if _, err := domain.PathKey(e.Path, 4096); err != nil || !domain.ValidChecksum(e.SHA256) || e.Size < 0 {
			return Bundle{}, domain.ErrInvalid
		}
		sums.WriteString(e.SHA256 + "  " + e.Path + "\n")
	}
	key := ed25519.NewKeyFromSeed(seed)
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(key, []byte(sums.String()))) + "\n"
	der, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		return Bundle{}, err
	}
	pub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	out := Bundle{Entries: append([]Entry{}, entries...), KeyID: keyID}
	for _, e := range []struct {
		name string
		data []byte
	}{{"SHA256SUMS.txt", []byte(sums.String())}, {"SHA256SUMS.sig", []byte(sig)}, {"SHA256SUMS.pub", pub}} {
		out.Entries = append(out.Entries, Entry{Path: e.name, Size: int64(len(e.data)), CRC32: crc32.ChecksumIEEE(e.data), Data: e.data})
	}
	return out, nil
}
func put(b *bytes.Buffer, v any) {
	if e := binary.Write(b, binary.LittleEndian, v); e != nil {
		panic(e)
	}
}
func local(e Entry) []byte {
	b := new(bytes.Buffer)
	put(b, uint32(0x04034b50))
	for _, n := range []uint16{45, 0x800, 0, 0, 33} {
		put(b, n)
	}
	put(b, e.CRC32)
	put(b, uint32(0xffffffff))
	put(b, uint32(0xffffffff))
	put(b, uint16(len(e.Path)))
	put(b, uint16(20))
	b.WriteString(e.Path)
	put(b, uint16(1))
	put(b, uint16(16))
	put(b, uint64(e.Size))
	put(b, uint64(e.Size))
	return b.Bytes()
}
func central(e Entry, offset int64) []byte {
	b := new(bytes.Buffer)
	put(b, uint32(0x02014b50))
	for _, n := range []uint16{(3 << 8) | 45, 45, 0x800, 0, 0, 33} {
		put(b, n)
	}
	put(b, e.CRC32)
	put(b, uint32(0xffffffff))
	put(b, uint32(0xffffffff))
	for _, n := range []uint16{uint16(len(e.Path)), 28, 0, 0, 0} {
		put(b, n)
	}
	put(b, uint32(0100600<<16))
	put(b, uint32(0xffffffff))
	b.WriteString(e.Path)
	put(b, uint16(1))
	put(b, uint16(24))
	put(b, uint64(e.Size))
	put(b, uint64(e.Size))
	put(b, uint64(offset))
	return b.Bytes()
}
func footer(count, offset, size int64) []byte {
	b := new(bytes.Buffer)
	put(b, uint32(0x06064b50))
	put(b, uint64(44))
	put(b, uint16(45))
	put(b, uint16(45))
	put(b, uint32(0))
	put(b, uint32(0))
	put(b, uint64(count))
	put(b, uint64(count))
	put(b, uint64(size))
	put(b, uint64(offset))
	put(b, uint32(0x07064b50))
	put(b, uint32(0))
	put(b, uint64(offset+size))
	put(b, uint32(1))
	put(b, uint32(0x06054b50))
	put(b, uint16(0))
	put(b, uint16(0))
	put(b, uint16(0xffff))
	put(b, uint16(0xffff))
	put(b, uint32(0xffffffff))
	put(b, uint32(0xffffffff))
	put(b, uint16(0))
	return b.Bytes()
}
func (b Bundle) Size() int64 {
	var n int64 = 98
	for _, e := range b.Entries {
		n += int64(124+2*len(e.Path)) + e.Size
	}
	return n
}

// WriteRange reads only overlapping payload chunks; central directory bytes are reproducible.
func (b Bundle) WriteRange(start, end int64, w io.Writer, read func(Entry, int64, int64, io.Writer) error) error {
	if start < 0 || end < start || end > b.Size() {
		return domain.ErrInvalid
	}
	var pos int64
	emit := func(data []byte) error {
		from, to := max(start, pos), min(end, pos+int64(len(data)))
		var err error
		if from < to {
			n, e := w.Write(data[from-pos : to-pos])
			err = e
			if e == nil && int64(n) != to-from {
				err = io.ErrShortWrite
			}
		}
		pos += int64(len(data))
		return err
	}
	offsets := make([]int64, len(b.Entries))
	for i, e := range b.Entries {
		offsets[i] = pos
		if err := emit(local(e)); err != nil {
			return err
		}
		from, to := max(start, pos), min(end, pos+e.Size)
		if from < to {
			if e.FileID != "" {
				if err := read(e, from-pos, to-pos, w); err != nil {
					return err
				}
			} else {
				n, err := w.Write(e.Data[from-pos : to-pos])
				if err != nil {
					return err
				}
				if int64(n) != to-from {
					return io.ErrShortWrite
				}
			}
		}
		pos += e.Size
	}
	directory := pos
	for i, e := range b.Entries {
		if err := emit(central(e, offsets[i])); err != nil {
			return err
		}
	}
	return emit(footer(int64(len(b.Entries)), directory, pos-directory))
}
