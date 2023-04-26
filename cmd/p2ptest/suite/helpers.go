package suite

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

func createHashes(num int) []common.Hash {
	hashes := make([]common.Hash, num)

	for i := 0; i < num; i++ {
		hashes[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	return hashes
}
