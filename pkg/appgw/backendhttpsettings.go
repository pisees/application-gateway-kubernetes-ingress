// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (builder *appGwConfigBuilder) BackendHTTPSettingsCollection(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	backendIDs := utils.NewUnorderedSet()
	serviceBackendPairsMap := make(map[backendIdentifier](utils.UnorderedSet))

	for _, ingress := range ingressList {
		defIngressBackend := ingress.Spec.Backend
		if defIngressBackend != nil {
			backendIDs.Insert(generateBackendID(ingress, nil, nil, defIngressBackend))
		}
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP == nil {
				// skip no http rule
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				path := &rule.HTTP.Paths[pathIdx]
				backendIDs.Insert(generateBackendID(ingress, rule, path, &path.Backend))
			}
		}
	}

	unresolvedBackendID := make([]backendIdentifier, 0)
	backendIDs.ForEach(func(backendIDInterface interface{}) {
		backendID := backendIDInterface.(backendIdentifier)
		resolvedBackendPorts := utils.NewUnorderedSet()

		service := builder.k8sContext.GetService(backendID.serviceKey())
		if service == nil {
			glog.V(1).Infof("unable to get the service [%s]", backendID.serviceKey())
			resolvedBackendPorts.Insert(serviceBackendPortPair{
				ServicePort: backendID.Backend.ServicePort.IntVal,
				BackendPort: backendID.Backend.ServicePort.IntVal,
			})
		} else {
			for _, sp := range service.Spec.Ports {
				// find the backend port number
				// check if any service ports matches the specified ports
				if sp.Protocol != v1.ProtocolTCP {
					// ignore UDP ports
					continue
				}
				if fmt.Sprint(sp.Port) == backendID.Backend.ServicePort.String() ||
					sp.Name == backendID.Backend.ServicePort.String() ||
					sp.TargetPort.String() == backendID.Backend.ServicePort.String() {
					// matched a service port with a port from the service

					if sp.TargetPort.String() == "" {
						// targetPort is not defined, by default targetPort == port
						resolvedBackendPorts.Insert(serviceBackendPortPair{
							ServicePort: sp.Port,
							BackendPort: sp.Port,
						})
					} else {
						// target port is defined as name or port number
						if sp.TargetPort.Type == intstr.Int {
							// port is defined as port number
							resolvedBackendPorts.Insert(serviceBackendPortPair{
								ServicePort: sp.Port,
								BackendPort: sp.TargetPort.IntVal,
							})
						} else {
							// if service port is defined by name, need to resolve
							targetPortName := sp.TargetPort.StrVal
							glog.V(1).Infof("resolving port name %s", targetPortName)
							targetPortsResolved := builder.resolvePortName(targetPortName, &backendID)
							targetPortsResolved.ForEach(func(targetPortInterface interface{}) {
								targetPort := targetPortInterface.(int32)
								resolvedBackendPorts.Insert(serviceBackendPortPair{
									ServicePort: sp.Port,
									BackendPort: targetPort,
								})
							})
						}
					}
					break
				}
			}
		}

		if resolvedBackendPorts.Size() == 0 {
			glog.V(1).Infof("unable to resolve any backend port for service [%s]", backendID.serviceKey())
			unresolvedBackendID = append(unresolvedBackendID, backendID)
			return
		}

		if serviceBackendPairsMap[backendID] == nil {
			serviceBackendPairsMap[backendID] = utils.NewUnorderedSet()
		}
		serviceBackendPairsMap[backendID] = serviceBackendPairsMap[backendID].Union(resolvedBackendPorts)
	})

	if len(unresolvedBackendID) > 0 {
		return builder, errors.New("unable to resolve backend port for some services")
	}

	probeID := builder.appGwIdentifier.probeID(defaultProbeName)
	httpSettingsCollection := make([](network.ApplicationGatewayBackendHTTPSettings), 0)
	httpSettingsCollection = append(httpSettingsCollection, defaultBackendHTTPSettings(probeID))

	// enforce single pair relationship between service port and backend port
	for backendID, serviceBackendPairs := range serviceBackendPairsMap {
		if serviceBackendPairs.Size() > 1 {
			// more than one possible backend port exposed through ingress
			glog.Warningf("service:port [%s:%s] has more than one service-backend port binding",
				backendID.serviceKey(), backendID.Backend.ServicePort.String())
			return builder, errors.New("more than one service-backend port binding is not allowed")
		}

		var uniquePair serviceBackendPortPair
		serviceBackendPairs.ForEach(func(pairI interface{}) {
			uniquePair = pairI.(serviceBackendPortPair)
		})

		builder.serviceBackendPairMap[backendID] = uniquePair

		probeName := builder.probesMap[backendID].Name
		probeID := builder.appGwIdentifier.probeID(*probeName)
		httpSettingsName := generateHTTPSettingsName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), uniquePair.BackendPort, backendID.Ingress.Name)
		glog.Infof("Created a new HTTP setting w/ name: %s\n", httpSettingsName)
		httpSettingsPort := uniquePair.BackendPort
		backendPathPrefix := to.StringPtr(annotations.BackendPathPrefix(backendID.Ingress))
		httpSettings := network.ApplicationGatewayBackendHTTPSettings{
			Etag: to.StringPtr("*"),
			Name: &httpSettingsName,
			ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
				Protocol: network.HTTP,
				Port:     &httpSettingsPort,
				Path:     backendPathPrefix,
				Probe:    resourceRef(probeID),
			},
		}
		// other settings should come from annotations
		httpSettingsCollection = append(httpSettingsCollection, httpSettings)
		builder.backendHTTPSettingsMap[backendID] = &httpSettings
	}

	builder.appGwConfig.BackendHTTPSettingsCollection = &httpSettingsCollection

	return builder, nil
}
