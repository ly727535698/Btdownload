package torrent

import (
	"fmt"
	"io"
)

/*
握手消息：1byte(表示第二块的长度：0x13) 	part1
		19byte(协议)  				part2
		8byte(保留位，全是空的，为协议拓展预留) 	part3
		20byte(InfoSHA) 			part4
		20byte(peerId)				part5
 */


const(
	Reserved int = 8		//保留位
	HsMsgLen int = SHALEN + IDLEN + Reserved	//握手消息长度(不包含前2part)
)

type HandshakeMsg struct {
	PreStr string	//协议
	InfoSHA [SHALEN]byte
	PeerId [IDLEN]byte
}

func NewHandshakeMsg(infoSHA [SHALEN]byte, peerId [IDLEN]byte) *HandshakeMsg{
	return &HandshakeMsg{
		PreStr:  "BitTorrent protocol",
		InfoSHA: infoSHA,
		PeerId:  peerId,
	}
}

func WriteHandShake(w io.Writer, msg *HandshakeMsg) (int, error){
	//创建缓冲区，长度为握手消息的长度
	buf := make([]byte, len(msg.PreStr) + HsMsgLen + 1)
	buf[0] = byte(len(msg.PreStr))	//给第一位赋值，为协议的长度
	curr := 1
	curr += copy(buf[curr:], []byte(msg.PreStr))	//把协议part写入缓冲区
	curr += copy(buf[curr:], make([]byte, Reserved))	//加上8byte的保留位，值为空
	curr += copy(buf[curr:], msg.InfoSHA[:])	//把infoSHA写入缓冲区
	curr += copy(buf[curr:], msg.PeerId[:])		//把PeerId写入缓冲区
	return w.Write(buf)
}

func ReadHandShake(r io.Reader)(*HandshakeMsg, error){
	lenBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lenBuf)	//从r中读取长度为len(lenBuf)，填充入lenBuf
	if err != nil{
		return nil, err
	}

	//取出part1
	prelen := int(lenBuf[0])
	if prelen == 0{
		err := fmt.Errorf("prelen cannot be 0")
		return nil, err
	}

	//取出part2
	preBuf := make([]byte, prelen)
	_, err = io.ReadFull(r, preBuf)
	if err != nil{
		return nil, err
	}

	//取出part3
	resBuf := make([]byte, Reserved)
	_, err = io.ReadFull(r, resBuf)
	if err != nil{
		return nil, err
	}

	//取出part4
	infoBuf := make([]byte, SHALEN)
	_, err = io.ReadFull(r, infoBuf)
	if err != nil{
		return nil, err
	}
	//将切片中的值拷贝到数组中，得到和切片值相同的数组用于返回
	var infoSHA [SHALEN]byte
	copy(infoSHA[:], infoBuf)

	//取出part5
	peerIdBuf := make([]byte, IDLEN)
	_, err = io.ReadFull(r, peerIdBuf)
	if err != nil{
		return nil, err
	}
	var peerId [IDLEN]byte
	copy(peerId[:], peerIdBuf)

	return &HandshakeMsg{
		PreStr:  string(resBuf),
		InfoSHA: infoSHA,
		PeerId:  peerId,
	}, nil
}