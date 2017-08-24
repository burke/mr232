package mr232

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type MR232 struct {
	s  *Serial
	mu sync.Mutex

	stop      chan struct{}
	done      chan struct{}
	listeners []chan string
	Lines     chan string
}

func Start(device string) (*MR232, error) {
	s, err := StartSerial(device)
	m := &MR232{
		s:     s,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
		Lines: make(chan string, 256),
	}

	go filterLoop(m)

	return m, err
}

func addListener(m *MR232) chan string {
	listener := make(chan string)
	m.mu.Lock()
	m.listeners = append(m.listeners, listener)
	m.mu.Unlock()
	return listener
}

func removeListener(m *MR232, listener chan string) {
	m.mu.Lock()
	var list []chan string
	for _, l := range m.listeners {
		if l != listener {
			list = append(list, l)
		}
	}
	m.listeners = list
	m.mu.Unlock()
}

func (m *MR232) GroupStatus(groupID int) (int, int, int, int, error) {
	listener := addListener(m)

	err := m.Send(fmt.Sprintf("stsg %d", groupID))
	if err != nil {
		return -1, -1, -1, -1, err
	}

	defer removeListener(m, listener)

	timeout := time.After(8 * time.Second)
	for {
		select {
		case line := <-listener:
			if strings.HasPrefix(line, fmt.Sprintf(">GS, %04d", groupID)) {
				var gid, curr, prev, a, b int
				_, err := fmt.Sscanf(line, ">GS, %d, %d, %d, %d, %d", &gid, &curr, &prev, &a, &b)
				if err != nil {
					return -1, -1, -1, -1, err
				}
				return curr, prev, a, b, nil
			}
		case <-timeout:
			return -1, -1, -1, -1, fmt.Errorf("timeout")
		}
	}
}

func (m *MR232) RampGroup(groupID int, toLevel int) error {
	return m.Send(fmt.Sprintf("rampg %d %d", groupID, toLevel))
}

func (m *MR232) Send(msg string) error {
	return m.s.Send(msg)
}

func (m *MR232) Close() error {
	close(m.stop)
	<-m.done
	return m.s.Close()
}

func filterLoop(m *MR232) {
	for {
		select {
		case line := <-m.s.Lines:
			for _, l := range m.listeners {
				l <- line
			}
			m.Lines <- line
		case <-m.stop:
			close(m.done)
			return
		}
	}
}
