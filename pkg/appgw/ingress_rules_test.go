package appgw

import (
	"testing"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

func TestIngressRules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up SSL redirect annotations")
}

var _ = Describe("Process ingress rules", func() {
	Context("without certificates", func() {
		certs := getCertsTestFixture()
		cb := makeConfigBuilderTestFixture(&certs)

		ingress := makeIngressTestFixture()
		ingress.Annotations[annotations.SslRedirectKey] = "one/two/three"
		ingress.Spec.TLS = nil // Ensure there are no certs

		// !! Action !!
		frontendListeners, frontendPorts, _ := cb.processIngressRules(&ingress)

		ingressList := []*v1beta1.Ingress{&ingress}
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		It("should have correct count of elements", func() {
			Expect(len(frontendListeners.ToSlice())).To(Equal(2))
			Expect(len(frontendPorts.ToSlice())).To(Equal(1))

		})

		It("should contain expected hosts", func() {
			felSlice := frontendListeners.ToSlice()
			hosts := []string{
				felSlice[0].(frontendListenerIdentifier).HostName,
				felSlice[1].(frontendListenerIdentifier).HostName,
			}
			Expect(hosts).To(ContainElement(testFixturesHost))
			Expect(hosts).To(ContainElement(testFixturesOtherHost))
		})

		It("should have expected ports", func() {
			expectedPorts := []int32{
				80,
				80,
			}
			ports := []int32{
				(frontendListeners.ToSlice()[0]).(frontendListenerIdentifier).FrontendPort,
				(frontendListeners.ToSlice()[1]).(frontendListenerIdentifier).FrontendPort,
			}
			Expect(ports).To(Equal(expectedPorts))
		})

		It("should succeed", func() {
			actualPort := frontendPorts.ToSlice()[0]
			Expect(actualPort).To(Equal(int32(80)))
		})

		// check the ApplicationGatewayRequestRoutingRule structs
		It("should have nil RequestRoutingRules", func() {
			Expect(cb.appGwConfig.RequestRoutingRules).To(BeNil())
		})

		// check the ApplicationGatewayRedirectConfiguration struct
		It("should configure listeners correctly", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			var feListener frontendListenerIdentifier
			for _, feListener = range azConfigMapKeys {
				if feListener.HostName == testFixturesHost {
					break
				}
			}
			expectedKey := frontendListenerIdentifier{
				FrontendPort: 80,
				HostName:     testFixturesHost,
			}

			Expect(len(azConfigMapKeys)).To(Equal(2))
			Expect(feListener).To(Equal(expectedKey))

			expectedVal := frontendListenerAzureConfig{
				Protocol: "Http",
				Secret: secretIdentifier{},
			}

			actualVal := httpListenersAzureConfigMap[feListener]
			Expect(*actualVal).To(Equal(expectedVal))
		})
	})

	Context("with attached certificates", func() {
		certs := getCertsTestFixture()
		cb := makeConfigBuilderTestFixture(&certs)

		ingress := makeIngressTestFixture()
		ingress.Annotations[annotations.SslRedirectKey] = "one/two/three"

		// !! Action !!
		frontendListeners, frontendPorts, _ := cb.processIngressRules(&ingress)

		ingressList := []*v1beta1.Ingress{&ingress}
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		It("should have two front end listener", func() {
			Expect(len(frontendListeners.ToSlice())).To(Equal(2))
		})
		It("should have 1 front end port", func() {
			Expect(len(frontendPorts.ToSlice())).To(Equal(1))

			expected := frontendListenerIdentifier{
				FrontendPort: 443,
				HostName:     testFixturesHost,
			}
			actual := (frontendListeners.ToSlice()[0]).(frontendListenerIdentifier)
			Expect(actual.FrontendPort).To(Equal(expected.FrontendPort))

			actualPort := frontendPorts.ToSlice()[0]
			Expect(actualPort).To(Equal(int32(443)))
		})

		// check the ApplicationGatewayRequestRoutingRule structs
		It("should succeed", func() {
			Expect(cb.appGwConfig.RequestRoutingRules).To(BeNil())
		})

		// check the ApplicationGatewayRedirectConfiguration struct
		It("should succeed", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(2))
			Expect(azConfigMapKeys[1].FrontendPort).To(Equal(int32(443)))

			actualVal := httpListenersAzureConfigMap[azConfigMapKeys[0]]

			expectedVal := frontendListenerAzureConfig{
				Protocol: "Https",
				Secret: secretIdentifier{
					Namespace: testFixturesNamespace,
					Name:      testFixturesNameOfSecret,
				},
				SslRedirectConfigurationName: "k8s-ag-ingress-" +
					testFixturesNamespace +
					"-" +
					testFixturesName +
					"-sslr",
			}

			Expect(*actualVal).To(Equal(expectedVal))
		})
	})
})

func getMapKeys(m *map[frontendListenerIdentifier]*frontendListenerAzureConfig) []frontendListenerIdentifier {
	keys := make([]frontendListenerIdentifier, 0, len(*m))
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}
