package appgw

import (
	"testing"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

func TestFrontendPorts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up SSL redirect annotations")
}

var _ = Describe("Process ingress rules", func() {
	Context("with many frontend ports", func() {
		certs := getCertsTestFixture()
		cb := makeConfigBuilderTestFixture(&certs)

		ing1 := makeIngressTestFixture()
		ing1.Annotations[annotations.SslRedirectKey] = "true"
		ing2 := makeIngressTestFixture()
		ing2.Annotations[annotations.SslRedirectKey] = "true"
		ingressList := []*v1beta1.Ingress{
			&ing1,
			&ing2,
		}

		ports := cb.getFrontendPorts(ingressList)

		It("should have correct count of ports", func() {
			Expect(len(*ports)).To(Equal(2))
		})

		It("should have correct count of ports", func() {
			// Get the HTTPS port only
			var httpsPort network.ApplicationGatewayFrontendPort
			for _, httpsPort = range *ports {
				if *httpsPort.Port == 443 {
					break
				}
			}
			Expect(*httpsPort.Port).To(Equal(int32(443)))
		})
	})
})
