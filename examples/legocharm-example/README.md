# LegoCharm Provider Example

This example demonstrates how to use the Terraform LegoCharm provider to create a user resource.

## Files

- **main.tf** - Provider configuration and user resource definition
- **variables.tf** - Variable definitions with validation rules
- **terraform.tfvars** - Example values for all variables
- **outputs.tf** - Output values for the created user

## Setup

1. **Update terraform.tfvars** with your LegoCharm server credentials:
   ```hcl
   legocharm_address  = "https://your-legocharm-server.com"
   legocharm_username = "your-admin-username"
   legocharm_password = "your-admin-password"
   ```

2. **Update user configuration** in terraform.tfvars:
   ```hcl
   username = "desired-username"
   password = "secure-password"
   email    = "user@example.com"
   ```

## Usage

### Initialize Terraform
```bash
terraform init
```

### Plan the deployment
```bash
terraform plan
```

### Apply the configuration
```bash
terraform apply
```

### View outputs
```bash
terraform output
```

### Destroy resources
```bash
terraform destroy
```

## Variables

### Provider Configuration
- `legocharm_address` - The address of the LegoCharm server (e.g., https://lego-certs.example.com)
- `legocharm_username` - Username for authenticating with the server
- `legocharm_password` - Password for authenticating with the server

### User Configuration
- `username` - Username for the new user (alphanumeric, underscore, hyphen only)
- `password` - Password for the new user (minimum 8 characters)
- `email` - Email address for the new user (optional, valid email format)

## Outputs

- `user_id` - The ID of the created user
- `user_username` - The username of the created user
- `user_email` - The email address of the created user

## Example Workflow

```bash
# Initialize the working directory
$ terraform init

# Check what changes will be made
$ terraform plan

# Create the user
$ terraform apply

# View the created user's ID
$ terraform output user_id

# Clean up when done
$ terraform destroy
```

## Notes

- Passwords are marked as sensitive in variables and outputs for security
- User configuration requires passwords to be at least 8 characters
- Usernames must contain only alphanumeric characters, underscores, and hyphens
- Email addresses must be in valid email format or left empty
