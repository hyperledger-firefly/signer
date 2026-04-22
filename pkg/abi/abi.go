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

/*
Tl;dr

	sampleABI, _ := ParseABI([]byte(`[
		{
			"name": "transfer",
			"inputs": [
				{"name": "recipient", "internalType": "address", "type": "address" },
				{"name": "amount", "internalType": "uint256", "type": "uint256"}
			],
			"outputs": [{"internalType": "bool", "type": "bool"}],
			"stateMutability": "nonpayable",
			"type": "function"
		}
	]`))
	transferABIFn := sampleABI.Functions()["transfer"]
	sampleABICallBytes, _ := transferABIFn.EncodeCallDataJSON([]byte(
		`{"recipient":"0x4a0d852ebb58fc88cb260bb270ae240f72edc45b","amount":"100000000000000000"}`,
	))
	fmt.Printf("ABI Call Bytes: %s\n", hex.EncodeToString(sampleABICallBytes))
	values, _ := transferABIFn.DecodeCallData(sampleABICallBytes)
	outputJSON, _ := NewSerializer().
		SetFormattingMode(FormatAsObjects).
		SetByteSerializer(HexByteSerializer0xPrefix).
		SerializeJSON(values)
	fmt.Printf("Back to JSON:  %s\n", outputJSON)

The abi package allows encoding and decoding of ABI encoded bytes, for the inputs/outputs
to EVM functions, and the parsing of EVM logs/events.

A high level summary of the API is as follows:

	                     [ ABI ]        - parse your ABI definition, using the Go model of the JSON format
	                        ↓
	                    (validate)      - all types in functions (methods), events and errors are validated
	                        ↓
	            [ ComponentType tree ]  - to build a "type tree" of all the arrays/tuples/elementary
	                        ↓
	[ JSON ] →  [ ComponentValue tree ] - which you combine with data (JSON or Go types) to get a "value tree"
	                        ↓
	                     (encode)       - the value tree can then be serialized into ABI encoded bytes
	                        ↓
	              [ ABI encoded bytes ] - so you can use these bytes to invoke EVM functions (signatures supported)
	                        ↓
	                     (decode)       - then you can decode ABI bytes from function outputs, or logs (event data)
	                        ↓
	[ JSON ] ← [ ComponentValue tree ]  - the value tree can be serialized back to JSON

Example:

	transferABI := `[
		{
			"inputs": [
				{
					"internalType": "address",
					"name": "recipient",
					"type": "address"
				},
				{
					"internalType": "uint256",
					"name": "amount",
					"type": "uint256"
				}
			],
			"name": "transfer",
			"outputs": [
				{
					"internalType": "bool",
					"name": "",
					"type": "bool"
				}
			],
			"stateMutability": "nonpayable",
			"type": "function"
		}
	]`

	// Parse the ABI definition
	var abi ABI
	_ = json.Unmarshal([]byte(transferABI), &abi)
	f := abi.Functions()["transfer"]

	// Parse some JSON input data conforming to the ABI
	encodedValueTree, _ := f.Inputs.ParseJSON([]byte(`{
		"recipient": "0x03706Ff580119B130E7D26C5e816913123C24d89",
		"amount": "1000000000000000000"
	}`))

	// We can serialize this directly to abi bytes
	abiData, _ := encodedValueTree.EncodeABIData()
	fmt.Println(hex.EncodeToString(abiData))
	// 00000000000000000000000003706ff580119b130e7d26c5e816913123c24d890000000000000000000000000000000000000000000000000de0b6b3a7640000

	// We can also serialize that to function call data, with the function selector prefix
	abiCallData, _ := f.EncodeCallData(encodedValueTree)

	// Decode those ABI bytes back again, verifying the function selector
	decodedValueTree, _ := f.DecodeCallData(abiCallData)

	// Serialize back to JSON with default formatting - note the keys are alphabetically ordered
	jsonData, _ := decodedValueTree.JSON()
	fmt.Println(string(jsonData))
	// {"amount":"1000000000000000000","recipient":"03706ff580119b130e7d26c5e816913123c24d89"}

	// Use a custom serializer to get ordered array output, hex integers, and 0x prefixes
	// - Check out FormatAsSelfDescribingArrays for a format with embedded type information
	jsonData2, _ := NewSerializer().
		SetFormattingMode(FormatAsFlatArrays).
		SetIntSerializer(HexIntSerializer0xPrefix).
		SetByteSerializer(HexByteSerializer0xPrefix).
		SerializeJSON(decodedValueTree)
	fmt.Println(string(jsonData2))
	// ["0x03706ff580119b130e7d26c5e816913123c24d89","0xde0b6b3a7640000"]

The package deliberately gives you access to perform all of the transitions individually.

For example, if you want to traverse the type tree itself to generate metadata for the ABI,
you can do that by calling TypeComponentTree() on the inputs/outputs of a function/event.

External data parsing tries to be flexible when coercing JSON data into a value tree:

- Bytes and Addresses can be any of:
  - Hex string without any prefix
  - Hex string with an "0x" prefix
  - A byte array

- Numbers can be any of:
  - A base10 formatted string without any prefix
  - A hex formatted string with an "0x" prefix
  - A number
  - Negative numbers are supported
  - Floating point numbers are supported (for ABI fixed/ufixed types)

- Boolean values can be any of:
  - A boolean
  - A string "true"/"false"

- Strings must be a string

When passing in an interface{} (instead of JSON directly) efforts are made to follow pointers,
and resolve types down to the basic types. For example detecting whether a struct conforms to
the fmt.Stringer interface.

For serialization back out from the value tree, to JSON, there is a pluggable formatting interface
with a number of built-in options as follows:

- Parameter serialization for function outputs / event log data (and nested tuples) can be:
  - Object based {"key1":"val1"}
  - Flat ordered array based ["val1"]
  - Self describing array based [{"name":"key1","type":"string","value":"val1"}]

- Number serialization can be:
  - Base 10 formatted string
  - Hex with "0x" prefix
  - Numeric up to the maximum safe Javascript values, then automatically switching to string

- Byte serialization can be:
  - Hex with "0x" prefix
  - Hex without any prefix
*/
package abi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"golang.org/x/crypto/sha3"
)

