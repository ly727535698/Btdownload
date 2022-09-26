package main


/*
	BT下载器编写流程
			1.编写Bencode库：用来做Bencode的序列化和反序列化
			2.根据Bencode库，编写种子文件(torrent文件)的解析库：得到tracker的url和种子文件的Info
			3.编写tracker模块：和tracker的url做网络交互，获取pears信息
			4.编写download模块：和所有的pears信息交互，下载和校验所有的文件片(piceces)
			5.编写assembler模块：把所有文件片拼装成最终文件
 */


func main(){

}
