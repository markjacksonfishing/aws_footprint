# AWS Footprint Collector Code Tour

More times than I care to count I have to step into a customer Amazon Web Services (AWS) account and get an understanding of what is there. Clicking through the UI is not optimal and remembering what I saw is even less optimal. So, I have had this code for some time and finally wanted to make it public. There are a bunch of scripts to do this, and maybe even better ones, but this is what I needed for an itch.

Like it, love it, or hate it. Just be nice.

This README provides a detailed tour of the AWS Footprint Collector code, which is used to retrieve information from an AWS account and generate a report of various resources across global and regional services. Below is a detailed breakdown of the code to help you understand its structure, logic, and functions.

Disclamer:This was not written by ChatGPT, but by me. I am just making it public.

## Overview
This Go program aims to give users a consolidated view of the AWS resources in an account by accessing information using the AWS SDK for Go v2. The program collects details from various AWS services such as EC2, S3, IAM, Lambda, RDS, and many others, and writes them to a text file. This makes it easier for users to audit or review the footprint of an AWS account without needing to click through the AWS Management Console.

The core features include:
- Prompting for AWS profile and region
- Loading AWS configurations using the specified profile and region
- Collecting global and regional AWS resources
- Saving the footprint information to a text file

## Code Structure
The code is divided into several components to do the functionality:

1. **Main Function**: This is the entry point of the program and handles the configuration setup, user input, and initial resource collection.
2. **Global Resource Collection**: Retrieves global resources such as S3 buckets, IAM users, IAM roles, and CloudFront distributions.
3. **Regional Resource Collection**: Retrieves regional resources such as EC2 instances, RDS instances, VPCs, Lambda functions, and many more.

### Dependencies
The code uses the AWS SDK for Go v2 to interact with AWS services. The following services are imported from the SDK:
- **EC2**: To gather instance, VPC, subnet, and security group information.
- **IAM**: To gather IAM users and roles.
- **S3**: To list all S3 buckets in the account.
- **RDS, Lambda, ECS, EKS, CloudFront, CloudWatch, SNS, SQS**: To gather specific resource information from these services.

### Code Walkthrough
Let's break down the code:

#### 1. Setup
- The `main()` function starts by prompting the user for the AWS profile and region to use.
- The AWS configuration (`cfg`) is loaded using `config.LoadDefaultConfig`, which allows you to specify the profile name.
- **Account ID Retrieval**: The STS service is used to retrieve the AWS account ID, which is then used in the report filename.

#### 2. File Creation
- A new text file is created to store the output (`aws_footprint_<accountID>.txt`), where the program will write all the retrieved information.

#### 3. Global Resources Collection (`collectGlobalResources`)
- **S3 Buckets**: Lists all S3 buckets across all regions.
- **IAM Users and Roles**: Retrieves all IAM users and roles using pagination to ensure all results are covered.
- **CloudFront Distributions**: Lists all CloudFront distributions, which are also global resources.

#### 4. Regional Resources Collection (`collectRegionalResources`)
The `collectRegionalResources` function handles resource collection for a specific region set by the user.

- **EC2 Instances, VPCs, Subnets, Security Groups**: Retrieves information about virtual resources in the specified region, such as instances, VPCs, subnets, and security groups.
- **Elastic Load Balancers (ELBv2)**: Lists load balancers in the selected region.
- **RDS Instances**: Retrieves information about all RDS database instances.
- **Lambda Functions**: Lists Lambda functions available in the selected region.
- **DynamoDB Tables**: Lists all DynamoDB tables, which are region-specific.
- **CloudWatch Alarms**: Retrieves active alarms within the region.
- **EBS Volumes**: Lists all attached and unattached EBS volumes.
- **SNS Topics and SQS Queues**: Lists SNS topics and SQS queues.
- **ECS and EKS Clusters**: Retrieves all ECS and EKS clusters.
- **ECR Repositories**: Lists ECR repositories within the specified region.

