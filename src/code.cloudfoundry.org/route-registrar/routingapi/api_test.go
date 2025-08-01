package routingapi

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"

	fakeuaa "code.cloudfoundry.org/route-registrar/routingapi/routingapifakes"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/route-registrar/config"
	"code.cloudfoundry.org/routing-api/fake_routing_api"
	"code.cloudfoundry.org/routing-api/models"
)

var _ = Describe("Routing API", func() {
	var (
		client    *fake_routing_api.FakeClient
		uaaClient *fakeuaa.FakeUaaClient

		api    *RoutingAPI
		logger lager.Logger

		port         uint16
		externalPort uint16
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("routing api test")
		uaaClient = &fakeuaa.FakeUaaClient{}
		uaaClient.FetchTokenReturns(&oauth2.Token{AccessToken: "my-token"}, nil)
		client = &fake_routing_api.FakeClient{}
		client.RouterGroupWithNameReturns(models.RouterGroup{Guid: "router-group-guid"}, nil)
		api = NewRoutingAPI(logger, uaaClient, client, 2*time.Minute)

		port = 1234
		externalPort = 5678
	})

	It("Sets SNI hostname if ServerCertDomainSAN is present.", func() {
		tcpRouteMapping, err := api.makeTcpRouteMapping(config.Route{
			Port:                &port,
			ExternalPort:        &externalPort,
			RouterGroup:         "my-router-group",
			ServerCertDomainSAN: "sniHostname",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(tcpRouteMapping.SniHostname).ToNot(BeNil())
		Expect(*tcpRouteMapping.SniHostname).To(Equal("sniHostname"))
	})

	It("SNI hostname nil if ServerCertDomainSAN is not present.", func() {
		tcpRouteMapping, err := api.makeTcpRouteMapping(config.Route{
			Port:         &port,
			ExternalPort: &externalPort,
			RouterGroup:  "my-router-group",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(tcpRouteMapping.SniHostname).To(BeNil())
	})

	It("Sets TerminateFrontendTLS if TerminateFrontendTLS is present.", func() {
		tcpRouteMapping, err := api.makeTcpRouteMapping(config.Route{
			Port:                 &port,
			ExternalPort:         &externalPort,
			TerminateFrontendTLS: true,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(tcpRouteMapping.TerminateFrontendTLS).To(BeTrue())
	})

	It("Sets TerminateFrontendTLS if TerminateFrontendTLS is not present.", func() {
		tcpRouteMapping, err := api.makeTcpRouteMapping(config.Route{
			Port:         &port,
			ExternalPort: &externalPort,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(tcpRouteMapping.TerminateFrontendTLS).To(BeFalse())
	})

	Context("Sets HostTlsPort", func() {
		var route config.Route

		BeforeEach(func() {
			route = config.Route{
				ExternalPort:         &externalPort,
				Port:                 &port,
				TerminateFrontendTLS: true,
				EnableBackendTLS:     true,
				ServerCertDomainSAN:  "sniHostname",
			}
		})

		It("when TerminateFrontendTLS, EnableBackendTLS are enabled and ServerCertDomainSAN is present", func() {
			tcpRouteMapping, err := api.makeTcpRouteMapping(route)
			Expect(err).NotTo(HaveOccurred())

			Expect(tcpRouteMapping.HostPort).To(BeNumerically(">", 0))
			Expect(tcpRouteMapping.HostTLSPort).To(BeNumerically("==", tcpRouteMapping.HostPort))
		})

		It("when TerminateFrontendTLS, EnableBackendTLS are enabled and ServerCertDomainSAN is not present", func() {
			route.ServerCertDomainSAN = ""
			tcpRouteMapping, err := api.makeTcpRouteMapping(route)
			Expect(err).NotTo(HaveOccurred())

			Expect(tcpRouteMapping.HostTLSPort).To(Equal(-1))
			Expect(tcpRouteMapping.HostTLSPort).NotTo(Equal(tcpRouteMapping.HostPort))
		})

		It("when TerminateFrontendTLS is disabled, EnableBackendTLS is enabled and ServerCertDomainSAN is present", func() {
			route.TerminateFrontendTLS = false
			tcpRouteMapping, err := api.makeTcpRouteMapping(route)
			Expect(err).NotTo(HaveOccurred())

			Expect(tcpRouteMapping.HostTLSPort).To(Equal(-1))
			Expect(tcpRouteMapping.HostTLSPort).NotTo(Equal(tcpRouteMapping.HostPort))
		})

		It("when TerminateFrontendTLS is enabled, EnableBackendTLS is disabled and ServerCertDomainSAN is present", func() {
			route.EnableBackendTLS = false
			tcpRouteMapping, err := api.makeTcpRouteMapping(route)
			Expect(err).NotTo(HaveOccurred())

			Expect(tcpRouteMapping.HostTLSPort).To(Equal(-1))
			Expect(tcpRouteMapping.HostTLSPort).NotTo(Equal(tcpRouteMapping.HostPort))
		})
	})

	It("Sets ALPNs if ALPNs are present.", func() {
		tcpRouteMapping, err := api.makeTcpRouteMapping(config.Route{
			Port:         &port,
			ExternalPort: &externalPort,
			ALPNs:        []string{"alpn1", "alpn2"},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(tcpRouteMapping.ALPNs).To(Equal("alpn1,alpn2"))
	})

	It("ALPNs empty if ALPNs is not present.", func() {
		tcpRouteMapping, err := api.makeTcpRouteMapping(config.Route{
			Port:         &port,
			ExternalPort: &externalPort,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(tcpRouteMapping.ALPNs).To(BeEmpty())
	})
})
