package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/chuma-beep/tutor.gguf/internal/renderer"
	"github.com/chuma-beep/tutor.gguf/internal/tui"
)

// Chat opens the interactive Bubble Tea shell. With no -tutor-url it starts
// the full stack itself (llama-servers + in-process RAG server) and tears it
// down when the shell exits.
func Chat(args []string) error {
	fs := flag.NewFlagSet("chat", flag.ContinueOnError)
	tutorURL := fs.String("tutor-url", "", "tutor RAG server base URL — omit to start the whole stack automatically")
	ascii := fs.Bool("ascii", false, "render math with ASCII fallbacks instead of Unicode")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *tutorURL == "" {
		mgr, err := startManaged(ctx)
		if err != nil {
			return err
		}
		defer mgr.stop()

		handler, err := NewRAGHandler(ctx, mgr.embedURL, mgr.genURL, resolveDBPath(""))
		if err != nil {
			return err
		}
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("pick port for local RAG server: %w", err)
		}
		srv := &http.Server{Handler: handler}
		go srv.Serve(l)
		defer srv.Close()
		*tutorURL = "http://127.0.0.1:" + strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
		fmt.Printf("local tutor server ready at %s\n", *tutorURL)
	}

	render := func(s string) string {
		return renderer.Render(s, *ascii)
	}

	err := tui.Run(tui.Options{
		TutorURL: *tutorURL,
		ASCII:    *ascii,
		Render:   render,
	})
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
