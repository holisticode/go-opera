package proto

import (
	"errors"
	"time"

	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type Sequence struct {
	Steps    []Interaction `json:"steps"`
	sender   sender
	receiver receiver
	logger *zap.Logger
}

type Interaction struct {
	Input  Input  `json:"input"`
	utput Output `json:"output"`
}

type Input struct {
	Code uint64 `json:"code"`
}

func (i Input) ToMsg() interface{} {
	switch i.Code {
	case gossip.HandshakeMsg: return gossip.HandshakeData{
		ProtocolVersion: 63,
		NetworkID: 0,
		Genesis: common.HexToHash("0x2c210befc091e71047cc7efb2b7789805c9dbd3081f08e67ecc9ca2236a510c0"),
	}
	}
	return nil
}

type Output struct {
	Content []byte `json:"content"`
}

var (
	ErrBytesDontMatch = errors.New("received and expected message do not match")
)

func (seq *Sequence) setup(urls []string) error {
	var err error


	seq.logger,err = zap.NewProduction()
	if seq.sender == nil {
		seq.sender,err = NewSender(urls)
		if err != nil {
			return err
		}
	}
	/*
	if seq.receiver == nil {
		r,err := NewReceiver() 
		if err != nil {
			fmt.Println(err)
		}
		seq.receiver = r
	}
	*/
	return nil
}

func (seq *Sequence) Run(urls []string) error {
    //defer seq.logger.Sync()

	if err := seq.setup(urls);err != nil {
		return err
	}
	seq.logger.Info("setup ok")
	for i, step := range seq.Steps {
		seq.logger.Info("sending", zap.Int("step",i))
		if err := seq.sender.Send(step.Input.ToMsg(), step.Input.Code); err != nil {
			seq.logger.Error("failed to send", zap.Error(err))
			return err
		}

		/*
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
		*/
	}

	time.Sleep(10*time.Second)

	return nil
}
