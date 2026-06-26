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

package ethsigner

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/hyperledger-firefly/common/pkg/ffapi"
	"github.com/hyperledger-firefly/signer/mocks/secp256k1mocks"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/rlp"
	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEncodeExistingLegacyEIP155(t *testing.T) {

	inputData, err := hex.DecodeString(
		"3674e15c00000000000000000000000000000000000000000000000000000000000000a03f04a4e93ded4d2aaa1a41d617e55c59ac5f1b28a47047e2a526e76d45eb9681d19642e9120d63a9b7f5f537565a430d8ad321ef1bc76689a4b3edc861c640fc00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000966665f73797374656d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d58747653456758626265506855684165364167426f3465796a7053434b437834515a4c50793548646a6177730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001a1f7502c8f8797999c0c6b9c2da653ea736598ed0daa856c47ae71411aa8fea2")
	assert.NoError(t, err)

	// The expected raw legacy transaction as queried from a node
	expectedRaw, err := hex.DecodeString(
		"f901e70380829e7e94497eedc4299dea2f2a364be10025d0ad0f702de380b901843674e15c00000000000000000000000000000000000000000000000000000000000000a03f04a4e93ded4d2aaa1a41d617e55c59ac5f1b28a47047e2a526e76d45eb9681d19642e9120d63a9b7f5f537565a430d8ad321ef1bc76689a4b3edc861c640fc00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000966665f73797374656d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d58747653456758626265506855684165364167426f3465796a7053434b437834515a4c50793548646a6177730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001a1f7502c8f8797999c0c6b9c2da653ea736598ed0daa856c47ae71411aa8fea2820feea002e6e9728373680d0a7d75f99697d3887069dd5db4b9581c42bfb5749fb5fc80a0032e8717112b372f41c4a2a46ad0ea807f56645990130cbbc60614f2240a3a1a")
	assert.NoError(t, err)

	txn := Transaction{
		Nonce:    ethtypes.NewHexInteger64(3),
		GasLimit: ethtypes.NewHexInteger64(40574),
		To:       ethtypes.MustNewAddress("0x497eedc4299dea2f2a364be10025d0ad0f702de3"),
		Data:     inputData,
	}
	sig := &secp256k1.SignatureData{
		V: big.NewInt(0xfee),
		R: new(big.Int),
		S: new(big.Int),
	}
	sig.R.SetString("0x2e6e9728373680d0a7d75f99697d3887069dd5db4b9581c42bfb5749fb5fc80", 0)
	sig.S.SetString("0x32e8717112b372f41c4a2a46ad0ea807f56645990130cbbc60614f2240a3a1a", 0)
	rlpList := txn.BuildLegacy()
	rlpList = txn.addSignature(rlpList, sig)
	raw := rlpList.Encode()

	assert.Equal(t, expectedRaw, raw)

}

func TestEncodeExistingEIP1559(t *testing.T) {

	// Sample from TX 0x61ca9c99c1d752fb3bda568b8566edf33ba93585c64a970566e6dfb540a5cbc1

	inputData, err := hex.DecodeString(
		"a0712d680000000000000000000000000000000000000000000000000000000000000001")
	assert.NoError(t, err)

	// The expected raw EIP1559 transaction as queried from a node
	expectedRaw, err := hex.DecodeString(
		"02f89701248459682f00854e58be5c3c8302b13d943c99f2a4b366d46bcf2277639a135a6d1288eceb878e1bc9bf040000a4a0712d680000000000000000000000000000000000000000000000000000000000000001c001a0ea6e1513d716146af3a02e1497fbe7fc3b2ffb08ccb4a1bfef4eaa2a122f62dfa00ddc23aec20948a55d3e1f8afd29b5570d8d279450a472b55561ef6afe4a07ff")
	assert.NoError(t, err)

	txn := Transaction{
		Nonce:                ethtypes.NewHexInteger64(0x24),
		MaxFeePerGas:         ethtypes.NewHexInteger64(0x4e58be5c3c),
		MaxPriorityFeePerGas: ethtypes.NewHexInteger64(0x59682f00),
		GasLimit:             ethtypes.NewHexInteger64(0x2b13d),
		Value:                ethtypes.NewHexInteger64(0x8e1bc9bf040000),
		To:                   ethtypes.MustNewAddress("0x3c99f2a4b366d46bcf2277639a135a6d1288eceb"),
		Data:                 inputData,
	}
	sig := &secp256k1.SignatureData{
		V: big.NewInt(1),
		R: new(big.Int),
		S: new(big.Int),
	}
	sig.R.SetString("0xea6e1513d716146af3a02e1497fbe7fc3b2ffb08ccb4a1bfef4eaa2a122f62df", 0)
	sig.S.SetString("0xddc23aec20948a55d3e1f8afd29b5570d8d279450a472b55561ef6afe4a07ff", 0)
	rlpList := txn.Build1559(1)
	rlpList = txn.addSignature(rlpList, sig)
	raw := append([]byte{TransactionType1559}, rlpList.Encode()...)

	assert.Equal(t, ethtypes.HexBytesPlain(expectedRaw).String(), ethtypes.HexBytesPlain(raw).String())

}

