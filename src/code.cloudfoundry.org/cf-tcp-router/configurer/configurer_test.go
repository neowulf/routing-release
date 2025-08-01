package configurer_test

import (
	"reflect"

	tlshelpers "code.cloudfoundry.org/cf-routing-test-helpers/tls"
	"code.cloudfoundry.org/cf-tcp-router/config"
	"code.cloudfoundry.org/cf-tcp-router/configurer"
	"code.cloudfoundry.org/cf-tcp-router/configurer/haproxy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configurer", func() {

	Describe("NewConfigurer", func() {
		var backendTlsCfg config.BackendTLSConfig
		var frontendTlsCfg []config.FrontendTLSConfig
		BeforeEach(func() {
			caFile, _ := tlshelpers.GenerateCa()
			_, clientCertFile, _, _ := tlshelpers.GenerateCaAndMutualTlsCerts()

			backendTlsCfg = config.BackendTLSConfig{
				CACertificatePath:    caFile,
				ClientCertAndKeyPath: clientCertFile,
			}
		})
		Context("when 'haproxy' tcp load balancer is passed", func() {
			It("should return haproxy configurer", func() {
				routeConfigurer := configurer.NewConfigurer(logger,
					configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "haproxy/fixtures/haproxy.cfg", nil, nil, backendTlsCfg, frontendTlsCfg)
				Expect(routeConfigurer).ShouldNot(BeNil())
				expectedType := reflect.PointerTo(reflect.TypeOf(haproxy.Configurer{}))
				value := reflect.ValueOf(routeConfigurer)
				Expect(value.Type()).To(Equal(expectedType))
			})

			Context("when invalid config file is passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "", nil, nil, backendTlsCfg, frontendTlsCfg)
					}).Should(Panic())
				})
			})

			Context("when invalid base config file is passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "", "haproxy/fixtures/haproxy.cfg", nil, nil, backendTlsCfg, frontendTlsCfg)
					}).Should(Panic())
				})
			})

			Context("when invalid CA file is passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "haproxy/fixtures/haproxy.cfg", nil, nil, config.BackendTLSConfig{CACertificatePath: "nonexistent/file"}, frontendTlsCfg)
					}).Should(Panic())
				})
			})

			Context("when invalid ClientCertAndKey file is passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "haproxy/fixtures/haproxy.cfg", nil, nil, config.BackendTLSConfig{ClientCertAndKeyPath: "nonexistent/file"}, frontendTlsCfg)
					}).Should(Panic())
				})
			})
			Context("when FrontendTLS is enabled and empty CertDir are passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "haproxy/fixtures/haproxy.cfg", nil, nil, config.BackendTLSConfig{}, []config.FrontendTLSConfig{config.FrontendTLSConfig{Enabled: true}})
					}).ShouldNot(Panic())
				})
			})
			Context("when FrontendTLS is enabled and invalid CertDir is passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "haproxy/fixtures/haproxy.cfg", nil, nil, config.BackendTLSConfig{}, []config.FrontendTLSConfig{config.FrontendTLSConfig{Enabled: true, CertificateDir: "foobar"}})
					}).Should(Panic())
				})
			})
			Context("when FrontendTLS is enabled and valid CertDir is passed", func() {
				It("should not panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "haproxy/fixtures/haproxy.cfg", nil, nil, config.BackendTLSConfig{}, []config.FrontendTLSConfig{config.FrontendTLSConfig{Enabled: true, CertificateDir: "fixtures"}})
					}).Should(Panic())
				})
			})
			Context("when empty CA + ClientCertAndKey paths are passed", func() {
				It("should not panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "haproxy/fixtures/haproxy.cfg", nil, nil, config.BackendTLSConfig{}, frontendTlsCfg)
					}).ShouldNot(Panic())
				})
			})
		})

		Context("when non-supported tcp load balancer is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, "not-supported", "some-base-config-file", "some-config-file", nil, nil, backendTlsCfg, frontendTlsCfg)
				}).Should(Panic())
			})
		})

		Context("when empty tcp load balancer is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, "", "some-base-config-file", "some-config-file", nil, nil, backendTlsCfg, frontendTlsCfg)
				}).Should(Panic())
			})
		})

	})
})
