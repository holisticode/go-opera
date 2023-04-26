package proto

import (
	"errors"
	"fmt"
	"time"

	"github.com/Fantom-foundation/go-opera/gossip"
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
	seq.logger.Debug("setup ok")
	for i, step := range seq.Steps {
		seq.logger.Info("running step: ", zap.String("step", step.Label))
		seq.logger.Debug("sending", zap.Int("step", i))
		if err := seq.p2pLayer.Send(step.Input.Msg, step.Input.Code); err != nil {
			seq.logger.Error("failed to send", zap.Error(err))
			return err
		}

		received := 0
		msgTimeout := time.NewTimer(MaxMessageTimeout)
		// for received < len(step.Output) {
		loop := true
		for loop {
			select {
			case <-msgTimeout.C:
				if received == len(step.Output) {
					seq.logger.Debug("timeout occurred but it was expected to not receive a message here; just continue")
					loop = false
					continue
				}
				return ErrMessageTimeout
			case msg := <-seq.p2pLayer.Receive():
				seq.logger.Debug("received message", zap.Uint64("code", msg.Code))
				if msg.Code != step.Output[received].Code {
					return ErrMessageCodesDontMatch
				}

				//var decodedMsg = reflect.New(reflect.TypeOf(step.Output[received].Msg)).Elem().Interface()
				var decodedMsg = step.Output[received].Msg
				if err := msg.Decode(decodedMsg); err != nil {
					fmt.Println(err)
					return ErrMessageFailedDecoding
				}
				if step.Output[received].Verify != nil {
					if err := step.Output[received].Verify(step.Input.Msg, decodedMsg); err != nil {
						return fmt.Errorf("message verification failed: %w", err)
					}
				}
				seq.logger.Debug("step matched")
				msgTimeout.Reset(MaxMessageTimeout)
				if msg.Code == gossip.HandshakeMsg {
					seq.p2pLayer.Send(step.Input.Msg, step.Input.Code)
				}
			}
			received++
			if received >= len(step.Output) {
				break
			}
		}

		seq.logger.Info("step successful")
	}

	return nil
}
