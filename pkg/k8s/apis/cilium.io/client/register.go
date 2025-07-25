// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

package client

import (
	_ "embed"
	"fmt"
	"log/slog"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	k8sconst "github.com/cilium/cilium/pkg/k8s/apis/cilium.io"
	k8sconstv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	k8sconstv2alpha1 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2alpha1"
	"github.com/cilium/cilium/pkg/k8s/apis/crdhelpers"
	"github.com/cilium/cilium/pkg/k8s/client"
	"github.com/cilium/cilium/pkg/k8s/synced"
	"github.com/cilium/cilium/pkg/logging"
	"github.com/cilium/cilium/pkg/logging/logfields"
	"github.com/cilium/cilium/pkg/versioncheck"
)

const (
	// CNPCRDName is the full name of the CNP CRD.
	CNPCRDName = k8sconstv2.CNPKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CCNPCRDName is the full name of the CCNP CRD.
	CCNPCRDName = k8sconstv2.CCNPKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CEPCRDName is the full name of the CEP CRD.
	CEPCRDName = k8sconstv2.CEPKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CIDCRDName is the full name of the CID CRD.
	CIDCRDName = k8sconstv2.CIDKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CNCRDName is the full name of the CN CRD.
	CNCRDName = k8sconstv2.CNKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CLRPCRDName is the full name of the CLRP CRD.
	CLRPCRDName = k8sconstv2.CLRPKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CEGPCRDName is the full name of the CEGP CRD.
	CEGPCRDName = k8sconstv2.CEGPKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CESCRDName is the full name of the CES CRD.
	CESCRDName = k8sconstv2alpha1.CESKindDefinition + "/" + k8sconstv2alpha1.CustomResourceDefinitionVersion

	// CCECCRDName is the full name of the CCEC CRD.
	CCECCRDName = k8sconstv2.CCECKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CECCRDName is the full name of the CEC CRD.
	CECCRDName = k8sconstv2.CECKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CNCCRDName is the full name of the CiliumNodeConfig CRD.
	CNCCRDName = k8sconstv2.CNCKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// BGPPCRDName is the full name of the BGPP CRD.
	BGPPCRDName = k8sconstv2alpha1.BGPPKindDefinition + "/" + k8sconstv2alpha1.CustomResourceDefinitionVersion

	// BGPClusterConfigCRDName is the full name of the BGP Cluster Config CRD.
	BGPClusterConfigCRDName = k8sconstv2.BGPCCKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// BGPPeerConfigCRDName is the full name of the BGP PeerConfig CRD.
	BGPPeerConfigCRDName = k8sconstv2.BGPPCKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// BGPAdvertisementCRDName is the full name of the BGP Advertisement CRD.
	BGPAdvertisementCRDName = k8sconstv2.BGPAKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// BGPNodeConfigCRDName is the full name of the BGP Node Config CRD.
	BGPNodeConfigCRDName = k8sconstv2.BGPNCKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// BGPNodeConfigOverrideCRDName is the full name of the BGP Node Config Override CRD.
	BGPNodeConfigOverrideCRDName = k8sconstv2.BGPNCOKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// LBIPPoolCRDName is the full name of the BGPPool CRD.
	LBIPPoolCRDName = k8sconstv2.PoolKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// CNCCRDNameAlpha is the full name of the CiliumNodeConfig CRD.
	// TODO remove me when CNC CRD v2alpha1 will be deprecated.
	CNCCRDNameAlpha = k8sconstv2alpha1.CNCKindDefinition + "/" + k8sconstv2alpha1.CustomResourceDefinitionVersion

	// CCGCRDName is the full name of the CiliumCIDRGroup CRD.
	CCGCRDName = k8sconstv2.CCGKindDefinition + "/" + k8sconstv2.CustomResourceDefinitionVersion

	// L2AnnouncementCRDName is the full name of the CiliumL2AnnouncementPolicy CRD.
	L2AnnouncementCRDName = k8sconstv2alpha1.L2AnnouncementKindDefinition + "/" + k8sconstv2alpha1.CustomResourceDefinitionVersion

	// CPIPCRDName is the full name of the CiliumPodIPPool CRD.
	CPIPCRDName = k8sconstv2alpha1.CPIPKindDefinition + "/" + k8sconstv2alpha1.CustomResourceDefinitionVersion
	// CGCCCRDName is the full name of the CiliumGatewayClassConfig CRD.
	CGCCCRDName = k8sconstv2alpha1.CGCCKindDefinition + "/" + k8sconstv2alpha1.CustomResourceDefinitionVersion
)