// ABI "Application Binary Interface" is a list of the methods and events
// on the external interface of an EVM based smart contract - written in
// Solidity / Vyper.
//
// It is structured as a JSON array of ABI entries, each of which can be
// a function, event or error definition.
type ABI []*Entry

// EntryType is an enum of the possible ABI entry types
type EntryType string

const (
	Function    EntryType = "function"    // A function/method of the smart contract
	Constructor EntryType = "constructor" // The constructor
	Receive     EntryType = "receive"     // The "receive ether" function
	Fallback    EntryType = "fallback"    // The default function to invoke
	Event       EntryType = "event"       // An event the smart contract can emit
	Error       EntryType = "error"       // An error definition
)

type StateMutability string

const (
	Pure       StateMutability = "pure"       // Specified not to read blockchain state
	View       StateMutability = "view"       // Specified not to modify the blockchain state (read-only)
	Payable    StateMutability = "payable"    // The function accepts ether
	NonPayable StateMutability = "nonpayable" // The function does not accept ether
)

type ParameterArray []*Parameter

// Entry is an individual entry in an ABI - a function, event or error.
//
// Defines the name / inputs / outputs which can be used to generate the signature
// of the function/event, and used to encode input data, or decode output data.
type Entry struct {
	Type            EntryType       `ffstruct:"EthABIEntry" json:"type,omitempty"`            // Type of the entry - there are multiple function sub-types, events and errors
	Name            string          `ffstruct:"EthABIEntry" json:"name,omitempty"`            // Name of the function/event/error
	Payable         bool            `ffstruct:"EthABIEntry" json:"payable,omitempty"`         // Functions only: Superseded by stateMutability payable/nonpayable
	Constant        bool            `ffstruct:"EthABIEntry" json:"constant,omitempty"`        // Functions only: Superseded by stateMutability pure/view
	Anonymous       bool            `ffstruct:"EthABIEntry" json:"anonymous,omitempty"`       // Events only: The event is emitted without a signature (topic[0] is not generated)
	StateMutability StateMutability `ffstruct:"EthABIEntry" json:"stateMutability,omitempty"` // How the function interacts with the blockchain state
	Inputs          ParameterArray  `ffstruct:"EthABIEntry" json:"inputs"`                    // The list of input parameters to a function, or fields of an event / error
	Outputs         ParameterArray  `ffstruct:"EthABIEntry" json:"outputs"`                   // Functions only: The list of return values from a function
}

