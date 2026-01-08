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

/**
* putUint16
* @param b []byte, v uint16
* @return void
**/
func putUint16(b []byte, v uint16) {
	binary.BigEndian.PutUint16(b, v)
}

/**
* @param b []byte
* @return uint32
**/
func getUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

/**
* @param b []byte
* @return uint16
**/
func getUint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

/**
* chunkKeys
* @param keys []string - the slice of keys to chunk
* @param workers int - number of worker goroutines
* @return [][]string - chunked keys
**/
func chunkKeys(keys []string, workers int) [][]string {
	if workers <= 0 {
		return nil
	}

	chunks := make([][]string, workers)
	for i, k := range keys {
		idx := i % workers
		chunks[idx] = append(chunks[idx], k)
	}
	return chunks
}
