/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package awstasks

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"k8s.io/klog/v2"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/awsup"
	"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
	"k8s.io/kops/upup/pkg/fi/cloudup/terraformWriter"
)

// DNSZone is a zone object in a dns provider
// +kops:fitask
type DNSZone struct {
	Name      *string
	Lifecycle fi.Lifecycle

	DNSName *string
	ZoneID  *string

	Private    *bool
	PrivateVPC *VPC
}

var _ fi.CompareWithID = &DNSZone{}

func (e *DNSZone) CompareWithID() *string {
	return e.Name
}

func (e *DNSZone) Find(c *fi.CloudupContext) (*DNSZone, error) {
	cloud := c.T.Cloud.(awsup.AWSCloud)

	z, err := e.findExisting(c.Context(), cloud)
	if err != nil {
		return nil, err
	}

	if z == nil {
		return nil, nil
	}

	actual := &DNSZone{}
	actual.Name = e.Name
	if z.HostedZone.Name != nil {
		actual.DNSName = fi.PtrTo(strings.TrimSuffix(*z.HostedZone.Name, "."))
	}
	if z.HostedZone.Id != nil {
		actual.ZoneID = fi.PtrTo(strings.TrimPrefix(*z.HostedZone.Id, "/hostedzone/"))
	}
	actual.Private = aws.Bool(z.HostedZone.Config.PrivateZone)

	// If the zone is private, but we don't want it to be, that will be an error
	// e.PrivateVPC won't be set, so we can't find the "right" VPC (without cheating)
	if e.PrivateVPC != nil {
		for _, vpc := range z.VPCs {
			if cloud.Region() != string(vpc.VPCRegion) {
				continue
			}

			if aws.ToString(e.PrivateVPC.ID) == aws.ToString(vpc.VPCId) {
				actual.PrivateVPC = e.PrivateVPC
			}
		}
	}

	if e.ZoneID == nil {
		e.ZoneID = actual.ZoneID
	}
	if e.DNSName == nil {
		e.DNSName = actual.DNSName
	}

	// Avoid spurious changes
	actual.Lifecycle = e.Lifecycle

	return actual, nil
}

func (e *DNSZone) findExisting(ctx context.Context, cloud awsup.AWSCloud) (*route53.GetHostedZoneOutput, error) {
	findID := ""
	if e.ZoneID != nil {
		request := &route53.GetHostedZoneInput{
			Id: e.ZoneID,
		}

		response, err := cloud.Route53().GetHostedZone(ctx, request)
		if err != nil {
			if awsup.AWSErrorCode(err) == "NoSuchHostedZone" {
				return nil, nil
			} else {
				return nil, fmt.Errorf("error fetching DNS HostedZone %q: %v", findID, err)
			}
		} else {
			return response, nil
		}
	}

	findName := fi.ValueOf(e.DNSName)
	if findName == "" {
		return nil, nil
	}
	if !strings.HasSuffix(findName, ".") {
		findName += "."
	}
	request := &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(findName),
	}

	response, err := cloud.Route53().ListHostedZonesByName(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error listing DNS HostedZones: %v", err)
	}

	var zones []route53types.HostedZone
	for _, zone := range response.HostedZones {
		if aws.ToString(zone.Name) == findName && zone.Config.PrivateZone == fi.ValueOf(e.Private) {
			zones = append(zones, zone)
		}
	}

	if len(zones) == 0 {
		return nil, nil
	} else if len(zones) != 1 {
		return nil, fmt.Errorf("found multiple hosted zones matched name %q", findName)
	} else {
		request := &route53.GetHostedZoneInput{
			Id: zones[0].Id,
		}

		response, err := cloud.Route53().GetHostedZone(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("error fetching DNS HostedZone by id %q: %v", *request.Id, err)
		}

		return response, nil
	}
}

func (e *DNSZone) Run(c *fi.CloudupContext) error {
	return fi.CloudupDefaultDeltaRunMethod(e, c)
}

func (s *DNSZone) CheckChanges(a, e, changes *DNSZone) error {
	if fi.ValueOf(e.Name) == "" {
		return fi.RequiredField("Name")
	}
	return nil
}

