package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"

	"mypaas/internal/errs"
)

const (
	maxInputBytes     = 8 * 1024
	sessionDuration   = 30 * time.Minute
	sessionIdlePeriod = 10 * time.Minute
	watchInterval     = 30 * time.Second
)

type SessionInfo struct {
	ID        uuid.UUID `json:"id"`
	Shell     string    `json:"shell"`
	StartedAt time.Time `json:"startedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Event struct {
	Type string
	Data string
}

type Service struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*session
}

type session struct {
	info         SessionInfo
	cmd          *exec.Cmd
	terminal     *os.File
	cancel       context.CancelFunc
	done         chan struct{}
	finishOnce   sync.Once
	mu           sync.Mutex
	subscribers  map[chan Event]struct{}
	lastActivity time.Time
}

func NewService() *Service {
	return &Service{sessions: make(map[uuid.UUID]*session)}
}

func (s *Service) Start(ctx context.Context) (SessionInfo, error) {
	path, args, label, err := shellCommand()
	if err != nil {
		return SessionInfo{}, err
	}

	sessionCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	cmd := exec.CommandContext(sessionCtx, path, args...)
	if runtime.GOOS != "windows" {
		cmd.Dir = "/"
	}
	cmd.Env = append(os.Environ(), "TERM=dumb", "PS1=mypaas$ ")

	terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 120})
	if err != nil {
		cancel()
		return SessionInfo{}, fmt.Errorf("start shell pty: %w", err)
	}

	startedAt := time.Now().UTC()
	sess := &session{
		info: SessionInfo{
			ID:        uuid.New(),
			Shell:     label,
			StartedAt: startedAt,
			ExpiresAt: startedAt.Add(sessionDuration),
		},
		cmd:          cmd,
		terminal:     terminal,
		cancel:       cancel,
		done:         make(chan struct{}),
		subscribers:  make(map[chan Event]struct{}),
		lastActivity: startedAt,
	}

	s.mu.Lock()
	s.sessions[sess.info.ID] = sess
	s.mu.Unlock()

	go s.consume(sess)
	go s.watch(sess)
	return sess.info, nil
}

func (s *Service) Subscribe(id uuid.UUID) (<-chan Event, <-chan struct{}, SessionInfo, func(), error) {
	s.mu.RLock()
	sess := s.sessions[id]
	s.mu.RUnlock()
	if sess == nil {
		return nil, nil, SessionInfo{}, func() {}, errs.ErrShellSessionNotFound
	}

	events := make(chan Event, 64)
	sess.mu.Lock()
	sess.subscribers[events] = struct{}{}
	info := sess.info
	sess.mu.Unlock()

	unsubscribe := func() {
		sess.mu.Lock()
		delete(sess.subscribers, events)
		sess.mu.Unlock()
	}
	return events, sess.done, info, unsubscribe, nil
}

func (s *Service) Write(id uuid.UUID, data string) error {
	if len([]byte(data)) > maxInputBytes {
		return errs.ErrShellInputTooLarge
	}

	s.mu.RLock()
	sess := s.sessions[id]
	s.mu.RUnlock()
	if sess == nil {
		return errs.ErrShellSessionNotFound
	}

	sess.mu.Lock()
	select {
	case <-sess.done:
		sess.mu.Unlock()
		return errs.ErrShellSessionClosed
	default:
	}
	sess.lastActivity = time.Now().UTC()
	terminal := sess.terminal
	sess.mu.Unlock()

	if _, err := io.WriteString(terminal, data); err != nil {
		select {
		case <-sess.done:
			return errs.ErrShellSessionClosed
		default:
		}
		return fmt.Errorf("write shell input: %w", err)
	}
	return nil
}

func (s *Service) Stop(id uuid.UUID) error {
	s.mu.RLock()
	sess := s.sessions[id]
	s.mu.RUnlock()
	if sess == nil {
		return errs.ErrShellSessionNotFound
	}
	sess.stop()
	return nil
}

func (s *Service) Close() {
	s.mu.RLock()
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.mu.RUnlock()
	for _, sess := range sessions {
		sess.stop()
	}
}

func (s *Service) consume(sess *session) {
	buffer := make([]byte, 16*1024)
	for {
		n, err := sess.terminal.Read(buffer)
		if n > 0 {
			sess.publish(Event{Type: "output", Data: string(buffer[:n])})
		}
		if err != nil {
			break
		}
	}

	_ = sess.cmd.Wait()
	sess.publish(Event{Type: "exit", Data: "Shell session ended."})
	sess.finish()

	s.mu.Lock()
	delete(s.sessions, sess.info.ID)
	s.mu.Unlock()
}

func (s *Service) watch(sess *session) {
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-sess.done:
			return
		case now := <-ticker.C:
			sess.mu.Lock()
			idle := now.Sub(sess.lastActivity) >= sessionIdlePeriod
			expired := !now.Before(sess.info.ExpiresAt)
			sess.mu.Unlock()
			if idle || expired {
				sess.stop()
				return
			}
		}
	}
}

func (sess *session) publish(event Event) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	for subscriber := range sess.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (sess *session) stop() {
	sess.mu.Lock()
	terminal := sess.terminal
	cmd := sess.cmd
	sess.mu.Unlock()

	_ = terminal.Close()
	sess.cancel()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (sess *session) finish() {
	sess.finishOnce.Do(func() {
		_ = sess.terminal.Close()
		sess.cancel()
		close(sess.done)
	})
}

func shellCommand() (string, []string, string, error) {
	if runtime.GOOS == "windows" {
		path, err := exec.LookPath("cmd.exe")
		if err != nil {
			return "", nil, "", fmt.Errorf("find host shell: %w", err)
		}
		return path, []string{"/Q"}, "cmd.exe", nil
	}

	for _, candidate := range []string{"bash", "sh"} {
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		if strings.HasSuffix(path, "/bash") {
			return path, []string{"--noprofile", "--norc", "-i"}, "bash", nil
		}
		return path, []string{"-i"}, "sh", nil
	}
	return "", nil, "", fmt.Errorf("find host shell: %w", errs.ErrNotFound)
}
