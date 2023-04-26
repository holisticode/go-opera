package proto

import (
	"encoding/json"
	"os"

	"go.uber.org/zap"
)

func LoadSequence(getSequence func() *Sequence, logger *zap.Logger) (*Sequence, error) {
	s := getSequence()
	s.logger = logger
	return s, nil
}

func LoadJSONSequence(sequenceDescriptionPath string, logger *zap.Logger) (*Sequence, error) {
	seqDesc, err := os.ReadFile(sequenceDescriptionPath)
	if err != nil {
		return nil, err
	}

	var sequence *Sequence

	if err := json.Unmarshal(seqDesc, &sequence); err != nil {
		return nil, err
	}

	sequence.logger = logger

	return sequence, nil
}
