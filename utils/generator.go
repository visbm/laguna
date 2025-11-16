package utils

import "sync/atomic"

type Generator struct {
	id atomic.Uint64
}

func NewIDGenerator() *Generator {
	return &Generator{}
}

func NewIDGeneratorWithStart(st uint64) *Generator {
	g := NewIDGenerator()
	g.id.Store(st)
	return g
}

func (tx *Generator) NextID() uint64 {
	return tx.id.Add(1)
}
