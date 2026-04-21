package config

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Signer struct {
	key    *ecdsa.PrivateKey
	signer types.Signer
}

func NewBaseSigner() (*Signer, error) {

	keyHex := os.Getenv("LIQUIDATOR_BASE_PRIVATE_KEY")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__BASE_PRIVATE_KEY not set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Signer{
		key:    key,
		signer: types.NewLondonSigner(big.NewInt(8453)),
	}, nil
}

func (s *Signer) Sign(tx *types.Transaction) (*types.Transaction, error) {
	return types.SignTx(tx, s.signer, s.key)
}

func NewMainnetSigner() (*Signer, error) {
	return nil, nil
}

func NewArbitrumSigner() (*Signer, error) {
	return nil, nil
}

func NewOptimismSigner() (*Signer, error) {
	return nil, nil
}
