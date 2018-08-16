output "ca_certificate" {
  value = "${base64decode(acsengine_kubernetes_cluster.cluster.ca_certificate)}"
}

output "client_certificate" {
  value = "${base64decode(acsengine_kubernetes_cluster.cluster.client_certificate)}"
}

output "kube_config" {
  value = "${base64decode(acsengine_kubernetes_cluster.cluster.kube_config_raw)}"
}