// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

package linuxrouting

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"testing"

	"github.com/cilium/hive/hivetest"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"

	"github.com/cilium/cilium/pkg/datapath/linux/linux_defaults"
	"github.com/cilium/cilium/pkg/datapath/linux/safenetlink"
	"github.com/cilium/cilium/pkg/testutils"
	"github.com/cilium/cilium/pkg/testutils/netns"
)

type MigrateSuite struct {
	// rpdb interface mock
	OnRuleList func(int) ([]netlink.Rule, error)
	OnRuleAdd  func(*netlink.Rule) error
	OnRuleDel  func(*netlink.Rule) error

	OnRouteListFiltered func(int, *netlink.Route, uint64) ([]netlink.Route, error)
	OnRouteAdd          func(*netlink.Route) error
	OnRouteDel          func(*netlink.Route) error
	OnRouteReplace      func(*netlink.Route) error

	OnLinkList    func() ([]netlink.Link, error)
	OnLinkByIndex func(int) (netlink.Link, error)

	// interfaceDB interface mock
	OnGetInterfaceNumberByMAC func(mac string) (int, error)
	OnGetMACByInterfaceNumber func(ifaceNum int) (string, error)
}

func setupMigrateSuite(tb testing.TB) *MigrateSuite {
	testutils.PrivilegedTest(tb)
	return &MigrateSuite{}
}

// n is the number of devices, routes, and rules that will be created in
// setUpRoutingTable() as fixtures for this test suite.
const n = 5

func TestPrivilegedMigrateENIDatapathUpgradeSuccess(t *testing.T) {
	m := setupMigrateSuite(t)
	logger := hivetest.Logger(t)
	// First, we need to setupMigrateSuite the Linux routing policy database to mimic a
	// broken setupMigrateSuite (1). Then we will call MigrateENIDatapath (2).

	// This test case will cover the successful path. We will create:
	//   - One rule with the old priority referencing the old table ID.
	//   - One route with the old table ID.
	// After we call MigrateENIDatapath(), we assert that:
	//   - The rule has switched to the new priority and references the new
	//     table ID.
	//   - The route has the new table ID.

	ns := netns.NewNetNS(t)
	ns.Do(func() error {
		// (1) Setting up the routing table.

		// Pick an arbitrary iface index. In the old table ID scheme, we used this
		// index as the table ID. All the old rules and routes will be set up with
		// this table ID.
		index := 5
		tableID := 11

		// (1) Setting up the routing table for testing upgrade.
		//
		// The reason we pass index twice is because we want to use the ifindex as
		// the table ID.
		devIfNumLookup, _ := setUpRoutingTable(t, index, index, linux_defaults.RulePriorityEgress)

		// Set up the rpdb mocks to just forward to netlink implementation.
		m.defaultNetlinkMock()

		// Set up the interfaceDB mock. We don't actually need to search by MAC
		// address in this test because we only have just one device. The actual
		// implementation will search the CiliumNode resource for the ENI device
		// matching.
		m.OnGetInterfaceNumberByMAC = func(mac string) (int, error) {
			// In setUpRoutingTable(), we used an arbitrary scheme that maps
			// each device created with an interface number of loop count (i)
			// plus one.
			return devIfNumLookup[mac], nil
		}

		// (2) Make the call to modifying the routing table.
		mig := migrator{logger: logger, rpdb: m, getter: m}
		migrated, failed := mig.MigrateENIDatapath(false)
		require.Equal(t, n, migrated)
		require.Equal(t, 0, failed)

		routes, err := safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: index,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Empty(t, routes) // We don't expect any routes with the old table ID.

		routes, err = safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: tableID,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Len(t, routes, 1) // We only expect one route that we created above in the setupMigrateSuite.
		require.NotEqual(t, index, routes[0].Table)

		rules, err := findRulesByPriority(linux_defaults.RulePriorityEgress)
		require.NoError(t, err)
		require.Empty(t, rules) // We don't expect any rules from old priority.

		rules, err = findRulesByPriority(linux_defaults.RulePriorityEgressv2)
		require.NoError(t, err)
		require.Len(t, rules, 5) // We expect all rules to be migrated to new priority.
		require.NotEqual(t, index, rules[0].Table)
		return nil
	})
}

