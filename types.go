package mr232

import (
	"errors"
	"fmt"
	"strings"
)

const (
	MultiLocationController     uint8 = 33
	RoomPresetController        uint8 = 34
	HousePresetController       uint8 = 35
	RoomRemotePresetController  uint8 = 36
	HouseRemotePresetController uint8 = 37

	DimmableSwitchController    uint8 = 65
	NonDimmableSwitchController uint8 = 66
	FanController               uint8 = 67
)

var (
	ErrNoLoadDetected       = errors.New("No-Load Detected")
	ErrOverloadDetected     = errors.New("Overload Detected")
	ErrShortCircuitDetected = errors.New("Short Circuit Detected")
)

func mapErrorCode(code uint8) error {
	switch code {
	case 0:
		return nil
	case 134:
		return ErrNoLoadDetected
	case 132:
		return ErrOverloadDetected
	case 130:
		return ErrShortCircuitDetected
	default:
		return fmt.Errorf("Unknown Error Code: %d", code)
	}
}

type Message interface {
	String() string
}

type GenericMessage struct {
	s string
}

type GroupStatusMessage struct {
	*GenericMessage

	GroupID          uint16
	CurrentLevel     uint8
	LastNonZeroLevel uint8
	DeviceType       uint8
	LastError        error
}

type RampGroupMessage struct {
	*GenericMessage

	GroupID     uint16
	TargetLevel uint8
	RampRate    uint8
}

type CancelRampMessage struct {
	*GenericMessage

	GroupID      uint16
	CurrentLevel uint8
}

func (g *GenericMessage) String() string {
	return g.s
}

func parseMessage(line string) Message {
	if strings.HasPrefix(line, ">GS, ") {
		return parseGroupStatusMessage(line)
	} else if strings.HasPrefix(line, "RG, ") {
		return parseRampGroupMessage(line)
	} else if strings.HasPrefix(line, "CR, ") {
		return parseCancelRampMessage(line)
	}
	return &GenericMessage{line}
}

func parseGroupStatusMessage(line string) Message {
	var gid uint16
	var curr, prev, deviceType, lastErrCode uint8

	_, err := fmt.Sscanf(
		line,
		">GS, %d, %d, %d, %d, %d",
		&gid, &curr, &prev, &deviceType, &lastErrCode,
	)
	if err == nil {
		return &GroupStatusMessage{
			GenericMessage: &GenericMessage{s: line},

			GroupID:          gid,
			CurrentLevel:     curr,
			LastNonZeroLevel: prev,
			DeviceType:       deviceType,
			LastError:        mapErrorCode(lastErrCode),
		}
	}
	return &GenericMessage{line}
}

func parseRampGroupMessage(line string) Message {
	var gid uint16
	var level, rate uint8

	_, err := fmt.Sscanf(line, "RG, %d, %d, %d", &gid, &level, &rate)
	if err == nil {
		return &RampGroupMessage{
			GenericMessage: &GenericMessage{s: line},

			GroupID:     gid,
			TargetLevel: level,
			RampRate:    rate,
		}
	}
	return &GenericMessage{line}
}

func parseCancelRampMessage(line string) Message {
	var gid uint16
	var level uint8

	_, err := fmt.Sscanf(line, "CR, %d, %d", &gid, &level)
	if err == nil {
		return &CancelRampMessage{
			GenericMessage: &GenericMessage{s: line},

			GroupID:      gid,
			CurrentLevel: level,
		}
	}
	return &GenericMessage{line}
}
