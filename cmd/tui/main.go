package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/chuma-beep/tutor.gguf/internal/renderer"
	"github.com/chuma-beep/tutor.gguf/internal/tui"
)

func main() {
	tutorURL := flag.String("tutor-url", "http://localhost:8082", "tutor RAG server base URL")
	ascii := flag.Bool("ascii", false, "render math with ASCII fallbacks instead of Unicode")
	flag.Parse()

	render := func(s string) string {
		return renderer.Render(s, *ascii)
	}

	if err := tui.Run(tui.Options{
		TutorURL: *tutorURL,
		ASCII:    *ascii,
		Render:   render,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
