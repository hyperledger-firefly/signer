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

import (
	"github.com/hyperledger-firefly/common/pkg/i18n"
	"golang.org/x/text/language"
)

var ffc = func(key, translation, fieldType string) i18n.ConfigMessageKey {
	return i18n.FFC(language.AmericanEnglish, key, translation, fieldType)
}

//revive:disable
var (
	ConfigFileWalletEnabled                      = ffc("config.fileWallet.enabled", "Whether the Keystore V3 filesystem wallet is enabled", "boolean")
	ConfigFileWalletPath                         = ffc("config.fileWallet.path", "Path on the filesystem where the metadata files (and/or key files) are located", "string")
	ConfigFileWalletFilenamesPrimaryBatchRegex   = ffc("config.fileWallet.filenames.primaryMatchRegex", "Regular expression run against key/metadata filenames to extract the address (takes precedence over primaryExt)", "regexp")
	ConfigFileWalletFilenamesWith0xPrefix        = ffc("config.fileWallet.filenames.with0xPrefix", "When true and passwordExt is used, password filenames will be generated with an 0x prefix", "boolean")
	ConfigFileWalletFilenamesPrimaryExt          = ffc("config.fileWallet.filenames.primaryExt", "Extension for key/metadata files named by <ADDRESS>.<EXT>", "string")
	ConfigFileWalletFilenamesPasswordExt         = ffc("config.fileWallet.filenames.passwordExt", "Optional to use to look up password files, that sit next to the key files directly. Alternative to metadata when you have a password per keystore", "string")
	ConfigFileWalletFilenamesPasswordPath        = ffc("config.fileWallet.filenames.passwordPath", "Optional directory in which to look for the password files, when passwordExt is configured. Default is the wallet directory", "string")
	ConfigFileWalletFilenamesPasswordTrimSpace   = ffc("config.fileWallet.filenames.passwordTrimSpace", "Whether to trim leading/trailing whitespace (such as a newline) from the password when loaded from file", "boolean")
	ConfigFileWalletDefaultPasswordFile          = ffc("config.fileWallet.defaultPasswordFile", "Optional default password file to use, if one is not specified individually for the key (via metadata, or file extension)", "string")
	ConfigFileWalletDisableListener              = ffc("config.fileWallet.disableListener", "Disable the filesystem listener that automatically detects the creation of new keystore files", "boolean")
	ConfigFileWalletSignerCacheSize              = ffc("config.fileWallet.signerCacheSize", "Maximum of signing keys to hold in memory", "number")
	ConfigFileWalletSignerCacheTTL               = ffc("config.fileWallet.signerCacheTTL", "How long ot leave an unused signing key in memory", "duration")
	ConfigFileWalletMetadataFormat               = ffc("config.fileWallet.metadata.format", "Set this if the primary key file is a metadata file. Supported formats: auto (from extension) / filename / toml / yaml / json (please quote \"0x...\" strings in YAML)", "string")
	ConfigFileWalletMetadataKeyFileProperty      = ffc("config.fileWallet.metadata.keyFileProperty", "Go template to look up the key-file path from the metadata. Example: '{{ index .signing \"key-file\" }}'", "go-template")
	ConfigFileWalletMetadataPasswordFileProperty = ffc("config.fileWallet.metadata.passwordFileProperty", "Go template to look up the password-file path from the metadata", "go-template")

	ConfigServerAddress      = ffc("config.server.address", "Local address for the JSON/RPC server to listen on", "string")
	ConfigServerPort         = ffc("config.server.port", "Port for the JSON/RPC server to listen on", "number")
	ConfigAPIPublicURL       = ffc("config.server.publicURL", "External address callers should access API over", "string")
	ConfigServerReadTimeout  = ffc("config.server.readTimeout", "The maximum time to wait when reading from an HTTP connection", "duration")
	ConfigServerWriteTimeout = ffc("config.server.writeTimeout", "The maximum time to wait when writing to a HTTP connection", "duration")
	ConfigAPIShutdownTimeout = ffc("config.server.shutdownTimeout", "The maximum amount of time to wait for any open HTTP requests to finish before shutting down the HTTP server", i18n.TimeDurationType)

	ConfigBackendChainID  = ffc("config.backend.chainId", "Optionally set the Chain ID of the blockchain. Otherwise the Network ID will be queried, and used as the Chain ID in signing", "number")
	ConfigBackendURL      = ffc("config.backend.url", "URL for the backend JSON/RPC server / blockchain node", "url")
	ConfigBackendProxyURL = ffc("config.backend.proxy.url", "Optional HTTP proxy URL", "url")
)
