# Notes on Scaling Agent Pool Node Counts

Like ACS Engine, this provider allows you to scale your agent pools up or down. This does not mean you can change the number of agent pools. You also cannot scale the master count after cluster creation.

If you would like to scale up or down your cluster, just change the value `count` in an agent pool profile to another positive integer, whether it's higher or lower than before.

```
agent_pool_profiles {
    name    = "agentpool1"
    count   = 1
    vm_size = "Standard_D2_v2"
}
```

```
agent_pool_profiles {
    name    = "agentpool1"
    count   = 2
    vm_size = "Standard_D2_v2"
}
```

When you run `terraform plan`, you should see that only a change will be made, not a creation of a new resource. You can now run `terraform apply` to apply the update to your cluster.