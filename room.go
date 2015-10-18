package main

import (
	"github.com/gorilla/websocket"
	"github.com/mxjxn/gochat/trace"
	"log"
	"net/http"
)

type room struct {
	// messages to be sent to all clients
	forward chan []byte
	// join is a channel for clients joining
	join chan *client
	// leave is a channel for clients leaving
	leave chan *client
	//clients is a map holding all the clients currently in the room
	clients map[*client]bool

	// tracer will print room events to console
	tracer trace.Tracer
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			//joining
			r.clients[client] = true
			r.tracer.Trace("New client joined")
		case client := <-r.leave:
			// leaving
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("Client Left")
		case msg := <-r.forward:
			r.tracer.Trace("Message Recieved: ", string(msg))
			for client := range r.clients {
				select {
				case client.send <- msg:
					//send the message
					r.tracer.Trace(" -- sent to client")
				default:
					//failed to send
					delete(r.clients, client)
					close(client.send)
					r.tracer.Trace(" -- failed to send, cleaned up client")

				}
			}
		}
	}
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
		tracer:  trace.Off(),
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
		log.Fatal("ServeHTTP:", err)
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