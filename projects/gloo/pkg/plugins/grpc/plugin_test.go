package grpc

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/solo-io/gloo/pkg/utils"
	v1 "github.com/solo-io/gloo/projects/gloo/pkg/api/v1"
	pluginsv1 "github.com/solo-io/gloo/projects/gloo/pkg/api/v1/options"
	v1grpc "github.com/solo-io/gloo/projects/gloo/pkg/api/v1/options/grpc"
	v1static "github.com/solo-io/gloo/projects/gloo/pkg/api/v1/options/static"
	transformapi "github.com/solo-io/gloo/projects/gloo/pkg/api/v1/options/transformation"
	"github.com/solo-io/gloo/projects/gloo/pkg/plugins"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"

	envoycluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoyroute "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/gogo/protobuf/types"
)

var _ = Describe("Plugin", func() {

	var (
		p            *plugin
		params       plugins.Params
		upstream     *v1.Upstream
		upstreamSpec *v1static.UpstreamSpec
		out          *envoycluster.Cluster
		grpcSpec     *pluginsv1.ServiceSpec_Grpc
	)

	BeforeEach(func() {
		b := false
		p = NewPlugin(&b)
		out = new(envoycluster.Cluster)

		grpcSpec = &pluginsv1.ServiceSpec_Grpc{
			Grpc: &v1grpc.ServiceSpec{
				GrpcServices: []*v1grpc.ServiceSpec_GrpcService{{
					PackageName:   "foo",
					ServiceName:   "bar",
					FunctionNames: []string{"func"},
				}},
			},
		}

		p.Init(plugins.InitParams{})
		upstreamSpec = &v1static.UpstreamSpec{
			ServiceSpec: &pluginsv1.ServiceSpec{
				PluginType: grpcSpec,
			},
			Hosts: []*v1static.Host{{
				Addr: "localhost",
				Port: 1234,
			}},
		}
		upstream = &v1.Upstream{
			Metadata: core.Metadata{
				Name:      "test",
				Namespace: "default",
			},
			UpstreamType: &v1.Upstream_Static{
				Static: upstreamSpec,
			},
		}

	})
	Context("upstream", func() {
		It("should not mark non-grpc upstreams as http2", func() {
			upstreamSpec.ServiceSpec.PluginType = nil
			err := p.ProcessUpstream(params, upstream, out)
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Http2ProtocolOptions).To(BeNil())
		})

		It("should mark grpc upstreams as http2", func() {
			err := p.ProcessUpstream(params, upstream, out)
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Http2ProtocolOptions).NotTo(BeNil())
		})
	})

	Context("route", func() {

		ps := &transformapi.Parameters{
			Path: &types.StringValue{Value: "/{what}/{ ever }/{nested.field}/too"},
			Headers: map[string]string{
				"header-simple":            "{simple}",
				"header-simple-with-space": "{ simple_with_space }",
				"header-nested":            "{something.nested}",
			},
		}

		It("should process route", func() {

			var routeParams plugins.RouteParams
			routeIn := &v1.Route{
				Action: &v1.Route_RouteAction{
					RouteAction: &v1.RouteAction{
						Destination: &v1.RouteAction_Single{
							Single: &v1.Destination{
								DestinationSpec: &v1.DestinationSpec{
									DestinationType: &v1.DestinationSpec_Grpc{
										Grpc: &v1grpc.DestinationSpec{
											Package:    "foo",
											Service:    "bar",
											Function:   "func",
											Parameters: ps,
										},
									},
								},
								DestinationType: &v1.Destination_Upstream{
									Upstream: utils.ResourceRefPtr(upstream.Metadata.Ref()),
								},
							},
						},
					},
				},
			}

			routeOut := &envoyroute.Route{
				Match: &envoyroute.RouteMatch{
					PathSpecifier: &envoyroute.RouteMatch_Prefix{Prefix: "/"},
				},
				Action: &envoyroute.Route_Route{
					Route: &envoyroute.RouteAction{},
				},
			}
			err := p.ProcessUpstream(params, upstream, out)
			Expect(err).NotTo(HaveOccurred())
			err = p.ProcessRoute(routeParams, routeIn, routeOut)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
