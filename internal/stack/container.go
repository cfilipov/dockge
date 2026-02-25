package stack

import (
	"strings"

	"github.com/cfilipov/dockge/internal/docker"
)

// ContainerSimpleJSON is the JSON representation of a container in the container list.
type ContainerSimpleJSON struct {
	Name                  string `json:"name"`
	ServiceName           string `json:"serviceName"`
	StackName             string `json:"stackName"`
	State                 string `json:"state"`
	Health                string `json:"health"`
	Image                 string `json:"image"`
	ImageUpdatesAvailable bool   `json:"imageUpdatesAvailable"`
	RecreateNecessary     bool   `json:"recreateNecessary"`
	IsManagedByDockge     bool   `json:"isManagedByDockge"`
}

// BuildContainerListJSON converts container data into the flat JSON array
// the frontend expects. Enriches each container with update/recreate status using
// O(1) map lookups (no additional Docker or registry calls).
//
// recreateMap is the stack-level recreate flag (stack → needs recreation).
// imagesByStack maps stackName → serviceName → image from compose YAML.
func BuildContainerListJSON(
	containersByProject map[string][]docker.Container,
	stacks map[string]*Stack,
	serviceUpdates map[string]bool,
	recreateMap map[string]bool,
	imagesByStack map[string]map[string]string,
) []ContainerSimpleJSON {
	// Count total containers for pre-allocation
	total := 0
	for _, cs := range containersByProject {
		total += len(cs)
	}

	result := make([]ContainerSimpleJSON, 0, total)

	for project, containers := range containersByProject {
		standalone := project == "_standalone"

		// Look up stack for isManagedByDockge (standalone containers are never managed)
		managed := false
		if !standalone {
			if s, ok := stacks[project]; ok {
				managed = s.IsManagedByDockge
			}
		}

		// Only fetch compose images when the stack-level recreate flag is set
		var composeImages map[string]string
		stackRecreate := recreateMap[project]
		if stackRecreate {
			composeImages = imagesByStack[project]
		}

		for _, c := range containers {
			stackName := project
			svc := c.Service
			if standalone {
				stackName = ""
				if svc == "" {
					svc = c.Name
				}
			} else if svc == "" {
				svc = extractServiceFromName(c.Name)
			}

			// Recreate check: only when stack-level flag says recreation is needed
			recreate := false
			if stackRecreate && !standalone {
				if compImg, ok := composeImages[svc]; ok && c.Image != "" && compImg != "" && c.Image != compImg {
					recreate = true
				}
			}

			// Image update check
			hasUpdate := false
			if serviceUpdates != nil && !standalone {
				hasUpdate = serviceUpdates[project+"/"+svc]
			}

			result = append(result, ContainerSimpleJSON{
				Name:                  c.Name,
				ServiceName:           svc,
				StackName:             stackName,
				State:                 strings.ToLower(c.State),
				Health:                strings.ToLower(c.Health),
				Image:                 c.Image,
				ImageUpdatesAvailable: hasUpdate,
				RecreateNecessary:     recreate,
				IsManagedByDockge:     managed,
			})
		}
	}

	return result
}
