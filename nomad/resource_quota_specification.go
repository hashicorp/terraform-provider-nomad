// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceQuotaSpecification() *schema.Resource {
	return &schema.Resource{
		Create: resourceQuotaSpecificationWrite,
		Update: resourceQuotaSpecificationWrite,
		Delete: resourceQuotaSpecificationDelete,
		Read:   resourceQuotaSpecificationRead,
		Exists: resourceQuotaSpecificationExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Description: "Manages a quota specification in a Nomad cluster.",

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Unique name for this quota specification.",
				Required:    true,
				Type:        schema.TypeString,
				ForceNew:    true,
			},

			"description": {
				Description: "Description for this quota specification.",
				Optional:    true,
				Type:        schema.TypeString,
			},

			"limits": {
				Description: "Limits encapsulated by this quota specification.",
				Required:    true,
				Type:        schema.TypeSet,
				Elem:        resourceQuotaSpecificationLimits(),
			},
		},
	}
}

func resourceQuotaSpecificationLimits() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"region": {
				Description: "Region in which this limit has affect.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"region_limit": {
				Required:    true,
				Description: "The limit applied to this region.",
				Type:        schema.TypeSet,
				MaxItems:    1,
				Elem:        resourceQuotaSpecificationRegionLimits(),
			},
		},
	}
}

func resourceQuotaSpecificationRegionLimits() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"cpu": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cores": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"memory_mb": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"memory_max_mb": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"devices": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceQuotaSpecificationDevices(),
			},
			"node_pools": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceQuotaSpecificationNodePools(),
			},
			"storage": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem:     resourceQuotaSpecificationStorage(),
			},
		},
	}
}

func resourceQuotaSpecificationDevices() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"count": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

func resourceQuotaSpecificationStorage() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"variables_mb": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"host_volumes_mb": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

func resourceQuotaSpecificationNodePools() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"node_pool": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cpu": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cores": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"memory_mb": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"memory_max_mb": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"devices": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceQuotaSpecificationDevices(),
			},
			"storage": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem:     resourceQuotaSpecificationStorage(),
			},
		},
	}
}

func resourceQuotaSpecificationWrite(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client

	spec := api.QuotaSpec{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}
	limits, err := expandQuotaLimits(d)
	if err != nil {
		return err
	}
	spec.Limits = limits

	log.Printf("[DEBUG] Upserting quota specification %q", spec.Name)
	_, err = client.Quotas().Register(&spec, nil)
	if err != nil {
		return fmt.Errorf("error upserting quota specification %q: %s", spec.Name, err.Error())
	}
	log.Printf("[DEBUG] Upserted quota specification %q", spec.Name)
	d.SetId(spec.Name)

	return resourceQuotaSpecificationRead(d, meta)
}

func resourceQuotaSpecificationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client
	name := d.Id()

	// check if quota is attached to a namespace
	nss, _, err := client.Namespaces().List(nil)
	if err != nil {
		return err
	}
	for _, ns := range nss {
		if ns.Quota == name {
			// dissasociate the quota with its ns
			ns.Quota = ""
			_, err := client.Namespaces().Register(ns, nil)
			if err != nil {
				return fmt.Errorf(
					"error disassociating quota spec %q with namespace %s: %w",
					name, ns.Name, err,
				)
			}
		}
	}

	// delete the quota spec
	log.Printf("[DEBUG] Deleting quota specification %q", name)
	_, err = client.Quotas().Delete(name, nil)
	if err != nil {
		return fmt.Errorf("error deleting quota specification %q: %s", name, err.Error())
	}
	log.Printf("[DEBUG] Deleted quota specification %q", name)

	return nil
}

func resourceQuotaSpecificationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client
	name := d.Id()

	// retrieve the policy
	log.Printf("[DEBUG] Reading quota specification %q", name)
	spec, _, err := client.Quotas().Info(name, nil)
	if err != nil {
		// we have Exists, so no need to handle 404
		return fmt.Errorf("error reading quota specification %q: %s", name, err.Error())
	}
	log.Printf("[DEBUG] Read quota specification %q", name)

	d.Set("name", spec.Name)
	d.Set("description", spec.Description)
	err = d.Set("limits", flattenQuotaLimits(spec.Limits))
	if err != nil {
		return fmt.Errorf("error setting quota specification limits for %q: %s", name, err.Error())
	}

	return nil
}

func resourceQuotaSpecificationExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(ProviderConfig).client

	name := d.Id()
	log.Printf("[DEBUG] Checking if quota specification %q exists", name)
	resp, _, err := client.Quotas().Info(name, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return true, fmt.Errorf("error checking for quota specification %q: %#v", name, err)
	}
	// just to be safe
	if resp == nil {
		log.Printf("[DEBUG] Response was nil, so assuming quota specification %q doesn't exist", name)
		return false, nil
	}

	return true, nil
}

func flattenQuotaLimits(limits []*api.QuotaLimit) *schema.Set {
	var results []interface{}
	for _, limit := range limits {
		res := map[string]interface{}{
			"region":       limit.Region,
			"region_limit": flattenQuotaRegionLimit(limit.RegionLimit),
		}
		results = append(results, res)
	}
	return schema.NewSet(schema.HashResource(resourceQuotaSpecificationLimits()), results)
}

func flattenQuotaRegionLimit(limit *api.QuotaResources) *schema.Set {
	if limit == nil {
		return nil
	}
	result := map[string]interface{}{}
	if limit.CPU != nil {
		result["cpu"] = *limit.CPU
	}
	if limit.Cores != nil {
		result["cores"] = *limit.Cores
	}
	if limit.MemoryMB != nil {
		result["memory_mb"] = *limit.MemoryMB
	}
	if limit.MemoryMaxMB != nil {
		result["memory_max_mb"] = *limit.MemoryMaxMB
	}
	if len(limit.Devices) > 0 {
		result["devices"] = flattenQuotaDevices(limit.Devices)
	}
	if len(limit.NodePools) > 0 {
		result["node_pools"] = flattenQuotaNodePools(limit.NodePools)
	}
	if limit.Storage != nil {
		result["storage"] = []interface{}{
			map[string]interface{}{
				"variables_mb":    limit.Storage.VariablesMB,
				"host_volumes_mb": limit.Storage.HostVolumesMB,
			},
		}
	}
	return schema.NewSet(schema.HashResource(resourceQuotaSpecificationRegionLimits()),
		[]interface{}{result})
}

func flattenQuotaDevices(devices []*api.RequestedDevice) []any {
	result := make([]any, 0, len(devices))
	for _, d := range devices {
		dev := map[string]interface{}{
			"name": d.Name,
		}
		if d.Count != nil {
			dev["count"] = int(*d.Count)
		}
		result = append(result, dev)
	}
	return result
}

func flattenQuotaNodePools(pools []*api.NodePoolLimit) []any {
	result := make([]any, 0, len(pools))
	for _, p := range pools {
		pool := map[string]interface{}{
			"node_pool": p.NodePool,
		}
		if p.CPU != nil {
			pool["cpu"] = *p.CPU
		}
		if p.Cores != nil {
			pool["cores"] = *p.Cores
		}
		if p.MemoryMB != nil {
			pool["memory_mb"] = *p.MemoryMB
		}
		if p.MemoryMaxMB != nil {
			pool["memory_max_mb"] = *p.MemoryMaxMB
		}
		if len(p.Devices) > 0 {
			pool["devices"] = flattenQuotaDevices(p.Devices)
		}
		if p.Storage != nil {
			pool["storage"] = []interface{}{
				map[string]interface{}{
					"variables_mb":    p.Storage.VariablesMB,
					"host_volumes_mb": p.Storage.HostVolumesMB,
				},
			}
		}
		result = append(result, pool)
	}
	return result
}

func expandQuotaLimits(d *schema.ResourceData) ([]*api.QuotaLimit, error) {
	configs := d.Get("limits")
	limits := configs.(*schema.Set).List()
	results := make([]*api.QuotaLimit, 0, len(limits))
	for _, lim := range limits {
		limit, ok := lim.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected limit to be a map[string]interface{}, got %T instead", lim)
		}
		region, ok := limit["region"].(string)
		if !ok {
			return nil, fmt.Errorf("expected region to be string, got %T instead", limit["region"])
		}
		res := &api.QuotaLimit{
			Region: region,
		}
		regLimit, err := expandRegionLimit(limit["region_limit"])
		if err != nil {
			return nil, fmt.Errorf("error parsing region limit for region %q: %s", limit["region"], err.Error())
		}
		res.RegionLimit = regLimit
		results = append(results, res)
	}
	return results, nil
}

