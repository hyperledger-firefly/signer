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
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/hyperledger-firefly/common/pkg/i18n"
	"github.com/hyperledger-firefly/common/pkg/log"
	"github.com/hyperledger-firefly/signer/internal/signermsgs"
)

func (w *fsWallet) startFilesystemListener(ctx context.Context) error {
	if w.conf.DisableListener {
		log.L(ctx).Debugf("Filesystem listener disabled")
		close(w.fsListenerDone)
		return nil
	}
	watcher, err := fsnotify.NewWatcher()
	if err == nil {
		go w.fsListenerLoop(ctx, func() {
			_ = watcher.Close()
			close(w.fsListenerDone)
		}, watcher.Events, watcher.Errors)
		err = watcher.Add(w.conf.Path)
	}
	if err != nil {
		log.L(ctx).Errorf("Failed to start filesystem listener: %s", err)
		return i18n.WrapError(ctx, err, signermsgs.MsgFailedToStartListener, err)
	}
	return nil
}

func (w *fsWallet) fsListenerLoop(ctx context.Context, done func(), events chan fsnotify.Event, errors chan error) {
	defer done()

	for {
		select {
		case <-ctx.Done():
			log.L(ctx).Infof("File listener exiting")
			return
		case event, ok := <-events:
			if ok {
				log.L(ctx).Tracef("FSEvent [%s]: %s", event.Op, event.Name)
				fi, err := os.Stat(event.Name)
				if err == nil {
					_ = w.notifyNewFiles(ctx, fi)
				}
			}
		case err, ok := <-errors:
			if ok {
				log.L(ctx).Errorf("FSEvent error: %s", err)
			}
		}
	}
}