func TestPrivilegedMigrateENIDatapathUpgradeFailure(t *testing.T) {
	logger := hivetest.Logger(t)
	// This test case will cover one failure path where we successfully migrate
	// all the old rules and routes, but fail to cleanup the old rule. This
	// test case will be set up identically to the successful case. After we
	// call MigrateENIDatapath(), we assert that we failed to migrate 1 rule.
	// We assert that the revert of the upgrade was successfully as well,
	// meaning we expect the old rules and routes to be reinstated.
	m := setupMigrateSuite(t)

	ns := netns.NewNetNS(t)
	ns.Do(func() error {
		index := 5
		devIfNumLookup, _ := setUpRoutingTable(t, index, index, linux_defaults.RulePriorityEgress)

		m.defaultNetlinkMock()

		// Here we inject the error on deleting a rule. The first call we want to
		// fail, but the second we want to succeed, because that will be the
		// revert.
		var onRuleDelCount int
		m.OnRuleDel = func(r *netlink.Rule) error {
			if onRuleDelCount == 0 {
				onRuleDelCount++
				return errors.New("fake error")
			}
			return netlink.RuleDel(r)
		}

		// Set up the interfaceDB mock. We don't actually need to search by MAC
		// address in this test because we only have just one device. The actual
		// implementation will search the CiliumNode resource for the ENI device
		// matching.
		m.OnGetInterfaceNumberByMAC = func(mac string) (int, error) {
			// In setUpRoutingTable(), we used an arbitrary scheme that maps
			// each device created with an interface number of loop count (i)
			// plus one.
			return devIfNumLookup[mac], nil
		}

		mig := migrator{logger: logger, rpdb: m, getter: m}
		migrated, failed := mig.MigrateENIDatapath(false)
		require.Equal(t, 4, migrated)
		require.Equal(t, 1, failed)

		tableID := 11
		routes, err := safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: index,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Len(t, routes, 1) // We expect old route to be untouched b/c we failed.
		require.Equal(t, index, routes[0].Table)

		routes, err = safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: tableID,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Empty(t, routes) // We don't expect any routes under new table ID b/c of revert.

		rules, err := findRulesByPriority(linux_defaults.RulePriorityEgress)
		require.NoError(t, err)
		require.Len(t, rules, 1) // We expect the old rule to be reinstated.
		require.Equal(t, index, rules[0].Table)

		rules, err = findRulesByPriority(linux_defaults.RulePriorityEgressv2)
		require.NoError(t, err)
		require.Len(t, rules, 4) // We expect the rest of the rules to be upgraded.
		return nil
	})
}

func TestPrivilegedMigrateENIDatapathDowngradeSuccess(t *testing.T) {
	// This test case will cover the successful downgrade path. We will create:
	//   - One rule with the new priority referencing the new table ID.
	//   - One route with the new table ID.
	// After we call MigrateENIDatapath(), we assert that:
	//   - The rule has switched to the old priority and references the old
	//     table ID.
	//   - The route has the old table ID.
	m := setupMigrateSuite(t)
	logger := hivetest.Logger(t)
	ns := netns.NewNetNS(t)
	ns.Do(func() error {
		// (1) Setting up the routing table.

		// Pick an arbitrary table ID. In the new table ID scheme, it is the
		// interface number + an offset of 10
		// (linux_defaults.RouteTableInterfacesOffset).
		//
		// Pick an ifindex and table ID.
		index := 5
		tableID := 11

		// (1) Setting up the routing table for testing downgrade, hence creating
		// rules with RulePriorityEgressv2.
		_, devMACLookup := setUpRoutingTable(t, index, tableID, linux_defaults.RulePriorityEgressv2)

		// Set up the rpdb mocks to just forward to netlink implementation.
		m.defaultNetlinkMock()

		// Set up the interfaceDB mock. The MAC address returned is coming from the
		// dummy ENI device we set up in setUpRoutingTable(). The actual
		// implementation will search the CiliumNode resource for the ENI device
		// matching.
		m.OnGetMACByInterfaceNumber = func(i int) (string, error) {
			// In setUpRoutingTable(), we used an arbitrary scheme for the
			// device name. It is simply the loop counter.
			return devMACLookup[fmt.Sprintf("gotestdummy%d", i)], nil
		}

		// (2) Make the call to modifying the routing table.
		mig := migrator{logger: logger, rpdb: m, getter: m}
		migrated, failed := mig.MigrateENIDatapath(true)
		require.Equal(t, n, migrated)
		require.Equal(t, 0, failed)

		routes, err := safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: tableID,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Empty(t, routes) // We don't expect any routes with the new table ID.

		routes, err = safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: index,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Len(t, routes, 1) // We only expect one route with the old table ID.
		require.NotEqual(t, tableID, routes[0].Table)

		rules, err := findRulesByPriority(linux_defaults.RulePriorityEgressv2)
		require.NoError(t, err)
		require.Empty(t, rules) // We don't expect any rules with this priority.

		rules, err = findRulesByPriority(linux_defaults.RulePriorityEgress)
		require.NoError(t, err)
		require.Len(t, rules, 5) // We expect all rules to have the original priority.
		require.NotEqual(t, tableID, rules[0].Table)
		return nil
	})
}

