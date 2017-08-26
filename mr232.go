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

func (m *MR232) BindGroup(groupID int) (chan struct{}, error) {
	return sendThenWaitForCancel(m, fmt.Sprintf("bindg %d", groupID))
}

func (m *MR232) BindHouse(presetID int) (chan struct{}, error) {
	return sendThenWaitForCancel(m, fmt.Sprintf("bindh %d", presetID))
}

func (m *MR232) BindRoom(roomID, presetID int) (chan struct{}, error) {
	return sendThenWaitForCancel(m, fmt.Sprintf("bindr %d %d", roomID, presetID))
}

func (m *MR232) CancelProportionalRampToRoom(roomID int) error {
	panic("not yet implemented")
}

func (m *MR232) CancelRampToGroup(groupID int) error {
	panic("not yet implemented")
}

func (m *MR232) GetDeviceIDs() error {
	panic("not yet implemented")
}

func (m *MR232) GetDeviceIDList() error {
	panic("not yet implemented")
}

func (m *MR232) Help() error {
	return m.Send("help")
}

func (m *MR232) IdentifyRoom(roomID int, timeInterval int) error {
	panic("not yet implemented")
}

func (m *MR232) IdentifyGroup(groupID int, timeInterval int) error {
	panic("not yet implemented")
}

func (m *MR232) SetGroupToLastNonZero(groupID int) error {
	panic("not yet implemented")
}

func (m *MR232) SetRoomToLastNonZero(roomID int) error {
	panic("not yet implemented")
}

func (m *MR232) LockHouse(locked bool) error {
	panic("not yet implemented")
}

func (m *MR232) LockRoom(locked bool, roomID int) error {
	panic("not yet implemented")
}

func (m *MR232) OverrideHouseToPreset(presetID int) error {
	panic("not yet implemented")
}

func (m *MR232) OverrideRoomToPreset(roomID int, presetID int) error {
	panic("not yet implemented")
}

func (m *MR232) Panic(enable bool) error {
	if enable {
		return m.Send("panic 1")
	}
	return m.Send("panic 0")
}

func (m *MR232) ProportionalRampRoomDown() error {
	panic("not yet implemented")
}

func (m *MR232) ProportionalRampRoomUp() error {
	panic("not yet implemented")
}

func (m *MR232) RecallHousePreset(presetID int) error {
	return m.Send(fmt.Sprintf("rchp %d", presetID))
}

func (m *MR232) RecallRoomPreset(roomID, presetID int) error {
	return m.Send(fmt.Sprintf("rcrp %d %d", roomID, presetID))
}

// default rampRate = 50
func (m *MR232) RampGroup(groupID int, toLevel, rampRate int) error {
	return m.Send(fmt.Sprintf("rampg %d %d %d", groupID, toLevel, rampRate))
}

func (m *MR232) RampRoom(roomID int, toLevel, rampRate int) error {
	return m.Send(fmt.Sprintf("rampr %d %d %d", roomID, toLevel, rampRate))
}

func (m *MR232) RevertOverrideHousePreset() error {
	panic("not yet implemented")
}

func (m *MR232) RevertOverrideRoomPreset() error {
	panic("not yet implemented")
}

func (m *MR232) SaveRoomPreset() error {
	panic("not yet implemented")
}

func (m *MR232) SaveHousePreset() error {
	panic("not yet implemented")
}

func (m *MR232) SetBuildingID() error {
	panic("not yet implemented")
}

func (m *MR232) SetHouseID() error {
	panic("not yet implemented")
}

func (m *MR232) TerminalSetup() error {
	panic("not yet implemented")
}

func (m *MR232) Reset() error {
	panic("not yet implemented")
}

func sendThenWaitForCancel(m *MR232, msg string) (chan struct{}, error) {
	m.mu.Lock()
	err := m.lockedSend(msg)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}

	ch := make(chan struct{})

	go func() {
		<-ch
		m.lockedSend("") // Executing, Press Enter to Stop...
		m.mu.Unlock()
	}()

	return ch, nil
}

func (m *MR232) GroupStatus(groupID int) (*GroupStatusMessage, error) {
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

func (m *MR232) Send(msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lockedSend(msg)
}

func (m *MR232) lockedSend(msg string) error {
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
