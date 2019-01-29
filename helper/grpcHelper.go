package helper

import (
	"errors"
	"golang.org/x/net/context"
	"sync"
	"time"

	"github.com/integration-system/isp-lib/logger"
	"github.com/integration-system/isp-lib/structure"
	"github.com/processout/grpc-go-pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var (
	isInitializing = false
	routingConfigs *roundRobinBalancer
	lock           = sync.RWMutex{}
)

type roundRobinBalancer struct {
	next     int
	addrList []structure.AddressConfiguration
	conList  []*grpc.ClientConn
	mu       sync.Mutex
}

func (rrb *roundRobinBalancer) Next() *grpc.ClientConn {

	l := len(rrb.addrList)
	if l == 1 {
		return rrb.conList[0]
	}

	rrb.mu.Lock()
	sc := rrb.conList[rrb.next]
	rrb.next = (rrb.next + 1) % l
	rrb.mu.Unlock()
	return sc
}

func (rrb *roundRobinBalancer) NextNumber() int {
	l := len(rrb.addrList)
	if l == 1 {
		return 0
	}
	rrb.next = (rrb.next + 1) % l
	return rrb.next
}

func IsInited() bool {
	return !(routingConfigs == nil || !isInitializing)
}

func CheckReconnectWhenConfigChanged(remoteAddresses []structure.AddressConfiguration) bool {
	lock.Lock()
	defer lock.Unlock()

	oldProxyAddress := ""
	newProxyAddress := ""
	if routingConfigs != nil && routingConfigs.addrList != nil {
		for _, v := range routingConfigs.addrList {
			oldProxyAddress += v.GetAddress()
		}
	}
	if remoteAddresses != nil {
		for _, v := range remoteAddresses {
			newProxyAddress += v.GetAddress()
		}
	}
	if newProxyAddress == "" {
		logger.Info("All ROUTERs have gone away")
		CloseAllConnections()
		routingConfigs = &roundRobinBalancer{}
		isInitializing = false
		return false
	} else if newProxyAddress != oldProxyAddress {
		logger.Infof("New ROUTER addresses was received: %v", newProxyAddress)
		CloseAllConnections()
		return initConnection(remoteAddresses)
	}

	return true
}

func CloseAllConnections() {
	if routingConfigs != nil {
		for _, v := range routingConfigs.conList {
			v.Close()
		}
	}
}

func GetRoutersAndStatus() interface{} {
	result := make(map[string]string, 0)
	if routingConfigs == nil || !isInitializing {
		return result
	}
	for j := 0; j < len(routingConfigs.addrList); j++ {
		if len(routingConfigs.conList) > j {
			result[routingConfigs.addrList[j].GetAddress()] = routingConfigs.conList[j].GetState().String()
		} else {
			result[routingConfigs.addrList[j].GetAddress()] = "UNKNOWN"
		}
	}
	return result
}

func GetGrpcConnection() (*grpc.ClientConn, error) {
	lock.RLock()
	defer lock.RUnlock()

	if routingConfigs == nil || !isInitializing {
		return nil, errors.New("No one ROUTER was found in routes")
	}
	for j := 0; j < len(routingConfigs.conList)*3; j++ {
		conn := routingConfigs.Next()
		state := conn.GetState()
		if state == connectivity.Ready || state == connectivity.Connecting {
			return conn, nil
		}
	}
	return nil, errors.New("No one ACTIVE ROUTER was found in routes")
}

func initConnection(remoteAddresses []structure.AddressConfiguration) bool {
	isInitializing = true
	logger.Infof("Connection to: %s is established", remoteAddresses)
	return initRoutes(remoteAddresses)
}

func initRoutes(addresses []structure.AddressConfiguration) bool {
	connections := make([]*grpc.ClientConn, 0, len(addresses))
	for _, address := range addresses {
		conn, err := getConnFactory(address.GetAddress())()
		if err == nil {
			connections = append(connections, conn)
		}
	}
	routingConfigs = &roundRobinBalancer{
		addrList: addresses,
		conList:  connections,
	}
	return len(routingConfigs.conList) > 0
}

func createConnPools(addr structure.AddressConfiguration) (*grpcpool.Pool, error) {
	pool, err := grpcpool.New(getConnFactory(addr.GetAddress()), 1, 3, 30*time.Minute)
	if err != nil {
		logger.Errorf("Could not connect to %s, %s", addr, err)
	}
	return pool, err
}

func getConnFactory(addr string) grpcpool.Factory {
	return func() (*grpc.ClientConn, error) {
		// grpclb, pick_first, round_robin
		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, 3*time.Second)
		return grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure())
	}
}
