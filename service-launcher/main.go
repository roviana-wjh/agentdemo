package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	blogURL  = "http://127.0.0.1:8080/web/"
	agentURL = "http://127.0.0.1:8081/health"
)

type serviceSpec struct {
	name       string
	directory  string
	binaries   []string
	healthURL  string
	healthText string
	port       string
	logName    string
	startLimit time.Duration
}

type childProcess struct {
	name string
	cmd  *exec.Cmd
	done chan error
	log  *os.File
}

func main() {
	verifyOnly := hasArgument("--verify")
	fmt.Println("========================================")
	fmt.Println("       AI 智能博客服务一键启动器")
	fmt.Println("========================================")

	workspace, err := findWorkspace()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("工作目录：%s\n\n", workspace)

	logDir := filepath.Join(workspace, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fatal(fmt.Errorf("创建日志目录失败: %w", err))
	}

	specs := []serviceSpec{
		{
			name: "Agent Service", directory: filepath.Join(workspace, "go-agent-main"),
			binaries:  []string{"go-agent-current.exe", "go-agent.exe"},
			healthURL: agentURL, healthText: "server is running", port: "8081", logName: "agent-service.log", startLimit: 60 * time.Second,
		},
		{
			name: "Blog Service", directory: filepath.Join(workspace, "blog-api"),
			binaries:  []string{"blog-api.exe", "awesomeProject1.exe"},
			healthURL: blogURL, healthText: "墨屿", port: "8080", logName: "blog-service.log", startLimit: 30 * time.Second,
		},
	}

	children := make([]*childProcess, 0, len(specs))
	for _, spec := range specs {
		child, reused, startErr := ensureService(spec, logDir)
		if startErr != nil {
			stopChildren(children)
			fatal(startErr)
		}
		if reused {
			fmt.Printf("[正常] %s 已在运行，直接复用。\n", spec.name)
		} else {
			children = append(children, child)
			fmt.Printf("[正常] %s 启动成功。\n", spec.name)
		}
	}

	fmt.Println()
	fmt.Printf("博客地址：%s\n", blogURL)
	fmt.Printf("日志目录：%s\n", logDir)
	if len(children) > 0 {
		fmt.Println("关闭此窗口或按 Ctrl+C，可停止本次启动的服务。")
	} else {
		fmt.Println("两个服务均由其他进程运行；关闭本窗口不会停止它们。")
	}
	if verifyOnly {
		fmt.Println("\n[验证] 两个服务均已通过健康检查，正在清理测试进程……")
		stopChildren(children)
		fmt.Println("[验证] 启动器自检通过。")
		return
	}
	openBrowser(blogURL)

	if len(children) == 0 {
		waitForEnter()
		return
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	exitChannel := make(chan string, len(children))
	for _, child := range children {
		current := child
		go func() {
			err := <-current.done
			if err != nil {
				exitChannel <- fmt.Sprintf("%s 已退出：%v", current.name, err)
			} else {
				exitChannel <- fmt.Sprintf("%s 已退出", current.name)
			}
		}()
	}

	select {
	case <-signalChannel:
		fmt.Println("\n正在停止服务……")
	case message := <-exitChannel:
		fmt.Printf("\n[异常] %s\n", message)
	}
	stopChildren(children)
	fmt.Println("服务已停止。")
}

func findWorkspace() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法确定启动器位置: %w", err)
	}
	directory := filepath.Dir(executable)
	for attempts := 0; attempts < 4; attempts++ {
		if isDirectory(filepath.Join(directory, "blog-api")) && isDirectory(filepath.Join(directory, "go-agent-main")) {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", errors.New("启动器旁未找到 blog-api 和 go-agent-main 目录，请将 EXE 放回项目根目录")
}

func ensureService(spec serviceSpec, logDir string) (*childProcess, bool, error) {
	if isHealthy(spec.healthURL, spec.healthText) {
		return nil, true, nil
	}
	if portInUse(spec.port) {
		return nil, false, fmt.Errorf("%s 端口 %s 已被其他程序占用，但健康检查失败", spec.name, spec.port)
	}

	binary, err := findBinary(spec.directory, spec.binaries)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", spec.name, err)
	}
	logPath := filepath.Join(logDir, spec.logName)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, false, fmt.Errorf("打开 %s 日志失败: %w", spec.name, err)
	}
	_, _ = fmt.Fprintf(logFile, "\n\n===== launcher start %s =====\n", time.Now().Format(time.RFC3339))

	cmd := exec.Command(binary)
	cmd.Dir = spec.directory
	cmd.Env = os.Environ()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, false, fmt.Errorf("启动 %s 失败: %w", spec.name, err)
	}
	child := &childProcess{name: spec.name, cmd: cmd, done: make(chan error, 1), log: logFile}
	go func() { child.done <- cmd.Wait(); _ = logFile.Close() }()

	fmt.Printf("[启动] %s（PID %d），等待健康检查……\n", spec.name, cmd.Process.Pid)
	deadline := time.Now().Add(spec.startLimit)
	for time.Now().Before(deadline) {
		if isHealthy(spec.healthURL, spec.healthText) {
			return child, false, nil
		}
		select {
		case processErr := <-child.done:
			return nil, false, fmt.Errorf("%s 启动后立即退出：%v；请查看 %s", spec.name, processErr, logPath)
		case <-time.After(500 * time.Millisecond):
		}
	}
	_ = cmd.Process.Kill()
	return nil, false, fmt.Errorf("%s 在 %s 内未通过健康检查；请查看 %s", spec.name, spec.startLimit, logPath)
}

func findBinary(directory string, candidates []string) (string, error) {
	for _, name := range candidates {
		path := filepath.Join(directory, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("未找到可执行文件（尝试：%s）", strings.Join(candidates, "、"))
}

func isHealthy(url, expectedText string) bool {
	client := http.Client{Timeout: 1500 * time.Millisecond}
	response, err := client.Get(url)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return false
	}
	if expectedText == "" {
		return true
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 128<<10))
	return err == nil && strings.Contains(string(body), expectedText)
}

func portInUse(port string) bool {
	connection, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = connection.Close()
	return true
}

func stopChildren(children []*childProcess) {
	for index := len(children) - 1; index >= 0; index-- {
		child := children[index]
		if child != nil && child.cmd != nil && child.cmd.Process != nil {
			_ = child.cmd.Process.Kill()
		}
	}
}

func openBrowser(url string) {
	if runtime.GOOS != "windows" {
		return
	}
	command := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	_ = command.Start()
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasArgument(expected string) bool {
	for _, argument := range os.Args[1:] {
		if strings.EqualFold(strings.TrimSpace(argument), expected) {
			return true
		}
	}
	return false
}

func fatal(err error) {
	fmt.Printf("\n[启动失败] %v\n", err)
	fmt.Println("按 Enter 键关闭窗口……")
	_, _ = fmt.Scanln()
	os.Exit(1)
}

func waitForEnter() {
	fmt.Println("按 Enter 键关闭启动器……")
	_, _ = fmt.Scanln()
}
