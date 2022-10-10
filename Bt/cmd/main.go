package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"go_code/Bt/torrent"
	"os"

)

/*
	BT下载器编写流程
			1.编写Bencode库：用来做Bencode的序列化和反序列化
			2.根据Bencode库，编写种子文件(torrent文件)的解析库：得到tracker的url和种子文件的Info
			3.编写tracker模块：和tracker的url做网络交互，获取pears信息
			4.编写download模块：和所有的pears信息交互，下载和校验所有的文件片(piceces)
			5.编写assembler模块：把所有文件片拼装成最终文件
 */


func main(){
	//1.解析torrent文件
	file, err := os.Open(os.Args[1])
	if err != nil{
		fmt.Println("open file error")
		return
	}
	defer file.Close()
	tf, err := torrent.ParseFile(bufio.NewReader(file))
	if err != nil{
		fmt.Println("parse file error")
		return
	}

	//随机peer
	var peerId [torrent.IDLEN]byte
	_, _ = rand.Read(peerId[:])

	//连接tracker并获取peer
	peers := torrent.FindPeers(tf, peerId)
	if len(peers) == 0{
		fmt.Println("can not find peers")
		return
	}

	//生成任务
	task := &torrent.TorrentTask{
		PeerId:   peerId,
		PeerList: peers,
		InfoSHA:  tf.InfoSHA,
		FileName: tf.FileName,
		FileLen:  tf.FileLen,
		PieceLen: tf.PieceLen,
		PieceSHA: tf.PieceSHA,
	}

	//下载并生成文件
	torrent.Download(task)
}