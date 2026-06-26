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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"testing/iotest"

	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
	"github.com/stretchr/testify/assert"
)

const samplePrivateKey = `f6d5b8eb66ac39a39004209b7da586e3f95ecd1265172850b15e305c5d1fe424`

const sampleWallet = `{
	"address": "5d093e9b41911be5f5c4cf91b108bac5d130fa83",
	"crypto": {
	  "cipher": "aes-128-ctr",
	  "ciphertext": "a28e5f6fd3189ef220f658392af0e967f17931530ac5b79376ed5be7d8adfb5a",
	  "cipherparams": {
		"iv": "7babf856e25f812d9dbc133e3122a1fc"
	  },
	  "kdf": "scrypt",
	  "kdfparams": {
		"dklen": 32,
		"n": 262144,
		"p": 1,
		"r": 8,
		"salt": "2844947e39e03785cad3ccda776279dbf5a86a5df9cb6d0ab5773bfcb7cbe3b7"
	  },
	  "mac": "69ed15cbb03a29ec194bdbd2c2d8084c62be620d5b3b0f668ed9aa1f45dbaf99"
	},
	"id": "307cc063-2344-426a-b992-3b72d5d5be0b",
	"version": 3
  }`

const sampleWalletPbkdf2 = `{
    "address": "08327c2085530f3a90db40174beff14f1fc96b22",
    "id": "174d997a-d737-4cf4-b8ff-d26eaf1b9201",
    "version": 3,
    "crypto": {
        "cipher": "es-128-ctr",
        "ciphertext": "ff36c3ad1dfda68ef4f65f62b6101638b6ed8fcb61954ae058a690d4ed8c4563",
        "cipherparams": {
            "iv": "169c176944db19d27b2e297c4e3f0f1c"
        },
        "kdf": "pbkdf2",
        "mac": "5b403923bc4945264dad3043da1a90adef979f97c2c353f1ba8cdb0123831fd0",
        "kdfparams": {
            "dklen": 32,
            "c": 4096,
            "prf": "hmac-sha256",
            "salt": "3f395aa93f6dc374081d19931dc3d98b61f935d2e8dd54df60f27685716dd1f9"
        }
    }
}`

func TestLoadSampleWallet(t *testing.T) {
	w, err := ReadWalletFile([]byte(sampleWallet), []byte("correcthorsebatterystaple"))
	assert.NoError(t, err)

	keypair := w.KeyPair()
	assert.Equal(t, samplePrivateKey, hex.EncodeToString(keypair.PrivateKeyBytes()))
}

func TestMustReadBytesPanic(t *testing.T) {
	assert.Panics(t, func() {
		mustReadBytes(100, iotest.ErrReader(fmt.Errorf("pop")))
	})
}

func TestReadWalletFileBadJSON(t *testing.T) {
	_, err := ReadWalletFile([]byte(`!!not json`), []byte(""))
	assert.Regexp(t, "invalid wallet file", err)
}

func TestReadWalletFileMissingID(t *testing.T) {
	_, err := ReadWalletFile([]byte(`{}`), []byte(""))
	assert.Regexp(t, "missing keyfile id", err)
}

func TestReadWalletFileBadVersion(t *testing.T) {
	_, err := ReadWalletFile([]byte(`{"id":"6A2175E5-E553-4E25-AD1B-569A3BB0C3FD", "version": 1}`), []byte(""))
	assert.Regexp(t, "incorrect keyfile version", err)
}

func TestReadWalletFileBadKDF(t *testing.T) {
	_, err := ReadWalletFile([]byte(`{"id":"6A2175E5-E553-4E25-AD1B-569A3BB0C3FD", "version": 3, "crypto": {
		"kdf": "unknown"
	}}`), []byte(""))
	assert.Regexp(t, "unsupported kdf", err)
}

func TestWalletFileScryptJSON(t *testing.T) {
	w, err := ReadWalletFile([]byte(sampleWallet), []byte("correcthorsebatterystaple"))
	assert.NoError(t, err)
	j := w.JSON()
	w2, err := ReadWalletFile(j, []byte("correcthorsebatterystaple"))
	assert.NoError(t, err)
	assert.Equal(t, w, w2)
}

func TestWalletFilePbkdf2JSON(t *testing.T) {
	w, err := ReadWalletFile([]byte(sampleWalletPbkdf2), []byte("myPrecious"))
	assert.NoError(t, err)
	j := w.JSON()
	w2, err := ReadWalletFile(j, []byte("myPrecious"))
	assert.NoError(t, err)
	assert.Equal(t, w, w2)
}

func TestWalletFileCustomBytes(t *testing.T) {
	customBytes := ([]byte)("planet refuse wheel robot position venue predict bring solid paper salmon bind")

	w := NewWalletFileCustomBytesStandard("correcthorsebatterystaple", customBytes)

	w, err := ReadWalletFile(w.JSON(), []byte("correcthorsebatterystaple"))
	assert.NoError(t, err)
	j := w.JSON()
	w2, err := ReadWalletFile(j, []byte("correcthorsebatterystaple"))
	assert.NoError(t, err)
	assert.Equal(t, w, w2)

	assert.Equal(t, customBytes, w.PrivateKey())

	first32 := ([]byte)("planet refuse wheel robot positi")
	kp, _ := secp256k1.NewSecp256k1KeyPair(first32)
	assert.NoError(t, err)
	assert.Equal(t, kp.Address, w2.KeyPair().Address)
}

func TestWalletFileCustomBytesLight(t *testing.T) {
	customBytes := ([]byte)("less than 32 bytes")

	w := NewWalletFileCustomBytesLight("correcthorsebatterystaple", customBytes)

	w, err := ReadWalletFile(w.JSON(), []byte("correcthorsebatterystaple"))
	assert.NoError(t, err)
	j := w.JSON()
	w2, err := ReadWalletFile(j, []byte("correcthorsebatterystaple"))
	assert.NoError(t, err)
	assert.Equal(t, w, w2)

	assert.Equal(t, customBytes, w.PrivateKey())

	zeroToTheRight := ([]byte)("less than 32 bytes")
	kp, _ := secp256k1.NewSecp256k1KeyPair(zeroToTheRight)
	assert.NoError(t, err)
	assert.Equal(t, kp.Address, w2.KeyPair().Address)
}

func TestMarshalWalletJSONFail(t *testing.T) {
	_, err := marshalWalletJSON(&walletFileBase{}, map[bool]bool{false: true})
	assert.Error(t, err)
}

func TestWalletFileCustomBytesUnsetAddress(t *testing.T) {
	customBytes := ([]byte)("something deterministic for testing")

	w := NewWalletFileCustomBytesLight("correcthorsebatterystaple", customBytes)

	w.Metadata()["address"] = nil
	w.Metadata()["myKeyIdentifier"] = "something I know works for me"
	w.Metadata()["id"] = "attempting to set this does not work"
	w.Metadata()["version"] = 42

	jsonBytes, err := json.Marshal(w)
	assert.NoError(t, err)

	var roundTripBackFromJSON map[string]interface{}
	err = json.Unmarshal(jsonBytes, &roundTripBackFromJSON)
	assert.NoError(t, err)

	_, hasAddress := roundTripBackFromJSON["address"]
	assert.False(t, hasAddress)
	assert.Equal(t, "something I know works for me", roundTripBackFromJSON["myKeyIdentifier"])
	assert.Equal(t, float64(w.GetVersion()), roundTripBackFromJSON["version"])
	assert.Equal(t, w.GetID().String(), roundTripBackFromJSON["id"])

}
