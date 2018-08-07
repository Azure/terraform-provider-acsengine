package utils

import "fmt"

// ACSEngineK8sClusterKubeConfig provides a example kube config for tests
func ACSEngineK8sClusterKubeConfig(dnsPrefix string, location string) string {
	return fmt.Sprintf(`    {
        "apiVersion": "v1",
        "clusters": [
            {
                "cluster": {
                    "certificate-authority-data": "0123",
                    "server": "https://%s.%s.cloudapp.azure.com"
                },
                "name": "%s"
            }
        ],
        "contexts": [
            {
                "context": {
                    "cluster": "%s",
                    "user": "%s-admin"
                },
                "name": "%s"
            }
        ],
        "current-context": "%s",
        "kind": "Config",
        "users": [
            {
                "name": "%s-admin",
                "user": {"client-certificate-data":"4567","client-key-data":"8910"}
            }
        ]
    }`, dnsPrefix, location, dnsPrefix, dnsPrefix, dnsPrefix, dnsPrefix, dnsPrefix, dnsPrefix)
}

// ACSEngineK8sClusterAPIModel ...
func ACSEngineK8sClusterAPIModel(name string, location string, dnsPrefix string) string {
	return fmt.Sprintf(`{
		"apiVersion": "vlabs",
		"location": "%s",
		"name": "%s",
		"tags": {
		  "Dep": "IT"
		},
		"properties": {
		  "orchestratorProfile": {
			"orchestratorType": "Kubernetes",
			"orchestratorRelease": "1.9",
			"orchestratorVersion": "1.9.0",
			"kubernetesConfig": {
			  "kubernetesImageBase": "k8s.gcr.io/",
			  "clusterSubnet": "10.244.0.0/16",
			  "dnsServiceIP": "10.0.0.10",
			  "serviceCidr": "10.0.0.0/16",
			  "networkPlugin": "kubenet",
			  "dockerBridgeSubnet": "172.17.0.1/16",
			  "useInstanceMetadata": true,
			  "enableRbac": true,
			  "enableSecureKubelet": true,
			  "privateCluster": {
				"enabled": false
			  },
			  "gchighthreshold": 85,
			  "gclowthreshold": 80,
			  "etcdVersion": "3.2.23",
			  "etcdDiskSizeGB": "256",
			  "addons": [
				{
				  "name": "tiller",
				  "enabled": true,
				  "containers": [
					{
					  "name": "tiller",
					  "cpuRequests": "50m",
					  "memoryRequests": "150Mi",
					  "cpuLimits": "50m",
					  "memoryLimits": "150Mi"
					}
				  ],
				  "config": {
					"max-history": "0"
				  }
				},
				{
				  "name": "aci-connector",
				  "enabled": false,
				  "containers": [
					{
					  "name": "aci-connector",
					  "cpuRequests": "50m",
					  "memoryRequests": "150Mi",
					  "cpuLimits": "50m",
					  "memoryLimits": "150Mi"
					}
				  ],
				  "config": {
					"nodeName": "aci-connector",
					"os": "Linux",
					"region": "westus",
					"taint": "azure.com/aci"
				  }
				},
				{
				  "name": "cluster-autoscaler",
				  "enabled": false,
				  "containers": [
					{
					  "name": "cluster-autoscaler",
					  "cpuRequests": "100m",
					  "memoryRequests": "300Mi",
					  "cpuLimits": "100m",
					  "memoryLimits": "300Mi"
					}
				  ],
				  "config": {
					"maxNodes": "5",
					"minNodes": "1"
				  }
				},
				{
				  "name": "keyvault-flexvolume",
				  "enabled": false,
				  "containers": [
					{
					  "name": "keyvault-flexvolume",
					  "cpuRequests": "50m",
					  "memoryRequests": "10Mi",
					  "cpuLimits": "50m",
					  "memoryLimits": "10Mi"
					}
				  ]
				},
				{
				  "name": "kubernetes-dashboard",
				  "enabled": true,
				  "containers": [
					{
					  "name": "kubernetes-dashboard",
					  "cpuRequests": "300m",
					  "memoryRequests": "150Mi",
					  "cpuLimits": "300m",
					  "memoryLimits": "150Mi"
					}
				  ]
				},
				{
				  "name": "rescheduler",
				  "enabled": false,
				  "containers": [
					{
					  "name": "rescheduler",
					  "cpuRequests": "10m",
					  "memoryRequests": "100Mi",
					  "cpuLimits": "10m",
					  "memoryLimits": "100Mi"
					}
				  ]
				},
				{
				  "name": "metrics-server",
				  "enabled": true,
				  "containers": [
					{
					  "name": "metrics-server"
					}
				  ]
				},
				{
				  "name": "nvidia-device-plugin",
				  "containers": [
					{
					  "name": "nvidia-device-plugin",
					  "cpuRequests": "50m",
					  "memoryRequests": "10Mi",
					  "cpuLimits": "50m",
					  "memoryLimits": "10Mi"
					}
				  ]
				},
				{
				  "name": "container-monitoring",
				  "enabled": false,
				  "containers": [
					{
					  "name": "omsagent",
					  "image": "microsoft/oms:June21st",
					  "cpuRequests": "50m",
					  "memoryRequests": "100Mi",
					  "cpuLimits": "150m",
					  "memoryLimits": "500Mi"
					}
				  ],
				  "config": {
					"dockerProviderVersion": "2.0.0-3",
					"omsAgentVersion": "1.6.0-42"
				  }
				},
				{
				  "name": "azure-cni-networkmonitor",
				  "enabled": false,
				  "containers": [
					{
					  "name": "azure-cni-networkmonitor"
					}
				  ]
				},
				{
				  "name": "azure-npm-daemonset",
				  "enabled": false,
				  "containers": [
					{
					  "name": "azure-npm-daemonset"
					}
				  ]
				}
			  ],
			  "kubeletConfig": {
				"--address": "0.0.0.0",
				"--allow-privileged": "true",
				"--anonymous-auth": "false",
				"--authorization-mode": "Webhook",
				"--azure-container-registry-config": "/etc/kubernetes/azure.json",
				"--cadvisor-port": "0",
				"--cgroups-per-qos": "true",
				"--client-ca-file": "/etc/kubernetes/certs/ca.crt",
				"--cloud-config": "/etc/kubernetes/azure.json",
				"--cloud-provider": "azure",
				"--cluster-dns": "10.0.0.10",
				"--cluster-domain": "cluster.local",
				"--enforce-node-allocatable": "pods",
				"--event-qps": "0",
				"--eviction-hard": "memory.available<100Mi,nodefs.available<10,nodefs.inodesFree<5",
				"--feature-gates": "",
				"--image-gc-high-threshold": "85",
				"--image-gc-low-threshold": "80",
				"--image-pull-progress-deadline": "30m",
				"--keep-terminated-pod-volumes": "false",
				"--kubeconfig": "/var/lib/kubelet/kubeconfig",
				"--max-pods": "110",
				"--network-plugin": "kubenet",
				"--node-status-update-frequency": "10s",
				"--non-masquerade-cidr": "10.244.0.0/16",
				"--pod-infra-container-image": "k8s.gcr.io/pause-amd64:3.1",
				"--pod-manifest-path": "/etc/kubernetes/manifests"
			  },
			  "controllerManagerConfig": {
				"--allocate-node-cidrs": "true",
				"--cloud-config": "/etc/kubernetes/azure.json",
				"--cloud-provider": "azure",
				"--cluster-cidr": "10.244.0.0/16",
				"--cluster-name": "clustermaster",
				"--cluster-signing-cert-file": "/etc/kubernetes/certs/ca.crt",
				"--cluster-signing-key-file": "/etc/kubernetes/certs/ca.key",
				"--configure-cloud-routes": "true",
				"--feature-gates": "ServiceNodeExclusion=true",
				"--kubeconfig": "/var/lib/kubelet/kubeconfig",
				"--leader-elect": "true",
				"--node-monitor-grace-period": "40s",
				"--pod-eviction-timeout": "5m0s",
				"--profiling": "false",
				"--root-ca-file": "/etc/kubernetes/certs/ca.crt",
				"--route-reconciliation-period": "10s",
				"--service-account-private-key-file": "/etc/kubernetes/certs/apiserver.key",
				"--terminated-pod-gc-threshold": "5000",
				"--use-service-account-credentials": "true",
				"--v": "2"
			  },
			  "cloudControllerManagerConfig": {
				"--allocate-node-cidrs": "true",
				"--cloud-config": "/etc/kubernetes/azure.json",
				"--cloud-provider": "azure",
				"--cluster-cidr": "10.244.0.0/16",
				"--cluster-name": "clustermaster",
				"--configure-cloud-routes": "true",
				"--kubeconfig": "/var/lib/kubelet/kubeconfig",
				"--leader-elect": "true",
				"--route-reconciliation-period": "10s",
				"--v": "2"
			  },
			  "apiServerConfig": {
				"--admission-control": "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,AlwaysPullImages,ExtendedResourceToleration",
				"--advertise-address": "<kubernetesAPIServerIP>",
				"--allow-privileged": "true",
				"--anonymous-auth": "false",
				"--audit-log-maxage": "30",
				"--audit-log-maxbackup": "10",
				"--audit-log-maxsize": "100",
				"--audit-log-path": "/var/log/kubeaudit/audit.log",
				"--audit-policy-file": "/etc/kubernetes/addons/audit-policy.yaml",
				"--authorization-mode": "Node,RBAC",
				"--bind-address": "0.0.0.0",
				"--client-ca-file": "/etc/kubernetes/certs/ca.crt",
				"--cloud-config": "/etc/kubernetes/azure.json",
				"--cloud-provider": "azure",
				"--etcd-cafile": "/etc/kubernetes/certs/ca.crt",
				"--etcd-certfile": "/etc/kubernetes/certs/etcdclient.crt",
				"--etcd-keyfile": "/etc/kubernetes/certs/etcdclient.key",
				"--etcd-servers": "https://127.0.0.1:2379",
				"--insecure-port": "8080",
				"--kubelet-client-certificate": "/etc/kubernetes/certs/client.crt",
				"--kubelet-client-key": "/etc/kubernetes/certs/client.key",
				"--profiling": "false",
				"--proxy-client-cert-file": "/etc/kubernetes/certs/proxy.crt",
				"--proxy-client-key-file": "/etc/kubernetes/certs/proxy.key",
				"--repair-malformed-updates": "false",
				"--requestheader-allowed-names": "",
				"--requestheader-client-ca-file": "/etc/kubernetes/certs/proxy-ca.crt",
				"--requestheader-extra-headers-prefix": "X-Remote-Extra-",
				"--requestheader-group-headers": "X-Remote-Group",
				"--requestheader-username-headers": "X-Remote-User",
				"--secure-port": "443",
				"--service-account-key-file": "/etc/kubernetes/certs/apiserver.key",
				"--service-account-lookup": "true",
				"--service-cluster-ip-range": "10.0.0.0/16",
				"--storage-backend": "etcd3",
				"--tls-cert-file": "/etc/kubernetes/certs/apiserver.crt",
				"--tls-private-key-file": "/etc/kubernetes/certs/apiserver.key",
				"--v": "4"
			  },
			  "schedulerConfig": {
				"--kubeconfig": "/var/lib/kubelet/kubeconfig",
				"--leader-elect": "true",
				"--profiling": "false",
				"--v": "2"
			  }
			}
		  },
		  "masterProfile": {
			"count": 1,
			"dnsPrefix": "%s",
			"subjectAltNames": null,
			"vmSize": "Standard_D2_v2",
			"firstConsecutiveStaticIP": "10.240.255.5",
			"storageProfile": "ManagedDisks",
			"oauthEnabled": false,
			"preProvisionExtension": null,
			"extensions": [],
			"distro": "ubuntu",
			"kubernetesConfig": {
			  "kubeletConfig": {
				"--address": "0.0.0.0",
				"--allow-privileged": "true",
				"--anonymous-auth": "false",
				"--authorization-mode": "Webhook",
				"--azure-container-registry-config": "/etc/kubernetes/azure.json",
				"--cadvisor-port": "0",
				"--cgroups-per-qos": "true",
				"--client-ca-file": "/etc/kubernetes/certs/ca.crt",
				"--cloud-config": "/etc/kubernetes/azure.json",
				"--cloud-provider": "azure",
				"--cluster-dns": "10.0.0.10",
				"--cluster-domain": "cluster.local",
				"--enforce-node-allocatable": "pods",
				"--event-qps": "0",
				"--eviction-hard": "memory.available<100Mi,nodefs.available<10,nodefs.inodesFree<5",
				"--feature-gates": "",
				"--image-gc-high-threshold": "85",
				"--image-gc-low-threshold": "80",
				"--image-pull-progress-deadline": "30m",
				"--keep-terminated-pod-volumes": "false",
				"--kubeconfig": "/var/lib/kubelet/kubeconfig",
				"--max-pods": "110",
				"--network-plugin": "kubenet",
				"--node-status-update-frequency": "10s",
				"--non-masquerade-cidr": "10.244.0.0/16",
				"--pod-infra-container-image": "k8s.gcr.io/pause-amd64:3.1",
				"--pod-manifest-path": "/etc/kubernetes/manifests"
			  }
			}
		  },
		  "agentPoolProfiles": [
			{
			  "name": "agentpool1",
			  "count": 1,
			  "vmSize": "Standard_D2_v2",
			  "osDiskSizeGB": 40,
			  "osType": "Linux",
			  "availabilityProfile": "AvailabilitySet",
			  "storageProfile": "ManagedDisks",
			  "distro": "ubuntu",
			  "kubernetesConfig": {
				"kubeletConfig": {
				  "--address": "0.0.0.0",
				  "--allow-privileged": "true",
				  "--anonymous-auth": "false",
				  "--authorization-mode": "Webhook",
				  "--azure-container-registry-config": "/etc/kubernetes/azure.json",
				  "--cadvisor-port": "0",
				  "--cgroups-per-qos": "true",
				  "--client-ca-file": "/etc/kubernetes/certs/ca.crt",
				  "--cloud-config": "/etc/kubernetes/azure.json",
				  "--cloud-provider": "azure",
				  "--cluster-dns": "10.0.0.10",
				  "--cluster-domain": "cluster.local",
				  "--enforce-node-allocatable": "pods",
				  "--event-qps": "0",
				  "--eviction-hard": "memory.available<100Mi,nodefs.available<10,nodefs.inodesFree<5",
				  "--feature-gates": "",
				  "--image-gc-high-threshold": "85",
				  "--image-gc-low-threshold": "80",
				  "--image-pull-progress-deadline": "30m",
				  "--keep-terminated-pod-volumes": "false",
				  "--kubeconfig": "/var/lib/kubelet/kubeconfig",
				  "--max-pods": "110",
				  "--network-plugin": "kubenet",
				  "--node-status-update-frequency": "10s",
				  "--non-masquerade-cidr": "10.244.0.0/16",
				  "--pod-infra-container-image": "k8s.gcr.io/pause-amd64:3.1",
				  "--pod-manifest-path": "/etc/kubernetes/manifests"
				}
			  },
			  "acceleratedNetworkingEnabled": true,
			  "fqdn": "",
			  "preProvisionExtension": null,
			  "extensions": []
			}
		  ],
		  "linuxProfile": {
			"adminUsername": "azureuser",
			"ssh": {
			  "publicKeys": [
				{
				  "keyData": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDOy4wZwM/jl/1A9JaX6GLyOCg4evKegWSg9p2d25WZOXCeDeHyfMde7iIM9yrTZRFnp5veLOLGzmsE+IrcA2M5dF12Q7B93m0wkqEznSCJ1jwUi3gI5KGZsuSsRiXwZ1treSdrn8V+KFCSWlWrMBWmcdoSo3tnqcseqY+2qwfAoALJnFJn4dOe4vBm3HVDvASXkIh5VZWj4JTBpfQhYWVm6vUQyWKD7vAy9/93BIPD0mPo6jsvCEDsShrGnxllzqtHyodddprZ1KtHF9fIcZCSIk/mdUV/ejkfuZMxSHYQYOEJ8p7FbNOnJ0UeF4LZaeeb2C51J2akRKO1EqYnRtp1 shanalily@shanalily-Precision-Tower-3620"
				}
			  ]
			}
		  },
		  "servicePrincipalProfile": {
			"clientId": "5780a851-c0c2-4c01-8baf-d074745be7f4",
			"secret": "cc7a82b8-1c52-41c2-ae29-b1ed596eb32d"
		  },
		  "certificateProfile": {
			"caCertificate": "test ca cert\n",
			"caPrivateKey": "test ca private key\n",
			"apiServerPrivateKey": "test api server private key\n",
			"clientCertificate": "client cert\n",
			"clientPrivateKey": "client private key\n",
			"kubeConfigCertificate": "kube config cert\n",
			"kubeConfigPrivateKey": "kube config private key\n",
			"etcdServerCertificate": "etcd server cert\n",
			"etcdServerPrivateKey": "etcd server private key\n",
			"etcdClientCertificate": "etcd client cert\n",
			"etcdClientPrivateKey": "etcd client private key\n",
			"etcdPeerCertificates": [
			  "etcd peer cert\n"
			],
			"etcdPeerPrivateKeys": [
			  "etcd peer private key\n"
			]
		  }
		}
	  }`, location, name, dnsPrefix)
}
