package datasource

import (
	"net"
	"testing"
)

func TestCoreOSVersion(t *testing.T) {
	ds := ForTest(t)

	version, err := ds.GetClusterVariable("coreos-version")

	if err != nil {
		t.Error("error when try get coreos version:", err)
	}

	if version != "1068.2.0" {
		t.Error("invalid coreos version")
	}

}

func TestMachine(t *testing.T) {
	ds := ForTest(t)

	if err := ds.WhileMaster(); err != nil {
		t.Error("failed to register as the master instance:", err)
	}

	mac1, _ := net.ParseMAC("FF:FF:FF:FF:FF:FF")
	mac2, _ := net.ParseMAC("FF:FF:FF:FF:FF:FE")

	machine1, err := ds.MachineInterface(mac1).Machine(true, nil)
	if err != nil {
		t.Error("error in creating first machine:", err)
	}

	machine2, err := ds.MachineInterface(mac2).Machine(true, nil)
	if err != nil {
		t.Error("error in creating second machine:", err)
	}

	if machine1.IP == nil {
		t.Error("unexpected nil value for machine1.IP")
	}

	if machine2.IP == nil {
		t.Error("unexpected nil value for machine2.IP")
	}

	if machine1.IP.Equal(machine2.IP) {
		t.Error("two machines got same IP address:", machine1.IP.String())
	}

	machine3, err := ds.MachineInterface(mac2).Machine(true, nil)
	if err != nil {
		t.Error("error in creating third machine:", err)
	}

	if !machine2.IP.Equal(machine3.IP) {
		t.Error("same MAC address got two different IPs:", machine2.IP.String(), machine3.IP.String())
	}

	if err := ds.Shutdown(); err != nil {
		t.Error("failed to shutdown:", err)
	}
}
