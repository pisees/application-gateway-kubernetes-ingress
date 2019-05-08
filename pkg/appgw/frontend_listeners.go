package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) getFrontendListeners(ingressList []*v1beta1.Ingress) (*[]network.ApplicationGatewayHTTPListener, map[frontendListenerIdentifier]*network.ApplicationGatewayHTTPListener) {
	// TODO(draychev): this is for compatibility w/ RequestRoutingRules and should be removed ASAP
	legacyMap := make(map[frontendListenerIdentifier]*network.ApplicationGatewayHTTPListener)

	frontendListeners := builder.getFrontendListenersSet(ingressList)
	listenerConfigs := builder.getListenerConfigs(ingressList)
	var httpListeners []network.ApplicationGatewayHTTPListener

	for _, frontendListenerID := range frontendListeners.ToSlice() {
		listener := frontendListenerID.(frontendListenerIdentifier)
		azListenerConfig := builder.makeFELAzureConfig(listener, listenerConfigs)
		httpListener := builder.makeListener(listener, azListenerConfig.Protocol)

		if azListenerConfig.Protocol == network.HTTPS {
			sslCertificateName := azListenerConfig.Secret.secretFullName()
			sslCertificateID := builder.appGwIdentifier.sslCertificateID(sslCertificateName)

			httpListener.SslCertificate = resourceRef(sslCertificateID)

			if len(*httpListener.ApplicationGatewayHTTPListenerPropertiesFormat.HostName) != 0 {
				httpListener.RequireServerNameIndication = to.BoolPtr(true)
			}
		}

		if len(*httpListener.ApplicationGatewayHTTPListenerPropertiesFormat.HostName) != 0 {
			httpListeners = append([]network.ApplicationGatewayHTTPListener{httpListener}, httpListeners...)
		} else {
			httpListeners = append(httpListeners, httpListener)
		}

		legacyMap[listener] = &httpListener
	}
	// TODO(draychev): The second parameter is for compatibility w/ RequestRoutingRules and should be removed ASAP
	return &httpListeners, legacyMap
}

func (builder *appGwConfigBuilder) getFrontendListenersSet(ingressList []*v1beta1.Ingress) utils.UnorderedSet {
	allListeners := make(map[frontendListenerIdentifier]interface{})
	for _, ingress := range ingressList {
		feListeners, _, _ := builder.processIngressRules(ingress)
		for _, listener := range feListeners.ToSlice() {
			l := listener.(frontendListenerIdentifier)
			allListeners[l] = nil
		}
	}

	if len(allListeners) == 0 {
		dflt := defaultFrontendListenerIdentifier()
		allListeners[dflt] = nil
	}

	// TODO(draychev): Swap UnorderedSet with go-sets
	frontendListeners := utils.NewUnorderedSet()
	for listener := range allListeners {
		frontendListeners.Insert(listener)
	}
	return frontendListeners
}

func (builder *appGwConfigBuilder) makeListener(listener frontendListenerIdentifier, protocol network.ApplicationGatewayProtocol) network.ApplicationGatewayHTTPListener {
	frontendPortName := generateFrontendPortName(listener.FrontendPort)
	frontendPortID := builder.appGwIdentifier.frontendPortID(frontendPortName)

	feConfigs := *builder.appGwConfig.FrontendIPConfigurations
	firstConfig := feConfigs[0]

	return network.ApplicationGatewayHTTPListener{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(generateHTTPListenerName(listener)),
		ApplicationGatewayHTTPListenerPropertiesFormat: &network.ApplicationGatewayHTTPListenerPropertiesFormat{
			// TODO: expose this to external configuration
			FrontendIPConfiguration: resourceRef(*firstConfig.ID),
			FrontendPort:            resourceRef(frontendPortID),
			Protocol:                protocol,
			HostName:                &listener.HostName,
		},
	}
}

func (builder *appGwConfigBuilder) makeFELAzureConfig(listener frontendListenerIdentifier, azConfigPerListener map[frontendListenerIdentifier]*frontendListenerAzureConfig) *frontendListenerAzureConfig {
	if config := azConfigPerListener[listener]; config != nil {
		return config
	}
	// Default config
	return &frontendListenerAzureConfig{
		Protocol: network.HTTP,
	}
}

func (builder *appGwConfigBuilder) getListenerConfigs(ingressList []*v1beta1.Ingress) map[frontendListenerIdentifier]*frontendListenerAzureConfig {
	httpListenersAzureConfigMap := make(map[frontendListenerIdentifier]*frontendListenerAzureConfig)

	for _, ingress := range ingressList {
		_, _, configMap := builder.processIngressRules(ingress)
		for k, v := range configMap {
			httpListenersAzureConfigMap[k] = v
		}
	}

	return httpListenersAzureConfigMap
}
