package gossip

import (
	"fmt"
	"testing"
)

// this file has been created by fabio for testing. If it appears somehow in your tree, it can and should be removed, it's a mistake.

func TestFabioEnv(t *testing.T) {
	env := newTestEnv(42, 5)
	fmt.Println(fmt.Sprintf("%v", env.store.GetEpoch()))
}
