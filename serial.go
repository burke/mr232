package mr232

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type Serial struct {
	Lines chan string

	m    sync.Mutex
	f    *os.File
	stop chan struct{}
	done chan struct{}
}

func (s *Serial) Send(msg string) error {
	s.m.Lock()
	defer s.m.Unlock()

	n, err := s.f.Write([]byte(msg + "\r"))
	if err != nil {
		return err
	}
	if n != len(msg)+1 {
		return fmt.Errorf("short write")
	}
	return nil
}

func configureTTY(f *os.File) error {
	t := syscall.Termios{
		Iflag:  syscall.IGNPAR,
		Oflag:  0,
		Cflag:  syscall.CREAD | syscall.CLOCAL | syscall.B38400 | syscall.CS8,
		Lflag:  0,
		Cc:     [ccWidth]uint8{syscall.VMIN: 0, syscall.VTIME: 5},
		Ispeed: syscall.B38400,
		Ospeed: syscall.B38400,
	}

	if _, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()), setTermios, uintptr(unsafe.Pointer(&t)), 0, 0, 0,
	); errno != 0 {
		return fmt.Errorf("errno %d", errno)
	}
	return nil
}

func readLoop(s *Serial) {
	line := ""
	buf := make([]byte, 1)
	for {
		select {
		case <-s.stop:
			close(s.done)
			return
		default:
		}

		n, err := s.f.Read(buf)
		if n == 1 {
			line += string(buf)
			if strings.HasSuffix(line, "\r\n") {
				trimmed := strings.TrimSuffix(line, "\r\n")
				s.Lines <- trimmed
				line = ""
			}
		}
		if err != nil {
			if err == io.EOF {
				continue // read timed out. just start another.
			}
			panic(err)
		}
	}
}

func StartSerial(device string) (*Serial, error) {
	f, err := os.OpenFile(device, syscall.O_RDWR|syscall.O_NOCTTY, 0666)
	if err != nil {
		return nil, err
	}

	if err := configureTTY(f); err != nil {
		return nil, err
	}

	s := &Serial{
		f:     f,
		Lines: make(chan string, 256),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}

	_, _ = ioutil.ReadAll(s.f) // drain any existing buffered data

	go readLoop(s)
	return s, nil
}

func (s *Serial) Close() error {
	close(s.stop)
	<-s.done
	return s.f.Close()
}
