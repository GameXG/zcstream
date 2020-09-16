package zcstream

import (
	"fmt"
	"github.com/gamexg/go-mempool"
	"io"
	"sync"
)

type ZeroCopyReadStream struct {
	m              sync.Mutex
	r              io.Reader
	buf            []byte
	dataOffset     int
	dataSize       int
	defaultBufSize int
	maxSize        int
	memPool        MemPool
}

type MemPool interface {
	Malloc(size int) []byte
	Free([]byte)
}

var defaultMemPool = new(memPool)

type memPool struct {
}

func (p *memPool) Malloc(size int) []byte {
	return mempool.Get(size)
}

func (p *memPool) Free(v []byte) {
	_ = mempool.Put(v)
}

// 初始化零拷贝读
// 参数：
//		r			数据来源
//		bufSize		内部缓冲区默认大小，小于0则使用默认值 4096
//		maxSize		安全限制，单次允许读取的最大大小，0 表示不存在限制
//		malloc		内存申请函数，允许返回比 size 大的内存
func NewZeroCopyReadStream(r io.Reader, defaultBufSize int, maxSize int, memPool MemPool) *ZeroCopyReadStream {
	if defaultBufSize <= 0 {
		defaultBufSize = 4096
	}

	if memPool == nil {
		memPool = defaultMemPool
	}

	return &ZeroCopyReadStream{
		r: r,
		//buf:            make([]byte, defaultBufSize),
		dataOffset:     0,
		dataSize:       0,
		defaultBufSize: defaultBufSize,
		maxSize:        maxSize,
		memPool:        memPool,
	}
}

// 释放资源
func (s *ZeroCopyReadStream) Close() {
	s.memPool.Free(s.buf)
	s.buf = nil
}

// 零拷贝读取特定长度数据
// 会使用内部缓冲区
// 注意：重复调用可能使得上次的输出失效
//	参数：
//		size		必须要读取到的数据长度，如果读取不到这个长度，则会返回读取到的数据+错误信息
//	返回值：
//		[]byte		读取到的数据，使用的内部缓冲区，注意，下次调用时数据可能被覆盖
//		error		是否发生了读取错误，即使发生了错误，[]byte也可能有数据
func (s *ZeroCopyReadStream) ZeroCopyReadFull(size int) ([]byte, error) {
	s.m.Lock()
	defer s.m.Unlock()

	r := s.r
	defaultSize := s.defaultBufSize
	maxSize := s.maxSize

	// 缓冲区有足够的数据
	if s.dataSize >= size {
		dataOffset := s.dataOffset
		s.dataOffset += size
		s.dataSize -= size
		return s.buf[dataOffset : dataOffset+size], nil
	}

	// 安全检查
	if maxSize > 0 && size > maxSize {
		return nil, fmt.Errorf("%v exceeds maxSize(%v) limit", size, maxSize)
	}

	// 检查缓冲区剩余空间是否足够
	if size > len(s.buf)-s.dataOffset {
		// 检查是否需要新申请 bug
		if size > len(s.buf) {
			nsize := defaultSize
			if size > nsize {
				nsize = size
			}

			nbuf := s.memPool.Malloc(nsize)
			copy(nbuf, s.buf[s.dataOffset:s.dataOffset+s.dataSize])
			s.dataOffset = 0
			s.memPool.Free(s.buf)
			s.buf = nbuf
		} else {
			// 仅移动数据即可
			copy(s.buf, s.buf[s.dataOffset:s.dataOffset+s.dataSize])
			s.dataOffset = 0
		}

	}

	dstBuf := s.buf[s.dataOffset+s.dataSize:]
	rMin := size - s.dataSize
	n, err := io.ReadAtLeast(r, dstBuf, rMin)
	s.dataSize += n

	osize := s.dataSize + n
	if osize > size {
		osize = size
	}

	dataOffset := s.dataOffset
	s.dataOffset += osize
	s.dataSize -= osize

	return s.buf[dataOffset : dataOffset+osize], err
}
