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

package fswallet

import (
	"context"

	"github.com/hyperledger-firefly/common/pkg/config"
	"github.com/hyperledger-firefly/signer/pkg/keystorev3"
)

const (
	// ConfigPath the path of the Keystore V3 wallet path
	ConfigPath = "path"
	// ConfigFilenamesWith0xPrefix whether or not to use the 0x prefix on filenames, when using passwordExt password
	ConfigFilenamesWith0xPrefix = "filenames.with0xPrefix"
	// ConfigFilenamesPrimaryExt extension to append to the "from" address string to find the file (see metadata section for file types). All filenames must be lower case on disk.
	ConfigFilenamesPrimaryExt = "filenames.primaryExt"
	// ConfigFilenamesPrimaryMatchRegex allows filenames where the address can be extracted with a regular expression. Takes precedence over primaryExt
	ConfigFilenamesPrimaryMatchRegex = "filenames.primaryMatchRegex"
	// ConfigFilenamesPasswordExt extension to append to the "from" address string to find the password file (if not using a metadata file to specify the password file)
	ConfigFilenamesPasswordExt = "filenames.passwordExt"
	// ConfigFilenamesPasswordPath directory path where the password files should be found - default is the same path as the primary file
	ConfigFilenamesPasswordPath = "filenames.passwordPath"
	// ConfigFilenamesPasswordTrimSpace whether to trim whitespace from passwords loaded from files (such as trailing newline characters)
	ConfigFilenamesPasswordTrimSpace = "filenames.passwordTrimSpace"
	// ConfigDefaultPasswordFile default password file to use if neither the metadata, or passwordExtension find a password
	ConfigDefaultPasswordFile = "defaultPasswordFile"
	// ConfigDisableListener disable the filesystem listener that detects newly added keys automatically
	ConfigDisableListener = "disableListener"
	// ConfigSignerCacheSize the number of signing keys to keep in memory
	ConfigSignerCacheSize = "signerCacheSize"
	// ConfigSignerCacheTTL the time to keep an unused signing key in memory
	ConfigSignerCacheTTL = "signerCacheTTL"
	// ConfigMetadataFormat format to parse the metadata - supported: auto (from extension) / filename / toml / yaml / json (please quote "0x..." strings in YAML)
	ConfigMetadataFormat = "metadata.format"
	// ConfigMetadataKeyFileProperty use for toml/yaml/json to find the name of the file containing the keystorev3 file
	ConfigMetadataKeyFileProperty = "metadata.keyFileProperty"
	// ConfigMetadataPasswordFileProperty use for toml/yaml to find the name of the file containing the keystorev3 file
	ConfigMetadataPasswordFileProperty = "metadata.passwordFileProperty"
)

type Config struct {
	Path                string
	DefaultPasswordFile string
	SignerCacheSize     string
	SignerCacheTTL      string
	DisableListener     bool
	Filenames           FilenamesConfig
	Metadata            MetadataConfig
}

type ConfigGeneric struct {
	Config
	WalletFileValidator func(ctx context.Context, addrString string, kv3 keystorev3.WalletFile) error
	AddressValidator    func(ctx context.Context, addrString string) (string, error)
}

type FilenamesConfig struct {
	PrimaryMatchRegex string
	PrimaryExt        string
	PasswordExt       string
	PasswordPath      string
	PasswordTrimSpace bool
	With0xPrefix      bool
}

type MetadataConfig struct {
	Format               string
	KeyFileProperty      string
	PasswordFileProperty string
}

func InitConfig(section config.Section) {
	section.AddKnownKey(ConfigPath)
	section.AddKnownKey(ConfigFilenamesPrimaryExt)
	section.AddKnownKey(ConfigFilenamesPrimaryMatchRegex)
	section.AddKnownKey(ConfigFilenamesPasswordExt)
	section.AddKnownKey(ConfigFilenamesPasswordPath)
	section.AddKnownKey(ConfigFilenamesPasswordTrimSpace, true)
	section.AddKnownKey(ConfigFilenamesWith0xPrefix)
	section.AddKnownKey(ConfigDisableListener)
	section.AddKnownKey(ConfigDefaultPasswordFile)
	section.AddKnownKey(ConfigSignerCacheSize, 250)
	section.AddKnownKey(ConfigSignerCacheTTL, "24h")
	section.AddKnownKey(ConfigMetadataFormat, `auto`)
	section.AddKnownKey(ConfigMetadataKeyFileProperty)
	section.AddKnownKey(ConfigMetadataPasswordFileProperty)
}

func ReadConfig(section config.Section) *Config {
	return &Config{
		Path:                section.GetString(ConfigPath),
		DefaultPasswordFile: section.GetString(ConfigDefaultPasswordFile),
		SignerCacheSize:     section.GetString(ConfigSignerCacheSize),
		SignerCacheTTL:      section.GetString(ConfigSignerCacheTTL),
		DisableListener:     section.GetBool(ConfigDisableListener),
		Filenames: FilenamesConfig{
			PrimaryExt:        section.GetString(ConfigFilenamesPrimaryExt),
			PrimaryMatchRegex: section.GetString(ConfigFilenamesPrimaryMatchRegex),
			PasswordExt:       section.GetString(ConfigFilenamesPasswordExt),
			PasswordPath:      section.GetString(ConfigFilenamesPasswordPath),
			PasswordTrimSpace: section.GetBool(ConfigFilenamesPasswordTrimSpace),
			With0xPrefix:      section.GetBool(ConfigFilenamesWith0xPrefix),
		},
		Metadata: MetadataConfig{
			Format:               section.GetString(ConfigMetadataFormat),
			KeyFileProperty:      section.GetString(ConfigMetadataKeyFileProperty),
			PasswordFileProperty: section.GetString(ConfigMetadataPasswordFileProperty),
		},
	}
}
