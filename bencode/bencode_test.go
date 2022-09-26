package bencode

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestString(t *testing.T){
	val := "abc"
	buf := new(bytes.Buffer)
	wlen := EncodeString(buf, val)
	assert.Equal(t, 5, wlen)

	val = ""
	for i := 0; i < 20; i++ {
		val += string(byte('a' + i))
	}
	buf.Reset()
	wlen = EncodeString(buf, val)
	assert.Equal(t, 23, wlen)
	str, _ := DecodeString(buf)
	assert.Equal(t, val, str)
}

func TestInt(t *testing.T) {
	val := 999
	buf := new(bytes.Buffer)
	wLen := EncodeInt(buf, val)
	assert.Equal(t, 5, wLen)
	iv, _ := DecodeInt(buf)
	assert.Equal(t, val, iv)

	val = 0
	buf.Reset()
	wLen = EncodeInt(buf, val)
	assert.Equal(t, 3, wLen)
	iv, _ = DecodeInt(buf)
	assert.Equal(t, val, iv)

	val = -99
	buf.Reset()
	wLen = EncodeInt(buf, val)
	assert.Equal(t, 5, wLen)
	iv, _ = DecodeInt(buf)
	assert.Equal(t, val, iv)
}