type CRDList struct {
	Name     string
	FullName string
}

// CustomResourceDefinitionList returns a map of CRDs
func CustomResourceDefinitionList() map[string]*CRDList {
	return map[string]*CRDList{
		synced.CRDResourceName(k8sconstv2.CNPName): {
			Name:     CNPCRDName,
			FullName: k8sconstv2.CNPName,
		},
		synced.CRDResourceName(k8sconstv2.CCNPName): {
			Name:     CCNPCRDName,
			FullName: k8sconstv2.CCNPName,
		},
		synced.CRDResourceName(k8sconstv2.CNName): {
			Name:     CNCRDName,
			FullName: k8sconstv2.CNName,
		},
		synced.CRDResourceName(k8sconstv2.CIDName): {
			Name:     CIDCRDName,
			FullName: k8sconstv2.CIDName,
		},
		synced.CRDResourceName(k8sconstv2.CEPName): {
			Name:     CEPCRDName,
			FullName: k8sconstv2.CEPName,
		},
		synced.CRDResourceName(k8sconstv2.CLRPName): {
			Name:     CLRPCRDName,
			FullName: k8sconstv2.CLRPName,
		},
		synced.CRDResourceName(k8sconstv2.CEGPName): {
			Name:     CEGPCRDName,
			FullName: k8sconstv2.CEGPName,
		},
		synced.CRDResourceName(k8sconstv2.CNCName): {
			Name:     CNCCRDName,
			FullName: k8sconstv2.CNCName,
		},
		synced.CRDResourceName(k8sconstv2alpha1.CESName): {
			Name:     CESCRDName,
			FullName: k8sconstv2alpha1.CESName,
		},
		synced.CRDResourceName(k8sconstv2.CCECName): {
			Name:     CCECCRDName,
			FullName: k8sconstv2.CCECName,
		},
		synced.CRDResourceName(k8sconstv2.CECName): {
			Name:     CECCRDName,
			FullName: k8sconstv2.CECName,
		},
		synced.CRDResourceName(k8sconstv2alpha1.BGPPName): {
			Name:     BGPPCRDName,
			FullName: k8sconstv2alpha1.BGPPName,
		},
		synced.CRDResourceName(k8sconstv2.BGPCCName): {
			Name:     BGPClusterConfigCRDName,
			FullName: k8sconstv2.BGPCCName,
		},
		synced.CRDResourceName(k8sconstv2.BGPPCName): {
			Name:     BGPPeerConfigCRDName,
			FullName: k8sconstv2.BGPPCName,
		},
		synced.CRDResourceName(k8sconstv2.BGPAName): {
			Name:     BGPAdvertisementCRDName,
			FullName: k8sconstv2.BGPAName,
		},
		synced.CRDResourceName(k8sconstv2.BGPNCName): {
			Name:     BGPNodeConfigCRDName,
			FullName: k8sconstv2.BGPNCName,
		},
		synced.CRDResourceName(k8sconstv2.BGPNCOName): {
			Name:     BGPNodeConfigOverrideCRDName,
			FullName: k8sconstv2.BGPNCOName,
		},
		synced.CRDResourceName(k8sconstv2.LBIPPoolName): {
			Name:     LBIPPoolCRDName,
			FullName: k8sconstv2.LBIPPoolName,
		},
		synced.CRDResourceName(k8sconstv2.CCGName): {
			Name:     CCGCRDName,
			FullName: k8sconstv2.CCGName,
		},
		synced.CRDResourceName(k8sconstv2alpha1.L2AnnouncementName): {
			Name:     L2AnnouncementCRDName,
			FullName: k8sconstv2alpha1.L2AnnouncementName,
		},
		synced.CRDResourceName(k8sconstv2alpha1.CPIPName): {
			Name:     CPIPCRDName,
			FullName: k8sconstv2alpha1.CPIPName,
		},
		synced.CRDResourceName(k8sconstv2alpha1.CGCCName): {
			Name:     CGCCCRDName,
			FullName: k8sconstv2alpha1.CGCCName,
		},
	}
}