func TestSignLegacyEIP155(t *testing.T) {

	inputData, err := hex.DecodeString(
		"3674e15c00000000000000000000000000000000000000000000000000000000000000a03f04a4e93ded4d2aaa1a41d617e55c59ac5f1b28a47047e2a526e76d45eb9681d19642e9120d63a9b7f5f537565a430d8ad321ef1bc76689a4b3edc861c640fc00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000966665f73797374656d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d58747653456758626265506855684165364167426f3465796a7053434b437834515a4c50793548646a6177730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001a1f7502c8f8797999c0c6b9c2da653ea736598ed0daa856c47ae71411aa8fea2")
	assert.NoError(t, err)

	txn := Transaction{
		Nonce:    ethtypes.NewHexInteger64(3),
		GasPrice: ethtypes.NewHexInteger64(100000000),
		GasLimit: ethtypes.NewHexInteger64(40574),
		To:       ethtypes.MustNewAddress("0x497eedc4299dea2f2a364be10025d0ad0f702de3"),
		Data:     inputData,
		Value:    ethtypes.NewHexInteger64(100000000),
	}
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	raw, err := txn.SignLegacyEIP155(keypair, 1001)
	assert.NoError(t, err)

	signer, txr, err := RecoverRawTransaction(context.Background(), raw, 1001)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address.String(), signer.String())
	jsonCompare(t, txn, *txr)

	_, _, err = RecoverRawTransaction(context.Background(), raw, 1002)
	assert.Regexp(t, "FF22085", err)

}

func TestSignAutoEIP1559(t *testing.T) {

	inputData, err := hex.DecodeString(
		"3674e15c00000000000000000000000000000000000000000000000000000000000000a03f04a4e93ded4d2aaa1a41d617e55c59ac5f1b28a47047e2a526e76d45eb9681d19642e9120d63a9b7f5f537565a430d8ad321ef1bc76689a4b3edc861c640fc00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000966665f73797374656d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d58747653456758626265506855684165364167426f3465796a7053434b437834515a4c50793548646a6177730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001a1f7502c8f8797999c0c6b9c2da653ea736598ed0daa856c47ae71411aa8fea2")
	assert.NoError(t, err)

	txn := Transaction{
		Nonce:                ethtypes.NewHexInteger64(3),
		MaxPriorityFeePerGas: ethtypes.NewHexInteger64(123456780),
		MaxFeePerGas:         ethtypes.NewHexInteger64(150000000),
		GasLimit:             ethtypes.NewHexInteger64(40574),
		To:                   ethtypes.MustNewAddress("0x497eedc4299dea2f2a364be10025d0ad0f702de3"),
		Data:                 inputData,
		Value:                ethtypes.NewHexInteger64(100000000),
	}
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	raw, err := txn.Sign(keypair, 1001)
	assert.NoError(t, err)

	signer, txr, err := RecoverRawTransaction(context.Background(), raw, 1001)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address.String(), signer.String())
	jsonCompare(t, txn, *txr)

}

