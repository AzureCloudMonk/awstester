package ec2

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"go.uber.org/zap"
)

const vpcCIDR = "192.168.0.0/16"

func (md *embedded) createVPC() (err error) {
	var output *ec2.CreateVpcOutput
	output, err = md.ec2.CreateVpc(&ec2.CreateVpcInput{
		CidrBlock: aws.String(vpcCIDR),
	})
	if err != nil {
		return err
	}
	md.cfg.VPCID = *output.Vpc.VpcId
	md.lg.Info(
		"created VPC",
		zap.String("vpc-id", md.cfg.VPCID),
	)

	if err = md.modifyVPC(); err != nil {
		return err
	}

	var op *ec2.CreateInternetGatewayOutput
	op, err = md.ec2.CreateInternetGateway(&ec2.CreateInternetGatewayInput{})
	if err != nil {
		return err
	}
	md.cfg.InternetGatewayID = *op.InternetGateway.InternetGatewayId
	md.lg.Info(
		"created internet gateway",
		zap.String("internet-gateway-id", md.cfg.InternetGatewayID),
	)
	_, err = md.ec2.AttachInternetGateway(&ec2.AttachInternetGatewayInput{
		VpcId:             output.Vpc.VpcId,
		InternetGatewayId: aws.String(md.cfg.InternetGatewayID),
	})
	if err != nil {
		return err
	}
	md.lg.Info(
		"attached internet gateway to VPC",
		zap.String("internet-gateway-id", md.cfg.InternetGatewayID),
		zap.String("vpc-id", md.cfg.VPCID),
	)

	var do *ec2.DescribeRouteTablesOutput
	do, err = md.ec2.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: aws.StringSlice([]string{md.cfg.VPCID}),
			},
		},
	})
	if err != nil {
		return err
	}
	for _, tb := range do.RouteTables {
		md.cfg.RouteTableIDs = append(md.cfg.RouteTableIDs, *tb.RouteTableId)
		_, err = md.ec2.CreateRoute(&ec2.CreateRouteInput{
			RouteTableId:         tb.RouteTableId,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			GatewayId:            aws.String(md.cfg.InternetGatewayID),
		})
		if err != nil {
			return err
		}
		md.lg.Info(
			"created a route for route table",
			zap.String("vpc-id", md.cfg.VPCID),
		)
	}

	return md.cfg.Sync()
}

func (md *embedded) modifyVPC() (err error) {
	_, err = md.ec2.ModifyVpcAttribute(&ec2.ModifyVpcAttributeInput{
		EnableDnsHostnames: &ec2.AttributeBooleanValue{Value: aws.Bool(true)},
		VpcId:              aws.String(md.cfg.VPCID),
	})
	if err != nil {
		return err
	}
	md.lg.Info(
		"modified VPC attribute",
		zap.String("vpc-id", md.cfg.VPCID),
	)
	return nil
}

func (md *embedded) deleteVPC() error {
	if md.cfg.VPCID == "" {
		return errors.New("cannot delete VPC without VPC ID")
	}

	_, err := md.ec2.DetachInternetGateway(&ec2.DetachInternetGatewayInput{
		VpcId:             aws.String(md.cfg.VPCID),
		InternetGatewayId: aws.String(md.cfg.InternetGatewayID),
	})
	if err != nil {
		return err
	}
	md.lg.Info(
		"detached internet gateway to VPC",
		zap.String("internet-gateway-id", md.cfg.InternetGatewayID),
		zap.String("vpc-id", md.cfg.VPCID),
	)
	_, err = md.ec2.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(md.cfg.InternetGatewayID),
	})
	if err != nil {
		return err
	}
	_, err = md.ec2.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: aws.String(md.cfg.VPCID),
	})
	if err != nil {
		awsErr, ok := err.(awserr.Error)
		if ok && awsErr.Code() == "InvalidVpcID.NotFound" {
			md.lg.Info(
				"VPC does not exist",
				zap.String("vpc-id", md.cfg.VPCID),
			)
			return nil
		}
	}
	md.lg.Info(
		"deleted internet gateway",
		zap.String("internet-gateway-id", md.cfg.InternetGatewayID),
	)

	time.Sleep(2 * time.Second)

	for _, id := range md.cfg.RouteTableIDs {
		_, err = md.ec2.DeleteRouteTable(&ec2.DeleteRouteTableInput{
			RouteTableId: aws.String(id),
		})
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "InvalidRouteTableID.NotFound" {
				md.lg.Info(
					"route table is already deleted",
					zap.String("route-table-id", id),
				)
				continue
			}
			return err
		}
	}
	md.lg.Info(
		"deleted route table",
		zap.Strings("route-table-ids", md.cfg.RouteTableIDs),
	)

	_, err = md.ec2.DescribeVpcs(&ec2.DescribeVpcsInput{
		VpcIds: aws.StringSlice([]string{md.cfg.VPCID}),
	})
	if err != nil {
		awsErr, ok := err.(awserr.Error)
		if ok && awsErr.Code() == "InvalidVpcID.NotFound" {
			md.lg.Info(
				"deleted VPC",
				zap.String("vpc-id", md.cfg.VPCID),
			)
			return nil
		}
		return err
	}
	return fmt.Errorf("deleted VPC but %q still exists", md.cfg.KeyName)
}