// Copyright © 2026 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kmswallet

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	_ "crypto/sha256"
	"encoding/asn1"
	"fmt"
	"math/big"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/hyperledger-firefly/common/pkg/config"
	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/hyperledger-firefly/signer/pkg/eip712"
	"github.com/hyperledger-firefly/signer/pkg/ethsigner"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
	"golang.org/x/crypto/sha3"

	btcec "github.com/btcsuite/btcd/btcec/v2"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

const (
	ConfigEnabled   = "enabled"
	ConfigKeyID     = "keyId"
	ConfigRegion    = "region"
	ConfigEndpoint  = "endpoint"
)

func InitConfig(c config.Section) {
	c.AddKnownKey(ConfigEnabled, false)
	c.AddKnownKey(ConfigKeyID, "")
	c.AddKnownKey(ConfigRegion, "")
	c.AddKnownKey(ConfigEndpoint, "")
}

func ReadConfig(c config.Section) *Config {
	return &Config{
		Enabled:  c.GetBool(ConfigEnabled),
		KeyID:    c.GetString(ConfigKeyID),
		Region:   c.GetString(ConfigRegion),
		Endpoint: c.GetString(ConfigEndpoint),
	}
}

type Config struct {
	Enabled  bool
	KeyID    string
	Region   string
	Endpoint string
}

type ConfigGeneric = Config

func NewKMSWallet(ctx context.Context, conf *Config) (*KMSWallet, error) {
	if conf.KeyID == "" {
		return nil, i18n.NewError(ctx, signermsgs.MsgNoWalletEnabled)
	}

	opts := []func(*awsconfig.LoadOptions) error{}
	if conf.Region != "" {
		opts = append(opts, awsconfig.WithRegion(conf.Region))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	kmsOpts := []func(*kms.Options){}
	if conf.Endpoint != "" {
		kmsOpts = append(kmsOpts, func(o *kms.Options) {
			o.BaseEndpoint = aws.String(conf.Endpoint)
		})
	}
	client := kms.NewFromConfig(cfg, kmsOpts...)

	return &KMSWallet{
		conf:   *conf,
		client: client,
	}, nil
}

type KMSWallet struct {
	conf   Config
	client *kms.Client

	mu          sync.RWMutex
	address     ethtypes.Address0xHex
	pubKeyBytes []byte
	initialized bool
}

func (w *KMSWallet) Initialize(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.initialized {
		return nil
	}

	out, err := w.client.GetPublicKey(ctx, &kms.GetPublicKeyInput{
		KeyId: aws.String(w.conf.KeyID),
	})
	if err != nil {
		return fmt.Errorf("kms GetPublicKey: %w", err)
	}
	if len(out.PublicKey) == 0 {
		return fmt.Errorf("kms returned empty public key")
	}

	pubKeyBytes := make([]byte, len(out.PublicKey))
	copy(pubKeyBytes, out.PublicKey)

	pubKey, err := unmarshalSecp256k1PublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("parsing KMS public key: %w", err)
	}

	addr := publicKeyToAddress(pubKey)

	w.pubKeyBytes = pubKeyBytes
	w.address = *addr
	w.initialized = true

	log.L(ctx).Infof("KMS wallet initialized: keyId=%s address=%s", w.conf.KeyID, addr)
	return nil
}

func (w *KMSWallet) Refresh(ctx context.Context) error {
	w.mu.Lock()
	w.initialized = false
	w.mu.Unlock()
	return w.Initialize(ctx)
}

func (w *KMSWallet) Close() error { return nil }

func (w *KMSWallet) GetAccounts(_ context.Context) ([]*ethtypes.Address0xHex, error) {
	if !w.initialized {
		return nil, fmt.Errorf("wallet not initialized")
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	return []*ethtypes.Address0xHex{&w.address}, nil
}

func (w *KMSWallet) Sign(ctx context.Context, txn *ethsigner.Transaction, chainID int64) ([]byte, error) {
	signer := &kmsSigner{wallet: w}
	return txn.Sign(signer, chainID)
}

func (w *KMSWallet) SignTypedDataV4(ctx context.Context, _ ethtypes.Address0xHex, payload *eip712.TypedData) (*ethsigner.EIP712Result, error) {
	signer := &kmsSigner{wallet: w}
	return ethsigner.SignTypedDataV4(ctx, signer, payload)
}

func (w *KMSWallet) signDirect(ctx context.Context, message []byte) (*secp256k1.SignatureData, error) {
	if len(message) != 32 {
		return nil, fmt.Errorf("KMS signDirect expects a 32-byte pre-hashed message, got %d bytes", len(message))
	}

	out, err := w.client.Sign(ctx, &kms.SignInput{
		KeyId:            aws.String(w.conf.KeyID),
		Message:          message,
		MessageType:      types.MessageTypeDigest,
		SigningAlgorithm: types.SigningAlgorithmSpecEcdsaSha256,
	})
	if err != nil {
		return nil, fmt.Errorf("kms Sign: %w", err)
	}

	r, s, err := parseDERSignature(out.Signature)
	if err != nil {
		return nil, fmt.Errorf("parsing DER signature from KMS: %w", err)
	}

	v, err := w.recoverYParity(message, r, s)
	if err != nil {
		return nil, fmt.Errorf("recovering y-parity: %w", err)
	}

	return &secp256k1.SignatureData{
		V: v,
		R: r,
		S: s,
	}, nil
}

func (w *KMSWallet) recoverYParity(hashedMessage []byte, r, s *big.Int) (*big.Int, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, parity := range []int64{27, 28} {
		sig := &secp256k1.SignatureData{
			V: big.NewInt(parity),
			R: r,
			S: s,
		}
		recovered, err := sig.RecoverDirect(hashedMessage, 0)
		if err != nil {
			continue
		}
		if recovered.String() == w.address.String() {
			return big.NewInt(parity), nil
		}
	}
	return nil, fmt.Errorf("could not recover y-parity — signature does not match the KMS public key")
}

func parseDERSignature(der []byte) (r, s *big.Int, err error) {
	var sig struct {
		R *asn1.RawValue
		S *asn1.RawValue
	}
	if _, err := asn1.Unmarshal(der, &sig); err != nil {
		return nil, nil, fmt.Errorf("asn1 unmarshal: %w", err)
	}
	r = new(big.Int).SetBytes(sig.R.Bytes)
	s = new(big.Int).SetBytes(sig.S.Bytes)
	return r, s, nil
}

func unmarshalSecp256k1PublicKey(raw []byte) (*ecdsa.PublicKey, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty public key")
	}
	curve := btcec.S256()
	x, y := elliptic.Unmarshal(curve, raw)
	if x == nil {
		return nil, fmt.Errorf("failed to unmarshal public key (len=%d, prefix=0x%02x)", len(raw), raw[0])
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func publicKeyToAddress(pubKey *ecdsa.PublicKey) *ethtypes.Address0xHex {
	pubBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubBytes[1:])
	a := new(ethtypes.Address0xHex)
	copy(a[:], hash.Sum(nil)[12:32])
	return a
}
