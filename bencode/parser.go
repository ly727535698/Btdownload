package bencode

import (
	"bufio"
	"io"
)
//Bencode -> Bobejct
func Parse(r io.Reader) (*Bobject, error){
	br, ok := r.(*bufio.Reader)
	if !ok{
		br = bufio.NewReader(r)
	}
	//从缓冲中，得到长度为1的byte切片，但不取出，不会破坏文件完整性
	b, err := br.Peek(1)
	if err != nil{
		return nil, err
	}

	var obj Bobject

	switch  {
	//string
	case b[0] >= '0' && b[0] <= '9':
		val, err := DecodeString(br)
		if err != nil{
			return nil, err
		}
		obj.btype = Bstr
		obj.bval = val

	//int
	case b[0] == 'i':
		val, err := DecodeInt(br)
		if err != nil{
			return nil, err
		}
		obj.btype = Bint
		obj.bval = val

	//list
	case b[0] == 'l':
		br.ReadByte()		//先取出'l', 'l'后是要转换的数据
		var list []*Bobject

		for{
			//当循环到'e'时，代表list全部转化完成
			p, _ := br.Peek(1)
			if p[0] == 'e'{
				br.ReadByte()
				break
			}

			objs, err := Parse(br)
			if err != nil{
				return nil, err
			}
			list = append(list, objs)
		}
		obj.btype = Blist
		obj.bval = list
	case b[0] == 'd':
		br.ReadByte()		//取出'd'
		dict := make(map[string]*Bobject)
		for {
			p, _ := br.Peek(1)
			if p[0] == 'e'{
				br.ReadByte()
				break
			}

			key, err := DecodeString(br)
			if err != nil{
				return nil, err
			}

			val, err := Parse(br)
			if err != nil{
				return nil, err
			}
			dict[key] = val
		}
		obj.btype = Bdict
		obj.bval = dict
	default:
		return nil, ErrIvd
	}
	return &obj, nil
}