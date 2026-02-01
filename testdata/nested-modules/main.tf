# Pattern: Nested modules - module calls another module
module "app" {
  source   = "./modules/app"
  app_name = "my-app"
}
