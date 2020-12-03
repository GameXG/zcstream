package zcstream

import (
	"fmt"
	"io"
	"sync"
)

// 零拷贝读
// 内部包含一个缓冲区，数据读取到缓冲区。
// 调用者使用数据时，直接提供对缓冲区的引用，节省一次拷贝。
// 和 ZeroCopyReadStream 的区别是， ZeroCopyReadStream 每次读取会尝试读满缓冲区，而 BufferRead 只读取调用者要求的数据的长度。
type BufferRead struct {
	m              sync.Mutex
	r              io.Reader
	buf            []byte
	defaultBufSize int
	maxSize        int
	memPool        MemPool
}

// 初始化读缓冲
// 参数：
//		r			数据来源
//		bufSize		内部缓冲区默认大小，小于0则使用默认值 4096
//		maxSize		安全限制，单次允许读取的最大大小，0 表示不存在限制
//		malloc		内存申请函数，允许返回比 size 大的内存
func NewBufferRead(r io.Reader, defaultBufSize int, maxSize int, memPool MemPool) *BufferRead {
	if defaultBufSize <= 0 {
		defaultBufSize = 4096
	}

	if memPool == nil {
		memPool = defaultMemPool
	}

	return &BufferRead{
		r: r,
		//buf:            make([]byte, defaultBufSize),
		defaultBufSize: defaultBufSize,
		maxSize:        maxSize,
		memPool:        memPool,
	}
}

// 读取特定长度数据
// 数据会读取到内部缓冲区
// 注意：重复调用必定会使得上次的输出失效
//	参数：
//		size		必须要读取到的数据长度，如果读取不到这个长度，则会返回读取到的数据+错误信息
//	返回值：
//		[]byte		读取到的数据，使用的内部缓冲区，注意，下次调用时数据会被覆盖
//		error		是否发生了读取错误，即使发生了错误，[]byte也可能有数据
func (s *BufferRead) BufferReadFull(size int) ([]byte, error) {
	s.m.Lock()
	defer s.m.Unlock()

	r := s.r
	buf := s.buf
	defaultSize := s.defaultBufSize
	maxSize := s.maxSize
	memPool := s.memPool

	if len(buf) < size {
		newSize := defaultSize

		if newSize < size {
			newSize = size
		}

		if maxSize > 0 && newSize > maxSize {
			return nil, fmt.Errorf("size %v  > maxSize %v", size, maxSize)
		}

		memPool.Free(buf)
		buf = memPool.Malloc(newSize)
		s.buf = buf
	}

	n, err := io.ReadFull(r, buf[:size])

	return buf[:n], err
}

// 释放资源
// 不关闭底层连接
func (s *BufferRead) Close() {
	s.Free()
}

func (s *BufferRead) Free() {
	s.m.Lock()
	defer s.m.Unlock()

	s.memPool.Free(s.buf)
	s.buf = nil
}
