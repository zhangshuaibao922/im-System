package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	//用户列表
	OnlineMap map[string]*User
	MapLock   sync.RWMutex
	//消息广播
	Message chan string
}

// NewServer 创建一个server接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// BroadCast 广播消息
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + ":" + user.Name + ":" + msg + "]"
	this.Message <- sendMsg
}

// ListenMessage 监听channel的goroutine，一旦有消息就全部发给所有在线的user
func (s *Server) ListenMessage() {
	for {
		msg := <-s.Message
		//将消息发送给全部在线的user
		s.MapLock.RLock()
		for _, user := range s.OnlineMap {
			user.C <- msg
		}
		s.MapLock.RUnlock()
	}
}
func (this *Server) Handler(conn net.Conn) {
	//当前链接业务
	fmt.Println("链接创建成功")
	user := NewUser(conn, this)

	user.Online()

	//监听用户是否活跃的channel
	isLive := make(chan bool)
	//接受客户端传递发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			read, err := conn.Read(buf)
			if err != nil && err != io.EOF {
				fmt.Println("conn.Read err:", err)
				return
			}
			if read == 0 {
				user.Offline()
				return
			}
			//提取用户的消息（“去除\n”）
			msg := string(buf[:read-1])
			//用户针对msg进行消息处理
			user.DoMessage(msg)
			//当前用户活跃
			isLive <- true
		}
	}()

	for {
		select {
		case <-isLive:
			//当前用户活跃、重制定时器
			//不做任何操作、更新下边的定时器
		case <-time.After(time.Second * 300):
			//已经超时
			//将当前的User强制的关闭
			user.SendMsg("done\n")
			close(user.C)
			//退出handler
			return //runtime.Goexit()
		}
	}

}

// Start 启动服务器
func (this *Server) Start() {
	//listen
	listen, err := net.Listen("tcp", this.Ip+":"+strconv.Itoa(this.Port))
	if err != nil {
		fmt.Println("net.listen err:", err)
		return
	}
	//close
	defer listen.Close()
	//启动监听msg的goroutine
	go this.ListenMessage()
	for {
		//accept
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("net.listen err:", err)
			continue
		}
		//do handler
		go this.Handler(conn)
	}
}
