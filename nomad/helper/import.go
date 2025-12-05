// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package helper

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	missingNamespaceImportErr = errors.New("missing namespace, the import ID should follow the pattern <id>@<namespace>")
	missingIDImportErr        = errors.New("missing resource ID, the import ID should follow the pattern <id>@<namespace>")
)

// NamespacedImporterContext imports a namespaced resource that doesn't have
// its namespace as part of the Terraform resource ID.
func NamespacedImporterContext(_ context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	namespacedID := d.Id()
	sepIdx := strings.LastIndex(namespacedID, "@")
	if sepIdx == -1 {
		return nil, missingNamespaceImportErr
	}

	ns := namespacedID[sepIdx+1:]
	if len(ns) == 0 {
		return nil, missingNamespaceImportErr
	}

	id := namespacedID[:sepIdx]
	if len(id) == 0 {
		return nil, missingIDImportErr
	}

	d.SetId(id)
	d.Set("namespace", ns)

	return []*schema.ResourceData{d}, nil
}
