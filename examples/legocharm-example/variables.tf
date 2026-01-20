# LegoCharm Provider Configuration
variable "legocharm_address" {
  description = "The address of the LegoCharm server (e.g., https://lego-certs.example.com)"
  type        = string
  sensitive   = false
}

variable "legocharm_username" {
  description = "Username for authenticating with the LegoCharm server"
  type        = string
  sensitive   = true
}

variable "legocharm_password" {
  description = "Password for authenticating with the LegoCharm server"
  type        = string
  sensitive   = true
}

# User Resource Configuration
variable "username" {
  description = "Username for the new LegoCharm user"
  type        = string
  validation {
    condition     = can(regex("^[a-zA-Z0-9_-]+$", var.username))
    error_message = "Username must contain only alphanumeric characters, underscores, and hyphens."
  }
}

variable "password" {
  description = "Password for the new LegoCharm user"
  type        = string
  sensitive   = true
  validation {
    condition     = length(var.password) >= 8
    error_message = "Password must be at least 8 characters long."
  }
}

variable "email" {
  description = "Email address for the new LegoCharm user (optional)"
  type        = string
  default     = null
  validation {
    condition     = var.email == "" || can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.email))
    error_message = "Email must be a valid email address or empty."
  }
}
