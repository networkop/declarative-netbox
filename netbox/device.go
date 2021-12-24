package netbox

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/netbox-community/go-netbox/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/netbox/models"
	netboxv1 "github.com/networkop/declarative-netbox/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Device struct {
	Data *netboxv1.Device
	NetboxServer
}

type ids struct {
	Role int64
	Type int64
	Site int64
}

func NewDevice(s NetboxServer, d *netboxv1.Device) *Device {
	return &Device{
		Data:         d,
		NetboxServer: s,
	}
}

// Get retrieves Devices from Netbox
func (d *Device) Get(ctx context.Context) ([]netboxv1.Device, error) {
	log := logr.FromContext(ctx)
	results := []netboxv1.Device{}

	devices, err := d.Client.Dcim.DcimDevicesList(&dcim.DcimDevicesListParams{
		Name:    &d.Data.Name,
		Context: ctx,
	}, nil)
	if err != nil {
		return results, fmt.Errorf("failed to DcimDevicesList, %+v", err)
	}
	log.V(1).Info("found devices", "count", devices.Payload.Count)

	for _, d := range devices.Payload.Results {
		results = append(results, netboxv1.Device{
			TypeMeta: metav1.TypeMeta{
				Kind:       netboxv1.DeviceKind,
				APIVersion: netboxv1.GroupVersion.String(),
			},

			ObjectMeta: metav1.ObjectMeta{
				Name: *d.Name,
			},
			Spec: netboxv1.DeviceSpec{
				Site:       *d.Site.Name,
				Role:       *d.DeviceRole.Name,
				DeviceType: *d.DeviceType.Model,
			},
			Status: netboxv1.DeviceStatus{
				ID:    &d.ID,
				State: netboxv1.DeviceReadyState,
			},
		})
	}

	return results, nil
}

// Apply creates or updates a device in Netbox
func (d *Device) Apply(ctx context.Context) error {
	_, found, err := d.exists(ctx)
	if err != nil {
		return err
	}

	if found {
		return d.update(ctx)
	}

	return d.create(ctx)
}

// Delete removes the device from Netbox
func (d *Device) Delete(ctx context.Context) error {
	log := logr.FromContext(ctx)

	// controller could've crashed before processing the delete event
	_, found, err := d.exists(ctx)
	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	response, err := d.Client.Dcim.DcimDevicesDelete(&dcim.DcimDevicesDeleteParams{
		ID:      *d.Data.Status.ID,
		Context: ctx,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to DcimDevicesDelete: %+v", err)
	}

	log.V(1).Info("deleted device", "name", d.Data.Name, "response", response)

	return nil
}

func (d *Device) Print(ctx context.Context) error {
	log := logr.FromContext(ctx)

	device, found, err := d.exists(ctx)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no device found")
	}
	log.V(1).Info("d.Print", "device", device)
	return nil
}

func (d *Device) create(ctx context.Context) error {
	log := logr.FromContext(ctx)

	IDs, err := d.resolveIDs(ctx)
	if err != nil {
		return err
	}

	createParams := &dcim.DcimDevicesCreateParams{
		Data: &models.WritableDeviceWithConfigContext{
			Name:       &d.Data.Name,
			DeviceRole: &IDs.Role,
			DeviceType: &IDs.Type,
			Site:       &IDs.Site,
			Tags:       []*models.NestedTag{},
		},
		Context: ctx,
	}

	netboxDevice, err := d.Client.Dcim.DcimDevicesCreate(createParams, nil)
	if err != nil {
		return err
	}
	log.V(1).Info("created device", "response", netboxDevice)

	return d.setStatus(netboxDevice.GetPayload())
}

func (d *Device) setStatus(response *models.DeviceWithConfigContext) error {
	if response.ID == 0 {
		return fmt.Errorf("unexpected Device ID: 0")
	}

	d.Data.Status.ID = &response.ID
	d.Data.Status.State = netboxv1.DeviceReadyState
	return nil
}

func (d *Device) update(ctx context.Context) error {
	log := logr.FromContext(ctx)

	IDs, err := d.resolveIDs(ctx)
	if err != nil {
		return err
	}

	updateParams := &dcim.DcimDevicesUpdateParams{
		Data: &models.WritableDeviceWithConfigContext{
			Name:       &d.Data.Name,
			DeviceRole: &IDs.Role,
			DeviceType: &IDs.Type,
			Site:       &IDs.Site,
			Tags:       []*models.NestedTag{},
		},
		ID:      *d.Data.Status.ID,
		Context: ctx,
	}

	netboxDevice, err := d.Client.Dcim.DcimDevicesUpdate(updateParams, nil)
	if err != nil {
		return err
	}
	log.V(1).Info("updated device", "response", netboxDevice)

	return d.setStatus(netboxDevice.GetPayload())
}

func (d *Device) exists(ctx context.Context) (*models.DeviceWithConfigContext, bool, error) {
	log := logr.FromContext(ctx)

	devices, err := d.Client.Dcim.DcimDevicesList(&dcim.DcimDevicesListParams{
		Name:    &d.Data.Name,
		Context: ctx,
	}, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to DcimDevicesList, %+v", err)
	}

	if *devices.Payload.Count > 1 {
		log.Info("found more than one matching device")
		return nil, false, fmt.Errorf("%d matching devices found, cannot proceed", *devices.Payload.Count)
	}

	if *devices.Payload.Count == 0 {
		return nil, false, nil
	}

	log.V(1).Info("found exactly one device")
	return devices.Payload.Results[0], true, nil
}

func (d *Device) resolveIDs(ctx context.Context) (*ids, error) {
	log := logr.FromContext(ctx)

	roleID, err := d.resolveNameToID(ctx, d.Data.Spec.Role, "role")
	if err != nil {
		return nil, err
	}
	log.V(1).Info("found role", "roleID", roleID)

	typeID, err := d.resolveNameToID(ctx, d.Data.Spec.DeviceType, "type")
	if err != nil {
		return nil, err
	}
	log.V(1).Info("found type", "typeID", typeID)

	siteID, err := d.resolveNameToID(ctx, d.Data.Spec.Site, "site")
	if err != nil {
		return nil, err
	}
	log.V(1).Info("found site", "siteID", siteID)

	return &ids{
		Role: roleID,
		Type: typeID,
		Site: siteID,
	}, nil
}
