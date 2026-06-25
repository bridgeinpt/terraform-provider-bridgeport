// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// providerConfig is prepended to every acceptance test's configuration. The
// provider reads its endpoint and token from BRIDGEPORT_ENDPOINT /
// BRIDGEPORT_TOKEN, which the acceptance harness (scripts/acc-harness.sh) sets
// after booting a disposable instance and minting a token — so the block is
// intentionally empty here.
const providerConfig = `
provider "bridgeport" {}
`

// testAccProtoV6ProviderFactories wires the in-process provider into the
// acceptance test framework. "test" is a sentinel version string.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"bridgeport": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck verifies the acceptance environment is wired up before any
// TF_ACC test runs. Acceptance tests talk to a real BridgePort instance.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("BRIDGEPORT_ENDPOINT") == "" {
		t.Fatal("BRIDGEPORT_ENDPOINT must be set for acceptance tests (see scripts/acc-harness.sh)")
	}
	if os.Getenv("BRIDGEPORT_TOKEN") == "" {
		t.Fatal("BRIDGEPORT_TOKEN must be set for acceptance tests (see scripts/acc-harness.sh)")
	}
}
