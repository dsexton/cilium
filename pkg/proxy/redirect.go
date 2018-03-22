// Copyright 2016-2018 Authors of Cilium
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

package proxy

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/cilium/cilium/pkg/completion"
	"github.com/cilium/cilium/pkg/lock"
	"github.com/cilium/cilium/pkg/maps/proxymap"
	"github.com/cilium/cilium/pkg/policy"
	"github.com/cilium/cilium/pkg/proxy/logger"
)

// RedirectImplementation is the generic proxy redirect interface that each
// proxy redirect type must implement
type RedirectImplementation interface {
	UpdateRules(wg *completion.WaitGroup) error
	Close(wg *completion.WaitGroup)
}

type Redirect struct {
	// The following fields are only written to during initialization, it
	// is safe to read these fields without locking the mutex

	// ProxyPort is the port the redirects redirects to where the proxy is
	// listening on
	ProxyPort      uint16
	endpointID     uint64
	id             string
	ingress        bool
	port           uint16
	source         logger.EndpointInfoSource
	parserType     policy.L7ParserType
	created        time.Time
	implementation RedirectImplementation

	// The following fields are updated while the redirect is alive, the
	// mutex must be held to read and write these fields
	mutex       lock.RWMutex
	lastUpdated time.Time
	rules       policy.L7DataMap
}

func newRedirect(port uint16, source logger.EndpointInfoSource, id string) *Redirect {
	return &Redirect{
		port:        port,
		source:      source,
		id:          id,
		created:     time.Now(),
		lastUpdated: time.Now(),
	}
}

// updateRules updates the rules of the redirect, Redirect.mutex must be held
func (r *Redirect) updateRules(l4 *policy.L4Filter) {
	r.rules = policy.L7DataMap{}
	for key, val := range l4.L7RulesPerEp {
		r.rules[key] = val
	}
}

func (r *Redirect) getLocation() string {
	if r.ingress {
		return "ingress"
	}

	return "egress"
}

func (r *Redirect) getRulesModel() []string {
	model := make([]string, len(r.rules))
	idx := 0
	for selector, rule := range r.rules {
		jsonSelector, _ := json.Marshal(selector)
		var jsonRule []byte

		switch r.parserType {
		case policy.ParserTypeHTTP:
			jsonRule, _ = json.Marshal(rule.HTTP)
		case policy.ParserTypeKafka:
			jsonRule, _ = json.Marshal(rule.Kafka)
		}

		model[idx] = fmt.Sprintf("from %s: %s", string(jsonSelector), string(jsonRule))
		idx++
	}
	return model
}

// removeProxyMapEntryOnClose is called after the proxy has closed a connection
// and will remove the proxymap entry for that connection
func (r *Redirect) removeProxyMapEntryOnClose(c net.Conn) error {
	key, err := getProxyMapKey(c, r.ProxyPort)
	if err != nil {
		return fmt.Errorf("unable to extract proxymap key: %s", err)
	}

	return proxymap.Delete(key)
}

// LocalEndpointInfoSource returns an EndpointInfoSource with the information on the local endpoint.
func (r *Redirect) LocalEndpointInfoSource() logger.EndpointInfoSource {
	return r.source
}
