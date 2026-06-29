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

package keystorev3

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
	"golang.org/x/crypto/scrypt"
)

const defaultR = 8

func readScryptWalletFile(jsonWallet []byte, password []byte, metadata map[string]interface{}) (WalletFile, error) {
	var w *walletFileScrypt
	if err := json.Unmarshal(jsonWallet, &w); err != nil {
		return nil, fmt.Errorf("invalid scrypt wallet file: %s", err)
	}
	w.metadata = metadata
	return w, w.decrypt(password)
}

func mustGenerateDerivedScryptKey(password string, salt []byte, n, p int) []byte {
	b, err := scrypt.Key([]byte(password), salt, n, defaultR, p, 16)
	if err != nil {
		panic(fmt.Sprintf("Scrypt failed: %s", err))
	}
	return b
}

// creates an ethereum address wallet file
func newScryptWalletFileSecp256k1(password string, keypair *secp256k1.KeyPair, n int, p int) WalletFile {
	wf := newScryptWalletFileBytes(password, keypair.PrivateKeyBytes(), n, p)
	wf.Metadata()["address"] = ethtypes.AddressPlainHex(keypair.Address).String()
	return wf
}

// this allows creation of any size/type of key in the store
func newScryptWalletFileBytes(password string, privateKey []byte, n int, p int) *walletFileScrypt {

	// Generate a sale for the scrypt
	salt := mustReadBytes(32, rand.Reader)

	// Do the scrypt derivation of the key with the salt from the password
	derivedKey := mustGenerateDerivedScryptKey(password, salt, n, p)

	// Generate a random Initialization Vector (IV) for the AES/CTR/128 key encryption
	iv := mustReadBytes(16 /* 128bit */, rand.Reader)

	// First 16 bytes of derived key are used as the encryption key
	encryptKey := derivedKey[0:16]

	// Encrypt the private key with the encryption key
	cipherText := mustAES128CtrEncrypt(encryptKey, iv, privateKey)

	// Last 16 bytes of derived key are used for the MAC
	mac := generateMac(derivedKey[16:32], cipherText)

	return &walletFileScrypt{
		walletFileBase: walletFileBase{
			walletFileCoreFields: walletFileCoreFields{
				ID:      fftypes.NewUUID(),
				Version: version3,
			},
			walletFileMetadata: walletFileMetadata{
				metadata: map[string]interface{}{},
			},
			privateKey: privateKey,
		},
		Crypto: cryptoScrypt{
			cryptoCommon: cryptoCommon{
				Cipher:     cipherAES128ctr,
				CipherText: cipherText,
				CipherParams: cipherParams{
					IV: iv,
				},
				KDF: kdfTypeScrypt,
				MAC: mac,
			},
			KDFParams: kdfParamsScrypt{
				DKLen: 32,
				N:     n,
				R:     defaultR,
				P:     p,
				Salt:  salt,
			},
		},
	}
}

func (w *walletFileScrypt) decrypt(password []byte) error {
	derivedKey, err := scrypt.Key(password, w.Crypto.KDFParams.Salt, w.Crypto.KDFParams.N, w.Crypto.KDFParams.R, w.Crypto.KDFParams.P, w.Crypto.KDFParams.DKLen)
	if err != nil {
		return fmt.Errorf("invalid scrypt keystore: %s", err)
	}
	w.privateKey, err = w.Crypto.decryptCommon(derivedKey)
	return err
}
