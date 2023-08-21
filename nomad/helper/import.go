package helper

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func NamespacedImporterContext(_ context.Context, d *schema.ResourceData, a any) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), " ", 2)

	var ns, jobID string
	switch len(parts) {
	case 1:
		jobID = parts[0]
	case 2:
		ns = parts[0]
		jobID = parts[1]
	}

	d.Set("namespace", ns)
	d.SetId(jobID)

	return []*schema.ResourceData{d}, nil
}