func TestPrivilegedMigrateENIDatapathDowngradeFailure(t *testing.T) {
	// This test case will cover one downgrade failure path where we failed to
	// migrate the rule to the old scheme. This test case will be set up
	// identically to the successful case. "New" meaning the rules and routes
	// using the new datapath scheme, hence downgrading. After we call
	// MigrateENIDatapath(), we assert that we failed to migrate 1 rule. We
	// assert that the revert of the downgrade was successfully as well,
	// meaning we expect the "newer" rules and routes to be reinstated.
	m := setupMigrateSuite(t)

	logger := hivetest.Logger(t)
	ns := netns.NewNetNS(t)
	ns.Do(func() error {
		index := 5
		tableID := 11
		_, devMACLookup := setUpRoutingTable(t, index, tableID, linux_defaults.RulePriorityEgressv2)

		m.defaultNetlinkMock()

		// Here we inject the error on adding a rule. The first call we want to
		// fail, but the second we want to succeed, because that will be the
		// revert.
		var onRuleAddCount int
		m.OnRuleAdd = func(r *netlink.Rule) error {
			if onRuleAddCount == 0 {
				onRuleAddCount++
				return errors.New("fake error")
			}
			return netlink.RuleAdd(r)
		}

		// Set up the interfaceDB mock. The MAC address returned is coming from the
		// dummy ENI device we set up in setUpRoutingTable().
		m.OnGetMACByInterfaceNumber = func(i int) (string, error) {
			// In setUpRoutingTable(), we used an arbitrary scheme for the
			// device name. It is simply the loop counter.
			return devMACLookup[fmt.Sprintf("gotestdummy%d", i)], nil
		}

		mig := migrator{logger: logger, rpdb: m, getter: m}
		migrated, failed := mig.MigrateENIDatapath(true)
		require.Equal(t, n-1, migrated) // One failed migration.
		require.Equal(t, 1, failed)

		routes, err := safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: tableID,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Len(t, routes, 1) // We expect "new" route to be untouched b/c we failed to delete.
		require.Equal(t, tableID, routes[0].Table)

		routes, err = safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: index,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Empty(t, routes) // We don't expect routes under original table ID b/c of revert.

		rules, err := findRulesByPriority(linux_defaults.RulePriorityEgressv2)
		require.NoError(t, err)
		require.Len(t, rules, 1) // We expect the "new" rule to be reinstated.
		require.Equal(t, tableID, rules[0].Table)

		rules, err = findRulesByPriority(linux_defaults.RulePriorityEgress)
		require.NoError(t, err)
		require.Len(t, rules, n-1) // Successfully migrated rules.
		return nil
	})
}

