package main

import (
	"log"
)

func setup(action string, requester Requester) {
	if action == "" {
		log.Println("You should provide one and only one argument for alarm. And it can only be 'arm', 'disarm' or 'partarm'.")
	}

	requester.RequestFeenstra(action)
}
