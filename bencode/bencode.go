package bencode

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

/*要求：
	序列化和反序列化
		从io.Reader中读出被bencode序列化后的文本(Bytes)，转化为Bobject
		将Bobject转化为bencode序列化后的文本，写入io.Writer中
 */


var (
	ErrNum = errors.New("expect num")
	ErrCol = errors.New("expect colon")
	ErrEpI = errors.New("expect char i")
	ErrEpE = errors.New("expect char e")
	ErrTyp = errors.New("wrong type")
	ErrIvd = errors.New("invalid bencode")
)

//Bobject表示

type BType uint8

//Bencode的type只有4中，所以byte足够
const(
	Bstr BType = 0x01
	Bint BType = 0x02
	Blist BType = 0x03
	Bdict BType = 0x04
)

//Bencode的value样式多，用interface接口。可用泛型
type BValue interface {

}

type Bobject struct {
	btype BType
	bval BValue
}

//对bval类型断言，返回该Bobject类型的对应类型
func (o *Bobject)Str() (string, error){
	if o.btype != Bstr{
		return "", ErrTyp
	}
	return o.bval.(string), nil
}

func (o *Bobject)Int() (int, error){
	if o.btype != Bint{
		return 0, ErrTyp
	}
	return o.bval.(int), nil
}

func (o *Bobject)List() ([]*Bobject, error){
	if o.btype != Blist{
		return nil, ErrTyp
	}
	return o.bval.([]*Bobject), nil
}

func (o *Bobject)Dict() (map[string]*Bobject, error){
	if o.btype != Bdict{
		return nil, ErrTyp
	}
	return o.bval.(map[string]*Bobject), nil
}

//Bobejct -> Bencode
func (o *Bobject)Bencode(w io.Writer) int{
	bw, ok := w.(*bufio.Writer)
	if !ok{
		bw = bufio.NewWriter(w)
	}
	wlen := 0	//写入writer的总长度

	switch o.btype{
	case Bstr:
		val, _ := o.Str()
		wlen += EncodeString(bw,val)
	case Bint:
		val, _ := o.Int()
		wlen += EncodeInt(bw,val)
	case Blist:
		bw.WriteByte('l')
		val, _ := o.List()
		for _, v := range val{
			wlen += v.Bencode(bw)
		}
		bw.WriteByte('e')
		wlen += 2
	case Bdict:
		bw.WriteByte('d')
		val, _ := o.Dict()
		for k, v := range val{
			wlen += EncodeString(bw, k)	//写入key
			wlen += v.Bencode(bw)		//写入value
		}
		bw.WriteByte('e')
		wlen += 2
	}
	bw.Flush()
	return wlen
}

//工具编写

//将buifo.Reader中的内容读出，转化为一个十进制数
func readDecimal(r *bufio.Reader)(val int, lenth int){
	b, _  := r.ReadByte()
	if b == 'i'{
		str, _ := r.ReadString('e')
		lenth = len(str)
		str  = str[:lenth - 1]
		lenth = len(str)
		val, _ = strconv.Atoi(str)
	}else{
		for b != ':'{
			val = val * 10 + int(b - '0')
			lenth++
			b, _ = r.ReadByte()
		}
		r.UnreadByte()
	}
	return val, lenth
}

//将一个十进制数以byte类型写入bufio.writer缓冲区
func writeDecimal(w *bufio.Writer, val int) (lenth int){
	val_str := strconv.Itoa(val)
	lenth = len(val_str)
	for i := 0; i < lenth; i++{
		w.WriteByte(val_str[i])
	}
	return lenth
}

/*
	string编码过程：
		1.将string长度以byte写入bufio.writer
		2.写入一个':'
		3.写入要写入的string
		4.最后用flush方法，将bufio缓冲中的信息写入io中
 */
func EncodeString(w io.Writer, val string) int{
	lenth := len(val)
	bw, ok := w.(*bufio.Writer)
	if !ok{
		bw = bufio.NewWriter(w)
	}
	wlen := writeDecimal(bw, lenth)

	bw.WriteByte(':')
	wlen++

	bw.WriteString(val)
	wlen += lenth

	err := bw.Flush()
	if err != nil{
		return 0
	}

	return wlen
}

/*
	int编码过程：
		1.写入一个'i'
		1.将val以byte写入bufio.writer
		2.写入一个'e'
		4.最后用flush方法，将bufio缓冲中的信息写入io中
*/
func EncodeInt(w io.Writer, val int) int{
	bw, ok := w.(*bufio.Writer)
	if !ok{
		bw = bufio.NewWriter(w)
	}

	bw.WriteByte('i')
	wlen := 1

	lenth := writeDecimal(bw, val)
	wlen += lenth

	bw.WriteByte('e')
	wlen++

	err := bw.Flush()
	if err != nil{
		return 0
	}
	return wlen
}

/*
	string解码过程：
		1. 判断有无缓冲，若没有则new一个
		2. 将缓冲的数字部分从缓冲中取出
		3. 取出缓冲中的':'
		4. 将缓冲中的字符串取出，字符串长度为num
*/
func DecodeString(r io.Reader)(val string, err error){
	br, ok := r.(*bufio.Reader)
	if !ok{
		br = bufio.NewReader(r)
	}

	num, lenth := readDecimal(br)
	if lenth == 0{
		return val, ErrNum
	}

	b, err := br.ReadByte()
	if b != ':'{
		return val, ErrCol
	}

	buf := make([]byte, num)
	_, err = io.ReadAtLeast(br, buf, num)
	val = string(buf)
	return
}

/*
	int解码过程：
		1. 判断有无缓冲，若没有则new一个
		2. 将缓冲的数字部分从缓冲中取出
*/
func DecodeInt(r io.Reader)(val int, err error){
	br, ok := r.(*bufio.Reader)
	if !ok{
		br = bufio.NewReader(r)
	}
	val, _ = readDecimal(br)
	return
}