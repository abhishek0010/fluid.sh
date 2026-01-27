package llm

// GetTools returns the list of tools available to the LLM.
func GetTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: Function{
				Name:        "list_sandboxes",
				Description: "List all existing sandboxes with their state and IP addresses.",
				Parameters: ParameterSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "create_sandbox",
				Description: "Create a new sandbox VM by cloning from a source VM.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"source_vm": {
							Type:        "string",
							Description: "The name of the source VM to clone from (e.g., 'ubuntu-base').",
						},
						"name": {
							Type:        "string",
							Description: "Optional name for the sandbox. If not provided, one will be generated.",
						},
						"host": {
							Type:        "string",
							Description: "Optional target host name for multi-host setups.",
						},
					},
					Required: []string{"source_vm"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "destroy_sandbox",
				Description: "Completely destroy a sandbox VM and remove its storage.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox to destroy.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "run_command",
				Description: "Execute a shell command inside a sandbox via SSH.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox to run the command in.",
						},
						"command": {
							Type:        "string",
							Description: "The shell command to execute.",
						},
					},
					Required: []string{"sandbox_id", "command"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "start_sandbox",
				Description: "Start a stopped sandbox VM.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox to start.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "stop_sandbox",
				Description: "Stop a running sandbox VM.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox to stop.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "get_sandbox",
				Description: "Get detailed information about a specific sandbox.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "list_vms",
				Description: "List available host VMs (base images) that can be cloned to create sandboxes. Does not include sandboxes - use list_sandboxes for those.",
				Parameters: ParameterSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "create_snapshot",
				Description: "Create a snapshot of the current sandbox state.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox.",
						},
						"name": {
							Type:        "string",
							Description: "Optional name for the snapshot.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "create_playbook",
				Description: "Create a new Ansible playbook.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"name": {
							Type:        "string",
							Description: "Name of the playbook.",
						},
						"hosts": {
							Type:        "string",
							Description: "Target hosts (default: 'all').",
						},
						"become": {
							Type:        "boolean",
							Description: "Whether to use privilege escalation (sudo).",
						},
					},
					Required: []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "add_playbook_task",
				Description: "Add a task to an Ansible playbook.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"playbook_id": {
							Type:        "string",
							Description: "The ID of the playbook.",
						},
						"name": {
							Type:        "string",
							Description: "Name of the task.",
						},
						"module": {
							Type:        "string",
							Description: "Ansible module to use (e.g., 'apt', 'shell', 'copy').",
						},
						"params": {
							Type:        "object",
							Description: "Parameters for the Ansible module.",
						},
					},
					Required: []string{"playbook_id", "name", "module"},
				},
			},
		},
	}
}
