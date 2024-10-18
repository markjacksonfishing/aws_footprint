package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/cloudfront"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/ecr"
    "github.com/aws/aws-sdk-go-v2/service/ecs"
    "github.com/aws/aws-sdk-go-v2/service/eks"
    "github.com/aws/aws-sdk-go-v2/service/iam"
    "github.com/aws/aws-sdk-go-v2/service/lambda"
    "github.com/aws/aws-sdk-go-v2/service/rds"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/sns"
    "github.com/aws/aws-sdk-go-v2/service/sqs"
    "github.com/aws/aws-sdk-go-v2/service/sts"
    elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

func main() {
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("Enter AWS profile name: ")
    profile, _ := reader.ReadString('\n')
    profile = strings.TrimSpace(profile)

    // Load the AWS configuration using the specified profile
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithSharedConfigProfile(profile))
    if err != nil {
        log.Fatalf("Failed to load configuration, %v", err)
    }

    // Prompt for AWS region
    fmt.Print("Enter AWS region (e.g., us-east-1): ")
    region, _ := reader.ReadString('\n')
    region = strings.TrimSpace(region)

    // Set the AWS region in the configuration
    cfg.Region = region

    // Get the AWS Account ID
    stsClient := sts.NewFromConfig(cfg)
    identity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
    if err != nil {
        log.Fatalf("Failed to get AWS account ID, %v", err)
    }
    accountID := aws.ToString(identity.Account)

    // Create a file to save the AWS footprint
    fileName := fmt.Sprintf("aws_footprint_%s.txt", accountID)
    file, err := os.Create(fileName)
    if err != nil {
        log.Fatalf("Failed to create output file, %v", err)
    }
    defer file.Close()

    // Start writing to the file
    file.WriteString(fmt.Sprintf("AWS Account ID: %s\n", accountID))

    // Retrieve and write global resources
    collectGlobalResources(cfg, file)

    // Collect resources for the specified region
    file.WriteString(fmt.Sprintf("\nRegion: %s\n", cfg.Region))
    collectRegionalResources(cfg, file)

    fmt.Printf("AWS footprint has been saved to %s\n", fileName)
}

func collectGlobalResources(cfg aws.Config, file *os.File) {
    ctx := context.TODO()

    // S3 Buckets
    s3Client := s3.NewFromConfig(cfg)
    s3Result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
    if err != nil {
        log.Printf("Failed to list S3 buckets, %v", err)
    } else {
        file.WriteString("\nS3 Buckets:\n")
        for _, bucket := range s3Result.Buckets {
            file.WriteString(fmt.Sprintf("- %s\n", aws.ToString(bucket.Name)))
        }
    }

    // IAM Users
    iamClient := iam.NewFromConfig(cfg)
    file.WriteString("\nIAM Users:\n")
    iamUsersPaginator := iam.NewListUsersPaginator(iamClient, &iam.ListUsersInput{})
    for iamUsersPaginator.HasMorePages() {
        page, err := iamUsersPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list IAM users, %v", err)
            break
        }
        for _, user := range page.Users {
            file.WriteString(fmt.Sprintf("- UserName: %s\n", aws.ToString(user.UserName)))
        }
    }

    // IAM Roles
    file.WriteString("\nIAM Roles:\n")
    iamRolesPaginator := iam.NewListRolesPaginator(iamClient, &iam.ListRolesInput{})
    for iamRolesPaginator.HasMorePages() {
        page, err := iamRolesPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list IAM roles, %v", err)
            break
        }
        for _, role := range page.Roles {
            file.WriteString(fmt.Sprintf("- RoleName: %s\n", aws.ToString(role.RoleName)))
        }
    }

    // CloudFront Distributions
    cfClient := cloudfront.NewFromConfig(cfg)
    cfPaginator := cloudfront.NewListDistributionsPaginator(cfClient, &cloudfront.ListDistributionsInput{})
    file.WriteString("\nCloudFront Distributions:\n")
    for cfPaginator.HasMorePages() {
        page, err := cfPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list CloudFront distributions, %v", err)
            break
        }
        if page.DistributionList.Items != nil {
            for _, dist := range page.DistributionList.Items {
                file.WriteString(fmt.Sprintf("- Distribution ID: %s\n", aws.ToString(dist.Id)))
            }
        }
    }
}

