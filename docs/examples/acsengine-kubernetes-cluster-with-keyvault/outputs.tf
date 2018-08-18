output "kube_config" {
  value = "${base64decode(acsengine_kubernetes_cluster.cluster.kube_config_raw)}"
}
