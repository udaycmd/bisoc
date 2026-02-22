package main

import (
	"log"
	"net/http"

	"github.com/udaycmd/bisoc"
)

var server = &bisoc.Server{}

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
