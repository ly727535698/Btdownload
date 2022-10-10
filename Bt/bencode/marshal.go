package bencode

import (
	"io"
	"reflect"
	"strings"
)

func Marshal(w io.Writer, s interface{}) int{
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr{
		v = v.Elem()
	}
	return  marshalValue(w, v)
}

func marshalValue(w io.Writer, v reflect.Value) int{
	len := 0
	switch  v.Kind() {
	case reflect.String:
		len += EncodeString(w, v.String())
	case reflect.Int:
		len += EncodeInt(w, int(v.Int()))
	case reflect.Slice:
		len += marshalList(w, v)
	case reflect.Struct:
		len += marshalDict(w, v)
	}
	return len
}

func marshalList(w io.Writer, vl reflect.Value) int{
	len := 2
	w.Write([]byte{'l'})
	for i := 0; i < vl.Len(); i++{
		ev := vl.Index(i)
		len += marshalValue(w, ev)
	}
	w.Write([]byte{'e'})
	return len
}

func marshalDict(w io.Writer, vd reflect.Value) int{
	len := 2
	w.Write([]byte{'d'})
	for i := 0; i < vd.NumField(); i++{
		fv := vd.Field(i)			//value
		ft := vd.Type().Field(i)
		key := ft.Tag.Get("bencode")
		if key == ""{
			key = strings.ToLower(ft.Name)
		}
		len += EncodeString(w, key)
		len += marshalValue(w, fv)
	}
	w.Write([]byte{'e'})
	return len
}