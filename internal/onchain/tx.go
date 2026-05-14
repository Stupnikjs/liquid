package onchain

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type TxParams struct {
	To          *common.Address
	Calldata    []byte
	Value       *big.Int
	GasEstimate uint64
}

// First Get Nonce and Gas price
// Then Send signed Tx
func SendSignedTx(ctx context.Context, client lqtypes.EthCaller, msgsender common.Address, signer *lqtypes.Signer, params TxParams) (common.Hash, error) {
	var nonce uint64
	var gasPrice *big.Int
	if err := client.EthCallCtx(ctx, []w3types.RPCCaller{
		eth.Nonce(msgsender, nil).Returns(&nonce),
		eth.GasPrice().Returns(&gasPrice),
	}); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: fetch params: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		To:        params.To,
		Data:      params.Calldata,
		Value:     params.Value,
		Gas:       params.GasEstimate * 12 / 10,
		GasTipCap: big.NewInt(1e9),
		GasFeeCap: new(big.Int).Add(gasPrice, big.NewInt(1e9)),
	})

	signedTx, err := signer.Sign(tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: sign: %w", err)
	}

	var receipt common.Hash
	if err := client.EthCallCtx(ctx, []w3types.RPCCaller{
		eth.SendTx(signedTx).Returns(&receipt),
	}); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: send: %w", err)
	}

	log.Printf("[tx] sent: %s", receipt.Hex())
	return receipt, nil
}
