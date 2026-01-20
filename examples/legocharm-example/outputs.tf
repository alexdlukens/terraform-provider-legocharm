# Output the created user details
output "user_id" {
  description = "The ID of the created user"
  value       = legocharm_user.example.id
}

output "user_username" {
  description = "The username of the created user"
  value       = legocharm_user.example.username
}

output "user_email" {
  description = "The email address of the created user"
  value       = legocharm_user.example.email
  sensitive   = false
}
