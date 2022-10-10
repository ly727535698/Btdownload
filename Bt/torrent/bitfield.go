package torrent

import "strconv"

type Bitfield []byte


func (bitfield Bitfield) HasPiece(index int)bool{
	byteIndex := index / 8
	offset := index % 8
	if byteIndex < 0 || byteIndex >= len(bitfield){
		return false
	}
	return bitfield[byteIndex] >> uint(7 - offset) & 1 != 0
}

func (bitfield Bitfield) SetPiece(index int){
	byteIndex := index / 8
	offset := index % 8
	if byteIndex < 0 || byteIndex >= len(bitfield){
		return
	}
	bitfield[byteIndex] |= 1 << uint(7 - offset)
}

func (bitfield Bitfield) String() string {
	str := "piece# "
	for i := 0; i < len(bitfield)*8; i++ {
		if bitfield.HasPiece(i) {
			str = str + strconv.Itoa(i) + " "
		}
	}
	return str
}