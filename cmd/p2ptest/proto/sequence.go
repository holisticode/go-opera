package proto

import (
	"fmt"
	"bytes"
	"errors"
)

type Sequence struct {
	Steps    []Interaction `json: "steps"`
	sender   sender
	receiver receiver
}

type Interaction struct {
	Input  Input  `json: "input"`
	Output Output `json: "output"`
}

type Input struct {
	Content []byte `json: "content"`
}

type Output struct {
	Content []byte `json: "content"`
}

var (
	ErrBytesDontMatch = errors.New("received and expected message do not match")
)

func (seq *Sequence) setup(urls []string) {
	if seq.sender == nil {
		seq.sender = NewSender(urls)
	}
	if seq.receiver == nil {
		r,err := NewReceiver() 
		if err != nil {
			fmt.Println(err)
		}
		seq.receiver = r
	}
}

func (seq *Sequence) Run(urls []string) error {
	seq.setup(urls)
	for _, step := range seq.Steps {
		if err := seq.sender.Send(step.Input.Content); err != nil {
			return err
		}

		cmsg, err := seq.receiver.Receive()
		if err != nil {
			return err
		}

		var msg []byte

		select {
		case msg = <- cmsg :
		}

		if bytes.Compare(msg, step.Output.Content) != 0 {
			return ErrBytesDontMatch
		}
	}

	return nil
}
