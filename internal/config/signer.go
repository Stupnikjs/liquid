package config

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// useless remplacer par un seul

func NewSigner(chainId int64) (*Signer, error) {

	keyHex := os.Getenv("BASE_PK")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__PRIVATE_KEY not set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Signer{
		key:    key,
		signer: types.NewLondonSigner(big.NewInt(chainId)),
	}, nil
}

func (s *Signer) Sign(tx *types.Transaction) (*types.Transaction, error) {
	return types.SignTx(tx, s.signer, s.key)
}
