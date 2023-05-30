package ethapi

import (
	"context"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicAbftAPI provides an API to access consensus related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicAbftAPI struct {
	b Backend
}

// NewPublicAbftAPI creates a new SFC protocol API.
func NewPublicAbftAPI(b Backend) *PublicAbftAPI {
	return &PublicAbftAPI{b}
}

func (s *PublicAbftAPI) GetValidators(ctx context.Context, epoch rpc.BlockNumber) (map[hexutil.Uint64]interface{}, error) {
	bs, es, err := s.b.GetEpochBlockState(ctx, epoch)
	if err != nil {
		return nil, err
	}
	if es == nil {
		return nil, nil
	}
	res := map[hexutil.Uint64]interface{}{}
	for _, vid := range es.Validators.IDs() {
		profiles := es.ValidatorProfiles
		if epoch == rpc.PendingBlockNumber {
			profiles = bs.NextValidatorProfiles
		}
		res[hexutil.Uint64(vid)] = map[string]interface{}{
			"weight": (*hexutil.Big)(profiles[vid].Weight),
			"pubkey": profiles[vid].PubKey.String(),
		}
	}
	return res, nil
}

// GetDowntime returns validator's downtime.
func (s *PublicAbftAPI) GetDowntime(ctx context.Context, validatorID hexutil.Uint) (map[string]interface{}, error) {
	blocks, period, err := s.b.GetDowntime(ctx, idx.ValidatorID(validatorID))
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"offlineBlocks": hexutil.Uint64(blocks),
		"offlineTime":   hexutil.Uint64(period),
	}, nil
}

// GetEpochUptime returns validator's epoch uptime in nanoseconds.
func (s *PublicAbftAPI) GetEpochUptime(ctx context.Context, validatorID hexutil.Uint) (hexutil.Uint64, error) {
	v, err := s.b.GetUptime(ctx, idx.ValidatorID(validatorID))
	if err != nil {
		return 0, err
	}
	if v == nil {
		return 0, nil
	}
	return hexutil.Uint64(v.Uint64()), nil
}

func (s *PublicAbftAPI) GetEpochBlockState(ctx context.Context, epoch hexutil.Uint) (map[string]interface{}, error) {
	ep := rpc.BlockNumber(epoch)
	_, es, err := s.b.GetEpochBlockState(ctx, ep)
	if err != nil {
		return nil, err
	}
	bytes, err := rlp.EncodeToBytes(es)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		/*
			"epoch" : hexutil.Uint64(es.Epoch),
			"esstart": hexutil.Uint64(es.EpochStart),
			"prevstart": hexutil.Uint64(es.PrevEpochStart),
			"stateroot" : hexutil.Bytes(es.EpochStateRoot[:]),
			"rules":   hexutil.Uint64(es.Rules),
		*/
		"rlp_encoded": hexutil.Bytes(bytes),
	}, nil
}

// GetOriginatedEpochFee returns validator's originated epoch fee.
func (s *PublicAbftAPI) GetOriginatedEpochFee(ctx context.Context, validatorID hexutil.Uint) (*hexutil.Big, error) {
	v, err := s.b.GetOriginatedFee(ctx, idx.ValidatorID(validatorID))
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(v), nil
}
