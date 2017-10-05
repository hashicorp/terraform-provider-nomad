package nomad

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceJobRegister,
		Update: resourceJobRegister,
		Delete: resourceJobDeregister,
		Read:   resourceJobRead,
		Exists: resourceJobExists,

		Schema: map[string]*schema.Schema{
			"jobspec": {
				Description:      "Job specification. If you want to point to a file use the file() function.",
				Required:         true,
				Type:             schema.TypeString,
				DiffSuppressFunc: jobspecDiffSuppress,
			},

			"deregister_on_destroy": {
				Description: "If true, the job will be deregistered on destroy.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},

			"deregister_on_id_change": {
				Description: "If true, the job will be deregistered when the job ID changes.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},
		},
	}
}

func resourceJobRegister(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	// Get the jobspec itself
	jobspecRaw := d.Get("jobspec").(string)

	// Parse it
	job, err := jobspec.Parse(strings.NewReader(jobspecRaw))
	if err != nil {
		return fmt.Errorf("error parsing jobspec: %s", err)
	}

	// Inject the Vault token if provided
	if providerConfig.vaultToken != nil {
		job.VaultToken = providerConfig.vaultToken
	}

	// Initialize
	job.Canonicalize()

	// If we have an ID and its not equal to this jobspec, then we
	// have to deregister the old job before we register the new job.
	prevId := d.Id()
	if !d.Get("deregister_on_id_change").(bool) {
		// If we aren't deregistering on ID change, just pretend we
		// don't have a prior ID.
		prevId = ""
	}

	if prevId != "" && prevId != *job.ID {
		log.Printf(
			"[INFO] Deregistering %q before registering %q",
			prevId, job.ID)

		log.Printf("[DEBUG] Deregistering job: %q", prevId)
		_, _, err := client.Jobs().Deregister(prevId, false, nil)
		if err != nil {
			return fmt.Errorf(
				"error deregistering previous job %q "+
					"before registering new job %q: %s",
				prevId, job.ID, err)
		}

		// Success! Clear our state.
		d.SetId("")
	}

	// Register the job
	_, _, err = client.Jobs().Register(job, nil)
	if err != nil {
		return fmt.Errorf("error applying jobspec: %s", err)
	}

	d.SetId(*job.ID)

	return nil
}

func resourceJobDeregister(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	// If deregistration is disabled, then do nothing
	if !d.Get("deregister_on_destroy").(bool) {
		log.Printf(
			"[WARN] Job %q will not deregister since 'deregister_on_destroy'"+
				" is false", d.Id())
		return nil
	}

	id := d.Id()
	log.Printf("[DEBUG] Deregistering job: %q", id)
	_, _, err := client.Jobs().Deregister(id, false, nil)
	if err != nil {
		return fmt.Errorf("error deregistering job: %s", err)
	}

	return nil
}

func resourceJobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Id()
	log.Printf("[DEBUG] Checking if job exists: %q", id)
	_, _, err := client.Jobs().Info(id, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return true, fmt.Errorf("error checking for job: %#v", err)
	}

	return true, nil
}

func resourceJobRead(d *schema.ResourceData, meta interface{}) error {
	// We don't do anything at the moment. Exists is used to
	// remove non-existent jobs but read doesn't have to do anything.
	return nil
}

// jobspecDiffSuppress is the DiffSuppressFunc used by the schema to
// check if two jobspecs are equal.
func jobspecDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	// Parse the old job
	oldJob, err := jobspec.Parse(strings.NewReader(old))
	if err != nil {
		return false
	}

	// Parse the new job
	newJob, err := jobspec.Parse(strings.NewReader(new))
	if err != nil {
		return false
	}

	// Init
	oldJob.Canonicalize()
	newJob.Canonicalize()

	// Check for jobspec equality
	return reflect.DeepEqual(oldJob, newJob)
}
