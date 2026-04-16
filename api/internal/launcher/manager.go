package launcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/runfivedev/runfive/internal/models"
	"github.com/runfivedev/runfive/internal/serverfs"
)

const (
	defaultLogCapacity   = 2000
	defaultTailLines     = 200
	startingWarmup       = 1200 * time.Millisecond
	stopGracePeriod      = 12 * time.Second
	stopTerminateTimeout = 5 * time.Second
	scannerBufferMax     = 1024 * 1024
)

var (
	// ErrServerNotFound is returned when the requested managed server does not exist.
	ErrServerNotFound = errors.New("server not found")
	// ErrAlreadyRunning is returned when Start is called for a running server.
	ErrAlreadyRunning = errors.New("server already running")
	// ErrNotRunning is returned when a command targets a stopped server.
	ErrNotRunning = errors.New("server is not running")
	// ErrStopFailed is returned when a server process refuses to exit even
	// after the final SIGKILL / taskkill /F attempt.
	ErrStopFailed = errors.New("server process did not exit after force kill")
)

type artifactResolver interface {
	ExecutablePath(version string) (string, error)
}

// Subscription delivers live console events for a single server.
type Subscription struct {
	C       <-chan models.ServerConsoleEvent
	closeFn func()
}

// Close unsubscribes the live console stream.
func (s *Subscription) Close() {
	if s != nil && s.closeFn != nil {
		s.closeFn()
	}
}

// Manager owns the in-memory runtime state for every server launched by the panel.
type Manager struct {
	registry  *serverfs.Registry
	artifacts artifactResolver

	mu      sync.RWMutex
	servers map[string]*serverRuntime
}

// NewManager creates a process manager bound to the filesystem server registry.
func NewManager(registry *serverfs.Registry, artifacts artifactResolver) *Manager {
	return &Manager{
		registry:  registry,
		artifacts: artifacts,
		servers:   make(map[string]*serverRuntime),
	}
}

// Start launches the fxserver binary for one managed server.
func (m *Manager) Start(id string) (models.ServerProcessStatus, error) {
	spec, ok := m.registry.LaunchSpec(id)
	if !ok {
		return models.ServerProcessStatus{}, ErrServerNotFound
	}

	executablePath, err := m.artifacts.ExecutablePath(spec.ArtifactVersion)
	if err != nil {
		return models.ServerProcessStatus{}, fmt.Errorf("resolve artifact executable: %w", err)
	}

	absExecutable, err := filepath.Abs(executablePath)
	if err != nil {
		return models.ServerProcessStatus{}, fmt.Errorf("resolve executable path: %w", err)
	}

	absConfig, err := filepath.Abs(spec.ConfigPath)
	if err != nil {
		return models.ServerProcessStatus{}, fmt.Errorf("resolve server.cfg path: %w", err)
	}

	absServerDir, err := filepath.Abs(spec.ServerDir)
	if err != nil {
		return models.ServerProcessStatus{}, fmt.Errorf("resolve server dir: %w", err)
	}

	args := make([]string, 0, 4)
	if spec.OneSync != "" {
		args = append(args, "+set", "onesync", spec.OneSync)
	}
	args = append(args, "+exec", absConfig)

	//nolint:gosec // executable path and args are derived from application-managed runtime files.
	cmd := exec.CommandContext(context.Background(), absExecutable, args...)
	cmd.Dir = absServerDir
	cmd.SysProcAttr = newProcessGroupAttr()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return models.ServerProcessStatus{}, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return models.ServerProcessStatus{}, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return models.ServerProcessStatus{}, fmt.Errorf("create stderr pipe: %w", err)
	}

	runtime, err := m.ensureRuntime(id)
	if err != nil {
		return models.ServerProcessStatus{}, err
	}

	runtime.mu.Lock()
	if runtime.status == models.ServerStatusStarting || runtime.status == models.ServerStatusRunning {
		status := runtime.snapshotLocked()
		runtime.mu.Unlock()
		return status, ErrAlreadyRunning
	}

	runtime.status = models.ServerStatusStarting
	runtime.exitCode = nil
	runtime.exitReason = ""
	runtime.updatedAt = time.Now().UTC()
	runtime.stopRequested = false
	runtime.stdin = nil
	runtime.cmd = nil
	runtime.pid = 0
	runtime.done = make(chan struct{})
	runtime.mu.Unlock()
	runtime.broadcastStatus()

	if err := cmd.Start(); err != nil {
		runtime.failStart(fmt.Errorf("start process: %w", err))
		return runtime.Status(), err
	}

	runtime.mu.Lock()
	runtime.cmd = cmd
	runtime.stdin = stdin
	runtime.pid = cmd.Process.Pid
	done := runtime.done
	runtime.mu.Unlock()

	runtime.appendSystem(fmt.Sprintf("Launching %s with artifact %s", spec.Name, spec.ArtifactVersion))

	go runtime.capturePipe("stdout", stdout)
	go runtime.capturePipe("stderr", stderr)
	go runtime.promoteToRunningAfter(startingWarmup, done)
	go runtime.waitForExit(cmd)

	return runtime.Status(), nil
}

