// Copyright Aeraki Authors
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

package etcd

import (
	"context"
	"time"

	"github.com/zhaohuabing/debounce"
	"google.golang.org/grpc/stats"

	"istio.io/pkg/log"
)

type connHandler struct {
	stats.Handler
	*debounce.Debouncer
}

func newConnHandler(callback func(), stop <-chan struct{}) *connHandler {
	handler := &connHandler{
		Debouncer: debounce.New(5*time.Second, 10*time.Second, callback, stop),
	}
	return handler
}
func (connHandler) TagRPC(context context.Context, tagInfo *stats.RPCTagInfo) context.Context {
	return context
}

// HandleRPC processes the RPC stats.
func (connHandler) HandleRPC(context.Context, stats.RPCStats) {}

func (connHandler) TagConn(context context.Context, info *stats.ConnTagInfo) context.Context {
	return context
}

// HandleConn processes the Conn stats.
func (c connHandler) HandleConn(context context.Context, connStats stats.ConnStats) {
	switch connStats.(type) {
	case *stats.ConnBegin:
		log.Infof("ConnBegin")
		c.Bounce()
	case *stats.ConnEnd:
		log.Infof("ConnEnd")
		c.Cancel()
	}
}
