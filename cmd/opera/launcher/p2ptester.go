package launcher

import (
	"flag"

	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/Fantom-foundation/go-opera/inter/validatorpk"
	"github.com/Fantom-foundation/go-opera/opera/genesis"
	"github.com/Fantom-foundation/go-opera/valkeystore"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"gopkg.in/urfave/cli.v1"
)

type P2PTestingNode struct {
	Node      *node.Node
	Service   *gossip.Service
	P2PServer *p2p.Server
	NodeClose func()
	Signer    valkeystore.SignerI
	Store     *gossip.Store
	Genesis   *genesis.Genesis
	PubKey    validatorpk.PubKey
}

func NewP2PTestingNode() *P2PTestingNode {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	app := cli.NewApp()

	fs.String("cache", "8000", "cache")
	fs.String("datadir", "/tmp/d", "datadir")
	fs.String("fakenet", "4/4", "fakenet")
	fs.Set("fakenet", "4/4")
	fs.Set("datadir", "/tmp/d")
	fs.Set("cache", "8000")
	ctx := cli.NewContext(app, fs, nil)
	cfg := makeAllConfigs(ctx)
	genesisStore := mayGetGenesisStore(ctx)

	return makeP2PTestNode(ctx, cfg, genesisStore)
}
