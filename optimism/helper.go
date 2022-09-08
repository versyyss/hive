package optimism

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/hive/hivesim"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

func BytesFile(name string, data []byte) hivesim.StartOption {
	return hivesim.WithDynamicFile(name, func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader(data)), nil
	})
}

func StringFile(name string, data string) hivesim.StartOption {
	return BytesFile(name, []byte(data))
}

var DefaultJWTPath = "/hive/input/jwt-secret.txt"
var defaultJWTSecret = common.Hash{42}
var defaultJWTFile = StringFile(DefaultJWTPath, defaultJWTSecret.String())
var DefaultP2PSequencerPrivPath = "/hive/input/p2p-sequencer-key.txt"
var DefaultP2PPrivPath = "/hive/input/p2p-key.txt"
var defaultP2PSequencerKey = common.Hash{32}
var defaultP2pSequencerKeyFile = StringFile(DefaultP2PSequencerPrivPath, strings.Replace(defaultP2PSequencerKey.Hex(), "0x", "", 1))

// HiveUnpackParams are hivesim.Params that have yet to be prefixed with "HIVE_UNPACK_".
//
// In the optimism monorepo we have many flags packages that define flags with namespaced env vars.
// We use these same env vars to configure the hive clients.
// But we need to add "HIVE_" to them, and then unpack them again in the hive client entrypoint, to use them.
//
// Within the client only params starting with "HIVE_UNPACK_" will be unpacked.
// E.g. "HIVE_UNPACK_OP_NODE_HELLO_WORLD" will become "OP_NODE_HELLO_WORLD"
type HiveUnpackParams hivesim.Params

func (u HiveUnpackParams) Params() hivesim.Params {
	out := make(hivesim.Params)
	for k, v := range u {
		out["HIVE_UNPACK_"+k] = v
	}
	return out
}
func (u HiveUnpackParams) Merge(other HiveUnpackParams) {
	for k, v := range other {
		u[k] = v
	}
}

func WaitBlock(ctx context.Context, client *ethclient.Client, n uint64) error {
	for {
		height, err := client.BlockNumber(ctx)
		if err != nil {
			return err
		}
		if height < n {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}

	return nil
}

func WaitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if receipt != nil && err == nil {
			return receipt, nil
		} else if err != nil && !errors.Is(err, ethereum.NotFound) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
