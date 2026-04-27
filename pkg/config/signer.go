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

func NewBaseSigner(chainId int64) (*Signer, error) {

	keyHex := os.Getenv("BASE_PK")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__BASE_PRIVATE_KEY not set")
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

func NewMainnetSigner() (*Signer, error) {
	return nil, nil
}

func NewArbitrumSigner(chainid int64) (*Signer, error) {
	keyHex := os.Getenv("ARB_PK")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__ARBITRUM_PRIVATE_KEY not set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Signer{
		key:    key,
		signer: types.NewLondonSigner(big.NewInt(chainid)),
	}, nil
}

func NewOptimismSigner(chainid int64) (*Signer, error) {
	keyHex := os.Getenv("OPT_PK")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__OPT_PRIVATE_KEY not set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Signer{
		key:    key,
		signer: types.NewLondonSigner(big.NewInt(chainid)),
	}, nil
}

func NewUniChainSigner(chainid int64) (*Signer, error) {
	keyHex := os.Getenv("OPT_PK")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__UNI_PRIVATE_KEY not set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Signer{
		key:    key,
		signer: types.NewLondonSigner(big.NewInt(chainid)),
	}, nil
}

func NewWorldChainSigner(chainid int64) (*Signer, error) {
	keyHex := os.Getenv("WORLD_PK")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__WORLD_PRIVATE_KEY not set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Signer{
		key:    key,
		signer: types.NewLondonSigner(big.NewInt(chainid)),
	}, nil
}

func NewKatanaSigner(chainid int64) (*Signer, error) {
	keyHex := os.Getenv("KATA_PK")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__KATANA_PRIVATE_KEY not set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Signer{
		key:    key,
		signer: types.NewLondonSigner(big.NewInt(chainid)),
	}, nil
}