func expandRegionLimit(limit interface{}) (*api.QuotaResources, error) {
	regLimits := limit.(*schema.Set).List()
	if len(regLimits) < 1 {
		return nil, nil
	}
	regLimit, ok := regLimits[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map[string]interface{} for region limit, got %T", regLimits[0])
	}
	var res api.QuotaResources
	if cpu, ok := regLimit["cpu"]; ok {
		c, ok := cpu.(int)
		if !ok {
			return nil, fmt.Errorf("expected CPU to be int, got %T", cpu)
		}
		res.CPU = &c
	}
	if cores, ok := regLimit["cores"]; ok {
		c, ok := cores.(int)
		if !ok {
			return nil, fmt.Errorf("expected cores to be int, got %T", cores)
		}
		res.Cores = &c
	}
	if mem, ok := regLimit["memory_mb"]; ok {
		m, ok := mem.(int)
		if !ok {
			return nil, fmt.Errorf("expected memory to be int, got %T", mem)
		}
		res.MemoryMB = &m
	}
	if memMax, ok := regLimit["memory_max_mb"]; ok {
		m, ok := memMax.(int)
		if !ok {
			return nil, fmt.Errorf("expected memory_max to be int, got %T", memMax)
		}
		res.MemoryMaxMB = &m
	}
	if devices, ok := regLimit["devices"]; ok {
		devList := devices.([]interface{})
		for _, d := range devList {
			dev := d.(map[string]interface{})
			rd := &api.RequestedDevice{
				Name: dev["name"].(string),
			}
			if count, ok := dev["count"]; ok {
				c := uint64(count.(int))
				rd.Count = &c
			}
			res.Devices = append(res.Devices, rd)
		}
	}
	if nodePools, ok := regLimit["node_pools"]; ok {
		poolList := nodePools.([]interface{})
		for _, p := range poolList {
			pool := p.(map[string]interface{})
			np := &api.NodePoolLimit{
				NodePool: pool["node_pool"].(string),
			}
			if cpu, ok := pool["cpu"]; ok {
				c, ok := cpu.(int)
				if !ok {
					return nil, fmt.Errorf("expected node_pool CPU to be int, got %T", cpu)
				}
				np.CPU = &c
			}
			if cores, ok := pool["cores"]; ok {
				c, ok := cores.(int)
				if !ok {
					return nil, fmt.Errorf("expected node_pool cores to be int, got %T", cores)
				}
				np.Cores = &c
			}
			if mem, ok := pool["memory_mb"]; ok {
				m, ok := mem.(int)
				if !ok {
					return nil, fmt.Errorf("expected node_pool memory to be int, got %T", mem)
				}
				np.MemoryMB = &m
			}
			if memMax, ok := pool["memory_max_mb"]; ok {
				m, ok := memMax.(int)
				if !ok {
					return nil, fmt.Errorf("expected node_pool memory_max to be int, got %T", memMax)
				}
				np.MemoryMaxMB = &m
			}
			if devices, ok := pool["devices"]; ok {
				devList := devices.([]interface{})
				for _, d := range devList {
					dev := d.(map[string]interface{})
					rd := &api.RequestedDevice{
						Name: dev["name"].(string),
					}
					if count, ok := dev["count"]; ok {
						c := uint64(count.(int))
						rd.Count = &c
					}
					np.Devices = append(np.Devices, rd)
				}
			}
			if storage, ok := pool["storage"]; ok {
				storageList := storage.([]interface{})
				if len(storageList) > 0 && storageList[0] != nil {
					storageMap := storageList[0].(map[string]interface{})
					np.Storage = &api.QuotaStorageResources{
						VariablesMB:   storageMap["variables_mb"].(int),
						HostVolumesMB: storageMap["host_volumes_mb"].(int),
					}
				}
			}
			res.NodePools = append(res.NodePools, np)
		}
	}
	if storage, ok := regLimit["storage"]; ok {
		storageList := storage.([]interface{})
		if len(storageList) > 0 && storageList[0] != nil {
			storageMap := storageList[0].(map[string]interface{})
			res.Storage = &api.QuotaStorageResources{
				VariablesMB:   storageMap["variables_mb"].(int),
				HostVolumesMB: storageMap["host_volumes_mb"].(int),
			}
		}
	}
	return &res, nil
}
