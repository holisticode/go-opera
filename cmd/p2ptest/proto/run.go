package proto

import (
	"encoding/json"
	"os"
)

func LoadSequence(sequenceDescriptionPath string) (*Sequence, error) {
	seqDesc, err := os.ReadFile(sequenceDescriptionPath)
	if err != nil {
		return nil, err
	}

	var sequence *Sequence

	if err := json.Unmarshal(seqDesc, &sequence); err != nil {
		return nil, err
	}

	return sequence, nil
}
