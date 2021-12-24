package netbox

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/go-logr/logr"
	"github.com/netbox-community/go-netbox/netbox"
	netboxClient "github.com/netbox-community/go-netbox/netbox/client"
	"github.com/netbox-community/go-netbox/netbox/client/dcim"
	netboxv1 "github.com/networkop/declarative-netbox/api/v1"
	"github.com/sirupsen/logrus"
)

type Action string

const (
	SetAction   Action = "set"
	UnsetAction Action = "unset"
)

type NetboxServer struct {
	Client *netboxClient.NetBoxAPI
}

func NewNetboxServer(url, token string) *NetboxServer {
	return &NetboxServer{
		Client: netbox.NewNetboxWithAPIKey(url, token),
	}
}

func (s *NetboxServer) Apply(ctx context.Context, object interface{}) error {
	log := logr.FromContext(ctx)

	switch o := object.(type) {
	case *netboxv1.Device:
		log.V(1).Info("identified type: Device")
		return NewDevice(*s, o).Apply(ctx)
	default:
		log.V(1).Info("identified type: default")
		log.Error(fmt.Errorf("unknown object type"), fmt.Sprintf("%T", o))
	}
	return nil
}

func (s *NetboxServer) Delete(ctx context.Context, object interface{}) error {
	log := logr.FromContext(ctx)

	switch o := object.(type) {
	case *netboxv1.Device:
		log.V(1).Info("identified type: Device")
		return NewDevice(*s, o).Delete(ctx)
	default:
		log.V(1).Info("identified type: device")
		log.Error(fmt.Errorf("unknown object type"), fmt.Sprintf("%T", o))
	}
	return nil
}

func (s *NetboxServer) Get(ctx context.Context, object interface{}) ([]netboxv1.Device, error) {
	log := logr.FromContext(ctx)

	switch o := object.(type) {
	case *netboxv1.Device:
		log.V(1).Info("identified type: Device")
		return NewDevice(*s, o).Get(ctx)
	default:
		log.V(1).Info("identified type: default")
		log.Error(fmt.Errorf("unknown object type"), fmt.Sprintf("%T", o))
	}
	return []netboxv1.Device{}, nil
}

func (s *NetboxServer) resolveNameToID(ctx context.Context, name, t string) (int64, error) {

	switch t {
	case "role":
		roles, err := s.Client.Dcim.DcimDeviceRolesList(&dcim.DcimDeviceRolesListParams{
			Name:    &name,
			Context: ctx,
		}, nil)
		if err != nil {
			return -1, err
		}
		if *roles.GetPayload().Count != 1 {
			return -1, fmt.Errorf("unexpected number of roles %q found: %d", name, *roles.GetPayload().Count)
		}
		return roles.GetPayload().Results[0].ID, nil
	case "type":
		models, err := s.Client.Dcim.DcimDeviceTypesList(&dcim.DcimDeviceTypesListParams{
			Model:   &name,
			Context: ctx,
		}, nil)
		if err != nil {
			return -1, err
		}
		if *models.GetPayload().Count != 1 {
			return -1, fmt.Errorf("unexpected number of models %q found: %d", name, *models.GetPayload().Count)
		}
		return models.GetPayload().Results[0].ID, nil
	case "site":
		sites, err := s.Client.Dcim.DcimSitesList(&dcim.DcimSitesListParams{
			Name:    &name,
			Context: ctx,
		}, nil)
		if err != nil {
			return -1, err
		}
		if *sites.GetPayload().Count != 1 {
			return -1, fmt.Errorf("unexpected number of sites %q found: %d", name, *sites.GetPayload().Count)
		}
		return sites.GetPayload().Results[0].ID, nil
	default:
		return -1, fmt.Errorf("unexpected type %q", t)
	}
}

func AuthCheck(s, t string) error {
	httpC := &http.Client{
		Timeout: time.Second * 20,
	}

	checkURL, err := url.Parse(s)
	if err != nil {
		return err
	}
	checkURL.Path = path.Join(checkURL.Path, "/api/dcim/sites/")

	req, err := http.NewRequest("GET", checkURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Token  "+t)
	resp, err := httpC.Do(req)
	if err != nil {
		return err
	}

	logrus.Println(resp.Status)

	if resp.StatusCode == 403 {
		return fmt.Errorf("authentication failed")
	}

	return nil
}
