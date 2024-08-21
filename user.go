package main

import (
	"fmt"
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	Conn   net.Conn
	Server *Server
}

// NewUser 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	user := &User{
		Conn:   conn,
		Name:   conn.RemoteAddr().String(),
		C:      make(chan string),
		Addr:   conn.RemoteAddr().String(),
		Server: server,
	}
	go user.ListenMessage()

	return user
}

// 用户的上线业务
func (this *User) Online() {
	//加入到onlinemap
	this.Server.MapLock.Lock()
	this.Server.OnlineMap[this.Name] = this
	this.Server.MapLock.Unlock()

	//广播当前用户上线消息
	this.Server.BroadCast(this, "已上线")
}

func (this *User) Offline() {
	//加入到onlinemap
	this.Server.MapLock.Lock()
	delete(this.Server.OnlineMap, this.Name)
	this.Server.MapLock.Unlock()

	//广播当前用户下线消息
	this.Server.BroadCast(this, "已下线")

}
func (this *User) SendMsg(msg string) {
	_, err := this.Conn.Write([]byte(msg))
	if err != nil {
		fmt.Println("conn.write error:", err)
		return
	}
}
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		//查询当前在线用户都有哪些
		this.Server.MapLock.Lock()
		for _, user := range this.Server.OnlineMap {
			onlieMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线\n"
			this.SendMsg(onlieMsg)
		}
		this.Server.MapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		//消息格式rename|张三
		newName := strings.Split(msg, "|")[1]
		//判断name是否存在
		_, ok := this.Server.OnlineMap[newName]
		if ok {
			this.SendMsg("当前用户名被占用\n")

		} else {
			this.Server.MapLock.Lock()
			delete(this.Server.OnlineMap, this.Name)
			this.Server.OnlineMap[newName] = this
			this.Server.MapLock.Unlock()

			this.Name = newName
			this.SendMsg("更新成功" + this.Name + "\n")
		}

	} else if len(msg) > 4 && msg[:3] == "to|" {
		//消息格式 to|张三|消息内容
		//获取对方的用户明
		remoteName := strings.Split(msg, "|")[1]
		//获取对方的User，User.sendMsg
		user, ok := this.Server.OnlineMap[remoteName]
		if !ok {
			this.SendMsg("该用户不存在\n")
			return
		}
		content := strings.Split(msg, "|")[2]
		if content == "" {
			this.SendMsg("无消息内容\n")
		}
		user.SendMsg(this.Name + "对你说：" + content + "\n")

	} else {
		this.Server.BroadCast(this, msg)
	}
}

// 监听当前User channel的方法，一旦有消息，就直接发送给对端客户端
func (u *User) ListenMessage() {
	//当u.C通道关闭后，不再进行监听并写入信息
	for msg := range u.C {
		_, err := u.Conn.Write([]byte(msg + "\n"))
		if err != nil {
			fmt.Println("conn.Write err:", err)
		}
	}
	//不监听后关闭conn，conn在这里关闭最合适
	err := u.Conn.Close()
	if err != nil {
		fmt.Println("conn.Close err:", err)
	}

}
