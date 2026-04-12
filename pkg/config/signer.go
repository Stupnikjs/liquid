package config

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
)

type Signer struct {
	key    *ecdsa.PrivateKey
	signer types.Signer
}

func NewSigner() (*Signer, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	keyHex := os.Getenv("LIQUIDATOR_PRIVATE_KEY")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR_PRIVATE_KEY not set")
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
