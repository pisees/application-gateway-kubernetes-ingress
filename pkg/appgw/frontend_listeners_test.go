package appgw

import (
	"testing"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

func TestFrontendListeners(t *testing.T) {
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

		listeners, _ := cb.getFrontendListeners(ingressList)

		It("should have correct count of listeners", func() {
			Expect(len(*listeners)).To(Equal(4))
		})

		It("should have correct values for listeners", func() {
			// Get the HTTPS listener for this test
			var listener network.ApplicationGatewayHTTPListener
			for _, listener = range *listeners {
				if listener.Protocol == "Https" && *listener.HostName == testFixturesHost {
					break
				}
			}
			Expect(*listener.HostName).To(Equal(testFixturesHost))
			portID := "/subscriptions//resourceGroups//providers/Microsoft.Network/" +
				"applicationGateways//frontEndPorts/k8s-ag-ingress-fp-443"
			Expect(*listener.FrontendPort.ID).To(Equal(portID))
		})
	})
})
