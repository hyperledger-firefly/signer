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

package signermsgs

var (
	ABIEntryAnonymous       = ffm("EthABIEntry.anonymous", "If the event is anonymous then the signature of the event does not take up a topic slot")
	ABIEntryType            = ffm("EthABIEntry.type", "The type of the ABI entry: 'event', 'error', 'function', 'constructor', 'receive', or 'fallback'")
	ABIEntryName            = ffm("EthABIEntry.name", "The name of the ABI entry")
	ABIEntryInputs          = ffm("EthABIEntry.inputs", "Array of ABI parameter definitions")
	ABIEntryOutputs         = ffm("EthABIEntry.outputs", "Array of ABI parameter definitions")
	ABIEntryStateMutability = ffm("EthABIEntry.stateMutability", "The state mutability of the function: 'pure', 'view', 'nonpayable' (the default) and 'payable'")
	ABIEntryConstant        = ffm("EthABIEntry.constant", "Functions only: Superseded by stateMutability payable/nonpayable")
	ABIEntryPayable         = ffm("EthABIEntry.payable", "Functions only: Superseded by stateMutability pure/view")

	ABIParameterName         = ffm("EthABIParameter.name", "The name of the parameter")
	ABIParameterType         = ffm("EthABIParameter.type", "The type of the parameter per the ABI specification")
	ABIParameterComponents   = ffm("EthABIParameter.components", "An array of components, if the parameter is a tuple")
	ABIParameterIndexed      = ffm("EthABIParameter.indexed", "Whether this parameter uses one of the topics, or is in the data area")
	ABIParameterInternalType = ffm("EthABIParameter.internalType", "Used by the solc compiler to include additional details - importantly the struct name for tuples")

	EthTransactionFrom                 = ffm("EthTransaction.from", "The from address (not encoded into the transaction directly, but used on this structure on input)")
	EthTransactionNonce                = ffm("EthTransaction.nonce", "Number used once (nonce) that specifies the sequence of this transaction in all transactions sent to the chain from this signing address")
	EthTransactionGasPrice             = ffm("EthTransaction.gasPrice", "The price per unit offered for the gas used when executing this transaction, if submitting to a chain that requires gas fees (in wei of the native chain token)")
	EthTransactionMaxPriorityFeePerGas = ffm("EthTransaction.maxPriorityFeePerGas", "Part of the EIP-1559 extension to transaction pricing. The amount provided to the miner of the block per unit of gas, in addition to the base fee (which is burned when the block is mined)")
	EthTransactionMaxFeePerGas         = ffm("EthTransaction.maxFeePerGas", "Part of the EIP-1559 extension to transaction pricing. The total amount you are willing to pay per unit of gas used by your contract, which is the total of the baseFeePerGas (determined by the chain at execution time) and the maxPriorityFeePerGas")
	EthTransactionGas                  = ffm("EthTransaction.gas", "The gas limit for execution of your transaction. Must be provided regardless of whether you paying a fee for the gas")
	EthTransactionTo                   = ffm("EthTransaction.to", "The target address of the transaction. Omitted for contract deployments")
	EthTransactionValue                = ffm("EthTransaction.value", "An optional amount of native token to transfer along with the transaction (in wei)")
	EthTransactionData                 = ffm("EthTransaction.data", "The encoded and signed transaction payload")

	EIP712ResultHash         = ffm("EIP712Result.hash", "The EIP-712 hash generated according to the Typed Data V4 algorithm")
	EIP712ResultSignatureRSV = ffm("EIP712Result.signatureRSV", "Hex encoded array of 65 bytes containing the R, S & V of the ECDSA signature. This is the standard signature encoding used in Ethereum recover utilities (note that some other utilities might expect a different encoding/packing of the data)")
	EIP712ResultV            = ffm("EIP712Result.v", "The V value of the ECDSA signature as a hex encoded integer")
	EIP712ResultR            = ffm("EIP712Result.r", "The R value of the ECDSA signature as a 32byte hex encoded array")
	EIP712ResultS            = ffm("EIP712Result.s", "The S value of the ECDSA signature as a 32byte hex encoded array")

	RevertErrorErrorEntry = ffm("RevertError.errorEntry", "The matched ABI error entry at this level of the error chain")
	RevertErrorPrefix     = ffm("RevertError.prefix", "The readable text prefix before the nested inner error, extracted from the outer Error(string) value")
	RevertErrorInnerError = ffm("RevertError.innerError", "The recursively decoded inner error, forming the next level of the error chain, or nil if this is the leaf")

	TypedDataDomain      = ffm("TypedData.domain", "The data to encode into the EIP712Domain as part fo signing the transaction")
	TypedDataMessage     = ffm("TypedData.message", "The data to encode into primaryType structure, with nested values for any sub-structures")
	TypedDataTypes       = ffm("TypedData.types", "Array of types to use when encoding, which must include the primaryType and the EIP712Domain (noting the primary type can be EIP712Domain if the message is empty)")
	TypedDataPrimaryType = ffm("TypedData.primaryType", "The primary type to begin encoding the EIP-712 hash from in the list of types, using the input message (unless set directly to EIP712Domain, in which case the message can be omitted)")
)
