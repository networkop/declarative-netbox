package cmd

func GetResources(c *Cli) map[string]*Resource {
	resources := make(map[string]*Resource)

	resources["device"] = NewDeviceResource(c)

	return resources
}
