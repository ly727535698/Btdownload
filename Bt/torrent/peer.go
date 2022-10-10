package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

/*
Peer通信机制：
	1. 握手：主要是交换信息：（协议，peerId(本机), InfoSHA(想要拿到的)）
		握手消息：1byte(表示第二块的长度：0x13) + 19byte(协议) + 8byte(保留位，全是空的，为协议拓展预留) + 20byte(InfoSHA) + 20byte(peerId)
	2. 获取peer数据情况：BitField表示，peer就是有这个文件的用户，用一个比特位，表示是否拥有这个piece
	3. 指定piece下载
 */

type MsgId uint8

const (
	MsgChoke       MsgId = 0	//只下载
	MsgUnchoke     MsgId = 1	//下载同时上传
	MsgInterested  MsgId = 2	//下载同时上传
	MsgNotInterest MsgId = 3	//不下载只上传
	MsgHave        MsgId = 4	//通知peer新增的piece
	MsgBitfield    MsgId = 5	//消息为Bitfield
	MsgRequest     MsgId = 6	//下载请求：指定下载的pieces，起始位置start，下载的长度
	MsgPiece       MsgId = 7	//返回请求要的piece的byte
	MsgCancel      MsgId = 8
)

type PeerMsg struct{
	Id MsgId			//消息类型
	Payload []byte		//消息内容
}

type PeerConn struct {
	net.Conn
	Choke bool
	bitField Bitfield
	peer PeerInfo
	peerId [IDLEN]byte
	infoSHA [SHALEN]byte
}

//将客户端和对端某个peer的conn抽象成PeerConn
func NewConn(peerInfo PeerInfo, infoSHA [SHALEN]byte, peerId [IDLEN]byte)(*PeerConn, error){
	//获取地址
	addr := net.JoinHostPort(peerInfo.Ip.String(), strconv.Itoa(int(peerInfo.Port)))
	tcpconn, err := net.DialTimeout("tcp", addr, 5 * time.Second)
	if err != nil{
		fmt.Println("set tcp conn failed: " + addr)
		return nil, err
	}

	//握手
	err = handshake(tcpconn, infoSHA, peerId)
	if err != nil{
		fmt.Println("handshake failed")
		tcpconn.Close()
		return nil, err
	}

	peerConn := &PeerConn{
		Conn: tcpconn,
		Choke:   true,
		peer:    peerInfo,
		peerId:  peerId,
		infoSHA: infoSHA,
	}

	//发送一个peerMsg，获取对端的BitField(资源拥有情况)
	err = fillBitfield(peerConn)
	if err != nil{
		fmt.Println("fill bitfield failed, " + err.Error())
		return nil, err
	}
	return peerConn, nil
}

//握手
func handshake(tcpconn net.Conn, infoSHA [SHALEN]byte, peerId [IDLEN]byte) error{
	//设置超时时间
	tcpconn.SetDeadline(time.Now().Add(3 * time.Second))
	defer tcpconn.SetDeadline(time.Time{})
	//1. 通过infoSHA和peerId，生成握手消息
	req := NewHandshakeMsg(infoSHA, peerId)
	_, err := WriteHandShake(tcpconn, req)
	if err != nil{
		fmt.Println("send handshake failed")
		return err
	}
	//读取回复的握手消息
	res, err := ReadHandShake(tcpconn)
	if err != nil{
		fmt.Println("read handshake failed")
		return err
	}

	//
	if !bytes.Equal(res.InfoSHA[:], infoSHA[:]){
		fmt.Println("check handshake failed")
		return fmt.Errorf("handshake msg error: " + string(res.InfoSHA[:]))
	}
	return nil
}

//发送一个peerMsg，获取对端的BitField(资源拥有情况)
func fillBitfield(peerConn *PeerConn) error{
	//设置超时时间
	peerConn.SetDeadline(time.Now().Add(5 * time.Second))
	defer peerConn.SetDeadline(time.Time{})

	msg, err := peerConn.ReadMsg()
	if err != nil{
		return err
	}

	if msg == nil{
		return fmt.Errorf("expected bitfield")
	}

	if msg.Id != MsgBitfield{
		return fmt.Errorf("expected bitfield, get " + strconv.Itoa(int(msg.Id)))
	}
	fmt.Println("fill bitfield : " + peerConn.peer.Ip.String())
	peerConn.bitField = msg.Payload
	return nil
}

func (peerConn *PeerConn)ReadMsg()(*PeerMsg, error){
	//获取msg的长度
	lenBuf := make([]byte,4)
	_, err := io.ReadFull(peerConn, lenBuf)
	if err != nil{
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)
	//keep alive msg
	if length == 0{
		return nil, nil
	}

	//获取消息内容
	msgBuf := make([]byte, length)
	_, err = io.ReadFull(peerConn, msgBuf)
	if err != nil{
		return nil, err
	}
	return &PeerMsg{
		Id:      MsgId(msgBuf[0]),
		Payload: msgBuf[1:],
	}, nil
}

const LenBytes uint32 = 4

//将消息写入conn中
func (peerConn *PeerConn)WriteMsg(msg *PeerMsg)(int, error){
	//var buf []byte
	//if msg == nil{
	//	buf = make([]byte, LenBytes)
	//}
	length := uint32(len(msg.Payload) + 1) 		//类型+内容的长度
	buf := make([]byte, length + LenBytes)
	binary.BigEndian.PutUint32(buf[0 : LenBytes], length)	//储存长度
	buf[LenBytes] = byte(msg.Id)	//将类型写入buf
	copy(buf[LenBytes + 1 :], msg.Payload)	//将内容写入buf
	return peerConn.Write(buf)		//将buf写入conn中
}

//生成请求消息
func NewRequestMsg(index int, offset int, length int) *PeerMsg{
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0 : 4], uint32(index))
	binary.BigEndian.PutUint32(payload[4 : 8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8 : 12], uint32(length))
	return &PeerMsg{MsgRequest, payload}
}

//得到peer所拥有的piece
func GetHaveIndex(msg *PeerMsg) (int, error){
	if msg.Id != MsgHave{
		return 0, fmt.Errorf("expected MsgHave (Id %d), got Id %d", MsgHave, msg.Id)
	}
	if len(msg.Payload) != 4{
		return 0, fmt.Errorf("expected payload length 4, got length %d", len(msg.Payload))
	}

	index := int(binary.BigEndian.Uint32(msg.Payload))
	return index, nil
}

//
func CopyPieceData(index int, buf []byte, msg *PeerMsg)(int, error){
	if msg.Id != MsgPiece{
		return 0, fmt.Errorf("expected MsgPiece (Id %d), got Id %d", MsgPiece, msg.Id)
	}
	if len(msg.Payload) < 8{
		return 0, fmt.Errorf("payload too short. %d < 8", len(msg.Payload))
	}

	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != index{
		return 0, fmt.Errorf("expected index %d, got %d", index, parsedIndex)
	}

	offset := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if offset >= len(buf) {
		return 0, fmt.Errorf("offset too high. %d >= %d", offset, len(buf))
	}
	data := msg.Payload[8:]
	if offset+len(data) > len(buf) {
		return 0, fmt.Errorf("data too large [%d] for offset %d with length %d", len(data), offset, len(buf))
	}
	copy(buf[offset:], data)
	return len(data), nil
}
