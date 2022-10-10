package torrent

import (
	"encoding/binary"
	"fmt"
	"go_code/Bt/bencode"

	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

//和tracker交互

const(
	PeerPort int = 6666
	IpLen int = 4
	PortLen int = 2
	PeerLen int = IpLen + PortLen
)

const IDLEN int = 20

type PeerInfo struct {
	Ip		net.IP
	Port	uint16
}
//tracker的响应
type TrackerResp struct {
	Interval	int		`bencode:"interval"`	//间隔
	Peers		string	`bencode:"peers"`
}
//构造url
func buildUrl(tf *TorrentFile, peerId [IDLEN]byte)(string, error){
	base, err := url.Parse(tf.Announce)
	if err != nil{
		fmt.Println("Announce Error: " + tf.Announce)
		return "", err
	}

	params := url.Values{
		"info_hash" : []string{string(tf.InfoSHA[:])},		//文件标识
		"peer_id" 	: []string{string(peerId[:])},			//下载器标识
		"port"		: []string{strconv.Itoa(PeerPort)},		//端口
		"uploaded"	: []string{"0"},						//上传
		"downloaded": []string{"0"},						//下载
		"compact"	: []string{"1"},
		"left"		: []string{strconv.Itoa(tf.FileLen)},	//剩余大小
	}
	//对url编码，生成完整的请求
	base.RawQuery = params.Encode()
	return base.String(), nil
}

//对ip和post做了一个紧凑排列
func buildPeerInfo(peers []byte) []PeerInfo{
	num := len(peers) / PeerLen
	if len(peers) % PeerLen != 0{
		fmt.Println("Received malformed peers")
		return nil
	}

	infos := make([]PeerInfo, num)
	for i := 0; i < num; i++{
		offset := i * PeerLen
		infos[i].Ip = net.IP(peers[offset : offset + IpLen])
		infos[i].Port = binary.BigEndian.Uint16(peers[offset + IpLen : offset + PeerLen])
	}
	return infos
}


func FindPeers(tf *TorrentFile, peerId [IDLEN]byte) []PeerInfo{
	//拿到请求
	url, err := buildUrl(tf, peerId)
	if err != nil{
		fmt.Println("Build Tracker Url Error: " + err.Error())
		return nil
	}

	//发送http的get请求
	cli := &http.Client{Timeout: 15 * time.Second}
	resp, err := cli.Get(url)	//resp也是一个bencode编码，所以需要先反序列化
	if err != nil{
		fmt.Println("Fail to Connect to Tracker: " + err.Error())
		return nil
	}
	defer resp.Body.Close()

	//
	trackResp := new(TrackerResp)
	err = bencode.Unmarshal(resp.Body, trackResp)
	if err != nil{
		fmt.Println("Tracker Response Error" + err.Error())
		return nil
	}

	return buildPeerInfo([]byte(trackResp.Peers))
}