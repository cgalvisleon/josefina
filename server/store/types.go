package store

import (
	"encoding/binary"
	"hash/crc32"
)

const (
	Active  byte = 1
	Deleted byte = 2
)

var crcTable = crc32.MakeTable(crc32.Castagnoli)

/**
* checksum
* @param b []byte
* @return uint32
**/
func checksum(b []byte) uint32 {
	return crc32.Checksum(b, crcTable)
}

/**
* putUint32
* @param b []byte, v uint32
* @return void
**/
func putUint32(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}
