package runtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// Mode selects which llama-server processes a Manager supervises.
type Mode int

const (
	ModeBoth Mode = iota
	ModeGenOnly
	ModeEmbedOnly
)

// Config describes one managed llama.cpp deployment.
type Config struct {
	LlamaServer string // path to the llama-server binary
	GenModel    string // path to the generation GGUF
	EmbedModel  string // path to the embedding GGUF
	Threads     int    // -t value; 0 lets llama.cpp auto-size
	Ctx         int    // -c value for the generation server
	LogDir      string // where child stdout/stderr is captured
	Mode        Mode
}

// Manager spawns llama-server processes, waits for their /health endpoints,
// and tears them down on Stop.
type Manager struct {
	cfg      Config
	genURL   string
	embedURL string
	cmds     []*exec.Cmd
	files    []*os.File
	stopped  bool
}

func New(cfg Config) *Manager { return &Manager{cfg: cfg} }

func (m *Manager) GenURL() string   { return m.genURL }
func (m *Manager) EmbedURL() string { return m.embedURL }

// DiscoverLlamaServer finds a usable llama-server binary.
// Precedence: $TUTOR_LLAMA_SERVER > ~/.tutor/bin/llama-server > $PATH.
func DiscoverLlamaServer() (string, error) {
	if p := os.Getenv("TUTOR_LLAMA_SERVER"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("TUTOR_LLAMA_SERVER=%s does not exist", p)
	}
	p := LlamaServerPath()
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	if lp, err := exec.LookPath("llama-server"); err == nil {
		return lp, nil
	}
	return "", fmt.Errorf("llama-server not found (checked $TUTOR_LLAMA_SERVER, %s, and $PATH) — run `tutor setup`", BinDir())
}

// Start launches the configured servers and blocks until each reports healthy
// via GET /health. A goroutine stops everything when ctx is cancelled, so the
// usual pattern is:
//
//	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
//	defer stop()
//	mgr := runtime.New(cfg)
//	if err := mgr.Start(ctx); err != nil { ... }
//	defer mgr.Stop()
func (m *Manager) Start(ctx context.Context) error {
	server := m.cfg.LlamaServer
	if server == "" {
		var err error
		if server, err = DiscoverLlamaServer(); err != nil {
			return err
		}
	}
	if m.cfg.LogDir == "" {
		m.cfg.LogDir = LogsDir()
	}
	if err := os.MkdirAll(m.cfg.LogDir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	if m.cfg.Mode != ModeEmbedOnly {
		port := freePort()
		cmd, err := m.spawn(server, "gen", []string{
			"-m", m.cfg.GenModel,
			"--port", strconv.Itoa(port),
			"-t", strconv.Itoa(m.cfg.Threads),
			"-c", strconv.Itoa(m.cfg.Ctx),
		})
		if err != nil {
			m.Stop()
			return err
		}
		m.genURL = "http://127.0.0.1:" + strconv.Itoa(port)
		logPath := filepath.Join(m.cfg.LogDir, "gen.log")
		if err := waitHealthy(ctx, "gen", m.genURL, cmd, logPath); err != nil {
			m.Stop()
			return err
		}
	}

	if m.cfg.Mode != ModeGenOnly {
		port := freePort()
		cmd, err := m.spawn(server, "embed", []string{
			"-m", m.cfg.EmbedModel,
			"--embeddings",
			"--batch-size", "2048",
			"--ubatch-size", "2048",
			"--port", strconv.Itoa(port),
			"-t", strconv.Itoa(m.cfg.Threads),
		})
		if err != nil {
			m.Stop()
			return err
		}
		m.embedURL = "http://127.0.0.1:" + strconv.Itoa(port)
		logPath := filepath.Join(m.cfg.LogDir, "embed.log")
		if err := waitHealthy(ctx, "embed", m.embedURL, cmd, logPath); err != nil {
			m.Stop()
			return err
		}
	}

	go func() {
		<-ctx.Done()
		m.Stop()
	}()
	return nil
}

// Stop kills every supervised process. Safe to call multiple times.
func (m *Manager) Stop() {
	if m.stopped {
		return
	}
	m.stopped = true
	for _, cmd := range m.cmds {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	for _, f := range m.files {
		f.Close()
	}
}

func (m *Manager) spawn(server, name string, args []string) (*exec.Cmd, error) {
	logFile, err := os.Create(filepath.Join(m.cfg.LogDir, name+".log"))
	if err != nil {
		return nil, fmt.Errorf("open %s log: %w", name, err)
	}
	cmd := exec.Command(server, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("start llama-server (%s): %w", name, err)
	}
	m.cmds = append(m.cmds, cmd)
	m.files = append(m.files, logFile)
	return cmd, nil
}

// waitHealthy polls {url}/health until it returns 200, the child exits early,
// ctx is cancelled, or ~5 minutes elapse (cold model loads can be slow).
func waitHealthy(ctx context.Context, name, url string, cmd *exec.Cmd, logPath string) error {
	exited := make(chan struct{})
	go func() {
		cmd.Wait()
		close(exited)
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(5 * time.Minute)
	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s server: %w", name, ctx.Err())
		case <-exited:
			return fmt.Errorf("%s server exited during startup — last log lines:\n%s", name, tailFile(logPath, 2048))
		case <-tick.C:
			resp, err := client.Get(url + "/health")
			if err == nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("%s server not healthy after 5m — logs: %s", name, logPath)
			}
		}
	}
}

// tailFile returns the last n bytes of path ("" when unreadable), so startup
// failures surface llama.cpp's own error message instead of nothing.
func tailFile(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	offset := int64(0)
	if info.Size() > int64(n) {
		offset = info.Size() - int64(n)
	}
	buf := make([]byte, int(info.Size()-offset))
	if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
		return ""
	}
	return string(buf)
}

func freePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
