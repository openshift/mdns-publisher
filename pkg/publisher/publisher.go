package publisher

import (
	"crypto/sha256"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/celebdor/zeroconf"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func Publish(ip net.IP, iface net.Interface, service Service, shutdown chan struct{}, waitGroup *sync.WaitGroup) (err error) {
	defer waitGroup.Done()
	svcName := truncateLongServiceName(service.Name)
	svcEntry := zeroconf.NewServiceEntry(svcName, service.SvcType, service.Domain)
	svcEntry.Port = service.Port
	if ip.To4() != nil {
		svcEntry.AddrIPv4 = append(svcEntry.AddrIPv4, ip)
	} else {
		svcEntry.AddrIPv6 = append(svcEntry.AddrIPv6, ip)
	}
	svcEntry.HostName = service.HostName
	log.WithFields(logrus.Fields{
		"name": svcEntry.Instance,
	}).Info("Zeroconf registering service")
	s, err := zeroconf.RegisterSvcEntry(svcEntry, []net.Interface{iface})
	if err != nil {
		log.Error("Failed to create zeroconf Server", err)
		return err
	}
	defer s.Shutdown()
	log.WithFields(logrus.Fields{
		"name": svcEntry.Instance,
		"ttl":  service.TTL,
	}).Info("Zeroconf setting service ttl")
	s.TTL(service.TTL)

	select {
	case <-shutdown:
		log.WithFields(logrus.Fields{
			"name": svcEntry.Instance,
		}).Info("Gracefully shutting down service")
	}

	return nil
}

func FindIface(ip net.IP) (iface net.Interface, err error) {
	var nw networkInterfacer = networkInterface{}
	return findIface(ip, nw)
}

func findIface(ip net.IP, nw networkInterfacer) (iface net.Interface, err error) {
	ifaces, err := nw.Interfaces()
	if err != nil {
		log.Printf("[ERR] mdns-publish: Failed retrieving system network interfaces %v.", err)
		return iface, err
	}
	for _, i := range ifaces {
		addrs, err := nw.Addrs(&i)
		if err != nil {
			log.Printf("[ERR] mdns-publish: Failed retrieving network addresses for interface %s: %v.", i.Name, err)
		}
		for _, addr := range addrs {
			currIP, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if currIP == nil {
				continue
			}
			if currIP.Equal(ip) {
				iface = i
				return iface, nil
			}
		}
	}
	return iface, fmt.Errorf("Couldn't find interface with IP address %s", ip)
}

// If the configured IP moves off the detected interface (perhaps because it
// got bridged), it causes communication issues for us. To address this, we
// exit and allow the service to be restarted where it will detect the new
// interface for the IP.
func ifaceCheck(ip net.IP, iface net.Interface, nw networkInterfacer, ifaceChanged chan struct{}) {
	for {
		i, err := findIface(ip, nw)
		if err != nil || i.Name != iface.Name {
			log.Printf("mdns-publish: Detected interface changed, exiting.")
			close(ifaceChanged)
			return
		}
		time.Sleep(5 * time.Second)
	}
}

func IfaceCheck(ip net.IP, iface net.Interface, ifaceChanged chan struct{}) {
	var nw networkInterfacer = networkInterface{}
	ifaceCheck(ip, iface, nw, ifaceChanged)
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

// Service names cannot be longer than 63 characters. If we get a service name
// that is longer, hash it to 32 characters, and sandwich that between the
// first 15 and last 16 characters of the original name. This way we preserve
// the likely unique parts of the name while ensuring it does not exceed
// the 63 character limit.
func truncateLongServiceName(serviceName string) (svcName string) {
	svcName = serviceName
	if len(svcName) > 63 {
		value := sha256.Sum256([]byte(svcName))
		hexValue := fmt.Sprintf("%x", value)[:32]
		svcName = svcName[:15] + string(hexValue[:]) + svcName[len(svcName)-16:]
		log.Printf("Truncating long service name '%s' to '%s'", serviceName, svcName)
	}
	return svcName
}

// networkInterfacer defines an interface for several net library functions. Production
// code will forward to net library functions, and unit tests will override the methods
// for testing purposes.
type networkInterfacer interface {
	Addrs(intf *net.Interface) ([]net.Addr, error)
	Interfaces() ([]net.Interface, error)
}

// networkInterface implements the networkInterfacer interface for production code, just
// wrapping the underlying net library function calls.
type networkInterface struct{}

func (_ networkInterface) Addrs(intf *net.Interface) ([]net.Addr, error) {
	return intf.Addrs()
}

func (_ networkInterface) Interfaces() ([]net.Interface, error) {
	return net.Interfaces()
}
