# mDNS-publisher

This program allows you to register .local mDNS services from a configuration
file controlling which IP they should be published under (as long as said IP is
configured in one of your network interfaces.

## Usage

You can place the configuration file in either:

    /etc/mdns/config.hcl
    /etc/mdns/config.yaml
    /etc/mdns/config.json
    /etc/mdns/config.toml

Or you can pass a path to the config:

    ./mdns-publisher --config /path/to/my/config_file.ext

## Configuration

Here you can see a sample configuration in Hashicorp's HCL format.
Alternatively, You can specify the configuration in YAML, JSON or TOML as long
as it is equivalent.

    bind_address = "192.168.2.21"
    service {
        name = "Etcd"
        host_name = "etcd-0.local."
        type = "_etcd-server-ssl._tcp"
        domain = "local."
        port = 2380
        ttl = 300
    }

You should always provide a *bind_address*, the address that will be propagated
for the service. All the service fields are mandatory at the moment. Note that
the mDNS library default *ttl* value is 3200.

## Building

    $ go get -v github.com/openshift-metalkube/mdns-publisher

## Building container

    sudo podman build -t your_user/mdns-publisher .

## Runing containerized

    sudo podman run -v /path/with/my/config:/etc/mdns --net host quay.io/openshift-metalkube/mdns-publisher
