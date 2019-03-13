/*
Declare as many services as you need
for your machine */

bind_address = "192.168.2.21"

service {
    name = "Etcd"
    host_name = "etcd-0.local."
    type = "_etcd-server-ssl._tcp"
    domain = "local."
    port = 2380
    ttl = 300
}

service {
    name = "workstation"
    host_name = "master-0.local."
    type = "_workstation._tcp"
    domain = "local."
    port = 42424
    ttl = 300
}

service {
    name = "EtcdWorkstation"
    host_name = "etcd-0.local."
    type = "_workstation._tcp"
    domain = "local."
    port = 42424
    ttl = 300
}
