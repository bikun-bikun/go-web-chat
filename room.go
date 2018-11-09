package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"trace"
)

type room struct {
	forward chan []byte      // 他のクライアントにメッセージを保存するためのチャネル
	join    chan *client     // チャットルームに参加しようとしているクライアントのためのチャネル
	leave   chan *client     // チャットルームから体質仕様としているクライアントのためのチャネル
	clients map[*client]bool // 在室しているすべてのクライアントが保持される
	tracer  trace.Tracer     //tracerはチャットルームで行われた操作ログを受け取ります。
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
			r.tracer.Trace("新しいクライアントが参加しました")
		case client := <-r.leave:
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("クライアントが退室しました")
		case msg := <-r.forward:
			for client := range r.clients {
				select {
				case client.send <- msg:
					//メッセージを送信
					r.tracer.Trace("-- クライアントに送信されました")
				default:
					delete(r.clients, client)
					close(client.send)
					r.tracer.Trace("-- 送信に失敗しました。クライアントをクリーンアップします")
				}
			}
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServerHTTP:", err)
		return
	}
	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}
