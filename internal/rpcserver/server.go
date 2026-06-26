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

package rpcserver

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hyperledger-firefly/common/pkg/config"
	"github.com/hyperledger-firefly/common/pkg/ffresty"
	"github.com/hyperledger-firefly/common/pkg/httpserver"
	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/signer/internal/signerconfig"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
	"github.com/hyperledger-firefly/signer/pkg/ethsigner"
	"github.com/hyperledger-firefly/signer/pkg/ethtypes"
	"github.com/hyperledger-firefly/signer/pkg/rpcbackend"
)

type Server interface {
	Start() error
	Stop()
	WaitStop() error
}

func NewServer(ctx context.Context, wallet ethsigner.Wallet) (ss Server, err error) {

	httpClient, err := ffresty.New(ctx, signerconfig.BackendConfig)
	if err != nil {
		return nil, err
	}
	s := &rpcServer{
		backend:       rpcbackend.NewRPCClient(httpClient),
		apiServerDone: make(chan error),
		wallet:        wallet,
		chainID:       config.GetInt64(signerconfig.BackendChainID),
	}
	s.ctx, s.cancelCtx = context.WithCancel(ctx)

	s.apiServer, err = httpserver.NewHTTPServer(ctx, "server", s.router(), s.apiServerDone, signerconfig.ServerConfig, signerconfig.CorsConfig)
	if err != nil {
		return nil, err
	}

	return s, err
}

type rpcServer struct {
	ctx       context.Context
	cancelCtx func()
	backend   rpcbackend.Backend

	started       bool
	apiServer     httpserver.HTTPServer
	apiServerDone chan error

	chainID int64
	wallet  ethsigner.Wallet
}

func (s *rpcServer) router() *mux.Router {
	mux := mux.NewRouter()
	mux.Path("/").Methods(http.MethodPost).Handler(http.HandlerFunc(s.rpcHandler))
	return mux
}

func (s *rpcServer) runAPIServer() {
	s.apiServer.ServeHTTP(s.ctx)
}

func (s *rpcServer) Start() error {
	if s.chainID < 0 {
		var chainID ethtypes.HexInteger
		rpcErr := s.backend.CallRPC(s.ctx, &chainID, "net_version")
		if rpcErr != nil {
			return i18n.WrapError(s.ctx, rpcErr.Error(), signermsgs.MsgQueryChainID)
		}
		s.chainID = chainID.BigInt().Int64()
	}

	err := s.wallet.Initialize(s.ctx)
	if err != nil {
		return err
	}
	go s.runAPIServer()
	s.started = true
	return nil
}

func (s *rpcServer) Stop() {
	s.cancelCtx()
}

func (s *rpcServer) WaitStop() (err error) {
	if s.started {
		s.started = false
		err = <-s.apiServerDone
	}
	return err
}
