# TFLint configuration file
# See https://github.com/terraform-linters/tflint/blob/master/docs/user-guide/config.md for options

plugin "google" {
  enabled = true
  version = "0.37.1"
  source  = "github.com/terraform-linters/tflint-ruleset-google"
}
