package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	netboxv1 "github.com/networkop/declarative-netbox/api/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDeviceResource(c *Cli) *Resource {

	resource := &Resource{
		Name: "device",
		Get: func() *cobra.Command {
			return DeviceGetCommand(c)
		},
	}

	return resource
}

func DeviceGetCommand(c *Cli) *cobra.Command {
	//var quiet bool
	var format string
	cmd := &cobra.Command{
		Use:     "device",
		Aliases: []string{"device", "devices"},
		Short:   "Get devices",
		RunE: func(cmd *cobra.Command, args []string) error {

			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			devices, err := c.netbox.Get(c.ctx, &netboxv1.Device{
				ObjectMeta: v1.ObjectMeta{
					Name: name,
				},
			})
			if err != nil {
				return err
			}
			devicePrintCommand(devices, format)
			return nil
		},
	}
	cmd.PersistentFlags().StringVarP(&format, "output", "o", "", strings.Join(allowedFormats(), "|"))
	//cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Only output IDs")
	return cmd
}

func devicePrintCommand(devices []netboxv1.Device, format string) {
	switch format {
	case "yaml":
		io.Copy(os.Stdout, devicePrintYaml(devices))
	case "json":
		io.Copy(os.Stdout, devicePrintJson(devices))
	default:
		devicePrintTable(devices)
	}
}

func devicePrintTable(devices []netboxv1.Device) {
	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"Name", "ID", "Type", "Role", "Site"})

	for _, d := range devices {
		tw.AppendRow(table.Row{d.Name, *d.Status.ID, d.Spec.DeviceType, d.Spec.Role, d.Spec.Site})
	}

	fmt.Println(tw.Render())
}

func devicePrintJson(devices []netboxv1.Device) io.Reader {
	var b bytes.Buffer
	jsonEncoder := json.NewEncoder(&b)
	jsonEncoder.SetIndent("", "  ")
	for _, d := range devices {
		if err := jsonEncoder.Encode(d); err != nil {
			log.Errorf("failed to encode json %s", err)
			continue
		}
	}
	return &b
}

func devicePrintYaml(devices []netboxv1.Device) io.Reader {
	var b bytes.Buffer

	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	for _, d := range devices {
		var jsonObj interface{}
		b, err := json.Marshal(d)
		if err != nil {
			log.Errorf("failed to marshal json %s", err)
			continue
		}

		if err := json.Unmarshal(b, &jsonObj); err != nil {
			log.Errorf("failed to unmarshal json %s", err)
			continue
		}
		if err := yamlEncoder.Encode(jsonObj); err != nil {
			log.Errorf("failed to encode yaml %s", err)
			continue
		}
	}

	return &b
}

func allowedFormats() []string {
	return []string{"json", "yaml"}
}
