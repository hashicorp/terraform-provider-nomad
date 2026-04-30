resource "nomad_sentinel_policy" "exec-only" {
  name        = "exec-only"
  description = "Only allow jobs that are based on an exec driver."

  policy = <<EOT
main = rule { all_drivers_exec }

# all_drivers_exec checks that all the drivers in use are exec
all_drivers_exec = rule {
    all job.task_groups as tg {
        all tg.tasks as task {
            task.driver is "exec"
        }
    }
}
EOT

  scope = "submit-job"

  # allow administrators to override
  enforcement_level = "soft-mandatory"
}
