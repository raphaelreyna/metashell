package metashell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	daemonproto "github.com/raphaelreyna/shelld/internal/rpc/go/daemon"

	. "github.com/raphaelreyna/shelld/internal/log"
)

type MetaShell struct {
	config Config

	cmd           *exec.Cmd
	ptmx          *os.File
	sigChan       chan os.Signal
	originalState *unix.Termios

	grpcConn *grpc.ClientConn
	client   daemonproto.MetashellDaemonClient
	ecStream daemonproto.MetashellDaemon_NewExitCodeStreamClient

	doneChan  chan error
	cancelCtx func()

	tty      string
	ExecPath string

	in        io.Reader
	out       *os.File
	cmdBuffer string
	scanner   *bufio.Scanner

	cmdIsRunning bool

	sync.RWMutex
}

func (ms *MetaShell) stop() {
	ms.cancelCtx()

	ms.ecStream.CloseSend()
	ms.grpcConn.Close()

	signal.Stop(ms.sigChan)
	close(ms.sigChan)

	if x := ms.ptmx; x != nil {
		x.Close()
	}

	restoreTTYSettings(int(os.Stdin.Fd()), ms.originalState)

	ms.doneChan <- nil
}

func (ms *MetaShell) Run(ctx context.Context) error {
	ctx, ms.cancelCtx = context.WithCancel(ctx)

	err := ms.ensureDaemon(ctx)
	if err != nil {
		return err
	}

	ms.grpcConn, err = grpc.Dial("unix://"+ms.config.socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer ms.grpcConn.Close()

	ms.cmd = exec.CommandContext(ctx, ms.config.ShellPath)
	ptmx, err := pty.Start(ms.cmd)
	if err != nil {
		return err
	}
	ms.tty = ms.cmd.Stdin.(*os.File).Name()

	ms.client = daemonproto.NewMetashellDaemonClient(ms.grpcConn)

	header := metadata.New(map[string]string{"TTY": ms.tty})
	ms.ecStream, err = ms.client.NewExitCodeStream(
		metadata.NewOutgoingContext(ctx, header), &empty.Empty{},
	)
	if err != nil {
		return err
	}

	go func() {
		var err error
		for {
			Log.Info().Msg("received exit code")
			_, err = ms.ecStream.Recv()
			if err != nil {
				Log.Error().
					Err(err).
					Msg("error reading from exit code stream")
				return
			}
			ms.Lock()
			ms.cmdIsRunning = false
			ms.Unlock()
		}
	}()

	// propagate os signals
	ms.sigChan = make(chan os.Signal, 1)
	ms.doneChan = make(chan error, 1)
	signal.Notify(ms.sigChan,
		syscall.SIGWINCH|
			syscall.SIGTERM|
			syscall.SIGKILL|
			syscall.SIGINT,
	)

	go func() {
		for sig := range ms.sigChan {
			switch sig {
			case syscall.SIGWINCH:
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					panic(err)
				}
			case syscall.SIGTERM:
				ms.cmd.Process.Signal(sig)
				ms.stop()
			case syscall.SIGKILL:
				ms.cmd.Process.Signal(sig)
				ms.stop()
			case syscall.SIGINT:
				ms.cmd.Process.Signal(sig)
			}
		}
	}()
	ms.sigChan <- syscall.SIGWINCH

	ms.originalState, err = setTTYSettings(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	ms.in = os.Stdin
	ms.out = ptmx

	go ms.start(ctx)
	go func() { _, _ = io.Copy(os.Stdout, ptmx) }()

	if _, err := fmt.Fprintf(ptmx, ". <(%s install)\n", ms.ExecPath); err != nil {
		return err
	}

	return <-ms.doneChan
}

func (ms *MetaShell) start(ctx context.Context) {
	ms.scanner = bufio.NewScanner(ms.in)

	ms.scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		switch len(data) {
		case 0:
		default:
			advance = 1
			token = data
		}
		return
	})

	for ms.scanner.Scan() {
		ms.RLock()
		cmdIsRunning := ms.cmdIsRunning
		ms.RUnlock()

		input := ms.scanner.Bytes()
		if !cmdIsRunning {
			switch input[0] {
			case 27: // ESC
				Log.Info().Msg("ESC")
			case 13: // \n
				Log.Info().Msg("registering")
				_, err := ms.client.RegisterCommandEntry(ctx, &daemonproto.CommandEntry{
					Command:   ms.cmdBuffer,
					Tty:       ms.tty,
					Timestamp: time.Now().Unix(),
				})
				if err != nil {
					Log.Error().
						Err(err).
						Msg("error registering command with daemon")
				}

				ms.cmdBuffer = ""
				ms.Lock()
				ms.cmdIsRunning = true
				ms.Unlock()

				ms.out.Write([]byte{13})
			default:
				ms.cmdBuffer += string(input)
				ms.out.Write(input)
			}
		} else {
			ms.out.Write(input)
		}
	}
}

func setTTYSettings(fd int) (*unix.Termios, error) {
	const ioctlReadTermios = unix.TCGETS
	const ioctlWriteTermios = unix.TCSETS

	termios, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return nil, err
	}

	old := termios

	// man termios(3)
	termios.Iflag &^= unix.IGNBRK |
		unix.BRKINT |
		unix.PARMRK |
		unix.ISTRIP |
		unix.INLCR |
		unix.IGNCR |
		unix.ICRNL |
		unix.IXON
	termios.Iflag &= unix.IUTF8
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO |
		unix.ECHONL |
		unix.ICANON |
		unix.ISIG |
		unix.IEXTEN
	termios.Cflag &^= unix.CSIZE |
		unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, termios); err != nil {
		return nil, err
	}

	return old, nil
}

func restoreTTYSettings(fd int, old *unix.Termios) error {
	const ioctlWriteTermios = unix.TCSETS
	return unix.IoctlSetTermios(fd, ioctlWriteTermios, old)
}

func (ms *MetaShell) ensureDaemon(ctx context.Context) error {
	_, err := os.Stat(ms.config.socketPath)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	cmd := exec.Command(os.Args[0], "daemon")
	_, err = cmd.CombinedOutput()
	if err == nil {
		time.Sleep(500 * time.Millisecond)
	}

	return err
}
