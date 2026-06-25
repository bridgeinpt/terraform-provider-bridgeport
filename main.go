// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/bridgeinpt/terraform-provider-bridgeport/internal/provider"
)

// version is set at build time via -ldflags by GoReleaser (see .goreleaser.yml).
// It defaults to "dev" for local builds.
var version = "dev"

// Regenerate the registry docs under docs/ from the schema + examples/.
//
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name bridgeport

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// This address must match the published registry namespace/name:
		// registry.terraform.io/providers/bridgeinpt/bridgeport
		Address: "registry.terraform.io/bridgeinpt/bridgeport",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
