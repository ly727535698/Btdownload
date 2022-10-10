package bencode

import (
	"errors"
	"io"
	"reflect"
	"strings"
)

func Unmarshal(r io.Reader, s interface{}) error{
	//从io中读入,并把内容解析成Bobject
	obj, err := Parse(r)
	if err != nil{
		return err
	}

	//
	p := reflect.ValueOf(s)
	//校验s是不是指针类型，因为需要对指向的Slice或Struct进行值修改，必须为指针
	if p.Kind() != reflect.Ptr{
		return errors.New("dest must be a pointer")
	}

	switch obj.btype {
	case Blist:
		//取得obj的val
		list, _ := obj.List()
		//因为传入的slice可能为空或者size、cap不同，所以这里创建了一个新的slice
		l := reflect.MakeSlice(p.Elem().Type(), len(list), len(list))
		//将传入的指针指向新创建的slice
		p.Elem().Set(l)
		err = unmarshalList(p, list)
		if err != nil{
			return err
		}
	case Bdict:
		dict, _ := obj.Dict()
		//传入一个struct，一定会指向struct，所以可以直接使用
		err = unmarshalDict(p, dict)
		if err != nil{
			return err
		}
	default:
		return errors.New("src code must be struct or slice")
	}
	return nil
}

func unmarshalList(p reflect.Value, list []*Bobject) error  {
	//判断p是否为指针类型，且p的元素要是切片类型
	if p.Kind() != reflect.Ptr || p.Elem().Type().Kind() != reflect.Slice{
		return errors.New("dest must be pointer to slice")
	}
	if len(list) == 0{
		return nil
	}

	v := p.Elem()
	//根据list的第一个元素的类型判断（？）
	switch list[0].btype {
	case Bstr:
		for i, obj := range list{
			val, err := obj.Str()
			if err != nil{
				return err
			}
			v.Index(i).SetString(val)
		}
	case Bint:
		for i, obj := range list{
			val, err := obj.Int()
			if err != nil{
				return err
			}
			v.Index(i).SetInt(int64(val))
		}
	case Blist:
		for i, obj := range list{
			val, err := obj.List()
			if err != nil{
				return err
			}
			//p中的元素v如果也是list，就需要再次判断是不是slice类型
			if v.Type().Elem().Kind() != reflect.Slice{
				return nil
			}
			//创建一个slice空指针
			lp := reflect.New(v.Type().Elem())
			//原因同上
			ls := reflect.MakeSlice(v.Type().Elem(), len(val),len(val))
			//让lp指向ls
			lp.Elem().Set(ls)
			err = unmarshalList(lp,val)
			if err != nil{
				return err
			}

			v.Index(i).Set(lp.Elem())
		}
	case Bdict:
		for i, obj := range list{
			val, err := obj.Dict()
			if err != nil{
				return nil
			}
			if v.Type().Elem().Kind() != reflect.Struct{
				return ErrTyp
			}
			//创建一个struct空指针
			dp := reflect.New(v.Type().Elem())
			err = unmarshalDict(dp, val)
			if err != nil {
				return err
			}
			v.Index(i).Set(dp.Elem())
		}
	}
	return nil
}

func unmarshalDict(p reflect.Value, dict map[string]*Bobject) error{
	//判断p是否为指针类型，且p的元素要是切片类型
	if p.Kind() != reflect.Ptr || p.Elem().Type().Kind() != reflect.Struct{
		return errors.New("dest must be pointer to struct")
	}

	v := p.Elem()
	n := v.NumField()	//得到v的字段数
	//遍历v的所有字段
	for i := 0; i < n; i++{
		fv := v.Field(i)
		//如果这个字段不能被修改就遍历下个字段
		if !fv.CanSet(){
			continue
		}
		//得到map的key
		ft := v.Type().Field(i)
		key := ft.Tag.Get("bencode")
		if key == ""{
			key = strings.ToLower(ft.Name)
		}

		obj := dict[key]
		if obj == nil{
			continue
		}

		switch obj.btype {
		case Bstr:
			if ft.Type.Kind() != reflect.String{
				break
			}
			val, _ := obj.Str()
			fv.SetString(val)
		case Bint:
			if ft.Type.Kind() != reflect.Int{
				break
			}
			val, _ := obj.Int()
			fv.SetInt(int64(val))
		case Blist:
			if ft.Type.Kind() != reflect.Slice{
				break
			}
			list, _ := obj.List()
			lp := reflect.New(ft.Type)
			ls := reflect.MakeSlice(ft.Type, len(list), len(list))
			lp.Elem().Set(ls)
			err := unmarshalList(lp, list)
			if err != nil{
				break
			}
			fv.Set(lp.Elem())
		case Bdict:
			if ft.Type.Kind() != reflect.Struct{
				break
			}
			dict, _ := obj.Dict()
			dp := reflect.New(ft.Type)
			err := unmarshalDict(dp, dict)
			if err != nil{
				break
			}
			fv.Set(dp.Elem())
		}
	}
	return nil
}