func TestPrivilegedMigrateENIDatapathPartial(t *testing.T) {
	// This test case will cover one case where we find a partial rule. It will
	// be set up with a rule with the newer priority and the user has indicated
	// compatbility=false, meaning they intend to upgrade. The fact that
	// there's already a rule with a newer priority indicates that a previous
	// migration has taken place and potentially failed. This simulates Cilium
	// starting up from a potentially failed previous migration.
	// After we call MigrateENIDatapath(), we assert that:
	//   - We still upgrade the remaining rules that need to be migrated.
	//   - We ignore the partially migrated rule.
	m := setupMigrateSuite(t)
	logger := hivetest.Logger(t)

	ns := netns.NewNetNS(t)
	ns.Do(func() error {
		index := 5
		// ifaceNumber := 1
		newTableID := 11

		devIfNumLookup, _ := setUpRoutingTable(t, index, index, linux_defaults.RulePriorityEgress)

		// Insert fake rule that has the newer priority to simulate it as
		// "partially migrated".
		err := exec.Command("ip", "rule", "add",
			"from", "10.1.0.0/24",
			"to", "all",
			"table", fmt.Sprintf("%d", newTableID),
			"priority", fmt.Sprintf("%d", linux_defaults.RulePriorityEgressv2)).Run()
		require.NoError(t, err)

		m.defaultNetlinkMock()

		m.OnGetInterfaceNumberByMAC = func(mac string) (int, error) {
			return devIfNumLookup[mac], nil
		}

		mig := migrator{logger: logger, rpdb: m, getter: m}
		migrated, failed := mig.MigrateENIDatapath(false)
		require.Equal(t, n, migrated)
		require.Equal(t, 0, failed)

		routes, err := safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: newTableID,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Len(t, routes, 1) // We expect one migrated route.
		require.Equal(t, newTableID, routes[0].Table)

		routes, err = safenetlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
			Table: index,
		}, netlink.RT_FILTER_TABLE)
		require.NoError(t, err)
		require.Empty(t, routes) // We don't expect any routes under old table ID.

		rules, err := findRulesByPriority(linux_defaults.RulePriorityEgressv2)
		require.NoError(t, err)
		require.Len(t, rules, n+1) // We expect all migrated rules and the partially migrated rule.
		require.Equal(t, newTableID, rules[0].Table)
		require.Equal(t, newTableID, rules[1].Table)

		rules, err = findRulesByPriority(linux_defaults.RulePriorityEgress)
		require.NoError(t, err)
		require.Empty(t, rules) // We don't expect any rules with the old priority.

		return nil
	})
}

// setUpRoutingTable initializes the routing table for this test suite. The
// starting ifindex, tableID, and the priority are passed in to give contron to
// the caller on the setupMigrateSuite. The two return values are:
//  1. Map of string to int, representing a mapping from MAC addrs to
//     interface numbers.
//  2. Map of string to string, representing a mapping from device name to MAC
//     addrs.
//
// (1) is used for the upgrade test cases where the GetInterfaceNumberByMAC
// mock is used. (2) is used for the downgrade test cases where the
// GetMACByInterfaceNumber mock is used. These maps are used in their
// respectives mocks to return the desired result data depending on the test.
func setUpRoutingTable(t *testing.T, ifindex, tableID, priority int) (map[string]int, map[string]string) {
	devIfNum := make(map[string]int)
	devMAC := make(map[string]string)

	// Create n sets of a dummy interface, a route, and a rule.
	//
	// Each dummy interface has a /24 from the private range of 172.16.0.0/20.
	//
	// Each route will be a default route to the gateway IP of the interface's
	// subnet.
	//
	// Each rule will be from the interface's subnet to all.
	for i := 1; i <= n; i++ {
		devName := fmt.Sprintf("gotestdummy%d", i)

		gw := net.ParseIP(fmt.Sprintf("172.16.%d.1", i))
		_, linkCIDR, err := net.ParseCIDR(fmt.Sprintf("172.16.%d.2/24", i))
		require.NoError(t, err)

		linkIndex := ifindex + (i - 1)
		newTableID := tableID + (i - 1)

		dummyTmpl := &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name:  devName,
				Index: linkIndex,
			},
		}
		require.NoError(t, netlink.LinkAdd(dummyTmpl))
		require.NoError(t, netlink.LinkSetUp(dummyTmpl))
		require.NoError(t, netlink.AddrAdd(dummyTmpl, &netlink.Addr{
			IPNet: linkCIDR,
		}))
		require.NoError(t, netlink.RouteAdd(&netlink.Route{
			Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
			Gw:        gw,
			LinkIndex: dummyTmpl.Index,
			Table:     newTableID,
		}))

		rule := netlink.NewRule()
		rule.Src = linkCIDR
		rule.Priority = priority
		rule.Table = newTableID
		require.NoError(t, netlink.RuleAdd(rule))

		// Return the MAC address of the dummy device, which acts as the ENI.
		link, err := safenetlink.LinkByName(devName)
		require.NoError(t, err)

		mac := link.Attrs().HardwareAddr.String()

		// Arbitrarily use an offset of 1 as the interface number. It doesn't
		// matter as long as we're consistent.
		devIfNum[mac] = i
		devMAC[devName] = mac
	}

	return devIfNum, devMAC
}

