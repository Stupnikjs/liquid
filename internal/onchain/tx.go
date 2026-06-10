package onchain

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/Stupnikjs/liquid/internal/config"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type TxParams struct {
	To          *common.Address
	Calldata    []byte
	Value       *big.Int
	GasEstimate uint64
}

func SendSignedTx(
	ctx context.Context,
	conn connector.Connector,
	msgsender common.Address,
	signer *config.Signer,
	params TxParams,
) (common.Hash, error) {

	// --- 1. Fetch nonce + gasPrice en batch ---
	var (
		nonceHex    hexutil.Uint64
		gasPriceHex hexutil.Big
	)
	batch := []rpc.BatchElem{
		{
			Method: "eth_getTransactionCount",
			Args:   []interface{}{msgsender, "latest"},
			Result: &nonceHex,
		},
		{
			Method: "eth_gasPrice",
			Args:   []interface{}{},
			Result: &gasPriceHex,
		},
	}
	if err := conn.CallCtx(ctx, batch); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: fetch params: %w", err)
	}
	for _, elem := range batch {
		if elem.Error != nil {
			return common.Hash{}, fmt.Errorf("SendSignedTx: fetch params: %w", elem.Error)
		}
	}

	nonce := uint64(nonceHex)
	gasPrice := gasPriceHex.ToInt()

	// --- 2. Construire et signer la tx ---
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

	// --- 3. Encoder et envoyer via eth_sendRawTransaction ---
	rawTx, err := signedTx.MarshalBinary()
	if err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: marshal: %w", err)
	}

	var hash common.Hash
	sendBatch := []rpc.BatchElem{
		{
			Method: "eth_sendRawTransaction",
			Args:   []interface{}{hexutil.Encode(rawTx)},
			Result: &hash,
		},
	}
	if err := conn.CallCtx(ctx, sendBatch); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: send: %w", err)
	}
	if sendBatch[0].Error != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: send: %w", sendBatch[0].Error)
	}

	log.Printf("[tx] sent: %s", hash.Hex())
	return hash, nil
}