// Parameter is an individual typed parameter input/output
type Parameter struct {
	Name         string         `ffstruct:"EthABIParameter" json:"name"`                   // The name of the argument - does not affect the signature
	Type         string         `ffstruct:"EthABIParameter" json:"type"`                   // The canonical type of the parameter
	InternalType string         `ffstruct:"EthABIParameter" json:"internalType,omitempty"` // Additional internal type information that might be generated by the compiler
	Components   ParameterArray `ffstruct:"EthABIParameter" json:"components,omitempty"`   // An ordered list (tuple) of nested elements for array/object types
	Indexed      bool           `ffstruct:"EthABIParameter" json:"indexed,omitempty"`      // Events only: Whether the parameter is indexed into one of the topics of the log, or in the log's data segment

	parsed *typeComponent // cached components
}

func (e *Entry) IsFunction() bool {
	switch e.Type {
	case Function, Constructor, Receive, Fallback:
		return true
	default:
		return false
	}
}

func ParseABI(data []byte) (ABI, error) {
	var abi ABI
	return abi, json.Unmarshal(data, &abi)
}

// Validate processes all the components of all the entries in this ABI, to build a parsing tree
func (a ABI) Validate() (err error) {
	return a.ValidateCtx(context.Background())
}

func (a ABI) ValidateCtx(ctx context.Context) (err error) {
	for _, e := range a {
		if err := e.ValidateCtx(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a ABI) Functions() map[string]*Entry {
	m := make(map[string]*Entry)
	for _, e := range a {
		if e.Name != "" && e.IsFunction() {
			m[e.Name] = e
		}
	}
	return m
}

func (a ABI) Constructor() *Entry {
	for _, e := range a {
		if e.Type == Constructor {
			return e
		}
	}
	return nil
}

func (a ABI) Events() map[string]*Entry {
	m := make(map[string]*Entry)
	for _, e := range a {
		if e.Name != "" && e.Type == Event {
			m[e.Name] = e
		}
	}
	return m
}

func (a ABI) Errors() map[string]*Entry {
	m := make(map[string]*Entry)
	for _, e := range a {
		if e.Name != "" && e.Type == Error {
			m[e.Name] = e
		}
	}
	return m
}

// FilterType returns a new ABI containing only the entries whose Type matches t,
// preserving their original order. Returns nil if no entries match.
// Use this to narrow an ABI before passing it to SelectorMap or other helpers,
// e.g. a.FilterType(Error) to obtain only error definitions.
func (a ABI) FilterType(t EntryType) ABI {
	var out ABI
	for _, e := range a {
		if e.Type == t {
			out = append(out, e)
		}
	}
	return out
}

// SelectorMap builds a map from 4-byte ABI selectors to their corresponding
// entries. When two entries in the ABI produce the same 4-byte selector, the
// first one wins. Entries for which selector generation fails are silently
// skipped.
//
// SelectorMap operates on all entry types — combine with FilterType to restrict
// to a specific type, e.g. a.FilterType(Error).SelectorMap().
func (a ABI) SelectorMap() map[[4]byte]*Entry {
	m := make(map[[4]byte]*Entry)
	for _, e := range a {
		sel := e.FunctionSelectorBytes()
		if len(sel) >= 4 {
			var key [4]byte
			copy(key[:], sel[:4])
			if _, exists := m[key]; !exists {
				m[key] = e
			}
		}
	}
	return m
}

// Returns the components value from the parsed error
func (a ABI) ParseError(revertData []byte) (*Entry, *ComponentValue, bool) {
	return a.ParseErrorCtx(context.Background(), revertData)
}

// ParseErrorCtx returns the matched Entry and decoded ComponentValue from the
// given revert data. The ABI's error entries are tried first, followed by the
// built-in Error(string) and Panic(uint256).
func (a ABI) ParseErrorCtx(ctx context.Context, revertData []byte) (*Entry, *ComponentValue, bool) {
	for _, source := range []ABI{a.FilterType(Error), defaultErrorEntries} {
		for _, e := range source {
			if cv, err := e.DecodeCallDataCtx(ctx, revertData); err == nil {
				return e, cv, true
			}
		}
	}
	return nil, nil, false
}

// ErrorFormatOption configures the behaviour of ErrorString and ErrorStringCtx.
type ErrorFormatOption struct {
	// There is a pattern used in some smart contracts, where a string error is used
	// to embed the raw bytes of an ABI encoded sub-error.
	// While uncommon, this pattern is supported by the library if you set this switch.
	// When set, every standard revert `Error(string)` error will be traversed to look for
	// eyecatcher (4 byte signatures) of other errors binary encoded within the string.
	SearchForWrappedBinaryErrors bool
}

// ErrorString formats raw EVM revert data as a human-readable string.
// Pass ErrorFormatOption{SearchForWrappedBinaryErrors: true} to decode inner errors
// that are binary-encoded within an Error(string) value.
func (a ABI) ErrorString(revertData []byte, options ...ErrorFormatOption) (string, bool) {
	return a.ErrorStringCtx(context.Background(), revertData, options...)
}

// ErrorStringCtx formats raw EVM revert data as a human-readable string.
// The ABI's own error entries are tried first, followed by the built-in
// Error(string) and Panic(uint256).
// Pass ErrorFormatOption{SearchForWrappedBinaryErrors: true} to decode inner errors
// that are binary-encoded within an Error(string) value.
func (a ABI) ErrorStringCtx(ctx context.Context, revertData []byte, options ...ErrorFormatOption) (string, bool) {
	searchBinary := false
	for _, o := range options {
		searchBinary = searchBinary || o.SearchForWrappedBinaryErrors
	}
	if searchBinary {
		r := a.DecodeRevertErrorCtx(ctx, revertData, ErrorFormatOption{SearchForWrappedBinaryErrors: true})
		if r == nil {
			return "", false
		}
		s := r.String()
		return s, s != ""
	}
	e, cv, ok := a.ParseErrorCtx(ctx, revertData)
	if !ok {
		return "", false
	}
	s := FormatErrorStringCtx(ctx, e, cv)
	return s, s != ""
}

// SanitizeBinaryString returns the input as a text string if it is entirely
// printable ASCII, or hex-encodes the entire input otherwise. This ensures the
// output is always safe for database TEXT columns and human-readable logging.
func SanitizeBinaryString(raw []byte) string {
	for _, b := range raw {
		if b < 32 || b >= 127 {
			return "0x" + hex.EncodeToString(raw)
		}
	}
	return string(raw)
}

func FormatErrorStringCtx(ctx context.Context, e *Entry, cv *ComponentValue) string {
	if e == nil || cv == nil {
		return ""
	}
	var ok bool
	var parsed []interface{}
	if res, err := NewSerializer().
		SetFormattingMode(FormatAsFlatArrays).
		SetIntSerializer(Base10StringIntSerializer).
		SetByteSerializer(HexByteSerializer0xPrefix).
		SetAddressSerializer(HexAddrSerializer0xPrefix).
		SerializeInterfaceCtx(ctx, cv); err == nil {
		parsed, ok = res.([]interface{})
	}
	buff := new(strings.Builder)
	if ok && parsed != nil {
		buff.WriteString(e.Name)
		buff.WriteRune('(')
		for i, c := range parsed {
			if i > 0 {
				buff.WriteRune(',')
			}
			b, _ := json.Marshal(c)
			buff.Write(b)
		}
		buff.WriteRune(')')
	}
	return buff.String()
}

// Validate processes all the components of all the parameters in this ABI entry
func (e *Entry) Validate() (err error) {
	return e.ValidateCtx(context.Background())
}

func (e *Entry) ValidateCtx(ctx context.Context) (err error) {
	for _, input := range e.Inputs {
		if err := input.ValidateCtx(ctx); err != nil {
			return err
		}
	}
	for _, output := range e.Outputs {
		if err := output.ValidateCtx(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ParseJSON takes external JSON data, and parses against the ABI to generate
// a component value tree.
//
// The component value tree can then be serialized to binary ABI data.
func (pa ParameterArray) ParseJSON(data []byte) (*ComponentValue, error) {
	return pa.ParseJSONCtx(context.Background(), data)
}

func (pa ParameterArray) ParseJSONCtx(ctx context.Context, data []byte) (*ComponentValue, error) {
	var jsonTree interface{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err := decoder.Decode(&jsonTree)
	if err != nil {
		return nil, err
	}
	return pa.ParseExternalDataCtx(ctx, jsonTree)
}

// ParseExternalData takes (non-ABI encoded) data input, such as an unmarshalled JSON structure,
// and traverses it against the ABI component type tree, to form a component value tree.
//
// The component value tree can then be serialized to binary ABI data.
func (pa ParameterArray) ParseExternalData(input interface{}) (cv *ComponentValue, err error) {
	return pa.ParseExternalDataCtx(context.Background(), input)
}

// TypeComponentTree returns the type component tree for the array (tuple) of individually typed parameters
func (pa ParameterArray) TypeComponentTree() (component TypeComponent, err error) {
	return pa.TypeComponentTreeCtx(context.Background())
}

func (pa ParameterArray) TypeComponentTreeCtx(ctx context.Context) (tc TypeComponent, err error) {
	component := &typeComponent{
		cType:         TupleComponent,
		tupleChildren: make([]*typeComponent, len(pa)),
	}
	for i, p := range pa {
		if component.tupleChildren[i], err = p.typeComponentTreeCtx(ctx); err != nil {
			return nil, err
		}
	}
	return component, nil
}

func (pa ParameterArray) ParseExternalDataCtx(ctx context.Context, input interface{}) (cv *ComponentValue, err error) {
	component, err := pa.TypeComponentTreeCtx(ctx)
	if err != nil {
		return nil, err
	}
	return walkInput(ctx, "", input, component.(*typeComponent))
}

// DecodeABIData takes ABI encoded bytes that conform to the parameter array, and decodes them
// into a value tree. We take the offset (rather than requiring you to generate a slice at the
// given offset) so that errors in parsing can be reported at an absolute offset.
func (pa ParameterArray) DecodeABIData(b []byte, offset int) (cv *ComponentValue, err error) {
	return pa.DecodeABIDataCtx(context.Background(), b, offset)
}

func (pa ParameterArray) DecodeABIDataCtx(ctx context.Context, b []byte, offset int) (cv *ComponentValue, err error) {
	component, err := pa.TypeComponentTreeCtx(ctx)
	if err != nil {
		return nil, err
	}
	_, cv, err = walkTupleABIBytes(ctx, b, offset, component.(*typeComponent))
	return cv, err
}

// EncodeABIData is a helper to go all the way from JSON data to encoded ABI bytes
func (pa ParameterArray) EncodeABIDataJSON(jsonData []byte) ([]byte, error) {
	return pa.EncodeABIDataJSONCtx(context.Background(), jsonData)
}

// EncodeABIData is a helper to go all the way from JSON data to encoded ABI bytes
func (pa ParameterArray) EncodeABIDataJSONCtx(ctx context.Context, jsonData []byte) ([]byte, error) {
	cv, err := pa.ParseJSONCtx(ctx, jsonData)
	if err != nil {
		return nil, err
	}
	return cv.EncodeABIDataCtx(ctx)
}

// EncodeABIData goes all the way from interface inputs, to encoded ABI bytes
// The TypeComponentTree is created and discarded within the function
func (pa ParameterArray) EncodeABIDataValues(v interface{}) ([]byte, error) {
	return pa.EncodeABIDataValuesCtx(context.Background(), v)
}

func (pa ParameterArray) EncodeABIDataValuesCtx(ctx context.Context, v interface{}) ([]byte, error) {
	cv, err := pa.ParseExternalDataCtx(ctx, v)
	if err != nil {
		return nil, err
	}
	return cv.EncodeABIDataCtx(ctx)
}

// String returns the signature string. If a Validate needs to be initiated, and that
// parse fails, then the error is logged, but is not returned
func (e *Entry) String() string {
	s, err := e.Signature()
	if err != nil {
		log.L(context.Background()).Warnf("ABI parsing failed: %s", err)
	}
	return s
}

// SolString returns a Solidity - like string, with an empty function definition,
// and any struct definitions separated afterwards with a ; (semicolon)
func (e *Entry) SolString() string {
	solStr, err := e.SolidityStringCtx(context.Background())
	if err != nil {
		log.L(context.Background()).Warnf("ABI parsing failed: %s", err)
	}
	return solStr
}

func (e *Entry) SolidityStringCtx(ctx context.Context) (string, error) {
	solDef, childStructs, err := e.SolidityDefCtx(ctx)
	if err != nil {
		return "", err
	}
	buff := new(strings.Builder)
	buff.WriteString(solDef)
	for _, e := range childStructs {
		buff.WriteString("; ")
		buff.WriteString(e)
	}
	return buff.String(), nil
}

func (e *Entry) Signature() (string, error) {
	return e.SignatureCtx(context.Background())
}

func (e *Entry) GenerateFunctionSelector() ([]byte, error) {
	return e.GenerateFunctionSelectorCtx(context.Background())
}

func (e *Entry) GenerateFunctionSelectorCtx(ctx context.Context) ([]byte, error) {
	hash := sha3.NewLegacyKeccak256()
	sig, err := e.SignatureCtx(ctx)
	if err != nil {
		return nil, err
	}
	hash.Write([]byte(sig))
	k := hash.Sum(nil)
	return k[0:4], nil
}

// FunctionSelectorBytes is a convenience function to get the ID as bytes.
// Will return all zeros on error (ensures non-nil)
func (e *Entry) FunctionSelectorBytes() ethtypes.HexBytes0xPrefix {
	id, err := e.GenerateFunctionSelector()
	if err != nil {
		log.L(context.Background()).Warnf("ABI parsing failed: %s", err)
		return []byte{0, 0, 0, 0}
	}
	return id
}

// EncodeCallDataValues is a helper to go straight from an JSON input, to binary encoded call data
func (e *Entry) EncodeCallDataJSON(jsonData []byte) ([]byte, error) {
	return e.EncodeCallDataJSONCtx(context.Background(), jsonData)
}

func (e *Entry) EncodeCallDataJSONCtx(ctx context.Context, jsonData []byte) ([]byte, error) {
	cv, err := e.Inputs.ParseJSONCtx(ctx, jsonData)
	if err != nil {
		return nil, err
	}
	return e.EncodeCallDataCtx(ctx, cv)
}

// EncodeCallDataValues is a helper to go straight from an interface input, such as a map or array, to call data
func (e *Entry) EncodeCallDataValues(values interface{}) ([]byte, error) {
	return e.EncodeCallDataValuesCtx(context.Background(), values)
}

func (e *Entry) EncodeCallDataValuesCtx(ctx context.Context, values interface{}) ([]byte, error) {
	cv, err := e.Inputs.ParseExternalDataCtx(ctx, values)
	if err != nil {
		return nil, err
	}
	return e.EncodeCallDataCtx(ctx, cv)
}

// EncodeCallData serializes the inputs of the entry, prefixed with the function selector
func (e *Entry) EncodeCallData(cv *ComponentValue) ([]byte, error) {
	return e.EncodeCallDataCtx(context.Background(), cv)
}

func (e *Entry) EncodeCallDataCtx(ctx context.Context, cv *ComponentValue) ([]byte, error) {

	id, err := e.GenerateFunctionSelectorCtx(ctx)
	if err != nil {
		return nil, err
	}

	cvData, err := cv.EncodeABIDataCtx(ctx)
	if err != nil {
		return nil, err
	}

	data := make([]byte, len(id)+len(cvData))
	copy(data, id)
	copy(data[len(id):], cvData)
	return data, nil

}

func (e *Entry) DecodeCallData(b []byte) (*ComponentValue, error) {
	return e.DecodeCallDataCtx(context.Background(), b)
}

func (e *Entry) DecodeCallDataCtx(ctx context.Context, b []byte) (*ComponentValue, error) {

	id, err := e.GenerateFunctionSelectorCtx(ctx)
	if err != nil {
		return nil, err
	}
	if len(b) < 4 {
		return nil, i18n.NewError(ctx, signermsgs.MsgNotEnoughBytesABISignature)
	}
	if !bytes.Equal(id, b[0:4]) {
		return nil, i18n.NewError(ctx, signermsgs.MsgIncorrectABISignatureID, e.String(), hex.EncodeToString(id), hex.EncodeToString(b[0:4]))
	}

	return e.Inputs.DecodeABIDataCtx(ctx, b, 4)

}

// SignatureHash returns the keccak hash of the signature as bytes
func (e *Entry) SignatureHash() (ethtypes.HexBytes0xPrefix, error) {
	return e.SignatureHashCtx(context.Background())
}

func (e *Entry) SignatureHashCtx(context.Context) (ethtypes.HexBytes0xPrefix, error) {
	hash := sha3.NewLegacyKeccak256()
	sig, err := e.SignatureCtx(context.Background())
	if err != nil {
		return nil, err
	}
	hash.Write([]byte(sig))
	return hash.Sum(nil), nil
}

// SignatureHashBytes is a convenience function to get the signature hash as bytes.
// Will return all zeros on error (ensures non-nil)
func (e *Entry) SignatureHashBytes() ethtypes.HexBytes0xPrefix {
	sh, err := e.SignatureHashCtx(context.Background())
	if err != nil {
		log.L(context.Background()).Errorf("Failed to generate signature: %s", err)
		sh = make(ethtypes.HexBytes0xPrefix, 32)
	}
	return sh
}

func (e *Entry) topicToValue(ctx context.Context, topicIdx int, topic ethtypes.HexBytes0xPrefix, input *typeComponent) (*ComponentValue, error) {
	et := input.ElementaryType().(*elementaryTypeInfo)
	if et != nil && et.fixed32 {
		// Directly encoded into topic
		return et.decodeABIData(ctx, fmt.Sprintf("topic[%d]", topicIdx), topic, 0, 0, input)
	}
	// For all other types it is just a hash of the output for indexing, so we can only
	// logically return it as a hex string. The Solidity developer has to include
	// the same data a second type non-indexed to get the real value.
	return &ComponentValue{
		Component: &typeComponent{
			cType:          ElementaryComponent,
			elementaryType: (ElementaryTypeBytes).(*elementaryTypeInfo),
			keyName:        input.keyName,
			parameter:      input.parameter, // the original parameter will obviously have a different type to Bytes
		},
		Value: []byte(topic),
	}, nil
}

// DecodeEventData takes the array of topics, and the event data, and builds a component value tree that parses
// the values against the ABI definition in the entry. Values are extracted from either the topic or data per
// the rules defined here: https://docs.soliditylang.org/en/v0.8.15/abi-spec.html
//
// If the event is non-anonymous, the signature hash of the event must match the first topic (or an error is thrown).
func (e *Entry) DecodeEventData(topics []ethtypes.HexBytes0xPrefix, data ethtypes.HexBytes0xPrefix) (*ComponentValue, error) {
	return e.DecodeEventDataCtx(context.Background(), topics, data)
}

func (e *Entry) DecodeEventDataCtx(ctx context.Context, topics []ethtypes.HexBytes0xPrefix, data ethtypes.HexBytes0xPrefix) (*ComponentValue, error) {
	typeTree, err := e.Inputs.TypeComponentTree()
	if err != nil {
		return nil, err
	}
	inputTypes := typeTree.TupleChildren()
	topicIdx := 0
	if !e.Anonymous && len(topics) >= 1 {
		sigHashBytes := e.SignatureHashBytes()
		if !bytes.Equal(topics[0], sigHashBytes) {
			return nil, i18n.NewError(ctx, signermsgs.MsgEventSignatureMismatch, e, topics[0], sigHashBytes)
		}
		topicIdx++
	}
	dataArgs := &typeComponent{
		cType:         TupleComponent,
		tupleChildren: make([]*typeComponent, 0, len(inputTypes)),
	}
	dataArgIndexMap := make(map[int]int)
	valueTree := &ComponentValue{
		Component: typeTree,
		Children:  make([]*ComponentValue, len(inputTypes)),
	}
	for idx, input := range inputTypes {
		if input.Parameter().Indexed {
			// Extract the value (or value hash) from the topic
			if topicIdx >= len(topics) {
				return nil, i18n.NewError(ctx, signermsgs.MsgEventsInsufficientTopics, idx, e)
			}
			topic := topics[topicIdx]
			topicIdx++
			valueTree.Children[idx], err = e.topicToValue(ctx, topicIdx, topic, input.(*typeComponent))
			if err != nil {
				return nil, err
			}
		} else {
			// Add this parameter to the list we expect to be encoded in the data, with a map
			// back to where we store the output in the tree.
			dataArgIndexMap[len(dataArgs.tupleChildren)] = idx
			dataArgs.tupleChildren = append(dataArgs.tupleChildren, input.(*typeComponent))
		}
	}
	// If we have data args, decode them
	if len(dataArgs.tupleChildren) > 0 {
		dataValueTree, err := dataArgs.DecodeABIDataCtx(ctx, data, 0)
		if err != nil {
			return nil, err
		}
		// Map back to their original positions
		for i, v := range dataValueTree.Children {
			targetIdx := dataArgIndexMap[i]
			valueTree.Children[targetIdx] = v
		}
	}
	return valueTree, nil
}

func (e *Entry) SignatureCtx(ctx context.Context) (string, error) {
	buff := new(strings.Builder)
	buff.WriteString(e.Name)
	buff.WriteRune('(')
	for i, p := range e.Inputs {
		if i > 0 {
			buff.WriteRune(',')
		}
		s, err := p.SignatureStringCtx(ctx)
		if err != nil {
			return "", err
		}
		buff.WriteString(s)
	}
	buff.WriteRune(')')
	return buff.String(), nil
}

func (e *Entry) SolidityDef() (string, []string, error) {
	return e.SolidityDefCtx(context.Background())
}

// SolidityDefCtx returns a Solidity-like descriptor of the entry, including its type
func (e *Entry) SolidityDefCtx(ctx context.Context) (string, []string, error) {
	// Everything apart from event and error is a type of function
	var fieldType SolFieldType
	switch e.Type {
	case Error, Event:
		fieldType = EventOrErrorField
	default:
		fieldType = FunctionInput
	}

	allChildStructs := []string{}
	buff := new(strings.Builder)
	buff.WriteString(string(e.Type))
	buff.WriteRune(' ')
	buff.WriteString(e.Name)
	buff.WriteRune('(')
	for i, p := range e.Inputs {
		if i > 0 {
			buff.WriteString(", ")
		}
		s, childStructs, err := p.SolidityDefCtx(ctx, fieldType)
		if err != nil {
			return "", nil, err
		}
		allChildStructs = append(allChildStructs, childStructs...)
		buff.WriteString(s)
	}
	buff.WriteRune(')')

	if fieldType == FunctionInput {
		buff.WriteString(" external")
		if e.StateMutability != "" &&
			// The state mutability nonpayable is reflected in Solidity by not specifying a state mutability modifier at all.
			e.StateMutability != NonPayable {
			buff.WriteRune(' ')
			buff.WriteString(string(e.StateMutability))
		}
		if len(e.Outputs) > 0 {
			buff.WriteString(" returns (")
			for i, p := range e.Outputs {
				if i > 0 {
					buff.WriteString(", ")
				}
				s, childStructs, err := p.SolidityDefCtx(ctx, fieldType)
				if err != nil {
					return "", nil, err
				}
				allChildStructs = append(allChildStructs, childStructs...)
				buff.WriteString(s)
			}
			buff.WriteRune(')')
		}
		buff.WriteString(" { }")
	}

	return buff.String(), allChildStructs, nil
}

// Validate processes all the components of the type of this ABI parameter.
// - The elementary type
// - The fixed/variable length array dimensions
// - The tuple component types (recursively)
func (p *Parameter) Validate() (err error) {
	return p.ValidateCtx(context.Background())
}

func (p *Parameter) ValidateCtx(ctx context.Context) (err error) {
	p.parsed, err = p.parseABIParameterComponents(ctx)
	return err
}

// SignatureString generates and returns the signature string of the ABI
// parameter. If Validate has not yet been called, it will be called on your behalf.
//
// Note if you have modified the structure since Validate was last called, you should
// call Validate again.
func (p *Parameter) SignatureString() (s string, err error) {
	return p.SignatureStringCtx(context.Background())
}

func (p *Parameter) SignatureStringCtx(ctx context.Context) (string, error) {
	// Ensure the type component tree has been parsed
	tc, err := p.TypeComponentTreeCtx(ctx)
	if err != nil {
		return "", err
	}
	return tc.String(), nil
}

func (p *Parameter) SolidityDefCtx(ctx context.Context, fieldType SolFieldType) (string, []string, error) {
	// Ensure the type component tree has been parsed
	tc, err := p.TypeComponentTreeCtx(ctx)
	if err != nil {
		return "", nil, err
	}
	solDef, childStructs := tc.SolidityParamDef(fieldType)
	return solDef, childStructs, nil
}

// String returns the signature string. If a Validate needs to be initiated, and that
// parse fails, then the error is logged, but is not returned
func (p *Parameter) String() string {
	s, err := p.SignatureString()
	if err != nil {
		log.L(context.Background()).Warnf("ABI parsing failed: %s", err)
	}
	return s
}

// ComponentTypeTree returns the root of the component tree for the parameter.
// If Validate has not yet been called, it will be called on your behalf.
//
// Note if you have modified the structure since Validate was last called, you should
// call Validate again.
func (p *Parameter) TypeComponentTree() (TypeComponent, error) {
	return p.TypeComponentTreeCtx(context.Background())
}

func (p *Parameter) TypeComponentTreeCtx(ctx context.Context) (TypeComponent, error) {
	tc, err := p.typeComponentTreeCtx(ctx)
	return TypeComponent(tc), err
}

func (p *Parameter) typeComponentTreeCtx(ctx context.Context) (*typeComponent, error) {
	if p.parsed == nil {
		if err := p.ValidateCtx(ctx); err != nil {
			return nil, err
		}
	}
	return p.parsed, nil
}
