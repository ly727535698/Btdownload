package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"time"
)

const(
	BLOCKSIZE = 16384
	MAXBACKLOG = 5
)

type TorrentTask struct {
	PeerId		[IDLEN]byte		//客户端ID
	PeerList 	[]PeerInfo		//从tracker获取到的peer
	InfoSHA 	[SHALEN]byte
	FileName 	string
	FileLen		int
	PieceLen 	int
	PieceSHA 	[][SHALEN]byte
}

type pieceTask struct {
	index	int
	sha		[SHALEN]byte
	length	int
}

//下载的中间状态
type taskState struct {
	index		int
	conn		*PeerConn
	requested	int		//发送的请求(字段)
	downloaded	int		//已下载的字段
	backlog		int		//并发度
	data		[]byte
}

type pieceResult struct {
	index	int
	data	[]byte
}

/*
	流程：
		1.把所有待下载的task放到一个channel中
		2.给每个peer起一个go协程
		3.每个协程从channel中获取一个task
		4.下载完成后将result放入一个channel中，再送去校验(SHA)
		5.校验完成后无误，写入最终的data中
 */
func Download(task *TorrentTask) error{
	fmt.Println("start downloading " + task.FileName)

	//初始化taskchannel
	taskQueue := make(chan *pieceTask, len(task.PieceSHA))
	//初始化resultchannel
	resultQueue := make(chan *pieceResult)

	//将所有任务遍历，放入channel中
	for index, sha := range task.PieceSHA{
		begin, end := task.getPieceBounds(index)
		taskQueue <- &pieceTask{
			index:  index,
			sha:    sha,
			length: (end - begin),
		}
	}

	//给每个peer起一个go协程
	for _, peer := range task.PeerList{
		go task.peerRoutine(peer, taskQueue, resultQueue)
	}

	//把result channel里的所有piece的信息写到缓存buf中
	buf := make([]byte, task.FileLen)
	count := 0
	for count <len(task.PieceSHA){
		res :=  <-resultQueue
		begin, end := task.getPieceBounds(res.index)
		copy(buf[begin : end], res.data)
		count++
		//打印piece下载进度
		percent := float64(count) / float64(len(task.PieceSHA)) * 100
		fmt.Printf("downloading, progress : (%0.2f%%)\n", percent)
	}

	close(taskQueue)
	close(resultQueue)

	//创建一个文件，把buf中的data写入文件中
	file, err := os.Create(task.FileName)
	if err != nil{
		fmt.Println("fail to create file: " + task.FileName)
		return err
	}
	_, err = file.Write(buf)
	if err != nil{
		fmt.Println("fail to write data")
		return err
	}

	return nil
}

//得到一段piece的起始和结束
func (task *TorrentTask)getPieceBounds(index int)(begin int, end int){
	begin = index * task.PieceLen
	end = begin + task.PieceLen
	//因为最后一段piece可能不足piecelen，所以如果end超出文件大小，则让end = filelen
	if end > task.FileLen{
		end = task.FileLen
	}
	return
}

func (task *TorrentTask)peerRoutine(peer PeerInfo, taskQueue chan *pieceTask, resultQueue chan *pieceResult){
	//获取和peer的连接，获取peer的bitField
	conn, err := NewConn(peer, task.InfoSHA, task.PeerId)
	if err != nil{
		return
	}
	defer conn.Close()

	fmt.Println("complete handshake with peer : " + peer.Ip.String())
	conn.WriteMsg(&PeerMsg{MsgInterested,nil})

	//只要有发生错误，就要把发生错误的task放回channel中
	for task := range taskQueue{
		//检查这个peer有无所需的piece,如果没有，那就将这个task重新放回channel中
		if !conn.bitField.HasPiece(task.index){
			taskQueue <- task
			continue
		}
		fmt.Printf("get task, index: %v, peer : %v\n", task.index, peer.Ip.String())
		res, err := downloadPiece(conn, task)
		if err != nil{
			taskQueue <- task
			fmt.Println("fail to download piece" + err.Error())
			return
		}
		if !checkPiece(task, res){
			taskQueue <- task
			continue
		}
		resultQueue <- res
	}
}

//下载piece
func downloadPiece(conn *PeerConn, task *pieceTask)(*pieceResult, error){
	state := &taskState{
		index:      task.index,
		conn:       conn,
		data:       make([]byte, task.length),
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	defer conn.SetDeadline(time.Time{})

	for state.downloaded < task.length{
		if !conn.Choke{
			for state.backlog < MAXBACKLOG && state.requested < task.length{
				length := BLOCKSIZE
				if task.length - state.requested < length{
					length = task.length - state.requested
				}
				msg := NewRequestMsg(state.index, state.requested, length)
				_ ,err := state.conn.WriteMsg(msg)
				if err != nil{
					return nil, err
				}
				state.backlog++
				state.requested += length
			}
		}
		err := state.handleMsg()
		if err != nil{
			return nil, err
		}
	}
	return &pieceResult{
		index: state.index,
		data:  state.data,
	},nil
}

//根据消息类型处理消息
func (state *taskState)handleMsg() error{
	msg, err := state.conn.ReadMsg()
	if err != nil{
		return err
	}
	if msg == nil{
		return nil
	}

	switch msg.Id {
	case MsgChoke:
		state.conn.Choke = true
	case MsgUnchoke:
		state.conn.Choke = false
	case MsgHave:
		index, err := GetHaveIndex(msg)
		if err != nil{
			return err
		}
		state.conn.bitField.SetPiece(index)
	case MsgPiece:
		n, err := CopyPieceData(state.index, state.data, msg)
		if err != nil{
			return err
		}
		state.downloaded += n
		state.backlog--
	}
	return nil
}

//校验piece的sha是否一致
func checkPiece(task *pieceTask, res *pieceResult) bool{
	sha := sha1.Sum(res.data)
	if !bytes.Equal(task.sha[:], sha[:]){
		fmt.Printf("check integrity failed, index :%v\n", res.index)
		return false
	}
	return true
}