// Stop attempts a graceful stop with a bounded fallback to termination/kill.
func (m *Manager) Stop(id string) (models.ServerProcessStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), stopGracePeriod+stopTerminateTimeout+3*time.Second)
	defer cancel()
	return m.StopWithContext(ctx, id)
}

// StopWithContext is Stop with caller-managed cancellation.
func (m *Manager) StopWithContext(ctx context.Context, id string) (models.ServerProcessStatus, error) {
	runtime, err := m.ensureRuntime(id)
	if err != nil {
		return models.ServerProcessStatus{}, err
	}

	pid, stdin, done, alreadyStopped := runtime.beginStop()
	if alreadyStopped {
		return runtime.Status(), nil
	}

	runtime.appendSystem("Stop requested by panel")
	if stdin != nil {
		_, _ = io.WriteString(stdin, "quit\n")
	}

	if waitForDone(ctx, done, stopGracePeriod) {
		return runtime.Status(), nil
	}

	runtime.appendSystem("Grace period elapsed, terminating server process tree")
	if err := terminateProcessTree(pid); err != nil {
		runtime.appendSystem(fmt.Sprintf("Terminate signal failed: %v", err))
	}
	if waitForDone(ctx, done, stopTerminateTimeout) {
		return runtime.Status(), nil
	}

	runtime.appendSystem("Server still alive, forcing process tree kill")
	if err := killProcessTree(pid); err != nil {
		runtime.appendSystem(fmt.Sprintf("Force kill failed: %v", err))
	}
	if !waitForDone(ctx, done, 2*time.Second) {
		runtime.appendSystem("Process still alive after force kill — panel gave up")
		return runtime.Status(), ErrStopFailed
	}

	return runtime.Status(), nil
}