func collectRegionalResources(cfg aws.Config, file *os.File) {
    ctx := context.TODO()

    // EC2 Client
    ec2Client := ec2.NewFromConfig(cfg)

    // EC2 Instances
    ec2Paginator := ec2.NewDescribeInstancesPaginator(ec2Client, &ec2.DescribeInstancesInput{})
    file.WriteString("\nEC2 Instances:\n")
    for ec2Paginator.HasMorePages() {
        page, err := ec2Paginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to describe EC2 instances in region %s, %v", cfg.Region, err)
            break
        }
        for _, reservation := range page.Reservations {
            for _, instance := range reservation.Instances {
                file.WriteString(fmt.Sprintf("- Instance ID: %s\n", aws.ToString(instance.InstanceId)))
            }
        }
    }

    // VPCs
    vpcResult, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
    if err != nil {
        log.Printf("Failed to describe VPCs in region %s, %v", cfg.Region, err)
    } else {
        file.WriteString("\nVPCs:\n")
        for _, vpc := range vpcResult.Vpcs {
            file.WriteString(fmt.Sprintf("- VPC ID: %s\n", aws.ToString(vpc.VpcId)))
        }
    }

    // Subnets
    subnetResult, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
    if err != nil {
        log.Printf("Failed to describe Subnets in region %s, %v", cfg.Region, err)
    } else {
        file.WriteString("\nSubnets:\n")
        for _, subnet := range subnetResult.Subnets {
            file.WriteString(fmt.Sprintf("- Subnet ID: %s\n", aws.ToString(subnet.SubnetId)))
        }
    }

    // Security Groups
    sgResult, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
    if err != nil {
        log.Printf("Failed to describe Security Groups in region %s, %v", cfg.Region, err)
    } else {
        file.WriteString("\nSecurity Groups:\n")
        for _, sg := range sgResult.SecurityGroups {
            file.WriteString(fmt.Sprintf("- Security Group ID: %s, Name: %s\n", aws.ToString(sg.GroupId), aws.ToString(sg.GroupName)))
        }
    }

    // Elastic Load Balancers (ELBv2)
    elbv2Client := elbv2.NewFromConfig(cfg)
    elbPaginator := elbv2.NewDescribeLoadBalancersPaginator(elbv2Client, &elbv2.DescribeLoadBalancersInput{})
    file.WriteString("\nLoad Balancers:\n")
    for elbPaginator.HasMorePages() {
        page, err := elbPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to describe Load Balancers in region %s, %v", cfg.Region, err)
            break
        }
        for _, lb := range page.LoadBalancers {
            file.WriteString(fmt.Sprintf("- Load Balancer Name: %s, Type: %s\n", aws.ToString(lb.LoadBalancerName), lb.Type))
        }
    }

    // RDS Instances
    rdsClient := rds.NewFromConfig(cfg)
    rdsPaginator := rds.NewDescribeDBInstancesPaginator(rdsClient, &rds.DescribeDBInstancesInput{})
    file.WriteString("\nRDS Instances:\n")
    for rdsPaginator.HasMorePages() {
        page, err := rdsPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to describe RDS instances in region %s, %v", cfg.Region, err)
            break
        }
        for _, dbInstance := range page.DBInstances {
            file.WriteString(fmt.Sprintf("- DB Instance Identifier: %s\n", aws.ToString(dbInstance.DBInstanceIdentifier)))
        }
    }

    // Lambda Functions
    lambdaClient := lambda.NewFromConfig(cfg)
    lambdaPaginator := lambda.NewListFunctionsPaginator(lambdaClient, &lambda.ListFunctionsInput{})
    file.WriteString("\nLambda Functions:\n")
    for lambdaPaginator.HasMorePages() {
        page, err := lambdaPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list Lambda functions in region %s, %v", cfg.Region, err)
            break
        }
        for _, function := range page.Functions {
            file.WriteString(fmt.Sprintf("- Function Name: %s\n", aws.ToString(function.FunctionName)))
        }
    }

    // DynamoDB Tables
    dynamoClient := dynamodb.NewFromConfig(cfg)
    dynamoPaginator := dynamodb.NewListTablesPaginator(dynamoClient, &dynamodb.ListTablesInput{})
    file.WriteString("\nDynamoDB Tables:\n")
    for dynamoPaginator.HasMorePages() {
        page, err := dynamoPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list DynamoDB tables in region %s, %v", cfg.Region, err)
            break
        }
        for _, tableName := range page.TableNames {
            file.WriteString(fmt.Sprintf("- Table Name: %s\n", tableName))
        }
    }

    // CloudWatch Alarms
    cwClient := cloudwatch.NewFromConfig(cfg)
    cwAlarmsPaginator := cloudwatch.NewDescribeAlarmsPaginator(cwClient, &cloudwatch.DescribeAlarmsInput{})
    file.WriteString("\nCloudWatch Alarms:\n")
    for cwAlarmsPaginator.HasMorePages() {
        page, err := cwAlarmsPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to describe CloudWatch alarms in region %s, %v", cfg.Region, err)
            break
        }
        for _, alarm := range page.MetricAlarms {
            file.WriteString(fmt.Sprintf("- Alarm Name: %s\n", aws.ToString(alarm.AlarmName)))
        }
    }

    // EBS Volumes
    ebsPaginator := ec2.NewDescribeVolumesPaginator(ec2Client, &ec2.DescribeVolumesInput{})
    file.WriteString("\nEBS Volumes:\n")
    for ebsPaginator.HasMorePages() {
        page, err := ebsPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to describe EBS volumes in region %s, %v", cfg.Region, err)
            break
        }
        for _, volume := range page.Volumes {
            file.WriteString(fmt.Sprintf("- Volume ID: %s\n", aws.ToString(volume.VolumeId)))
        }
    }

    // SNS Topics
    snsClient := sns.NewFromConfig(cfg)
    snsPaginator := sns.NewListTopicsPaginator(snsClient, &sns.ListTopicsInput{})
    file.WriteString("\nSNS Topics:\n")
    for snsPaginator.HasMorePages() {
        page, err := snsPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list SNS topics in region %s, %v", cfg.Region, err)
            break
        }
        for _, topic := range page.Topics {
            file.WriteString(fmt.Sprintf("- Topic ARN: %s\n", aws.ToString(topic.TopicArn)))
        }
    }

    // SQS Queues
    sqsClient := sqs.NewFromConfig(cfg)
    sqsPaginator := sqs.NewListQueuesPaginator(sqsClient, &sqs.ListQueuesInput{})
    file.WriteString("\nSQS Queues:\n")
    for sqsPaginator.HasMorePages() {
        page, err := sqsPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list SQS queues in region %s, %v", cfg.Region, err)
            break
        }
        for _, queueUrl := range page.QueueUrls {
            file.WriteString(fmt.Sprintf("- Queue URL: %s\n", queueUrl))
        }
    }

    // ECS Clusters
    ecsClient := ecs.NewFromConfig(cfg)
    ecsPaginator := ecs.NewListClustersPaginator(ecsClient, &ecs.ListClustersInput{})
    file.WriteString("\nECS Clusters:\n")
    for ecsPaginator.HasMorePages() {
        page, err := ecsPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list ECS clusters in region %s, %v", cfg.Region, err)
            break
        }
        for _, clusterArn := range page.ClusterArns {
            file.WriteString(fmt.Sprintf("- Cluster ARN: %s\n", clusterArn))
        }
    }

    // EKS Clusters
    eksClient := eks.NewFromConfig(cfg)
    eksPaginator := eks.NewListClustersPaginator(eksClient, &eks.ListClustersInput{})
    file.WriteString("\nEKS Clusters:\n")
    for eksPaginator.HasMorePages() {
        page, err := eksPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to list EKS clusters in region %s, %v", cfg.Region, err)
            break
        }
        for _, clusterName := range page.Clusters {
            file.WriteString(fmt.Sprintf("- Cluster Name: %s\n", clusterName))
        }
    }

    // ECR Repositories
    ecrClient := ecr.NewFromConfig(cfg)
    ecrPaginator := ecr.NewDescribeRepositoriesPaginator(ecrClient, &ecr.DescribeRepositoriesInput{})
    file.WriteString("\nECR Repositories:\n")
    for ecrPaginator.HasMorePages() {
        page, err := ecrPaginator.NextPage(ctx)
        if err != nil {
            log.Printf("Failed to describe ECR repositories in region %s, %v", cfg.Region, err)
            break
        }
        for _, repo := range page.Repositories {
            file.WriteString(fmt.Sprintf("- Repository Name: %s\n", aws.ToString(repo.RepositoryName)))
        }
    }
}
