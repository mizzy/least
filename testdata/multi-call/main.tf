# Pattern: Same module called multiple times
module "bucket_logs" {
  source      = "./modules/bucket"
  bucket_name = "logs-bucket"
  versioning  = false
}

module "bucket_data" {
  source      = "./modules/bucket"
  bucket_name = "data-bucket"
  versioning  = true
}

module "bucket_backup" {
  source      = "./modules/bucket"
  bucket_name = "backup-bucket"
  versioning  = true
}