func TestSignLegacyOriginal(t *testing.T) {

	inputData, err := hex.DecodeString(
		"3674e15c00000000000000000000000000000000000000000000000000000000000000a03f04a4e93ded4d2aaa1a41d617e55c59ac5f1b28a47047e2a526e76d45eb9681d19642e9120d63a9b7f5f537565a430d8ad321ef1bc76689a4b3edc861c640fc00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000966665f73797374656d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d58747653456758626265506855684165364167426f3465796a7053434b437834515a4c50793548646a6177730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001a1f7502c8f8797999c0c6b9c2da653ea736598ed0daa856c47ae71411aa8fea2")
	assert.NoError(t, err)

	txn := Transaction{
		Nonce:    ethtypes.NewHexInteger64(3),
		GasPrice: ethtypes.NewHexInteger64(100000000),
		GasLimit: ethtypes.NewHexInteger64(40574),
		To:       ethtypes.MustNewAddress("0x497eedc4299dea2f2a364be10025d0ad0f702de3"),
		Data:     inputData,
		Value:    ethtypes.NewHexInteger64(100000000),
	}
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)

	raw, err := txn.SignLegacyOriginal(keypair)
	assert.NoError(t, err)

	signer, txr, err := RecoverRawTransaction(context.Background(), raw, 1001)
	assert.NoError(t, err)
	assert.Equal(t, keypair.Address.String(), signer.String())
	jsonCompare(t, txn, *txr)

}

func TestSignFail(t *testing.T) {

	txn := Transaction{}

	_, err := txn.SignLegacyOriginal(nil)
	assert.Error(t, err)

	_, err = txn.SignLegacyEIP155(nil, 0)
	assert.Error(t, err)

	_, err = txn.SignEIP1559(nil, 0)
	assert.Error(t, err)

}

func TestSignNilSigner(t *testing.T) {
	txn := Transaction{}
	_, err := txn.Sign(nil, 1001)
	assert.Regexp(t, "FF22064", err)
}

func TestSignLegacyOriginalNilSigner(t *testing.T) {
	txn := Transaction{}
	_, err := txn.SignLegacyOriginal(nil)
	assert.Regexp(t, "FF22064", err)
}

func TestSignLegacyOriginaSignerError(t *testing.T) {
	txn := Transaction{}
	msn := &secp256k1mocks.Signer{}
	msn.On("Sign", mock.Anything).Return(nil, fmt.Errorf("pop"))
	_, err := txn.SignLegacyOriginal(msn)
	assert.Regexp(t, "pop", err)
}

func TestSignLegacyEIP155Error(t *testing.T) {
	txn := Transaction{}
	msn := &secp256k1mocks.Signer{}
	msn.On("Sign", mock.Anything).Return(nil, fmt.Errorf("pop"))
	_, err := txn.SignLegacyEIP155(msn, 12345)
	assert.Regexp(t, "pop", err)
}
func TestSignEIP1559Error(t *testing.T) {
	txn := Transaction{}
	msn := &secp256k1mocks.Signer{}
	msn.On("Sign", mock.Anything).Return(nil, fmt.Errorf("pop"))
	_, err := txn.SignEIP1559(msn, 12345)
	assert.Regexp(t, "pop", err)
}

func TestEthTXDocumented(t *testing.T) {
	ffapi.CheckObjectDocumented(&Transaction{})
}

func jsonCompare(t *testing.T, expected, actual interface{}) {
	expectedJSON, err := json.Marshal(expected)
	assert.NoError(t, err)
	actualJSON, err := json.Marshal(actual)
	assert.NoError(t, err)
	assert.JSONEq(t, (string)(expectedJSON), (string)(actualJSON))

}

func TestRecoverRawTransactionEmpty(t *testing.T) {
	_, _, err := RecoverRawTransaction(context.Background(), []byte{}, 1001)
	assert.Regexp(t, "FF22081", err)
}

func TestRecoverRawTransactionInvalidType(t *testing.T) {
	_, _, err := RecoverRawTransaction(context.Background(), []byte{0x03}, 1001)
	assert.Regexp(t, "FF22082.*0x03", err)
}

func TestRecoverLegacyTransactionEmpty(t *testing.T) {
	_, _, err := RecoverLegacyRawTransaction(context.Background(), []byte{}, 1001)
	assert.Regexp(t, "FF22083", err)
}

func TestRecoverLegacyBadData(t *testing.T) {
	_, _, err := RecoverLegacyRawTransaction(context.Background(), []byte{0xff}, 1001)
	assert.Regexp(t, "FF22083", err)
}

func TestRecoverLegacyBadStructure(t *testing.T) {
	_, _, err := RecoverLegacyRawTransaction(context.Background(), (rlp.List{
		rlp.WrapInt(big.NewInt(12345)),
	}).Encode(), 1001)
	assert.Regexp(t, "FF22083.*EOF", err)
}

