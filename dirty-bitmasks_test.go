package main

import (
	"log"
	"testing"
)

func TestDirtyBitMasks(t *testing.T) {
	Round1 := DirtySectorTracker{}
	Round1.Setup(1e7)
	for i := 0; i < 512; i++ {
		Round1.SetDirty(uint64(i))
	}

	set := make(map[uint64]bool)
	a := Round1.GetDirtySectors()
	for v := range a {
		set[v] = true
	}

	for i := 0; i < 512; i++ {
		if set[uint64(i)] == false {
			log.Printf("Sector %d should have been set as dirty", i)
			t.Fail()
		}
	}

}

func TestDirtyBitMasks2(t *testing.T) {
	Round1 := DirtySectorTracker{}
	Round1.Setup(1e7)
	for i := 0; i < 1024; i++ {
		if i%27 == 0 {

		} else {
			Round1.SetDirty(uint64(i))
		}
	}

	set := make(map[uint64]bool)
	a := Round1.GetDirtySectors()
	for v := range a {
		set[v] = true
	}

	for i := 0; i < 512; i++ {
		if set[uint64(i)] == (i%27 == 0) {
			log.Printf("Sector %d should not have been set", i)
			t.Fail()
		}
	}

}
