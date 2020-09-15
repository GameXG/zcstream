package protocol

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

type TestRead struct {
	r    io.Reader
	size int
}

func NewTestRead(r io.Reader, size int) *TestRead {
	return &TestRead{
		r:    r,
		size: size,
	}
}

func (r *TestRead) Read(p []byte) (int, error) {
	if len(p) > 3 {
		p = p[:3]
	}
	return r.r.Read(p)
}

func TestNewZeroCopyReadStream(t *testing.T) {
	data := make([]byte, 1024)
	_, err := rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}

	zr := NewZeroCopyReadStream(bytes.NewReader(data), 10, -1, nil)

	if zr == nil {
		t.Fatal(zr)
	}

	if zr.r == nil ||
		zr.memPool == nil ||
		zr.defaultBufSize != 10 {
		t.Errorf("%#v", zr)
	}
}

func TestZeroCopyReadStream_ZeroCopyRead(t *testing.T) {
	data := make([]byte, 10)
	_, err := rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}

	// 一点点的测试太麻烦了，
	// 直接暴力遍历所有可能吧
	for defaultBufSize := 0; defaultBufSize < 20; defaultBufSize++ {
		for size := 1; size < 20; size++ {
			for rs := 1; rs < 20; rs++ {
				r := NewTestRead(bytes.NewReader(data), rs)

				res := make([]byte, 0, 10)
				zr := NewZeroCopyReadStream(r, defaultBufSize, 0, nil)
				for {
					lsize := size
					if s := len(data) - len(res); s < lsize && s != 0 {
						lsize = s
					}

					buf, err := zr.ZeroCopyReadFull(lsize)
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
					res = append(res, buf...)
				}
				if bytes.Equal(data, res) == false {
					t.Errorf("\r\n%#v\r\n!=\r\n%#v", data, res)
				}
			}
		}
	}
}
