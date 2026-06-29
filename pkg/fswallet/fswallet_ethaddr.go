// Copyright © 2023 Kaleido, Inc.
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

package fswallet

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/hyperledger-firefly/signer/pkg/eip712"
	"github.com/hyperledger-firefly/signer/pkg/ethsigner"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/keystorev3"
	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
)

type SyncAddressCallback func(context.Context, ethtypes.Address0xHex) error

// This is a wrapper on the WalletStrIDs capability, that requires all lookups to keys
// to be ethereum addresses, and adds some other ethereum specific functionality.
//
// Originally this library only provided this utility, and later we made the concept
// of an address more generic.
type Wallet interface {
	ethsigner.WalletTypedData
	GetWalletFile(ctx context.Context, addr ethtypes.Address0xHex) (keystorev3.WalletFile, error)
	SetSyncAddressCallback(SyncAddressCallback)
	AddListener(listener chan<- ethtypes.Address0xHex)
}

type walletEthAddr struct {
	gw WalletGeneric
}

func NewFilesystemWallet(ctx context.Context, conf *Config, initialListeners ...chan<- ethtypes.Address0xHex) (ww Wallet, err error) {
	gw, err := NewFilesystemWalletGeneric(ctx, &ConfigGeneric{
		Config: *conf,
		WalletFileValidator: func(ctx context.Context, addrString string, kv3 keystorev3.WalletFile) error {
			addr, err := ethtypes.NewAddress(addrString)
			if err == nil {
				keypair := kv3.KeyPair()
				if keypair.Address != *addr {
					err = i18n.NewError(ctx, signermsgs.MsgAddressMismatch, keypair.Address, addr)
				}
			}
			return err
		},
		AddressValidator: func(ctx context.Context, addrString string) (standardized string, err error) {
			addr, err := ethtypes.NewAddress(addrString)
			if err == nil {
				standardized = addr.String() // ensures lookups match
			}
			return standardized, err
		},
	}, ethProxyListeners(initialListeners...)...)
	if err != nil {
		return nil, err
	}
	return &walletEthAddr{
		gw: gw,
	}, nil
}

func ethProxyListeners(listeners ...chan<- ethtypes.Address0xHex) []chan<- string {
	genericListeners := make([]chan<- string, len(listeners))
	for i, listener := range listeners {
		genericListener := make(chan string)
		go func() {
			for addrString := range genericListener {
				addr, _ := ethtypes.NewAddress(addrString)
				if addr != nil {
					listener <- *addr // note we have a validation function to ensure this
				}
			}
		}()
		genericListeners[i] = genericListener
	}
	return genericListeners
}

func (e *walletEthAddr) AddListener(listener chan<- ethtypes.Address0xHex) {
	e.gw.AddListener(ethProxyListeners(listener)[0])
}

func (e *walletEthAddr) Close() error {
	return e.gw.Close()
}

func (e *walletEthAddr) GetAccounts(ctx context.Context) (addrs []*ethtypes.Address0xHex, err error) {
	addrStrs, err := e.gw.GetAccounts(ctx)
	if err == nil {
		addrs = make([]*ethtypes.Address0xHex, len(addrStrs))
		for i, addrStr := range addrStrs {
			if err == nil {
				addrs[i], err = ethtypes.NewAddress(addrStr)
			}
		}
	}
	return addrs, err
}

func (e *walletEthAddr) GetWalletFile(ctx context.Context, addr ethtypes.Address0xHex) (keystorev3.WalletFile, error) {
	return e.gw.GetWalletFile(ctx, addr.String())
}

func (e *walletEthAddr) Initialize(ctx context.Context) error {
	return e.gw.Initialize(ctx)
}

func (e *walletEthAddr) Refresh(ctx context.Context) error {
	return e.gw.Refresh(ctx)
}

func (e *walletEthAddr) SetSyncAddressCallback(cb SyncAddressCallback) {
	e.gw.SetSyncCallback(func(ctx context.Context, s string) error {
		addr, err := ethtypes.NewAddress(s)
		if err == nil {
			err = cb(ctx, *addr)
		}
		return err
	})
}

func (e *walletEthAddr) getSignerForJSONAccount(ctx context.Context, rawAddrJSON json.RawMessage) (*secp256k1.KeyPair, error) {

	// We require an ethereum address in the "from" field
	var from ethtypes.Address0xHex
	err := json.Unmarshal(rawAddrJSON, &from)
	if err != nil {
		return nil, err
	}
	return e.getSignerForAddr(ctx, from)
}

func (e *walletEthAddr) getSignerForAddr(ctx context.Context, from ethtypes.Address0xHex) (*secp256k1.KeyPair, error) {

	wf, err := e.GetWalletFile(ctx, from)
	if err != nil {
		return nil, err
	}
	return wf.KeyPair(), nil

}

func (e *walletEthAddr) Sign(ctx context.Context, txn *ethsigner.Transaction, chainID int64) ([]byte, error) {
	keypair, err := e.getSignerForJSONAccount(ctx, txn.From)
	if err != nil {
		return nil, err
	}
	return txn.Sign(keypair, chainID)
}

func (e *walletEthAddr) SignTypedDataV4(ctx context.Context, from ethtypes.Address0xHex, payload *eip712.TypedData) (*ethsigner.EIP712Result, error) {
	keypair, err := e.getSignerForAddr(ctx, from)
	if err != nil {
		return nil, err
	}
	return ethsigner.SignTypedDataV4(ctx, keypair, payload)
}
