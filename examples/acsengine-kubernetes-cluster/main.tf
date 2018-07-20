provider "acsengine" {}

resource "acsengine_kubernetes_cluster" "cluster" {
  name               = "testcluster"
  resource_group     = "testRG"
  location           = "southcentralus"
  kubernetes_version = "1.9.0"

  master_profile {
    count           = 1
    dns_name_prefix = "clustermaster"
    vm_size         = "Standard_D2_v2"
  }

  agent_pool_profiles {
    name         = "agentpool1"
    count        = 1
    vm_size      = "Standard_D2_v2"
    os_disk_size = 40
  }

  linux_profile {
    admin_username = "azureuser"

    ssh {
      key_data = "ssh-rsa AAAAB3NzaC..."
    }
  }

  service_principal {
    client_id     = ""
    client_secret = ""
  }

  tags {
    Environment = "Production"
  }
}
