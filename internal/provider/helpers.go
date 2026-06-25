// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"

	bpclient "github.com/bridgeinpt/bridgeport/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// isNotFound reports whether err is a BridgePort API 404. Resources use it to
// drop deleted objects from state during Read instead of erroring.
func isNotFound(err error) bool {
	var apiErr *bpclient.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return false
}

// stringOrNull maps an SDK *string to a Terraform string, treating both nil and
// the empty string as null. Avoids spurious null-vs-"" diffs on optional fields.
func stringOrNull(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// listToStrings converts an optional Terraform list of strings to a Go slice. A
// null/unknown list yields a nil slice, so the SDK request omits the field.
func listToStrings(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := l.ElementsAs(ctx, &out, false)
	return out, diags
}

// parseTags converts the API's raw JSON-encoded tags string (e.g. `["a","b"]`)
// into a Terraform list. When the result is empty and the prior state was null,
// it preserves null to avoid a spurious null-vs-empty-list diff.
func parseTags(ctx context.Context, raw string, prior types.List) (types.List, diag.Diagnostics) {
	var tags []string
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &tags); err != nil {
			tags = nil // treat an unparseable value as no tags
		}
	}
	if len(tags) == 0 && prior.IsNull() {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, tags)
}
