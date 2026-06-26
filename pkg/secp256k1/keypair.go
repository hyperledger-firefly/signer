// Copyright © 2024 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secp256k1

import (
	btcec "github.com/btcsuite/btcd/btcec/v2" // ISC licensed
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"golang.org/x/crypto/sha3"
)

type KeyPair struct {
	PrivateKey *btcec.PrivateKey
	PublicKey  *btcec.PublicKey
	Address    ethtypes.Address0xHex
}

func (k *KeyPair) PrivateKeyBytes() []byte {
	return k.PrivateKey.Serialize()
}

func (k *KeyPair) PublicKeyBytes() []byte {
	// Remove the "04" Prefix byte when computing the address. This byte indicates that it is an uncompressed public key.
	return k.PublicKey.SerializeUncompressed()[1:]
}

func GenerateSecp256k1KeyPair() (*KeyPair, error) {
	// Generates key of curve S256() by default
	key, _ := btcec.NewPrivateKey()
	return wrapSecp256k1Key(key, key.PubKey()), nil
}

// Deprecated: Note there is no error condition returned by this function (use KeyPairFromBytes)
func NewSecp256k1KeyPair(b []byte) (*KeyPair, error) {
	return KeyPairFromBytes(b), nil
}

func KeyPairFromBytes(b []byte) *KeyPair {
	key, pubKey := btcec.PrivKeyFromBytes(b)
	return wrapSecp256k1Key(key, pubKey)
}

func wrapSecp256k1Key(key *btcec.PrivateKey, pubKey *btcec.PublicKey) *KeyPair {
	return &KeyPair{
		PrivateKey: key,
		PublicKey:  pubKey,
		Address:    *PublicKeyToAddress(pubKey),
	}
}

func PublicKeyToAddress(pubKey *btcec.PublicKey) *ethtypes.Address0xHex {
	// Take the hash of the public key to generate the address
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubKey.SerializeUncompressed()[1:])
	// Ethereum addresses only use the lower 20 bytes, so toss the rest away
	a := new(ethtypes.Address0xHex)
	copy(a[:], hash.Sum(nil)[12:32])
	return a
}
