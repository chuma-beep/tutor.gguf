// Command tutor is the single entrypoint for all tutor.gguf user-facing
// components: the RAG HTTP server, the corpus indexer, and the interactive
// terminal shell.
package main

import (
	"fmt"
	"os"

	"github.com/chuma-beep/tutor.gguf/internal/cli"
)

const usage = `tutor — offline math tutor

Usage:
  tutor <command> [flags]

Commands:
  setup   download models, llama.cpp, and corpus; build the index (idempotent)
  chat    interactive terminal shell — starts the whole stack automatically
  serve   run the RAG HTTP server (retrieval + generation on /v1/complete)
  index   ingest corpus sources into the vector store, then run a test query
  help    show this message

Run "tutor <command> -h" for command-specific flags.`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		return
	}

	var err error
	switch os.Args[1] {
	case "setup":
		err = cli.Setup(os.Args[2:])
	case "serve":
		err = cli.Serve(os.Args[2:])
	case "index":
		err = cli.Index(os.Args[2:])
	case "chat":
		err = cli.Chat(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Println(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s\n", os.Args[1], usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
