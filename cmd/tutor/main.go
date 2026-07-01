package main

import (
	"fmt"
	"log"

	"github.com/chuma-beep/tutor.gguf/internal/llm"
)

func main() {
	client := llm.NewClient("http://localhost:8080")

	prompt := "Solve step by step: What is the derivative of x^2 + 3x + 5?"

	fmt.Println("tutor.gguf — sending prompt...")

	response, err := client.Complete(prompt)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Println("Response:")
	fmt.Println(response)
}
