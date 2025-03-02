package daemon

import (
	"context"
	"sort"
	"strings"

	"github.com/NordSecurity/nordvpn-linux/daemon/pb"
	"github.com/NordSecurity/nordvpn-linux/internal"
)

// Cities provides cities command and autocompletion.
func (r *RPC) Cities(ctx context.Context, in *pb.CitiesRequest) (*pb.Payload, error) {
	// collect cities and sort them
	if value, ok := r.dm.GetAppData().CityNames[in.GetObfuscate()][in.GetProtocol()][strings.ToLower(in.GetCountry())]; ok {
		var namesList []string
		for city := range value.Iter() {
			namesList = append(namesList, city.(string))
		}
		sort.Strings(namesList)
		return &pb.Payload{
			Type: internal.CodeSuccess,
			Data: namesList,
		}, nil
	}
	return &pb.Payload{
		Type: internal.CodeEmptyPayloadError,
	}, nil
}
