// Copyright © 2022 Kaleido, Inc.
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
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/pbkdf2"
)

func TestPbkdf2Wallet(t *testing.T) {
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	salt := mustReadBytes(32, rand.Reader)
	derivedKey := pbkdf2.Key([]byte("myPrecious"), salt, 4096, 32, sha256.New)
	iv := mustReadBytes(16 /* 128bit */, rand.Reader)
	encryptKey := derivedKey[0:16]
	cipherText := mustAES128CtrEncrypt(encryptKey, iv, keypair.PrivateKeyBytes())
	mac := generateMac(derivedKey[16:32], cipherText)

	w1 := &walletFilePbkdf2{
		walletFileBase: walletFileBase{
			walletFileCoreFields: walletFileCoreFields{
				ID:      fftypes.NewUUID(),
				Version: version3,
			},
			walletFileMetadata: walletFileMetadata{
				metadata: map[string]interface{}{
					"address": ethtypes.AddressPlainHex(keypair.Address).String(),
				},
			},
		},
		Crypto: cryptoPbkdf2{
			cryptoCommon: cryptoCommon{
				Cipher:     cipherAES128ctr,
				CipherText: cipherText,
				CipherParams: cipherParams{
					IV: iv,
				},
				KDF: kdfTypePbkdf2,
				MAC: mac,
			},
			KDFParams: kdfParamsPbkdf2{
				PRF:   prfHmacSHA256,
				DKLen: 32,
				C:     4096,
				Salt:  salt,
			},
		},
	}

	wb1, err := json.Marshal(&w1)
	assert.NoError(t, err)

	w2, err := ReadWalletFile(wb1, []byte("myPrecious"))
	assert.NoError(t, err)

	assert.Equal(t, keypair.PrivateKeyBytes(), w2.KeyPair().PrivateKeyBytes())

}

func TestPbkdf2WalletFileDecryptInvalid(t *testing.T) {

	_, err := readPbkdf2WalletFile([]byte(`!! not json`), []byte(""), nil)
	assert.Regexp(t, "invalid pbkdf2 keystore", err)

}

func TestPbkdf2WalletFileUnsupportedPRF(t *testing.T) {

	_, err := readPbkdf2WalletFile([]byte(`{}`), []byte(""), nil)
	assert.Regexp(t, "invalid pbkdf2 wallet file: unsupported prf", err)

}