// CreateCustomResourceDefinitions creates our CRD objects in the Kubernetes
// cluster.
func CreateCustomResourceDefinitions(logger *slog.Logger, clientset apiextensionsclient.Interface) error {
	crds := CustomResourceDefinitionList()

	for _, r := range synced.AllCiliumCRDResourceNames() {
		if crd, ok := crds[r]; ok {
			if err := createCRD(logger, crd.Name, crd.FullName)(clientset); err != nil {
				return err
			}
		} else {
			logging.Fatal(logger, fmt.Sprintf("Unknown resource %s. Please update pkg/k8s/apis/cilium.io/client to understand this type.", r))
		}
	}

	return nil
}

var (
	//go:embed crds/v2/ciliumnetworkpolicies.yaml
	crdsCiliumnetworkpolicies []byte

	//go:embed crds/v2/ciliumclusterwidenetworkpolicies.yaml
	crdsCiliumclusterwidenetworkpolicies []byte

	//go:embed crds/v2/ciliumendpoints.yaml
	crdsCiliumendpoints []byte

	//go:embed crds/v2/ciliumidentities.yaml
	crdsCiliumidentities []byte

	//go:embed crds/v2/ciliumnodes.yaml
	crdsCiliumnodes []byte

	//go:embed crds/v2/ciliumlocalredirectpolicies.yaml
	crdsCiliumlocalredirectpolicies []byte

	//go:embed crds/v2/ciliumegressgatewaypolicies.yaml
	crdsv2Ciliumegressgatewaypolicies []byte

	//go:embed crds/v2/ciliumnodeconfigs.yaml
	crdsv2CiliumNodeConfigs []byte

	//go:embed crds/v2alpha1/ciliumendpointslices.yaml
	crdsv2Alpha1Ciliumendpointslices []byte

	//go:embed crds/v2/ciliumclusterwideenvoyconfigs.yaml
	crdsv2Ciliumclusterwideenvoyconfigs []byte

	//go:embed crds/v2/ciliumenvoyconfigs.yaml
	crdsv2Ciliumenvoyconfigs []byte

	//go:embed crds/v2alpha1/ciliumbgppeeringpolicies.yaml
	crdsv2Alpha1Ciliumbgppeeringpolicies []byte

	//go:embed crds/v2/ciliumbgpclusterconfigs.yaml
	crdsv2Ciliumbgpclusterconfigs []byte

	//go:embed crds/v2/ciliumbgppeerconfigs.yaml
	crdsv2Ciliumbgppeerconfigs []byte

	//go:embed crds/v2/ciliumbgpadvertisements.yaml
	crdsv2Ciliumbgpadvertisements []byte

	//go:embed crds/v2/ciliumbgpnodeconfigs.yaml
	crdsv2Ciliumbgpnodeconfigs []byte

	//go:embed crds/v2/ciliumbgpnodeconfigoverrides.yaml
	crdsv2Ciliumbgpnodeconfigoverrides []byte

	//go:embed crds/v2/ciliumloadbalancerippools.yaml
	crdsv2Ciliumloadbalancerippools []byte

	//go:embed crds/v2/ciliumcidrgroups.yaml
	crdsv2CiliumCIDRGroups []byte

	//go:embed crds/v2alpha1/ciliuml2announcementpolicies.yaml
	crdsv2Alpha1CiliumL2AnnouncementPolicies []byte

	//go:embed crds/v2alpha1/ciliumpodippools.yaml
	crdsv2Alpha1CiliumPodIPPools []byte

	//go:embed crds/v2alpha1/ciliumgatewayclassconfigs.yaml
	crdsv2Alpha1CiliumGatewayClassConfigs []byte
)

