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

package rlp

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
)

var loremIpsumRLPBytes = []byte{
	0xb8,
	0x38,
	'L',
	'o',
	'r',
	'e',
	'm',
	' ',
	'i',
	'p',
	's',
	'u',
	'm',
	' ',
	'd',
	'o',
	'l',
	'o',
	'r',
	' ',
	's',
	'i',
	't',
	' ',
	'a',
	'm',
	'e',
	't',
	',',
	' ',
	'c',
	'o',
	'n',
	's',
	'e',
	'c',
	't',
	'e',
	't',
	'u',
	'r',
	' ',
	'a',
	'd',
	'i',
	'p',
	'i',
	's',
	'i',
	'c',
	'i',
	'n',
	'g',
	' ',
	'e',
	'l',
	'i',
	't',
}

const loremIpsumString = "Lorem ipsum dolor sit amet, consectetur adipisicing elit"

func TestDecodeBigIntRoundTrip(t *testing.T) {

	value := int64(3000000000)
	b := WrapInt(big.NewInt(value)).Encode()
	decoded, pos, err := Decode(b)
	assert.NoError(t, err)
	assert.False(t, decoded.IsList(), 1)
	assert.Equal(t, len(b), pos)
	assert.Equal(t, value, decoded.(Data).Int().Int64())

}

func TestDecodeZeroLength(t *testing.T) {

	decoded, pos, err := Decode([]byte{})
	assert.NoError(t, err)
	assert.Equal(t, 0, pos)
	assert.Nil(t, decoded)

}

func TestDecodeShortString(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{0x83, 'd', 'o', 'g'},
	)
	assert.NoError(t, err)
	assert.Equal(t, 4, pos)
	assert.False(t, decoded.IsList())
	assert.Equal(t, "dog", string(decoded.(Data)))

}

func TestDecodeShortList(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{0xc8, 0x83, 'c', 'a', 't', 0x83, 'd', 'o', 'g'},
	)
	assert.NoError(t, err)
	assert.Equal(t, 9, pos)
	assert.True(t, decoded.IsList())
	assert.Len(t, decoded.(List), 2)
	assert.Equal(t, "cat", string(decoded.(List)[0].(Data)))
	assert.Equal(t, "dog", string(decoded.(List)[1].(Data)))

}

func TestDecodeEmptyString(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{0x80},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, pos)
	assert.False(t, decoded.IsList())
	assert.Equal(t, "", string(decoded.(Data)))
	assert.Zero(t, decoded.(Data).Int().Int64())
}

func TestDecodeEmptyList(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{0xc0},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, pos)
	assert.True(t, decoded.IsList())
	assert.Len(t, decoded, 0)

}

func TestDecodeOneNilByte(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{0x00},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, pos)
	assert.False(t, decoded.IsList())
	assert.Equal(t, Data{0x00}, decoded.(Data))

}

func TestDecodeInteger15(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{0x0f},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, pos)
	assert.False(t, decoded.IsList())
	assert.Equal(t, Data{0x0f}, decoded.(Data))
	assert.Equal(t, int64(15), decoded.(Data).Int().Int64())

}

func TestDecodeInteger1024(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{0x82, 0x04, 0x00},
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, pos)
	assert.False(t, decoded.IsList())
	assert.Equal(t, int64(1024), decoded.(Data).Int().Int64())

}

func TestDecodeNestedEmptyLists(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{
			0xc7,
			0xc0,
			0xc1,
			0xc0,
			0xc3,
			0xc0,
			0xc1,
			0xc0,
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 8, pos)
	assert.True(t, decoded.IsList())
	assert.Equal(t, List{
		List{},
		List{
			List{},
		},
		List{
			List{},
			List{
				List{},
			},
		},
	}, decoded)

}

func TestLoremIpsumString(t *testing.T) {

	decoded, pos, err := Decode(loremIpsumRLPBytes)
	assert.NoError(t, err)
	assert.Equal(t, len(loremIpsumRLPBytes), pos)
	assert.False(t, decoded.IsList())
	assert.Equal(t, loremIpsumString, string(decoded.(Data)))

}

