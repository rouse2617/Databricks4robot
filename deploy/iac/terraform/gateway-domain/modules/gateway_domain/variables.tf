variable "cloudflare_zone" {
  description = "Cloudflare DNS zone name."
  type        = string
}

variable "certificate_map_name" {
  description = "Existing Certificate Manager certificate map name."
  type        = string
}

variable "gateway_ip" {
  description = "Global external IP of the Gateway load balancer."
  type        = string
}

variable "frontend_domain" {
  description = "Frontend FQDN."
  type        = string
}

variable "api_domain" {
  description = "API FQDN."
  type        = string
}

variable "frontend_record_name" {
  description = "Cloudflare record name for frontend A record."
  type        = string
}

variable "api_record_name" {
  description = "Cloudflare record name for API A record."
  type        = string
}

variable "name_prefix" {
  description = "Prefix for Certificate Manager resource names."
  type        = string
}
