package main

import (
	"log"
	"net/http"

	"github.com/udaycmd/bisoc"
)

var server = &bisoc.Server{
	// allow any host to connect
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := server.Accept(w, r)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	log.Println("Connected!")
	defer c.Close()

	for {
		t, msg, err := c.RecvMsg()
		if err != nil {
			log.Printf("Error: %v\n", err)
			break
		}

		log.Println("Message from client: ", string(msg))

		err = c.SendMsg(t, string(msg))
		if err != nil {
			log.Printf("Error: %v", err)
			break
		}
	}

}

func main() {
	http.HandleFunc("/", echo)
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Printf("Error: %v\n", err)
	}
}
