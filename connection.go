package websocket

import (
	"errors"
	"fmt"
	gorillaWebsocket "github.com/gorilla/websocket"
	"sync"
)

type Connection struct{
	UniqueIdentification   UniqueIdentification
	wsConnect              *gorillaWebsocket.Conn
	inChan                 chan []byte
	outChan                chan []byte
	closeChan              chan byte
	mutex                  sync.Mutex  // 对closeChan关闭上锁
	isClosed               bool  // 防止closeChan被关闭多次
}

func InitConnection(wsConn *gorillaWebsocket.Conn)(conn *Connection ,err error){
	conn = &Connection{
		wsConnect:wsConn,
		inChan: make(chan []byte,10000),
		outChan: make(chan []byte,10000),
		closeChan: make(chan byte,1),
	}
	// 启动读协程
	go conn.readLoop()
	// 启动写协程
	go conn.writeLoop()
	return
}

//read message
func (conn *Connection)ReadMessage()(data []byte , err error){
	select{
	case data = <- conn.inChan:
	case <- conn.closeChan:
		err = errors.New("connection is closed")
	}
	return
}

func (conn *Connection)WriteMessage(data []byte)(err error) {
	select{
	case conn.outChan <- data:
		fmt.Println(string(data))
	case <- conn.closeChan:
		err = errors.New("connection is closed")
	}
	return
}

func (conn *Connection)Close(){
	// 线程安全，可多次调用
	_ = conn.wsConnect.Close()
	// 利用标记，让closeChan只关闭一次
	conn.mutex.Lock()
	if !conn.isClosed {
		close(conn.closeChan)
		conn.isClosed = true
	}
	conn.mutex.Unlock()
}

// 内部实现
func (conn *Connection)readLoop(){
	var(
		data []byte
		err error
	)
	for{
		if _, data , err = conn.wsConnect.ReadMessage(); err != nil{
			goto ERR
		}
		//阻塞在这里，等待inChan有空闲位置
		select{
		case conn.inChan <- data:
		case <- conn.closeChan:		// closeChan 感知 conn断开
			goto ERR
		}
	}
ERR:
	conn.Close()
}

func (conn *Connection)writeLoop(){
	var(
		data []byte
		err error
	)

	for{
		select{
		case data= <- conn.outChan:
		case <- conn.closeChan:
			goto ERR
		}

		if err = conn.wsConnect.WriteMessage(gorillaWebsocket.TextMessage , data); err != nil{
			goto ERR
		}
	}
ERR:
	conn.Close()
}