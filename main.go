package main

import (
	"log"
	
	"github.com/katatrina/rag-chatbot-go/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
