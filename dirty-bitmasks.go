package main

import (
	"math/bits"
	"sync"
)

// This will need 270MB~ per TB of disk being tracked
type DirtySectorTracker struct {
	TotalSizeOfDevice uint64
	Sectors           uint64
	dirtyTracker      []uint64
	// DirtySectors: CountDirty() must be called for this number to be updated
	DirtySectors int
	lock         *sync.Mutex
}

func (d *DirtySectorTracker) Setup(diskSize uint64) {
	d.lock = &sync.Mutex{}
	d.dirtyTracker = make([]uint64, ((diskSize)/64)+2)
}

func (d *DirtySectorTracker) SetDirty(sector uint) {
	arrayTarget := sector / 64
	bitTarget := sector % 64
	d.lock.Lock()
	defer d.lock.Unlock()
	block := d.dirtyTracker[arrayTarget]
	block |= (1 << bitTarget)
	d.dirtyTracker[arrayTarget] = block
}

func (d *DirtySectorTracker) CountDirty() {
	dirty := 0
	for i := 0; i < len(d.dirtyTracker); i++ {
		if d.dirtyTracker[i] != 0 {
			dirty += bits.OnesCount64(d.dirtyTracker[i])
		}
	}

	d.DirtySectors = dirty
}

// GetDirtyPages Gives a full list of sectors (in order) that have been marked as dirty
func (d *DirtySectorTracker) GetDirtyPages() chan uint64 {
	o := make(chan uint64)
	return o
}
