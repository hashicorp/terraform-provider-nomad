resource "nomad_scheduler_config" "config" {
  scheduler_algorithm             = "spread"
  memory_oversubscription_enabled = true
  preemption_config = {
    system_scheduler_enabled   = true
    batch_scheduler_enabled    = true
    service_scheduler_enabled  = true
    sysbatch_scheduler_enabled = true
  }
}