// ShutdownAll stops every process known to the manager before the API exits.
func (m *Manager) ShutdownAll(ctx context.Context) error {
	m.mu.RLock()
	ids := make([]string, 0, len(m.servers))
	for id := range m.servers {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	var firstErr error
	for _, id := range ids {
		if _, err := m.StopWithContext(ctx, id); err != nil && !errors.Is(err, ErrServerNotFound) && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// Status returns the current runtime state for one managed server.
func (m *Manager) Status(id string) (models.ServerProcessStatus, error) {
	runtime, err := m.ensureRuntime(id)
	if err != nil {
		return models.ServerProcessStatus{}, err
	}
	return runtime.Status(), nil
}

// Tail returns the most recent console lines for one managed server.
func (m *Manager) Tail(id string, n int) ([]models.ServerLogLine, error) {
	runtime, err := m.ensureRuntime(id)
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		n = defaultTailLines
	}
	return runtime.logs.tail(n), nil
}

// Subscribe registers a live console subscriber for one managed server.
func (m *Manager) Subscribe(id string) (*Subscription, error) {
	runtime, err := m.ensureRuntime(id)
	if err != nil {
		return nil, err
	}
	return runtime.subscribe(), nil
}

// SendCommand forwards one line of console input to the running server.
func (m *Manager) SendCommand(id, command string) error {
	runtime, err := m.ensureRuntime(id)
	if err != nil {
		return err
	}

	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}

	// The write is performed under stdinMu rather than r.mu so broadcast/status
	// readers are not blocked, but is still serialized against waitForExit's
	// pipe teardown which acquires the same lock before niling r.stdin.
	runtime.stdinMu.Lock()
	defer runtime.stdinMu.Unlock()

	runtime.mu.Lock()
	stdin := runtime.stdin
	running := runtime.status == models.ServerStatusStarting || runtime.status == models.ServerStatusRunning
	runtime.mu.Unlock()

	if !running || stdin == nil {
		return ErrNotRunning
	}

	if _, err := io.WriteString(stdin, command+"\n"); err != nil {
		return fmt.Errorf("write console input: %w", err)
	}
	runtime.appendCommand(command)
	return nil
}

func (m *Manager) ensureRuntime(id string) (*serverRuntime, error) {
	if !m.registry.HasServer(id) {
		return nil, ErrServerNotFound
	}

	m.mu.RLock()
	runtime, ok := m.servers[id]
	m.mu.RUnlock()
	if ok {
		return runtime, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if runtime, ok = m.servers[id]; ok {
		return runtime, nil
	}

	runtime = &serverRuntime{
		id:        id,
		status:    models.ServerStatusStopped,
		updatedAt: time.Now().UTC(),
		logs:      newRingBuffer(defaultLogCapacity),
		subs:      make(map[int]chan models.ServerConsoleEvent),
	}
	m.servers[id] = runtime
	return runtime, nil
}

type serverRuntime struct {
	id string

	// stdinMu serializes writes to the fxserver stdin pipe and protects against
	// use-after-close when waitForExit tears the pipe down. Always acquire
	// stdinMu BEFORE mu if both are needed, to avoid lock-order inversion.
	stdinMu sync.Mutex

	mu            sync.Mutex
	status        models.ServerStatus
	pid           int
	exitCode      *int
	exitReason    string
	updatedAt     time.Time
	stopRequested bool
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	done          chan struct{}
	nextSubID     int
	subs          map[int]chan models.ServerConsoleEvent
	logs          *ringBuffer
}

func (r *serverRuntime) Status() models.ServerProcessStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshotLocked()
}

func (r *serverRuntime) snapshotLocked() models.ServerProcessStatus {
	return models.ServerProcessStatus{
		ID:         r.id,
		Status:     r.status,
		PID:        r.pid,
		ExitCode:   cloneIntPtr(r.exitCode),
		ExitReason: r.exitReason,
		UpdatedAt:  r.updatedAt,
	}
}

func (r *serverRuntime) failStart(err error) {
	r.mu.Lock()
	r.status = models.ServerStatusCrashed
	r.exitCode = nil
	r.exitReason = err.Error()
	r.updatedAt = time.Now().UTC()
	if r.done != nil {
		close(r.done)
		r.done = nil
	}
	r.mu.Unlock()

	r.appendSystem(fmt.Sprintf("Launch failed: %v", err))
	r.broadcastStatus()
}

func (r *serverRuntime) beginStop() (pid int, stdin io.WriteCloser, done chan struct{}, alreadyStopped bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.status != models.ServerStatusStarting && r.status != models.ServerStatusRunning {
		return 0, nil, nil, true
	}

	r.stopRequested = true
	return r.pid, r.stdin, r.done, false
}

func (r *serverRuntime) capturePipe(stream string, pipe io.Reader) {
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerBufferMax)

	for scanner.Scan() {
		r.promoteToRunning()
		r.appendLine(stream, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		r.appendSystem(fmt.Sprintf("%s capture error: %v", stream, err))
	}
}

func (r *serverRuntime) promoteToRunning() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.status != models.ServerStatusStarting {
		return
	}

	r.status = models.ServerStatusRunning
	r.updatedAt = time.Now().UTC()
	go r.broadcastStatus()
}

func (r *serverRuntime) promoteToRunningAfter(delay time.Duration, done <-chan struct{}) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-done:
		return
	case <-timer.C:
		r.promoteToRunning()
	}
}

