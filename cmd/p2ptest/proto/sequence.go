package proto

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"
)

type Sequence struct {
	Steps    []Interaction `json:"steps"`
	p2pLayer p2pProtocolTest
	logger   *zap.Logger
}

type Interaction struct {
	Input  Input    `json:"input"`
	Output []Output `json:"output"`
	Label  string
}

type Input struct {
	Msg  interface{}
	Code uint64
}

type Output struct {
	//Content []byte `json:"content"`
	Msg    interface{}
	Code   uint64
	Verify func(input interface{}, output interface{}) error
}

var (
	ErrProtocolTimeout       = errors.New("timed out waiting for protocol to be established")
	ErrMessageCodesDontMatch = errors.New("received and expected message codes do not match")
	ErrMessageTimeout        = errors.New("did not receive a message during the expected time")
	ErrMessageFailedDecoding = errors.New("failed decoding of received message")
)

const (
	MaxMessageTimeout = 6 * time.Second
)

func (seq *Sequence) setup(urls []string) error {
	var err error

	if seq.p2pLayer == nil {
		seq.p2pLayer, err = New(urls, seq.logger)
		if err != nil {
			return err
		}
	}
	return nil
}

func (seq *Sequence) Run(urls []string) error {
	defer seq.logger.Sync()

	if err := seq.setup(urls); err != nil {
		return err
	}

	// we just ran AddPeer; wait for peer connection to be established
	select {
	case <-time.After(10 * time.Second):
		return ErrProtocolTimeout
	case <-seq.p2pLayer.Initalized():
	}
	seq.logger.Debug("protocol initialized; start sequence")

	for i, step := range seq.Steps {
		fmt.Println()
		seq.logger.Info("running step: ", zap.String("step", step.Label), zap.Int("step", i))
		if err := seq.p2pLayer.Send(step.Input.Msg, step.Input.Code); err != nil {
			seq.logger.Error("failed to send", zap.Error(err))
			return err
		}

		received := 0
		msgTimeout := time.NewTimer(MaxMessageTimeout)
		loop := true
		for received < len(step.Output) && loop {
			select {
			case <-msgTimeout.C:
				if received == len(step.Output) {
					seq.logger.Debug("timeout occurred but it was expected to not receive a message here; just continue")
					loop = false
					continue
				}
				return ErrMessageTimeout
			case msg := <-seq.p2pLayer.Receive():
				seq.logger.Debug("handling message", zap.Uint64("code", msg.Code))
				if msg.Code != step.Output[received].Code {
					return ErrMessageCodesDontMatch
				}

				//var decodedMsg = reflect.New(reflect.TypeOf(step.Output[received].Msg)).Elem().Interface()
				var decodedMsg = reflect.New(reflect.TypeOf(step.Output[received].Msg))
				if err := msg.Decode(decodedMsg.Interface()); err != nil {
					fmt.Println(err)
					return ErrMessageFailedDecoding
				}
				reflect.ValueOf(&step.Output[received].Msg).Elem().Set(decodedMsg.Elem())
				if step.Output[received].Verify != nil {
					if err := step.Output[received].Verify(step.Input.Msg, step.Output[received].Msg); err != nil {
						return fmt.Errorf("message verification failed: %w", err)
					}
				}
				seq.logger.Debug("step matched")
				msgTimeout.Reset(MaxMessageTimeout)
			}
			received++
		}

		seq.logger.Info("step successful")
	}

	return nil
}
