package main

type room struct {
	forward chan []byte      // 他のクライアントにメッセージを保存するためのチャネル
	join    chan *client     // チャットルームに参加しようとしているクライアントのためのチャネル
	leave   chan *client     // チャットルームから体質仕様としているクライアントのためのチャネル
	clients map[*client]bool // 在室しているすべてのクライアントが保持される
}
