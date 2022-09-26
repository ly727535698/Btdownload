package bencode

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func objAssertStr(t *testing.T, expect string, o *Bobject) {
	assert.Equal(t, Bstr, o.btype)
	str, err := o.Str()
	assert.Equal(t, nil, err)
	assert.Equal(t, expect, str)
}

func objAssertInt(t *testing.T, expect int, o *Bobject) {
	assert.Equal(t, Bint, o.btype)
	val, err := o.Int()
	assert.Equal(t, nil, err)
	assert.Equal(t, expect, val)
}

func TestParseString(t *testing.T) {
	var o *Bobject
	in := "3:abc"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	objAssertStr(t, "abc", o)

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestParseInt(t *testing.T) {
	var o *Bobject
	in := "i123e"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	objAssertInt(t, 123, o)

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestParseList(t *testing.T) {
	var o *Bobject
	var list []*Bobject
	in := "li123e6:archeri789ee"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	assert.Equal(t, Blist, o.btype)
	list, err := o.List()
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(list))
	objAssertInt(t, 123, list[0])
	objAssertStr(t, "archer", list[1])
	objAssertInt(t, 789, list[2])

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestParseMap(t *testing.T) {
	var o *Bobject
	var dict map[string]*Bobject
	in := "d4:name6:archer3:agei29ee"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	assert.Equal(t, Bdict, o.btype)
	dict, err := o.Dict()
	assert.Equal(t, nil, err)
	objAssertStr(t, "archer", dict["name"])
	objAssertInt(t, 29, dict["age"])

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
}

func TestParseComMap(t *testing.T) {
	var o *Bobject
	var dict map[string]*Bobject
	in := "d4:userd4:name6:archer3:agei29ee5:valueli80ei85ei90eee"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	assert.Equal(t, Bdict, o.btype)
	dict, err := o.Dict()
	assert.Equal(t, nil, err)
	assert.Equal(t, Bdict, dict["user"].btype)
	assert.Equal(t, Blist, dict["value"].btype)
}
