package agent.tools

# Default deny
default allow = false

# Allow k8s_get_resources in default namespace
allow {
    input.tool == "k8s_get_resources"
    input.namespace == "default"
}

# Allow k8s_get_logs in default namespace
allow {
    input.tool == "k8s_get_logs"
    input.namespace == "default"
}

# Allow creating PipelineRuns in default namespace
allow {
    input.tool == "tekton_create_pipelinerun"
    input.namespace == "default"
}

# Deny creating PipelineRuns with unsafe names
deny[msg] {
    input.tool == "tekton_create_pipelinerun"
    contains(input.name, "..")
    msg := "pipelinerun name cannot contain '..'"
}

# Deny creating PipelineRuns without required fields
deny[msg] {
    input.tool == "tekton_create_pipelinerun"
    not input.pipelineName
    msg := "pipelineName is required"
}
