package mr232

import (
	"fmt"
	"sync"
	"time"
)

type MR232 struct {
	s  *Serial
	mu sync.Mutex

	stop      chan struct{}
	done      chan struct{}
	listeners []chan Message
	Messages  chan Message
}

func Start(device string) (*MR232, error) {
	s, err := StartSerial(device)
	m := &MR232{
		s:        s,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		Messages: make(chan Message, 256),
	}

	go filterLoop(m)

	return m, err
}

func addListener(m *MR232) chan Message {
	listener := make(chan Message)
	m.mu.Lock()
	m.listeners = append(m.listeners, listener)
	m.mu.Unlock()
	return listener
}

func removeListener(m *MR232, listener chan Message) {
	m.mu.Lock()
	var list []chan Message
	for _, l := range m.listeners {
		if l != listener {
			list = append(list, l)
		}
	}
	m.listeners = list
	m.mu.Unlock()
}

func (m *MR232) GroupStatus(groupID uint16) (*GroupStatusMessage, error) {
	listener := addListener(m)

	err := m.Send(fmt.Sprintf("stsg %d", groupID))
	if err != nil {
		return nil, err
	}

	defer removeListener(m, listener)

	timeout := time.After(8 * time.Second)
	for {
		select {
		case msg := <-listener:
			if gsm, ok := msg.(*GroupStatusMessage); ok && gsm.GroupID == groupID {
				return gsm, nil
			}
		case <-timeout:
			return nil, fmt.Errorf("timeout")
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
			msg := parseMessage(line)
			for _, l := range m.listeners {
				l <- msg
			}
			m.Messages <- msg
		case <-m.stop:
			close(m.done)
			return
		}
	}
}
