// Copyright 2019 Authors of Cilium
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

package datapath

import (
	"context"
	"io"

	"github.com/cilium/cilium/pkg/datapath/loader/metrics"
	"github.com/sirupsen/logrus"
)

// Datapath is the interface to abstract all datapath interactions. The
// abstraction allows to implement the datapath requirements with multiple
// implementations
type Datapath interface {
	// Node must return the handler for node events
	Node() NodeHandler

	// LocalNodeAddressing must return the node addressing implementation
	// of the local node
	LocalNodeAddressing() NodeAddressing

	// WriteNodeConfig writes the implementation-specific configuration of
	// node-wide options into the specified writer.
	WriteNodeConfig(io.Writer, *LocalNodeConfiguration) error

	// WriteNetdevConfig writes the implementation-specific configuration
	// of configurable options to the specified writer. Options specified
	// here will apply to base programs and not to endpoints, though
	// endpoints may have equivalent configurable options.
	WriteNetdevConfig(io.Writer, DeviceConfiguration) error

	// WriteTemplateConfig writes the implementation-specific configuration
	// of configurable options for BPF templates to the specified writer.
	WriteTemplateConfig(w io.Writer, cfg EndpointConfiguration) error

	// WriteEndpointConfig writes the implementation-specific configuration
	// of configurable options for the endpoint to the specified writer.
	WriteEndpointConfig(w io.Writer, cfg EndpointConfiguration) error

	// InstallProxyRules creates the necessary datapath config (e.g., iptables
	// rules for redirecting host proxy traffic on a specific ProxyPort)
	InstallProxyRules(proxyPort uint16, ingress bool, name string) error

	// RemoveProxyRules creates the necessary datapath config (e.g., iptables
	// rules for redirecting host proxy traffic on a specific ProxyPort)
	RemoveProxyRules(proxyPort uint16, ingress bool, name string) error

	Loader() Loader
}

// Loader is an interface to abstract out loading of datapath programs.
type Loader interface {
	CallsMapPath(id uint16) string
	CompileAndLoad(ctx context.Context, ep Endpoint, stats *metrics.SpanStat) error
	CompileOrLoad(ctx context.Context, ep Endpoint, stats *metrics.SpanStat) error
	ReloadDatapath(ctx context.Context, ep Endpoint, stats *metrics.SpanStat) error
	EndpointHash(cfg EndpointConfiguration) (string, error)
	DeleteDatapath(ctx context.Context, ifName, direction string) error
	Unload(ep Endpoint)
}

// Endpoint provides access endpoint configuration information that is necessary
// to compile and load the datapath.
type Endpoint interface {
	EndpointConfiguration
	InterfaceName() string
	Logger(subsystem string) *logrus.Entry
	StateDir() string
	MapPath() string
}