// GetPregeneratedCRD returns the pregenerated CRD based on the requested CRD
// name. The pregenerated CRDs are generated by the controller-gen tool and
// serialized into binary form by go-bindata. This function retrieves CRDs from
// the binary form.
func GetPregeneratedCRD(logger *slog.Logger, crdName string) apiextensionsv1.CustomResourceDefinition {
	var (
		err      error
		crdBytes []byte
	)

	logAttr := slog.String("crdName", crdName)

	switch crdName {
	case CNPCRDName:
		crdBytes = crdsCiliumnetworkpolicies
	case CCNPCRDName:
		crdBytes = crdsCiliumclusterwidenetworkpolicies
	case CEPCRDName:
		crdBytes = crdsCiliumendpoints
	case CIDCRDName:
		crdBytes = crdsCiliumidentities
	case CNCRDName:
		crdBytes = crdsCiliumnodes
	case CLRPCRDName:
		crdBytes = crdsCiliumlocalredirectpolicies
	case CEGPCRDName:
		crdBytes = crdsv2Ciliumegressgatewaypolicies
	case CESCRDName:
		crdBytes = crdsv2Alpha1Ciliumendpointslices
	case CCECCRDName:
		crdBytes = crdsv2Ciliumclusterwideenvoyconfigs
	case CECCRDName:
		crdBytes = crdsv2Ciliumenvoyconfigs
	case BGPPCRDName:
		crdBytes = crdsv2Alpha1Ciliumbgppeeringpolicies
	case BGPClusterConfigCRDName:
		crdBytes = crdsv2Ciliumbgpclusterconfigs
	case BGPPeerConfigCRDName:
		crdBytes = crdsv2Ciliumbgppeerconfigs
	case BGPAdvertisementCRDName:
		crdBytes = crdsv2Ciliumbgpadvertisements
	case BGPNodeConfigCRDName:
		crdBytes = crdsv2Ciliumbgpnodeconfigs
	case BGPNodeConfigOverrideCRDName:
		crdBytes = crdsv2Ciliumbgpnodeconfigoverrides
	case LBIPPoolCRDName:
		crdBytes = crdsv2Ciliumloadbalancerippools
	case CNCCRDName:
		// Contains both v2 and v2alpha1 versions
		crdBytes = crdsv2CiliumNodeConfigs
	case CCGCRDName:
		crdBytes = crdsv2CiliumCIDRGroups
	case L2AnnouncementCRDName:
		crdBytes = crdsv2Alpha1CiliumL2AnnouncementPolicies
	case CPIPCRDName:
		crdBytes = crdsv2Alpha1CiliumPodIPPools
	case CGCCCRDName:
		crdBytes = crdsv2Alpha1CiliumGatewayClassConfigs
	default:
		logging.Fatal(logger, "Pregenerated CRD does not exist", logAttr)
	}

	ciliumCRD := apiextensionsv1.CustomResourceDefinition{}
	err = yaml.Unmarshal(crdBytes, &ciliumCRD)
	if err != nil {
		logging.Fatal(
			logger,
			"Error unmarshalling pregenerated CRD",
			slog.Any(logfields.Error, err),
			logAttr,
		)
	}

	return ciliumCRD
}

// createCRD creates and updates a CRD.
// It should be called on agent startup but is idempotent and safe to call again.
func createCRD(logger *slog.Logger, crdVersionedName string, crdMetaName string) func(clientset apiextensionsclient.Interface) error {
	return func(clientset apiextensionsclient.Interface) error {
		ciliumCRD := GetPregeneratedCRD(logger, crdVersionedName)

		return crdhelpers.CreateUpdateCRD(
			logger,
			clientset,
			constructV1CRD(crdMetaName, ciliumCRD),
			crdhelpers.NewDefaultPoller(),
			k8sconst.CustomResourceDefinitionSchemaVersionKey,
			versioncheck.MustVersion(k8sconst.CustomResourceDefinitionSchemaVersion),
		)
	}
}

func constructV1CRD(
	name string,
	template apiextensionsv1.CustomResourceDefinition,
) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				k8sconst.CustomResourceDefinitionSchemaVersionKey: k8sconst.CustomResourceDefinitionSchemaVersion,
			},
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: k8sconst.CustomResourceDefinitionGroup,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:       template.Spec.Names.Kind,
				Plural:     template.Spec.Names.Plural,
				ShortNames: template.Spec.Names.ShortNames,
				Singular:   template.Spec.Names.Singular,
				Categories: template.Spec.Names.Categories,
			},
			Scope:      template.Spec.Scope,
			Versions:   template.Spec.Versions,
			Conversion: template.Spec.Conversion, // conversion strategy is needed to support several versions of a same CRD
		},
	}
}

// RegisterCRDs registers all CRDs with the K8s apiserver.
func RegisterCRDs(logger *slog.Logger, clientset client.Clientset) error {
	if err := CreateCustomResourceDefinitions(logger, clientset); err != nil {
		return fmt.Errorf("Unable to create custom resource definition: %w", err)
	}

	return nil
}
