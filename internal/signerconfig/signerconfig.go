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

package signerconfig

import (
	"github.com/hyperledger-firefly/common/pkg/config"
	"github.com/hyperledger-firefly/common/pkg/httpserver"
	"github.com/hyperledger-firefly/common/pkg/wsclient"
	"github.com/hyperledger-firefly/signer/pkg/fswallet"
	"github.com/spf13/viper"
)

var ffc = config.AddRootKey

var (
	// BackendChainID optionally set the Chain ID manually (usually queries network ID)
	BackendChainID = ffc("backend.chainId")
	// FileWalletEnabled if the Keystore V3 wallet is enabled
	FileWalletEnabled = ffc("fileWallet.enabled")
)

var ServerConfig config.Section

var CorsConfig config.Section

var BackendConfig config.Section

var FileWalletConfig config.Section

func setDefaults() {
	viper.SetDefault(string(BackendChainID), -1)
	viper.SetDefault(string(FileWalletEnabled), true)
}

func Reset() {
	config.RootConfigReset(setDefaults)

	ServerConfig = config.RootSection("server")
	httpserver.InitHTTPConfig(ServerConfig, 8545)

	CorsConfig = config.RootSection("cors")
	httpserver.InitCORSConfig(CorsConfig)

	BackendConfig = config.RootSection("backend")
	wsclient.InitConfig(BackendConfig)

	FileWalletConfig = config.RootSection("fileWallet")
	fswallet.InitConfig(FileWalletConfig)

}
