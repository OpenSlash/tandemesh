package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type localPlane struct {
	cmd  *exec.Cmd
	URL  string
	Port int
}

func findPlaneBinary() string {
	if value := os.Getenv("TANDEMESH_PLANE_BIN"); value != "" {
		return value
	}
	name := "tandemesh-plane"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	candidates := []string{
		filepath.Join(filepath.Dir(os.Args[0]), name),
		filepath.Join("..", "..", "bin", name),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func startLocalPlane(localToken string) (*localPlane, error) {
	binary := findPlaneBinary()
	if binary == "" {
		return nil, fmt.Errorf("tandemesh-plane binary not found; set TANDEMESH_PLANE_BIN")
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	dataDir := dataDirPath()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"TANDEMESH_PROFILE=local",
		"TANDEMESH_AUTH=none",
		"TANDEMESH_USER=me",
		"TANDEMESH_LISTEN="+listenAddr()+":"+fmt.Sprint(port),
		"TANDEMESH_DB_PATH="+filepath.Join(dataDir, "tandemesh.db"),
		"TANDEMESH_LOCAL_TOKEN="+localToken,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if err := attachToJob(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "attach plane job: %v\n", err)
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/readyz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return &localPlane{cmd: cmd, URL: url, Port: port}, nil
			}
		}
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return nil, fmt.Errorf("tandemesh-plane exited during startup")
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	return nil, fmt.Errorf("tandemesh-plane did not become ready")
}

func (p *localPlane) Stop() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	_ = p.cmd.Process.Kill()
	_, _ = p.cmd.Process.Wait()
}
func listenAddr() string {
	if os.Getenv("TANDEMESH_LAN") == "1" {
		return "0.0.0.0"
	}
	return "127.0.0.1"
}
