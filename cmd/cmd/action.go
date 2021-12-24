package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	netboxv1 "github.com/networkop/declarative-netbox/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type action string

const (
	ApplyAction  action = "apply"
	DeleteAction action = "delete"
)

func Action(c *Cli, a action, fn string) error {

	f, err := os.Open(fn)
	if os.IsNotExist(err) {
		return fmt.Errorf("the path %q does not exist", fn)
	}
	defer f.Close()

	scheme, err := netboxv1.SchemeBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to build scheme for netboxv1")
	}
	codecFactory := serializer.NewCodecFactory(scheme)
	decoder := codecFactory.UniversalDeserializer()

	d := yaml.NewYAMLOrJSONDecoder(f, 4096)
	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error parsing %s: %v", fn, err)
		}
		ext.Raw = bytes.TrimSpace(ext.Raw)

		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}

		obj, gvk, err := decoder.Decode(ext.Raw, nil, nil)
		if err != nil {
			return fmt.Errorf("unable to decode %q: %v", fn, err)
		}
		log.Debugf("Obj: %+v, GVK: %+v", obj, gvk)

		switch a {
		case ApplyAction:
			if err := c.netbox.Apply(c.ctx, obj); err != nil {
				return fmt.Errorf("failed to apply netbox configuration: %s", err)
			}
		case DeleteAction:
			if err := c.netbox.Delete(c.ctx, obj); err != nil {
				return fmt.Errorf("failed to apply netbox configuration: %s", err)
			}
		default:
			return fmt.Errorf("unexpected action: %s", a)
		}

	}

}