func findRulesByPriority(prio int) ([]netlink.Rule, error) {
	rules, err := safenetlink.RuleList(netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}

	return filterRulesByPriority(rules, prio), nil
}

func (m *MigrateSuite) defaultNetlinkMock() {
	m.OnRuleList = func(family int) ([]netlink.Rule, error) { return safenetlink.RuleList(family) }
	m.OnRuleAdd = func(rule *netlink.Rule) error { return netlink.RuleAdd(rule) }
	m.OnRuleDel = func(rule *netlink.Rule) error { return netlink.RuleDel(rule) }
	m.OnRouteListFiltered = func(family int, filter *netlink.Route, mask uint64) ([]netlink.Route, error) {
		return safenetlink.RouteListFiltered(family, filter, mask)
	}
	m.OnRouteAdd = func(route *netlink.Route) error { return netlink.RouteAdd(route) }
	m.OnRouteDel = func(route *netlink.Route) error { return netlink.RouteDel(route) }
	m.OnRouteReplace = func(route *netlink.Route) error { return netlink.RouteReplace(route) }
	m.OnLinkList = func() ([]netlink.Link, error) { return safenetlink.LinkList() }
	m.OnLinkByIndex = func(ifindex int) (netlink.Link, error) { return netlink.LinkByIndex(ifindex) }
}

func (m *MigrateSuite) RuleList(family int) ([]netlink.Rule, error) {
	if m.OnRuleList != nil {
		return m.OnRuleList(family)
	}
	panic("OnRuleList should not have been called")
}

func (m *MigrateSuite) RuleAdd(rule *netlink.Rule) error {
	if m.OnRuleAdd != nil {
		return m.OnRuleAdd(rule)
	}
	panic("OnRuleAdd should not have been called")
}

func (m *MigrateSuite) RuleDel(rule *netlink.Rule) error {
	if m.OnRuleDel != nil {
		return m.OnRuleDel(rule)
	}
	panic("OnRuleDel should not have been called")
}

func (m *MigrateSuite) RouteListFiltered(family int, filter *netlink.Route, mask uint64) ([]netlink.Route, error) {
	if m.OnRouteListFiltered != nil {
		return m.OnRouteListFiltered(family, filter, mask)
	}
	panic("OnRouteListFiltered should not have been called")
}

func (m *MigrateSuite) RouteAdd(route *netlink.Route) error {
	if m.OnRouteAdd != nil {
		return m.OnRouteAdd(route)
	}
	panic("OnRouteAdd should not have been called")
}

func (m *MigrateSuite) RouteDel(route *netlink.Route) error {
	if m.OnRouteDel != nil {
		return m.OnRouteDel(route)
	}
	panic("OnRouteDel should not have been called")
}

func (m *MigrateSuite) RouteReplace(route *netlink.Route) error {
	if m.OnRouteReplace != nil {
		return m.OnRouteReplace(route)
	}
	panic("OnRouteReplace should not have been called")
}

func (m *MigrateSuite) LinkList() ([]netlink.Link, error) {
	if m.OnLinkList != nil {
		return m.OnLinkList()
	}
	panic("OnLinkList should not have been called")
}

func (m *MigrateSuite) LinkByIndex(ifindex int) (netlink.Link, error) {
	if m.OnLinkByIndex != nil {
		return m.OnLinkByIndex(ifindex)
	}
	panic("OnLinkByIndex should not have been called")
}

func (m *MigrateSuite) GetInterfaceNumberByMAC(mac string) (int, error) {
	if m.OnGetInterfaceNumberByMAC != nil {
		return m.OnGetInterfaceNumberByMAC(mac)
	}
	panic("OnGetInterfaceNumberByMAC should not have been called")
}

func (m *MigrateSuite) GetMACByInterfaceNumber(ifaceNum int) (string, error) {
	if m.OnGetMACByInterfaceNumber != nil {
		return m.OnGetMACByInterfaceNumber(ifaceNum)
	}
	panic("OnGetMACByInterfaceNumber should not have been called")
}