func (_ *DNSZone) RenderAWS(t *awsup.AWSAPITarget, a, e, changes *DNSZone) error {
	ctx := context.TODO()
	name := aws.ToString(e.DNSName)
	if a == nil {
		request := &route53.CreateHostedZoneInput{}
		request.Name = e.DNSName
		nonce := rand.Int63()
		request.CallerReference = aws.String(strconv.FormatInt(nonce, 10))

		if e.PrivateVPC != nil {
			request.VPC = &route53types.VPC{
				VPCId:     e.PrivateVPC.ID,
				VPCRegion: route53types.VPCRegion(t.Cloud.Region()),
			}
		}

		klog.V(2).Infof("Creating Route53 HostedZone with Name %q", name)

		response, err := t.Cloud.Route53().CreateHostedZone(ctx, request)
		if err != nil {
			return fmt.Errorf("error creating DNS HostedZone %q: %v", name, err)
		}

		e.ZoneID = response.HostedZone.Id
	} else {
		if changes.PrivateVPC != nil {
			request := &route53.AssociateVPCWithHostedZoneInput{
				HostedZoneId: a.ZoneID,
				VPC: &route53types.VPC{
					VPCId:     e.PrivateVPC.ID,
					VPCRegion: route53types.VPCRegion(t.Cloud.Region()),
				},
			}

			changes.PrivateVPC = nil

			klog.V(2).Infof("Updating DNSZone %q", name)

			_, err := t.Cloud.Route53().AssociateVPCWithHostedZone(ctx, request)
			if err != nil {
				return fmt.Errorf("error associating VPC with hosted zone %q: %v", name, err)
			}
		}

		empty := &DNSZone{}
		if !reflect.DeepEqual(empty, changes) {
			klog.Warningf("cannot apply changes to DNSZone %q: %v", name, changes)
		}
	}

	// We don't tag the zone - we expect it to be shared
	return nil
}

type terraformRoute53ZoneAssociation struct {
	ZoneID    *terraformWriter.Literal `cty:"zone_id"`
	VPCID     *terraformWriter.Literal `cty:"vpc_id"`
	Lifecycle *terraform.Lifecycle     `cty:"lifecycle"`
}

func (_ *DNSZone) RenderTerraform(t *terraform.TerraformTarget, a, e, changes *DNSZone) error {
	ctx := context.TODO()
	cloud := t.Cloud.(awsup.AWSCloud)

	dnsName := fi.ValueOf(e.DNSName)

	// As a special case, we check for an existing zone
	// It is really painful to have TF create a new one...
	// (you have to reconfigure the DNS NS records)
	klog.Infof("Check for existing route53 zone to re-use with name %q", dnsName)
	z, err := e.findExisting(ctx, cloud)
	if err != nil {
		return err
	}

	if z != nil {
		klog.Infof("Existing zone %q found; will configure TF to reuse", aws.ToString(z.HostedZone.Name))

		e.ZoneID = z.HostedZone.Id

		// If the user specifies dns=private we'll have a non-nil PrivateVPC that specifies the VPC
		// that should used with the private Route53 zone. If the zone doesn't already know about the
		// VPC, we add that association.
		if e.PrivateVPC != nil {
			assocNeeded := true
			var vpcName string
			if e.PrivateVPC.ID != nil {
				vpcName = *e.PrivateVPC.ID
				for _, vpc := range z.VPCs {
					if *vpc.VPCId == vpcName {
						klog.Infof("VPC %q already associated with zone %q", vpcName, aws.ToString(z.HostedZone.Name))
						assocNeeded = false
					}
				}
			} else {
				vpcName = *e.PrivateVPC.Name
			}

			if assocNeeded {
				klog.Infof("No association between VPC %q and zone %q; adding", vpcName, aws.ToString(z.HostedZone.Name))
				tf := &terraformRoute53ZoneAssociation{
					ZoneID: terraformWriter.LiteralFromStringValue(*e.ZoneID),
					VPCID:  e.PrivateVPC.TerraformLink(),
				}
				return t.RenderResource("aws_route53_zone_association", *e.Name, tf)
			}
		}

		return nil
	}

	// Because we expect most users to create their zones externally,
	// we now block hostedzone creation in terraform.
	// This lets us perform deeper DNS validation, but also solves the problem
	// that otherwise we don't know if TF created the hosted zone
	// (in which case we should output it) or whether it already existed (in which case we should not)
	// The root problem here is that TF doesn't have a strong notion of an unmanaged resource
	return fmt.Errorf("Creation of Route53 hosted zones is not supported for terraform")
}

func (e *DNSZone) TerraformLink() *terraformWriter.Literal {
	if e.ZoneID != nil {
		klog.V(4).Infof("reusing existing route53 zone with id %q", *e.ZoneID)
		return terraformWriter.LiteralFromStringValue(*e.ZoneID)
	}

	return terraformWriter.LiteralSelfLink("aws_route53_zone", *e.Name)
}
