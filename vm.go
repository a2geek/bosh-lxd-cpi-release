package main

import (
	"fmt"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/lxc/lxd/shared/api"
)

func (c CPI) CreateVM(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID,
	cloudProps apiv1.VMCloudProps, networks apiv1.Networks,
	associatedDiskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (apiv1.VMCID, error) {

	return apiv1.NewVMCID("vm-cid"), nil
}

func (c CPI) CreateVMV2(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID,
	cloudProps apiv1.VMCloudProps, networks apiv1.Networks,
	associatedDiskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (apiv1.VMCID, apiv1.Networks, error) {

	return apiv1.NewVMCID("vm-cid"), networks, nil
}

func (c CPI) DeleteVM(cid apiv1.VMCID) error {
	return nil
}

func (c CPI) CalculateVMCloudProperties(res apiv1.VMResources) (apiv1.VMCloudProps, error) {
	props := make(map[string]interface{})
	props["instance_type"] = fmt.Sprintf("c%d-m%d", res.CPU, res.RAM/1024)
	props["ephemeral_disk"] = res.EphemeralDiskSize
	return apiv1.NewVMCloudPropsFromMap(props), nil
}

func (c CPI) SetVMMetadata(cid apiv1.VMCID, metadata apiv1.VMMeta) error {
	return nil
}

func (c CPI) HasVM(cid apiv1.VMCID) (bool, error) {
	_, _, err := c.client.GetContainer(cid.AsString())
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c CPI) RebootVM(cid apiv1.VMCID) error {
	req := api.ContainerStatePut{
		Action:   "restart",
		Timeout:  30,
		Force:    true,
		Stateful: false,
	}

	op, err := c.client.UpdateContainerState(cid.AsString(), req, "")
	if err != nil {
		return bosherr.WrapError(err, "reboot vm")
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "reboot vm")
	}

	return nil
}
