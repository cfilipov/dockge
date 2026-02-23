package stack

import (
    "strings"

    "github.com/cfilipov/dockge/backend-go/internal/compose"
    "github.com/cfilipov/dockge/backend-go/internal/docker"
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

// BuildContainerListJSON converts cached container data into the flat JSON array
// the frontend expects. Enriches each container with update/recreate status using
// O(1) map lookups (no additional Docker or registry calls).
func BuildContainerListJSON(
    containersByProject map[string][]docker.Container,
    stacks map[string]*Stack,
    serviceUpdates map[string]bool,
    composeCache *compose.ComposeCache,
) []ContainerSimpleJSON {
    // Count total containers for pre-allocation
    total := 0
    for _, cs := range containersByProject {
        total += len(cs)
    }

    result := make([]ContainerSimpleJSON, 0, total)

    for project, containers := range containersByProject {
        // Look up stack for isManagedByDockge
        managed := false
        if s, ok := stacks[project]; ok {
            managed = s.IsManagedByDockge
        }

        // Get compose images for recreate comparison
        composeImages := composeCache.GetImages(project)

        for _, c := range containers {
            svc := c.Service
            if svc == "" {
                svc = extractServiceFromName(c.Name)
            }

            // Recreate check: running image vs compose.yaml image
            recreate := false
            if compImg, ok := composeImages[svc]; ok && c.Image != "" && compImg != "" && c.Image != compImg {
                recreate = true
            }

            // Image update check
            hasUpdate := false
            if serviceUpdates != nil {
                hasUpdate = serviceUpdates[project+"/"+svc]
            }

            result = append(result, ContainerSimpleJSON{
                Name:                  c.Name,
                ServiceName:           svc,
                StackName:             project,
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
