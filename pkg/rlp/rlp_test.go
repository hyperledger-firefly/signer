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

package rlp

import (
	"testing"

	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/stretchr/testify/assert"
)

func TestDataBytes(t *testing.T) {

	assert.Nil(t, ((List)(nil)).ToData())
	assert.Nil(t, ((Data)(nil)).ToData())
	assert.Equal(t, Data{0xff}, ((Data)([]byte{0xff})).ToData())

}

func TestDataIntOrZero(t *testing.T) {

	assert.Equal(t, int64(0), ((List)(nil)).ToData().IntOrZero().Int64())
	assert.Equal(t, int64(0xff), ((Data)([]byte{0xff})).ToData().IntOrZero().Int64())

}

func TestDataBytesNotNil(t *testing.T) {

	assert.Equal(t, []byte{}, ((List)(nil)).ToData().BytesNotNil())
	assert.Equal(t, []byte{0xff}, ((Data)([]byte{0xff})).ToData().BytesNotNil())

}

func TestAddress(t *testing.T) {

	assert.Nil(t, ((List)(nil)).ToData().Address())
	assert.Nil(t, (Data{0x00}).Address())
	assert.Equal(t, "0x4f78181c7fdc267d953a3cba8079f899d7f5ba78", (Data)(ethtypes.MustNewAddress("0x4F78181C7fdC267d953A3cBa8079f899D7F5BA78")[:]).Address().String())

}