### Key Concepts and Techniques
- **AWS SDK v2 Usage**: The program makes extensive use of AWS SDK v2 for Go to interact with AWS services.
- **Pagination**: Many AWS services return data in pages. The code uses paginator functions (`NewListUsersPaginator`, `NewDescribeInstancesPaginator`, etc.) to ensure that all data is collected, even for accounts with a large number of resources.
- **Context and Error Handling**: The code uses `context.TODO()` to manage request contexts and logs errors during resource collection to ensure robustness.
- **File Writing**: The collected information is written to a text file, making it easy to access and review the footprint of the AWS account.

## Usage Instructions
1. **Run the Program**: Compile and run the Go program in a terminal. You'll be prompted for an AWS profile name and a region.
   ```sh
   go run aws_footprint.go
   ```
2. **Provide AWS Credentials**: Make sure you have AWS credentials configured in `~/.aws/credentials` with the profile name you provide.
3. **Check the Output**: Once the program completes, you'll have a file named `aws_footprint_<accountID>.txt` in the current directory containing the details of all AWS resources it found.

## Customization
You may want to customize the following aspects of the code:
- **Additional Services**: If you need information from other AWS services, you can import the relevant service client from the AWS SDK and add a new function to collect the data.
- **Output Format**: Currently, the output is a simple text file. You could modify the program to output in JSON or another format if preferred.

## Limitations
- **Static Configuration**: The AWS profile and region must be provided manually, which might be inconvenient in automated scenarios. This could be modified to use environment variables or other configuration options.
- **Service Coverage**: The script only includes a subset of AWS services. Depending on your needs, you might need to extend it to cover additional services such as Glue, Redshift, or others.
- **Error Handling**: The program logs errors when retrieving information from services but continues executing. Depending on your use case, you might want to handle some errors more strictly.

## Dockerization

This repository also provides a Dockerized version of the AWS Footprint Collector. This allows you to run the program in an isolated environment without needing to install Go or the AWS SDK on your local machine.

### Dockerfile Overview

The provided `Dockerfile` is a multi-stage build that compiles the Go application and packages it into a small Alpine-based Docker image.

- **Stage 1**: Uses the official Go image to compile the `aws_footprint.go` program.
- **Stage 2**: Uses a minimal Alpine image to run the compiled Go binary, reducing the image size.

### Building the Docker Image

To build the Docker image locally, follow these steps:

1. Clone the repository and navigate to the project directory.
2. Run the following command to build the Docker image:

   ```bash
   docker build -t aws_footprint .
   ```

This will create a Docker image named `aws_footprint` on your local machine.

### Running the Docker Container

To run the AWS Footprint Collector inside a Docker container:

```bash
docker run -it --rm aws_footprint
```

The program will prompt you for an AWS profile and region, and will generate a report of AWS resources as described in previous sections.

### Pulling from Docker Hub

Alternatively, you can skip building the image locally and pull the pre-built image directly from Docker Hub:

```bash
docker pull jequals5/aws_footprint:latest
```

Once the image is pulled, you can run it using:

```bash
docker run -it --rm jequals5/aws_footprint:latest
```

This will execute the program and prompt for your AWS profile and region, similar to the locally built image.

### Using the Image

Make sure that your AWS credentials are configured properly and are accessible to Docker. You can pass environment variables or mount your AWS credentials file if necessary:

```bash
docker run -it --rm -v ~/.aws:/root/.aws jequals5/aws_footprint:latest
```

This command mounts your AWS credentials from your local machine into the container so it can authenticate with AWS.

---

## And So
This AWS Footprint Collector provides a basic yet extensible solution to retrieve information from an AWS account programmatically. It's a useful tool to help with initial account assessments, audits, or simply to get a lay of the land.

Feel free to tweak it to suit your needs or add additional services to enhance its utility. I hope this helps simplify your cloud operations!
