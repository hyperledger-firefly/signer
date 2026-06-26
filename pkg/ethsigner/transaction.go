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
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/rlp"
	"github.com/hyperledger-firefly/signer/pkg/secp256k1"
	"golang.org/x/crypto/sha3"
)

type TransactionSignaturePayload struct {
	rlpList rlp.List
	data    []byte
}

func (sp *TransactionSignaturePayload) Bytes() []byte {
	return sp.data
}

func (sp *TransactionSignaturePayload) Hash() ethtypes.HexBytes0xPrefix {
	msgHash := sha3.NewLegacyKeccak256()
	msgHash.Write(sp.data)
	return msgHash.Sum(nil)
}

const (
	TransactionTypeLegacy byte = 0x00
	TransactionType2930   byte = 0x01 // unused
	TransactionType1559   byte = 0x02
)

type Transaction struct {
	From                 json.RawMessage           `ffstruct:"EthTransaction" json:"from,omitempty"` // only here as a possible input to signing key selection (eth_sendTransaction)
	Nonce                *ethtypes.HexInteger      `ffstruct:"EthTransaction" json:"nonce,omitempty"`
	GasPrice             *ethtypes.HexInteger      `ffstruct:"EthTransaction" json:"gasPrice,omitempty"`
	MaxPriorityFeePerGas *ethtypes.HexInteger      `ffstruct:"EthTransaction" json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerGas         *ethtypes.HexInteger      `ffstruct:"EthTransaction" json:"maxFeePerGas,omitempty"`
	GasLimit             *ethtypes.HexInteger      `ffstruct:"EthTransaction" json:"gas,omitempty"` // note this is required for some methods (eth_estimateGas)
	To                   *ethtypes.Address0xHex    `ffstruct:"EthTransaction" json:"to,omitempty"`
	Value                *ethtypes.HexInteger      `ffstruct:"EthTransaction" json:"value,omitempty"`
	Data                 ethtypes.HexBytes0xPrefix `ffstruct:"EthTransaction" json:"data"`
}

type TransactionWithOriginalPayload struct {
	*Transaction
	Payload []byte `json:"-"`
}

func (t *Transaction) BuildLegacy() rlp.List {
	rlpList := make(rlp.List, 0, 6)
	rlpList = append(rlpList, rlp.WrapInt(t.Nonce.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.GasPrice.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.GasLimit.BigInt()))
	rlpList = append(rlpList, rlp.WrapAddress(t.To))
	rlpList = append(rlpList, rlp.WrapInt(t.Value.BigInt()))
	rlpList = append(rlpList, rlp.Data(t.Data))
	return rlpList
}

func AddEIP155HashValuesToRLPList(rlpList rlp.List, chainID int64) rlp.List {
	// These values go into the hash of the transaction
	rlpList = append(rlpList, rlp.WrapInt(big.NewInt(chainID)))
	rlpList = append(rlpList, rlp.WrapInt(big.NewInt(0)))
	rlpList = append(rlpList, rlp.WrapInt(big.NewInt(0)))
	return rlpList
}

func (t *Transaction) AddEIP155HashValues(rlpList rlp.List, chainID int64) rlp.List {
	return AddEIP155HashValuesToRLPList(rlpList, chainID)
}

func (t *Transaction) Build1559(chainID int64) rlp.List {
	rlpList := make(rlp.List, 0, 9)
	rlpList = append(rlpList, rlp.WrapInt(big.NewInt(chainID)))
	rlpList = append(rlpList, rlp.WrapInt(t.Nonce.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.MaxPriorityFeePerGas.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.MaxFeePerGas.BigInt()))
	rlpList = append(rlpList, rlp.WrapInt(t.GasLimit.BigInt()))
	rlpList = append(rlpList, rlp.WrapAddress(t.To))
	rlpList = append(rlpList, rlp.WrapInt(t.Value.BigInt()))
	rlpList = append(rlpList, rlp.Data(t.Data))
	rlpList = append(rlpList, rlp.List{}) // access list not currently supported
	return rlpList
}

// Automatically pick signer, based on input fields.
// - If either of the new EIP-1559 fields are set, use EIP-1559
// - By default use EIP-155 signing
// Never picks legacy-legacy (non EIP-155), or EIP-2930
func (t *Transaction) Sign(signer secp256k1.Signer, chainID int64) ([]byte, error) {
	if signer == nil {
		return nil, i18n.NewError(context.Background(), signermsgs.MsgInvalidSigner)
	}
	if t.MaxPriorityFeePerGas.BigInt().Sign() > 0 || t.MaxFeePerGas.BigInt().Sign() > 0 {
		return t.SignEIP1559(signer, chainID)
	}
	return t.SignLegacyEIP155(signer, chainID)
}

// Returns the bytes that would be used to sign the transaction, without actually
// perform the signing. Can be used with Recover to verify a signing result.
func (t *Transaction) SignaturePayload(chainID int64) (sp *TransactionSignaturePayload) {
	if t.MaxPriorityFeePerGas.BigInt().Sign() > 0 || t.MaxFeePerGas.BigInt().Sign() > 0 {
		return t.SignaturePayloadEIP1559(chainID)
	}
	return t.SignaturePayloadLegacyEIP155(chainID)
}

// SignaturePayloadLegacyOriginal returns the rlpList of fields that are signed, and the
// bytes. Note that for legacy and EIP-155 transactions (everything prior to EIP-2718),
// there is no transaction type byte added (so the bytes are exactly rlpList.Encode())
func (t *Transaction) SignaturePayloadLegacyOriginal() *TransactionSignaturePayload {
	rlpList := t.BuildLegacy()
	return &TransactionSignaturePayload{
		rlpList: rlpList,
		data:    rlpList.Encode(),
	}
}

// SignLegacyOriginal uses legacy transaction structure, with legacy V value (27/28)
func (t *Transaction) SignLegacyOriginal(signer secp256k1.Signer) ([]byte, error) {
	if signer == nil {
		return nil, i18n.NewError(context.Background(), signermsgs.MsgInvalidSigner)
	}
	signaturePayload := t.SignaturePayloadLegacyOriginal()
	sig, err := signer.Sign(signaturePayload.data)
	if err != nil {
		return nil, err
	}
	return t.FinalizeLegacyOriginalWithSignature(signaturePayload, sig)
}

func (t *Transaction) FinalizeLegacyOriginalWithSignature(signaturePayload *TransactionSignaturePayload, sig *secp256k1.SignatureData) ([]byte, error) {
	// Use the pre-EIP-155 V value (27/28)
	err := sig.UpdateOriginal()
	if err != nil {
		return nil, err
	}
	rlpList := t.addSignature(signaturePayload.rlpList, sig)
	return rlpList.Encode(), nil
}

// SignaturePayloadLegacyEIP155 returns the rlpList of fields that are signed, and the
// bytes. Note that for legacy and EIP-155 transactions (everything prior to EIP-2718),
// there is no transaction type byte added (so the bytes are exactly rlpList.Encode())
func (t *Transaction) SignaturePayloadLegacyEIP155(chainID int64) *TransactionSignaturePayload {
	rlpList := t.BuildLegacy()
	rlpList = t.AddEIP155HashValues(rlpList, chainID)
	return &TransactionSignaturePayload{
		rlpList: rlpList,
		data:    rlpList.Encode(),
	}
}

// SignLegacyEIP155 uses legacy transaction structure, with EIP-155 signing V value (2*ChainID + 35 + Y-parity)
func (t *Transaction) SignLegacyEIP155(signer secp256k1.Signer, chainID int64) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("invalid signer")
	}

	signaturePayload := t.SignaturePayloadLegacyEIP155(chainID)

	sig, err := signer.Sign(signaturePayload.data)
	if err != nil {
		return nil, err
	}
	return t.FinalizeLegacyEIP155WithSignature(signaturePayload, sig, chainID)
}

func (t *Transaction) FinalizeLegacyEIP155WithSignature(signaturePayload *TransactionSignaturePayload, sig *secp256k1.SignatureData, chainID int64) ([]byte, error) {
	// Use the EIP-155 V value, of (2*ChainID + 35 + Y-parity)
	err := sig.UpdateEIP155(chainID)
	if err != nil {
		return nil, err
	}
	rlpList := t.addSignature(signaturePayload.rlpList[0:6] /* we don't include the chainID+0+0 hash values in the payload */, sig)
	return rlpList.Encode(), nil
}

// SignaturePayloadEIP1559 returns the rlpList of fields that are signed, along with the full
// bytes for the signature / TX Hash - which have the transaction type prefixed
func (t *Transaction) SignaturePayloadEIP1559(chainID int64) *TransactionSignaturePayload {
	rlpList := t.Build1559(chainID)

	// The signature payload is the transaction type, concatenated with RLP list _excluding_ signature
	// keccak256(0x02 || rlp([chain_id, nonce, max_priority_fee_per_gas, max_fee_per_gas, gas_limit, destination, amount, data, access_list]))
	return &TransactionSignaturePayload{
		rlpList: rlpList,
		data:    append([]byte{TransactionType1559}, rlpList.Encode()...),
	}
}

// SignEIP1559 uses EIP-1559 transaction structure (with EIP-2718 transaction type byte), with EIP-2930 V value (0 / 1 - direct parity-Y)
func (t *Transaction) SignEIP1559(signer secp256k1.Signer, chainID int64) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("invalid signer")
	}

	signaturePayload := t.SignaturePayloadEIP1559(chainID)
	sig, err := signer.Sign(signaturePayload.data)
	if err != nil {
		return nil, err
	}
	return t.FinalizeEIP1559WithSignature(signaturePayload, sig)
}

func (t *Transaction) FinalizeEIP1559WithSignature(signaturePayload *TransactionSignaturePayload, sig *secp256k1.SignatureData) ([]byte, error) {
	// Use the direct 0/1 Y-parity value
	err := sig.UpdateEIP2930()
	if err != nil {
		return nil, err
	}

	// Now we need a new RLP array, _including_ signature
	// 0x02 || rlp([chain_id, nonce, max_priority_fee_per_gas, max_fee_per_gas, gas_limit, destination, amount, data, access_list, signature_y_parity, signature_r, signature_s])
	rlpList := t.addSignature(signaturePayload.rlpList, sig)
	return append([]byte{TransactionType1559}, rlpList.Encode()...), nil
}

func RecoverLegacyRawTransaction(ctx context.Context, rawTx ethtypes.HexBytes0xPrefix, chainID int64) (*ethtypes.Address0xHex, *TransactionWithOriginalPayload, error) {

	decoded, _, err := rlp.Decode(rawTx)
	if err != nil {
		log.L(ctx).Errorf("Invalid legacy transaction data '%s': %s", rawTx, err)
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgInvalidLegacyTransaction, err)
	}

	if decoded == nil || len(decoded.(rlp.List)) < 9 {
		log.L(ctx).Errorf("Invalid legacy transaction data '%s': EOF", rawTx)
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgInvalidLegacyTransaction, "EOF")
	}
	rlpList := decoded.(rlp.List)

	tx := &Transaction{
		Nonce:    (*ethtypes.HexInteger)(rlpList[0].ToData().Int()),
		GasPrice: (*ethtypes.HexInteger)(rlpList[1].ToData().Int()),
		GasLimit: (*ethtypes.HexInteger)(rlpList[2].ToData().Int()),
		To:       rlpList[3].ToData().Address(),
		Value:    (*ethtypes.HexInteger)(rlpList[4].ToData().Int()),
		Data:     ethtypes.HexBytes0xPrefix(rlpList[5].ToData()),
	}

	vValue := rlpList[6].ToData().Int().Int64()
	rValue := rlpList[7].ToData().BytesNotNil()
	sValue := rlpList[8].ToData().BytesNotNil()
	var message []byte
	if vValue != 27 && vValue != 28 {
		// Legacy with EIP155 extensions
		yParity := vValue - (chainID * 2) - 35
		if yParity != 0 && yParity != 1 {
			return nil, nil, i18n.NewError(ctx, signermsgs.MsgInvalidEIP155TransactionV, chainID, vValue)
		}

		signedRLPList := make(rlp.List, 6, 9)
		copy(signedRLPList, rlpList[0:6])
		signedRLPList = AddEIP155HashValuesToRLPList(signedRLPList, chainID)
		message = signedRLPList.Encode()
	} else {
		// Legacy original transaction
		message = (rlpList[0:6]).Encode()
	}

	return recoverCommon(tx, message, chainID, vValue, rValue, sValue)

}

func recoverCommon(tx *Transaction, message []byte, chainID int64, v int64, r, s []byte) (*ethtypes.Address0xHex, *TransactionWithOriginalPayload, error) {
	foundSig := &secp256k1.SignatureData{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int),
	}
	foundSig.V.SetInt64(v)
	foundSig.R.SetBytes(r)
	foundSig.S.SetBytes(s)

	signer, err := foundSig.Recover(message, chainID)
	if err != nil {
		return nil, nil, err
	}

	return signer, &TransactionWithOriginalPayload{
		Transaction: tx,
		Payload:     message,
	}, nil
}

func decodeEIP1559SignaturePayload(ctx context.Context, rawTx ethtypes.HexBytes0xPrefix, chainID int64, rlpMinLen int) (rlp.List, *Transaction, error) {
	if len(rawTx) == 0 || rawTx[0] != TransactionType1559 {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgInvalidEIP1559Transaction, "TransactionType")
	}

	rawTx = rawTx[1:]
	decoded, _, err := rlp.Decode(rawTx)
	if err != nil {
		log.L(ctx).Errorf("Invalid EIP-1559 transaction data '%s': %s", rawTx, err)
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgInvalidEIP1559Transaction, err)
	}
	rlpList := decoded.(rlp.List)

	if len(rlpList) < rlpMinLen {
		log.L(ctx).Errorf("Invalid EIP-1559 transaction data (%d RLP elements)", rlpList)
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgInvalidEIP1559Transaction, "EOF")
	}
	encodedChainID := rlpList[0].ToData().IntOrZero().Int64()
	if encodedChainID != chainID {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgInvalidChainID, chainID, encodedChainID)
	}
	return rlpList, &Transaction{
		Nonce:                (*ethtypes.HexInteger)(rlpList[1].ToData().Int()),
		MaxPriorityFeePerGas: (*ethtypes.HexInteger)(rlpList[2].ToData().Int()),
		MaxFeePerGas:         (*ethtypes.HexInteger)(rlpList[3].ToData().Int()),
		GasLimit:             (*ethtypes.HexInteger)(rlpList[4].ToData().Int()),
		To:                   rlpList[5].ToData().Address(),
		Value:                (*ethtypes.HexInteger)(rlpList[6].ToData().Int()),
		Data:                 ethtypes.HexBytes0xPrefix(rlpList[7].ToData()),
		// No access list support
	}, nil
}

func DecodeEIP1559SignaturePayload(ctx context.Context, rawTx ethtypes.HexBytes0xPrefix, chainID int64) (*Transaction, error) {
	_, tx, err := decodeEIP1559SignaturePayload(ctx, rawTx, chainID, 9 /* no signature data */)
	return tx, err
}

func RecoverEIP1559Transaction(ctx context.Context, rawTx ethtypes.HexBytes0xPrefix, chainID int64) (*ethtypes.Address0xHex, *TransactionWithOriginalPayload, error) {

	rlpList, tx, err := decodeEIP1559SignaturePayload(ctx, rawTx, chainID, 12 /* with signature data */)
	if err != nil {
		return nil, nil, err
	}

	return recoverCommon(tx,
		append([]byte{TransactionType1559}, (rlpList[0:9]).Encode()...),
		chainID,
		rlpList[9].ToData().Int().Int64(),
		rlpList[10].ToData().BytesNotNil(),
		rlpList[11].ToData().BytesNotNil(),
	)
}

func RecoverRawTransaction(ctx context.Context, rawTx ethtypes.HexBytes0xPrefix, chainID int64) (*ethtypes.Address0xHex, *TransactionWithOriginalPayload, error) {

	// The first byte of the payload (per EIP-2718) is either `>= 0xc0` for legacy transactions,
	// or a transaction type selector (up to `0x7f`).
	if len(rawTx) == 0 {
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgEmptyTransactionBytes)
	}
	txTypeByte := rawTx[0]
	switch {
	case txTypeByte >= 0xc7:
		return RecoverLegacyRawTransaction(ctx, rawTx, chainID)
	case txTypeByte == TransactionType1559:
		return RecoverEIP1559Transaction(ctx, rawTx, chainID)
	default:
		return nil, nil, i18n.NewError(ctx, signermsgs.MsgUnsupportedTransactionType, txTypeByte)
	}

}

func (t *Transaction) addSignature(rlpList rlp.List, sig *secp256k1.SignatureData) rlp.List {
	rlpList = append(rlpList, rlp.WrapInt(sig.V))
	rlpList = append(rlpList, rlp.WrapInt(sig.R))
	rlpList = append(rlpList, rlp.WrapInt(sig.S))
	return rlpList
}
