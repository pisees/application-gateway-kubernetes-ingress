// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const provisionStateExpected = "--provisionStateExpected--"
const rewriteRulesetID = "--RewriteRuleSet--"
const redirectConfigID = "/subscriptions//resourceGroups//providers/Microsoft.Network/applicationGateways//" +
	"redirectConfigurations/k8s-ag-ingress-" +
	testFixturesNamespace +
	"-" +
	testFixturesName +
	"-sslr"

func makeHTTPURLPathMap() network.ApplicationGatewayURLPathMap {
	return network.ApplicationGatewayURLPathMap{
		ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
			PathRules: &[]network.ApplicationGatewayPathRule{
				{
					ID:   to.StringPtr("-the-id-"),
					Type: to.StringPtr("-the-type-"),
					Etag: to.StringPtr("-the-etag-"),
					Name: to.StringPtr("/some/path"),
					ApplicationGatewayPathRulePropertiesFormat: &network.ApplicationGatewayPathRulePropertiesFormat{
						BackendAddressPool:    resourceRef("--BackendAddressPool--"),
						BackendHTTPSettings:   resourceRef("--BackendHTTPSettings--"),
						RedirectConfiguration: resourceRef("--RedirectConfiguration--"),
						RewriteRuleSet:        resourceRef(rewriteRulesetID),
						ProvisioningState:     to.StringPtr(provisionStateExpected),
					},
				},
			},
		},
	}
}

func TestGetSslRedirectConfigResourceReference(t *testing.T) {
	configBuilder := makeConfigBuilderTestFixture(nil)
	ingress := makeIngressTestFixture()
	actualID := *(configBuilder.getSslRedirectConfigResourceReference(&ingress).ID)
	if actualID != redirectConfigID {
		t.Error(fmt.Sprintf("\nExpected: %s\nActually: %s\n", redirectConfigID, actualID))
	}
}

func TestAddPathRulesZeroPathRules(t *testing.T) {
	configBuilder := makeConfigBuilderTestFixture(nil)
	ingress := makeIngressTestFixture()
	actualURLPathMap := makeHTTPURLPathMap()
	// Ensure there are no path rules defined for this test
	actualURLPathMap.PathRules = &[]network.ApplicationGatewayPathRule{}

	// Action -- will mutate actualURLPathMap struct
	configBuilder.addPathRules(&ingress, &actualURLPathMap)

	actualID := *(actualURLPathMap.DefaultRedirectConfiguration.ID)
	if actualID != redirectConfigID {
		t.Error(fmt.Sprintf("\nExpected: %+v\nActually: %+v\n", redirectConfigID, actualID))
	}

	if len(*actualURLPathMap.PathRules) != 0 {
		t.Error(fmt.Sprintf("Expected length of PathRules to be 0. It is %d", len(*actualURLPathMap.PathRules)))
	}

}

func TestAddPathRulesManyPathRules(t *testing.T) {
	configBuilder := makeConfigBuilderTestFixture(nil)
	ingress := makeIngressTestFixture()
	pathMap := makeHTTPURLPathMap()

	// Ensure the test is setup correctly
	if len(*pathMap.PathRules) != 1 {
		t.Error(fmt.Sprintf("Expected length of PathRules to be 1. It is %d", len(*pathMap.PathRules)))
	}

	// Action -- will mutate pathMap struct
	configBuilder.addPathRules(&ingress, &pathMap)

	// Ensure the test is setup correctly
	actual := *(*pathMap.PathRules)[0].ApplicationGatewayPathRulePropertiesFormat

	if actual.BackendAddressPool != nil {
		t.Error(fmt.Sprintf("BackendAddressPool expected to be nil. Its ID is %s\n", *actual.BackendAddressPool.ID))
	}

	if actual.BackendHTTPSettings != nil {
		t.Error(fmt.Sprintf("BackendHTTPSettings expected to be nil. Its ID is %s\n", *actual.BackendHTTPSettings.ID))
	}

	if *actual.RedirectConfiguration.ID != redirectConfigID {
		t.Error(fmt.Sprintf("RedirectConfiguration.ID expected to be %s Its ID is %s\n", redirectConfigID, *actual.RedirectConfiguration.ID))
	}

	if *actual.RewriteRuleSet.ID != rewriteRulesetID {
		t.Error(fmt.Sprintf("RewriteRuleSet expected to be %s Its ID is %s\n", rewriteRulesetID, *actual.RewriteRuleSet.ID))
	}

	if *actual.ProvisioningState != provisionStateExpected {
		t.Error(fmt.Sprintf("ProvisioningState expected to be %s. It is %s\n", provisionStateExpected, *actual.ProvisioningState))
	}
}
