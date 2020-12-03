package zcstream

import "github.com/gamexg/go-mempool"

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
