output "kube_config" {
  value = "${base64decode(acsengine_kubernetes_cluster.test.kube_config_raw)}"
}
