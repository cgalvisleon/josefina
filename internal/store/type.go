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
* checksum: Calcula el CRC de b.
* @param b []byte
* @return uint32
**/
func checksum(b []byte) uint32 {
	return crc32.Checksum(b, crcTable)
}

/**
* putUint32: Escribe v en b como uint32 big-endian.
* @param b []byte, v uint32
* @return void
**/
func putUint32(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}

/**
* putUint16: Escribe v en b como uint16 big-endian.
* @param b []byte, v uint16
* @return void
**/
func putUint16(b []byte, v uint16) {
	binary.BigEndian.PutUint16(b, v)
}

/**
* getUint32: Lee un uint32 big-endian de b.
* @param b []byte
* @return uint32
**/
func getUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

/**
* getUint16: Lee un uint16 big-endian de b.
* @param b []byte
* @return uint16
**/
func getUint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}
