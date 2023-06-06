// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceACLBindingRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceACLBindingRuleCreate,
		Update: resourceACLBindingRuleUpdate,
		Delete: resourceACLBindingRuleDelete,
		Read:   resourceACLBindingRuleRead,
		Exists: resourceACLBindingRuleExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Description: "Description for this ACL binding rule.",
				Optional:    true,
				Type:        schema.TypeString,
			},
			"auth_method": {
				Description: "Name of the auth method for which this rule applies to.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"selector": {
				Description: "A boolean expression that matches against verified identity attributes returned from the auth method during login.",
				Optional:    true,
				Type:        schema.TypeString,
			},
			"bind_type": {
				Description: `Adjusts how this binding rule is applied at login time. Valid values are "role" and "policy".`,
				Required:    true,
				Type:        schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					api.ACLBindingRuleBindTypeManagement,
					api.ACLBindingRuleBindTypePolicy,
					api.ACLBindingRuleBindTypeRole,
				}, false),
			},
			"bind_name": {
				Description: "Target of the binding.",
				Optional:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceACLBindingRuleCreate(d *schema.ResourceData, meta interface{}) error {
	err := validateNomadACLBindingRule(d)
	if err != nil {
		return err
	}

	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	aclBindingRule := generateNomadACLBindingRule(d)

	// Create our ACL Binding Rule.
	log.Print("[DEBUG] Creating ACL Binding Rule")
	aclBindingRuleCreateResp, _, err := client.ACLBindingRules().Create(aclBindingRule, nil)
	if err != nil {
		return fmt.Errorf("error creating ACL Binding Rule: %s", err.Error())
	}
	log.Printf("[DEBUG] Created ACL Binding Rule %q", aclBindingRuleCreateResp.ID)
	d.SetId(aclBindingRuleCreateResp.ID)

	return resourceACLBindingRuleRead(d, meta)
}

func resourceACLBindingRuleDelete(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	bindingRuleID := d.Id()

	// Delete the ACL binding rule.
	log.Printf("[DEBUG] Deleting ACL Binding Rule %q", bindingRuleID)
	_, err := client.ACLBindingRules().Delete(bindingRuleID, nil)
	if err != nil {
		return fmt.Errorf("error deleting ACL Binding Rule %q: %s", bindingRuleID, err.Error())
	}
	log.Printf("[DEBUG] Deleted ACL Binding Rule %q", bindingRuleID)

	d.SetId("")

	return nil
}

func resourceACLBindingRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	err := validateNomadACLBindingRule(d)
	if err != nil {
		return err
	}

	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	aclBindingRule := generateNomadACLBindingRule(d)

	// Perform the in-place update of the ACL binding rule.
	log.Printf("[DEBUG] Updating ACL Binding Rule %q", aclBindingRule.ID)
	_, _, err = client.ACLBindingRules().Update(aclBindingRule, nil)
	if err != nil {
		return fmt.Errorf("error updating ACL Binding Rule %q: %s", aclBindingRule.ID, err.Error())
	}
	log.Printf("[DEBUG] Updated ACL Binding Rule %q", aclBindingRule.ID)

	return resourceACLBindingRuleRead(d, meta)
}

func resourceACLBindingRuleRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	bindingRuleID := d.Id()

	// If the rule has not been created, the ID will be an empty string which
	// means we can skip attempting to perform the lookup.
	if bindingRuleID == "" {
		return nil
	}

	log.Printf("[DEBUG] Reading ACL Binding Rule %q", bindingRuleID)
	bindingRule, _, err := client.ACLBindingRules().Get(bindingRuleID, nil)
	if err != nil {
		return fmt.Errorf("error reading ACL Binding Rule %q: %s", bindingRuleID, err.Error())
	}
	log.Printf("[DEBUG] Read ACL Binding Rule %q", bindingRule.ID)

	_ = d.Set("description", bindingRule.Description)
	_ = d.Set("auth_method", bindingRule.AuthMethod)
	_ = d.Set("selector", bindingRule.Selector)
	_ = d.Set("bind_type", bindingRule.BindType)
	_ = d.Set("bind_name", bindingRule.BindName)

	return nil
}

func resourceACLBindingRuleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	bindingRuleID := d.Id()

	log.Printf("[DEBUG] Checking if ACL Binding Rule %q exists", bindingRuleID)
	_, _, err := client.ACLBindingRules().Get(bindingRuleID, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return true, fmt.Errorf("error checking for ACL Binding Rule %q: %#v", bindingRuleID, err)
	}

	return true, nil
}

func generateNomadACLBindingRule(d *schema.ResourceData) *api.ACLBindingRule {
	return &api.ACLBindingRule{
		ID:          d.Id(),
		Description: d.Get("description").(string),
		AuthMethod:  d.Get("auth_method").(string),
		Selector:    d.Get("selector").(string),
		BindType:    d.Get("bind_type").(string),
		BindName:    d.Get("bind_name").(string),
	}
}

func validateNomadACLBindingRule(d *schema.ResourceData) error {
	bindName := d.Get("bind_name").(string)
	bindType := d.Get("bind_type").(string)

	switch bindType {
	case api.ACLBindingRuleBindTypeManagement:
		if bindName != "" {
			return fmt.Errorf("error bind_name must not be defined if bind_type is '%q'", api.ACLBindingRuleBindTypeManagement)
		}
	default:
		if bindName == "" {
			return fmt.Errorf("error bind_name must be defined if bind_type is '%q'", bindType)
		}
	}

	return nil
}
