// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helper

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// NamespacedImporter returns a function that can be used to import resources
// where the namespace is not included in the resource ID.
func NamespacedImporter(readFunc schema.ReadFunc) schema.StateContextFunc {
	return func(_ context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
		namespacedID := d.Id()
		sepIdx := strings.LastIndex(d.Id(), "@")
		if sepIdx == -1 {
			readFunc(d, meta)
			return []*schema.ResourceData{d}, nil
		}

		d.SetId(namespacedID[:sepIdx])
		d.Set("namespace", namespacedID[sepIdx+1:])

		readFunc(d, meta)
		return []*schema.ResourceData{d}, nil
	}
}
