package compute

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Azure-Samples/azure-sdk-for-go-samples/helpers"
	"github.com/Azure-Samples/azure-sdk-for-go-samples/iam"
	"github.com/Azure-Samples/azure-sdk-for-go-samples/network"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-03-30/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	publisher = "Canonical"
	offer     = "UbuntuServer"
	sku       = "16.04.0-LTS"
)

var fakepubkey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIM97OXV9dTCZdZJkSk+W2r9XqmlbuGkZU1b1zyiAaXZq azureuser"

func getVMClient() (compute.VirtualMachinesClient, error) {
	token, _ := iam.GetResourceManagementToken(iam.OAuthGrantTypeServicePrincipal)
	vmClient := compute.NewVirtualMachinesClient(helpers.SubscriptionID())
	vmClient.Authorizer = autorest.NewBearerAuthorizer(token)
	return vmClient, nil
}

// CreateVM creates a new virtual machine with the specified name using the specified NIC.
// Username, password, and sshPublicKeyPath determine logon credentials.
func CreateVM(vmName, nicName, username, password, sshPublicKeyPath string) (<-chan compute.VirtualMachine, <-chan error) {
	nic, _ := network.GetNic(nicName)

	var sshKeyData string
	var err error
	if _, err = os.Stat(sshPublicKeyPath); err == nil {
		sshBytes, err := ioutil.ReadFile(sshPublicKeyPath)
		if err != nil {
			log.Fatalf("failed to read SSH key data: %v", err)
		}
		sshKeyData = string(sshBytes)
	} else {
		sshKeyData = fakepubkey
	}

	vmClient, _ := getVMClient()
	return vmClient.CreateOrUpdate(
		helpers.ResourceGroupName(),
		vmName,
		compute.VirtualMachine{
			Location: to.StringPtr(helpers.Location()),
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: compute.StandardDS1V2,
				},
				StorageProfile: &compute.StorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr(publisher),
						Offer:     to.StringPtr(offer),
						Sku:       to.StringPtr(sku),
						Version:   to.StringPtr("latest"),
					},
				},
				OsProfile: &compute.OSProfile{
					ComputerName:  to.StringPtr(vmName),
					AdminUsername: to.StringPtr(username),
					AdminPassword: to.StringPtr(password),
					LinuxConfiguration: &compute.LinuxConfiguration{
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								compute.SSHPublicKey{
									Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", username)),
									KeyData: to.StringPtr(sshKeyData),
								},
							},
						},
					},
				},
				NetworkProfile: &compute.NetworkProfile{
					NetworkInterfaces: &[]compute.NetworkInterfaceReference{
						compute.NetworkInterfaceReference{
							ID: nic.ID,
							NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
								Primary: to.BoolPtr(true),
							},
						},
					},
				},
			},
		},
		nil,
	)
}
