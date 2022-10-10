package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"go_code/Bt/bencode"
	"io"

)

type rawInfo struct {
	Length 			int		`bencode:"length"`
	Name			string	`bencode:"name"`
	PieceLength		int		`bencode:"piece length"`
	Pieces			string	`bencode:"pieces"`
}

//未经过加工的种子文件
type rawFile struct {
	Announce string		`bencode:"announce"`	//tracker的URL
	Info 	rawInfo		`bencode:"info"`
}

const SHALEN int = 20

type TorrentFile struct{
	Announce 	string			//tracker的URL
	InfoSHA 	[SHALEN]byte	//File的唯一标识
	FileName 	string			//制作本地文件时的文件名
	FileLen 	int				//tracker交互、校验用到。根据filelen可以计算还需下载多少。。
	//以下两个字段是校验时用到的
	PieceLen 	int
	PieceSHA 	[][SHALEN]byte
}

func ParseFile(r io.Reader)(*TorrentFile, error){
	raw := new(rawFile)
	err := bencode.Unmarshal(r, raw)
	if err != nil{
		fmt.Println("Fail to parse torrent file")
		return nil, err
	}

	ret := new(TorrentFile)
	ret.Announce = raw.Announce
	ret.FileName = raw.Info.Name
	ret.FileLen = raw.Info.Length
	ret.PieceLen = raw.Info.PieceLength

	//计算info的SHA
	buf := new(bytes.Buffer)
	wlen := bencode.Marshal(buf, raw.Info)
	if wlen == 0{
		fmt.Println("raw file info error")
	}
	ret.InfoSHA = sha1.Sum(buf.Bytes())

	// 计算pieces的SHA
	bys := []byte(raw.Info.Pieces)
	cnt := len(bys) / SHALEN
	hashes := make([][SHALEN]byte,cnt)
	for i := 0; i < cnt; i++{
		copy(hashes[i][:], bys[i * SHALEN : (i + 1) * SHALEN])
	}
	ret.PieceSHA =hashes

	return ret, nil

}