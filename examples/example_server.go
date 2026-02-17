package main

import (
	"fmt"
	"net/http"

	"github.com/udaycmd/bisoc"
)

var server = &bisoc.Server{}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := server.Accept(w, r)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// TODO: Read the message and echo it back to the client, and close the connection.
	_ = c
}

func main() {
	http.HandleFunc("/echo", echo)
	if err := http.ListenAndServe("localhost:3000", nil); err != nil {
		fmt.Println(err.Error())
	}
}