func TestDecodeNestedListsWithData(t *testing.T) {

	decoded, pos, err := Decode(
		[]byte{
			0xc6,
			0x82,
			0x7a,
			0x77,
			0xc1,
			0x04,
			0x01,
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 7, pos)
	assert.True(t, decoded.IsList())
	assert.Equal(t, List{
		WrapString("zw"),
		List{
			WrapInt(big.NewInt(4)),
		},
		WrapInt(big.NewInt(1)),
	}, decoded)

}

func TestDecodeLongerPayload(t *testing.T) {

	encoded, err := hex.DecodeString(
		"F86E12F86B80881BC16D674EC8000094CD2A3D9F938E13CD947EC05ABC7FE734D" +
			"F8DD8268609184E72A00064801BA0C52C114D4F5A3BA904A9B3036E5E118FE0DBB987" +
			"FE3955DA20F2CD8F6C21AB9CA06BA4C2874299A55AD947DBC98A25EE895AABF6B625C" +
			"26C435E84BFD70EDF2F69",
	)
	assert.NoError(t, err)

	decoded, pos, err := Decode(encoded)
	assert.NoError(t, err)
	assert.Equal(t, len(encoded), pos)
	assert.True(t, decoded.IsList())
	assert.Len(t, decoded.(List), 2)

	assert.Equal(t, List{
		MustWrapHex("0x12"),
		List{
			MustWrapHex("0x"),
			MustWrapHex("0x1bc16d674ec80000"),
			MustWrapHex("0xcd2a3d9f938e13cd947ec05abc7fe734df8dd826"),
			MustWrapHex("0x09184e72a000"),
			MustWrapHex("0x64"),
			MustWrapHex("0x"),
			MustWrapHex("0x1b"),
			MustWrapHex("0xc52c114d4f5a3ba904a9b3036e5e118fe0dbb987fe3955da20f2cd8f6c21ab9c"),
			MustWrapHex("0x6ba4c2874299a55ad947dbc98a25ee895aabf6b625c26c435e84bfd70edf2f69"),
		},
	}, decoded)

}

func TestDecodeBadShortDataSizeTooLarge(t *testing.T) {

	_, _, err := Decode([]byte{0xb7})
	assert.EqualError(t, err, "length mismatch in RLP for short data (pos=1 len=55)")

}

func TestDecodeBadLongDataSizeTooLarge(t *testing.T) {

	_, _, err := Decode([]byte{0xb8})
	assert.EqualError(t, err, "length mismatch in RLP for length bytes (list=false pos=1 len=1)")

	_, _, err = Decode([]byte{0xbb, 0x7f, 0xff, 0xff, 0xff})
	assert.EqualError(t, err, "length mismatch in RLP for data bytes (list=false pos=5 len=2147483647)")

}

func TestDecodeBadShortListSizeTooLarge(t *testing.T) {

	_, _, err := Decode([]byte{0xf6})
	assert.EqualError(t, err, "length mismatch in RLP for short list (pos=1 len=54)")

	_, _, err = Decode([]byte{0xf7})
	assert.EqualError(t, err, "length mismatch in RLP for short list (pos=1 len=55)")

}

func TestDecodeBadLongListSizeTooLarge(t *testing.T) {

	_, _, err := Decode([]byte{0xf8})
	assert.EqualError(t, err, "length mismatch in RLP for length bytes (list=true pos=1 len=1)")

	_, _, err = Decode([]byte{0xfb, 0x7f, 0xff, 0xff, 0xff})
	assert.EqualError(t, err, "length mismatch in RLP for data bytes (list=true pos=5 len=2147483647)")

}

func TestDecodeShortListBadChild(t *testing.T) {

	_, _, err := Decode([]byte{0xc1, 0xff})
	assert.EqualError(t, err, "length mismatch in RLP for length bytes (list=true pos=1 len=8)")

}

func TestDecodeLongListBadChild(t *testing.T) {

	_, _, err := Decode([]byte{0xf8, 0x01, 0xff})
	assert.EqualError(t, err, "length mismatch in RLP for length bytes (list=true pos=1 len=8)")

}

func TestExtractLongLenTooLong(t *testing.T) {
	rlpData := []byte{
		byte(0xc1),
		byte(0x09),
		byte(0xff),
		byte(0xff),
		byte(0xff),
		byte(0xff),
		byte(0xff),
		byte(0xff),
		byte(0xff),
		byte(0xff),
		byte(0xff),
	}
	_, _, err := extractLongLen(false, rlpData[0], 0, rlpData)
	assert.EqualError(t, err, "too many RLP bytes to decode")
}

func TestExtractLongZero(t *testing.T) {
	rlpData := []byte{
		byte(0xb7),
	}
	dataLen, newPos, err := extractLongLen(false, rlpData[0], 0, rlpData)
	assert.NoError(t, err)
	assert.Equal(t, 1, newPos)
	assert.Zero(t, dataLen)
}

func TestDecodeTX(t *testing.T) {

	// Sample (legacy) transaction info
	// {
	// 	blockHash: "0xf792398ef0d5fbd4cccff85778032ce17074123eb143e6c658e544bc1b76ff4f",
	// 	blockNumber: 4,
	// 	from: "0x5d093e9b41911be5f5c4cf91b108bac5d130fa83",
	// 	gas: 40574,
	// 	gasPrice: 0,
	// 	hash: "0xea4bf65ee1f2ae6df7259676f4dc30e28a879fa7e7519a86c2ed6b9c59a544d8",
	// 	input: "0x ... see below",
	// 	nonce: 3,
	// 	r: "0x2e6e9728373680d0a7d75f99697d3887069dd5db4b9581c42bfb5749fb5fc80",
	// 	s: "0x32e8717112b372f41c4a2a46ad0ea807f56645990130cbbc60614f2240a3a1a",
	// 	to: "0x497eedc4299dea2f2a364be10025d0ad0f702de3",
	// 	transactionIndex: 0,
	// 	type: "0x0",
	// 	v: "0xfee",
	// 	value: 0
	// }

	// The raw transaction
	encoded, err := hex.DecodeString(
		"f901e70380829e7e94497eedc4299dea2f2a364be10025d0ad0f702de380b901843674e15c00000000000000000000000000000000000000000000000000000000000000a03f04a4e93ded4d2aaa1a41d617e55c59ac5f1b28a47047e2a526e76d45eb9681d19642e9120d63a9b7f5f537565a430d8ad321ef1bc76689a4b3edc861c640fc00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000966665f73797374656d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d58747653456758626265506855684165364167426f3465796a7053434b437834515a4c50793548646a6177730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001a1f7502c8f8797999c0c6b9c2da653ea736598ed0daa856c47ae71411aa8fea2820feea002e6e9728373680d0a7d75f99697d3887069dd5db4b9581c42bfb5749fb5fc80a0032e8717112b372f41c4a2a46ad0ea807f56645990130cbbc60614f2240a3a1a")
	assert.NoError(t, err)

	// The input data
	inputData, err := hex.DecodeString(
		"3674e15c00000000000000000000000000000000000000000000000000000000000000a03f04a4e93ded4d2aaa1a41d617e55c59ac5f1b28a47047e2a526e76d45eb9681d19642e9120d63a9b7f5f537565a430d8ad321ef1bc76689a4b3edc861c640fc00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000966665f73797374656d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d58747653456758626265506855684165364167426f3465796a7053434b437834515a4c50793548646a6177730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001a1f7502c8f8797999c0c6b9c2da653ea736598ed0daa856c47ae71411aa8fea2")
	assert.NoError(t, err)

	elem, next, err := Decode(encoded)
	assert.NoError(t, err)
	assert.Equal(t, len(encoded), next)
	rlpList := elem.(List)
	assert.Len(t, rlpList, 9)
	// Nonce
	assert.Equal(t, int64(3), rlpList[0].(Data).Int().Int64())
	// Gas Price
	assert.Equal(t, int64(0), rlpList[1].(Data).Int().Int64())
	// Gas Limit
	assert.Equal(t, int64(40574), rlpList[2].(Data).Int().Int64())
	// To
	assert.Equal(t, "0x497eedc4299dea2f2a364be10025d0ad0f702de3", ethtypes.HexBytes0xPrefix(rlpList[3].(Data)).String())
	// Value
	assert.Equal(t, int64(0), rlpList[4].(Data).Int().Int64())
	// Data
	assert.Equal(t, inputData, []byte(rlpList[5].(Data)))
	// V
	assert.Equal(t, "0x0fee", ethtypes.HexBytes0xPrefix(rlpList[6].(Data)).String())
	// R
	assert.Equal(t, "0x02e6e9728373680d0a7d75f99697d3887069dd5db4b9581c42bfb5749fb5fc80", ethtypes.HexBytes0xPrefix(rlpList[7].(Data)).String())
	// S
	assert.Equal(t, "0x032e8717112b372f41c4a2a46ad0ea807f56645990130cbbc60614f2240a3a1a", ethtypes.HexBytes0xPrefix(rlpList[8].(Data)).String())
}
