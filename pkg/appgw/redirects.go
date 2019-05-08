package appgw

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) getRedirectConfigurations(ingressList []*v1beta1.Ingress) *[]network.ApplicationGatewayRedirectConfiguration {
	listenerConfigs := builder.getListenerConfigs(ingressList)
	var redirectConfigs []network.ApplicationGatewayRedirectConfiguration

	for _, fel := range builder.getFrontendListenersSet(ingressList).ToSlice() {
		listener := fel.(frontendListenerIdentifier)
		azListenerConfig := builder.makeFELAzureConfig(listener, listenerConfigs)

		isHTTPS := azListenerConfig.Protocol == network.HTTPS
		hasSslRedirectName := azListenerConfig.SslRedirectConfigurationName != ""

		if isHTTPS && hasSslRedirectName {
			targetListener := resourceRef(builder.appGwIdentifier.httpListenerID(generateHTTPListenerName(listener)))
			rc := makeSSLRedirectConfig(azListenerConfig, targetListener)
			redirectConfigs = append(redirectConfigs, rc)
		}
	}

	return &redirectConfigs
}

func makeSSLRedirectConfig(flc *frontendListenerAzureConfig, targetListener *network.SubResource) network.ApplicationGatewayRedirectConfiguration {
	rcp := network.ApplicationGatewayRedirectConfigurationPropertiesFormat{
		RedirectType:       network.Permanent,
		TargetListener:     targetListener,
		IncludePath:        to.BoolPtr(true),
		IncludeQueryString: to.BoolPtr(true),
	}

	return network.ApplicationGatewayRedirectConfiguration{
		Etag: to.StringPtr("*"),
		Name: &flc.SslRedirectConfigurationName,
		ApplicationGatewayRedirectConfigurationPropertiesFormat: &rcp,
	}
}