func (r *serverRuntime) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()

	exitCode := 0
	switch {
	case cmd.ProcessState != nil:
		exitCode = cmd.ProcessState.ExitCode()
	case err != nil:
		exitCode = -1
	}

	waitReason := "process exited cleanly"
	if err != nil {
		waitReason = err.Error()
		if exitCode >= 0 {
			waitReason = fmt.Sprintf("process exited with code %d", exitCode)
		}
	}

	// Serialize with SendCommand before niling the pipe so an in-flight write
	// cannot race the teardown.
	r.stdinMu.Lock()
	r.mu.Lock()
	stopRequested := r.stopRequested
	r.cmd = nil
	r.stdin = nil
	r.pid = 0
	r.updatedAt = time.Now().UTC()

	// A panel-initiated stop always lands in Stopped — the force-kill path
	// legitimately leaves exitCode != 0 (SIGTERM/SIGKILL/taskkill produce 143,
	// 137, 1 etc.) and that must not masquerade as a crash. The raw code is
	// kept in exitCode for diagnostics either way.
	switch {
	case stopRequested:
		r.status = models.ServerStatusStopped
		if exitCode != 0 {
			r.exitReason = fmt.Sprintf("stopped by panel (exit %d)", exitCode)
			r.exitCode = intPtr(exitCode)
		} else {
			r.exitReason = "stopped by panel"
			r.exitCode = nil
		}
	case exitCode != 0:
		r.status = models.ServerStatusCrashed
		r.exitReason = waitReason
		r.exitCode = intPtr(exitCode)
	default:
		r.status = models.ServerStatusStopped
		r.exitReason = waitReason
		r.exitCode = nil
	}

	done := r.done
	r.done = nil
	r.stopRequested = false
	r.mu.Unlock()
	r.stdinMu.Unlock()

	if done != nil {
		close(done)
	}

	if exitCode != 0 {
		r.appendSystem(fmt.Sprintf("Server exited with code %d: %s", exitCode, waitReason))
	}
	r.broadcastStatus()
}

func (r *serverRuntime) appendCommand(command string) {
	line := r.logs.add("stdin", command)
	r.broadcast(models.ServerConsoleEvent{
		Type: "line",
		Line: &line,
	})
}

func (r *serverRuntime) appendSystem(message string) {
	line := r.logs.add("system", message)
	r.broadcast(models.ServerConsoleEvent{
		Type: "line",
		Line: &line,
	})
}

func (r *serverRuntime) appendLine(stream, message string) {
	line := r.logs.add(stream, message)
	r.broadcast(models.ServerConsoleEvent{
		Type: "line",
		Line: &line,
	})
}

func (r *serverRuntime) broadcastStatus() {
	status := r.Status()
	r.broadcast(models.ServerConsoleEvent{
		Type:   "status",
		Status: &status,
	})
}

func (r *serverRuntime) subscribe() *Subscription {
	r.mu.Lock()
	id := r.nextSubID
	r.nextSubID++
	ch := make(chan models.ServerConsoleEvent, 256)
	r.subs[id] = ch
	r.mu.Unlock()

	return &Subscription{
		C: ch,
		closeFn: func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			if sub, ok := r.subs[id]; ok {
				delete(r.subs, id)
				close(sub)
			}
		},
	}
}

func (r *serverRuntime) broadcast(event models.ServerConsoleEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, ch := range r.subs {
		if offerEvent(ch, event) {
			continue
		}
		delete(r.subs, id)
		close(ch)
	}
}

// ringBuffer stores the most recent log lines in a fixed-capacity circular
// buffer. Writes are O(1) regardless of how many lines have passed through;
// older entries are overwritten in place once capacity is reached.
type ringBuffer struct {
	mu       sync.RWMutex
	capacity int
	nextID   int64
	buf      []models.ServerLogLine
	head     int // index of the next write slot
	size     int // number of valid entries currently stored
}

func newRingBuffer(capacity int) *ringBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &ringBuffer{
		capacity: capacity,
		buf:      make([]models.ServerLogLine, capacity),
	}
}

func (r *ringBuffer) add(stream, message string) models.ServerLogLine {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	line := models.ServerLogLine{
		ID:        r.nextID,
		Timestamp: time.Now().UTC(),
		Stream:    stream,
		Message:   message,
	}

	r.buf[r.head] = line
	r.head = (r.head + 1) % r.capacity
	if r.size < r.capacity {
		r.size++
	}

	return line
}

func (r *ringBuffer) tail(n int) []models.ServerLogLine {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n <= 0 || n > r.size {
		n = r.size
	}
	if n == 0 {
		return []models.ServerLogLine{}
	}

	out := make([]models.ServerLogLine, n)
	// The oldest of the returned `n` entries sits n slots before head.
	start := (r.head - n + r.capacity) % r.capacity
	for i := 0; i < n; i++ {
		out[i] = r.buf[(start+i)%r.capacity]
	}
	return out
}

func offerEvent(ch chan models.ServerConsoleEvent, event models.ServerConsoleEvent) bool {
	select {
	case ch <- event:
		return true
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case ch <- event:
		return true
	default:
		return false
	}
}

func waitForDone(ctx context.Context, done <-chan struct{}, timeout time.Duration) bool {
	if done == nil {
		return true
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		return false
	}
}

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func intPtr(v int) *int {
	return &v
}