func TestRecoverEIP1559TransactionEmpty(t *testing.T) {
	_, _, err := RecoverEIP1559Transaction(context.Background(), []byte{}, 1001)
	assert.Regexp(t, "FF22084.*TransactionType", err)
}

func TestRecoverEIP1559BadData(t *testing.T) {
	_, _, err := RecoverEIP1559Transaction(context.Background(), []byte{TransactionType1559, 0xff}, 1001)
	assert.Regexp(t, "FF22084", err)
}

func TestRecoverEIP1559BadStructure(t *testing.T) {
	_, _, err := RecoverEIP1559Transaction(context.Background(), append([]byte{TransactionType1559}, (rlp.List{
		rlp.WrapInt(big.NewInt(12345)),
	}).Encode()...), 1001)
	assert.Regexp(t, "FF22084.*EOF", err)
}

func TestDecodeEIP1559SignaturePayloadEmpty(t *testing.T) {
	_, err := DecodeEIP1559SignaturePayload(context.Background(), []byte{}, 1001)
	assert.Regexp(t, "FF22084.*TransactionType", err)
}

func TestRoundTripEIP1559Payload(t *testing.T) {

	txIn := Transaction{
		Nonce:                ethtypes.NewHexInteger64(3),
		GasLimit:             ethtypes.NewHexInteger64(40574),
		MaxPriorityFeePerGas: ethtypes.NewHexInteger64(505050),
		MaxFeePerGas:         ethtypes.NewHexInteger64(606060),
		To:                   ethtypes.MustNewAddress("0x497eedc4299dea2f2a364be10025d0ad0f702de3"),
		Data:                 ethtypes.MustNewHexBytes0xPrefix("0xfeedbeef"),
		Value:                ethtypes.NewHexInteger64(100000000),
	}
	abiData := txIn.SignaturePayloadEIP1559(1001).Bytes()

	txOut, err := DecodeEIP1559SignaturePayload(context.Background(), abiData, 1001)
	require.NoError(t, err)

	jsonIn, err := json.Marshal(txIn)
	require.NoError(t, err)
	jsonOut, err := json.Marshal(txOut)
	require.NoError(t, err)

	require.JSONEq(t, string(jsonIn), string(jsonOut))

}

func TestRecoverEIP1559BadChainID(t *testing.T) {
	_, _, err := RecoverEIP1559Transaction(context.Background(), append([]byte{TransactionType1559}, (rlp.List{
		rlp.WrapInt(big.NewInt(111)),
		rlp.WrapInt(big.NewInt(222)),
		rlp.WrapInt(big.NewInt(333)),
		rlp.WrapInt(big.NewInt(444)),
		rlp.WrapInt(big.NewInt(555)),
		rlp.WrapInt(big.NewInt(666)),
		rlp.WrapInt(big.NewInt(777)),
		rlp.WrapInt(big.NewInt(888)),
		rlp.WrapInt(big.NewInt(999)),
		rlp.WrapInt(big.NewInt(111)),
		rlp.WrapInt(big.NewInt(223)),
		rlp.WrapInt(big.NewInt(333)),
	}).Encode()...), 1001)
	assert.Regexp(t, "FF22086.*1,001.*111", err)
}

func TestRecoverEIP1559Signature(t *testing.T) {
	_, _, err := RecoverEIP1559Transaction(context.Background(), append([]byte{TransactionType1559}, (rlp.List{
		rlp.WrapInt(big.NewInt(1001)),
		rlp.WrapInt(big.NewInt(222)),
		rlp.WrapInt(big.NewInt(333)),
		rlp.WrapInt(big.NewInt(444)),
		rlp.WrapInt(big.NewInt(555)),
		rlp.WrapInt(big.NewInt(666)),
		rlp.WrapInt(big.NewInt(777)),
		rlp.WrapInt(big.NewInt(888)),
		rlp.WrapInt(big.NewInt(999)),
		rlp.WrapInt(big.NewInt(111)),
		rlp.WrapInt(big.NewInt(223)),
		rlp.WrapInt(big.NewInt(333)),
	}).Encode()...), 1001)
	assert.Regexp(t, "invalid", err)
}
