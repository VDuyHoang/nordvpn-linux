package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NordSecurity/nordvpn-linux/internal"
	"github.com/NordSecurity/nordvpn-linux/meshnet/pb"
	"github.com/NordSecurity/nordvpn-linux/nstrings"
	"github.com/NordSecurity/nordvpn-linux/slices"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

const (
	flagFilter            = "filter"
	externalFilter        = "external"
	internalFilter        = "internal"
	PeerListArgsUsageText = `

Press the Tab key to see auto-suggestions for filters.`
)

type keyval struct {
	Key   string
	Value string
}

func (c *cmd) MeshRefresh(ctx *cli.Context) error {
	resp, err := c.meshClient.RefreshMeshnet(context.Background(), &pb.Empty{})
	if err != nil {
		return formatError(err)
	}

	if err := getMeshnetResponseToError(resp); err != nil {
		return formatError(err)
	}

	color.Green("Refresh successful")
	return nil
}

func filterOnline(peers *pb.PeerList) *pb.PeerList {
	peers.Local = slices.Filter(peers.Local, func(p *pb.Peer) bool { return p.Status == 1 })
	peers.External = slices.Filter(peers.External, func(p *pb.Peer) bool { return p.Status == 1 })
	return peers
}
func filterOffline(peers *pb.PeerList) *pb.PeerList {
	peers.Local = slices.Filter(peers.Local, func(p *pb.Peer) bool { return p.Status == 0 })
	peers.External = slices.Filter(peers.External, func(p *pb.Peer) bool { return p.Status == 0 })
	return peers
}
func filterAllowsIncomingTraffic(peers *pb.PeerList) *pb.PeerList {
	peers.Local = slices.Filter(peers.Local, func(p *pb.Peer) bool { return p.IsInboundAllowed })
	peers.External = slices.Filter(peers.External, func(p *pb.Peer) bool { return p.IsInboundAllowed })
	return peers
}
func filterAllowsRouting(peers *pb.PeerList) *pb.PeerList {
	peers.Local = slices.Filter(peers.Local, func(p *pb.Peer) bool { return p.IsRoutable })
	peers.External = slices.Filter(peers.External, func(p *pb.Peer) bool { return p.IsRoutable })
	return peers
}
func filterIncomingTrafficAllowed(peers *pb.PeerList) *pb.PeerList {
	peers.Local = slices.Filter(peers.Local, func(p *pb.Peer) bool { return p.DoIAllowInbound })
	peers.External = slices.Filter(peers.External, func(p *pb.Peer) bool { return p.DoIAllowInbound })
	return peers
}
func filterRoutingAllowed(peers *pb.PeerList) *pb.PeerList {
	peers.Local = slices.Filter(peers.Local, func(p *pb.Peer) bool { return p.DoIAllowRouting })
	peers.External = slices.Filter(peers.External, func(p *pb.Peer) bool { return p.DoIAllowRouting })
	return peers
}
func filterInternalExternal(peers *pb.PeerList) *pb.PeerList {
	return peers
}

var availableFilters map[string]func(*pb.PeerList) *pb.PeerList = map[string]func(*pb.PeerList) *pb.PeerList{
	"online":                   filterOnline,
	"offline":                  filterOffline,
	"allows-incoming-traffic":  filterAllowsIncomingTraffic,
	"allows-routing":           filterAllowsRouting,
	"incoming-traffic-allowed": filterIncomingTrafficAllowed,
	"routing-allowed":          filterRoutingAllowed,
	"internal":                 filterInternalExternal,
	"external":                 filterInternalExternal,
}

// MeshPeerList queries the peer list from the meshnet service, and
// displays it to stdout
func (c *cmd) MeshPeerList(ctx *cli.Context) error {
	resp, err := c.meshClient.GetPeers(
		context.Background(),
		&pb.Empty{},
	)
	if err != nil {
		return formatError(err)
	}
	peers, err := getPeersResponseToPeerList(resp)
	if err != nil {
		return formatError(err)
	}
	if ctx.IsSet(flagFilter) {
		condition := ""
		for _, value := range strings.Split(ctx.String(flagFilter), ",") {
			filtersFunc, ok := availableFilters[value]
			if !ok {
				return formatError(errors.New(internal.FilterNonExistentErrorMessage))
			}
			peers = filtersFunc(peers)
			if value == internalFilter || value == externalFilter {
				condition = value
			}
		}
		fmt.Println(strings.TrimSpace(peersToOutputString(peers, condition)))
	} else {
		fmt.Println(strings.TrimSpace(peersToOutputString(peers, "")))
	}
	return nil
}

func (c *cmd) FiltersAutoComplete(ctx *cli.Context) {
	for key := range availableFilters {
		fmt.Println(key)
	}
}

func peersToOutputString(peers *pb.PeerList, condition string) string {
	var builder strings.Builder
	boldCol := color.New(color.Bold)
	builder.WriteString(boldCol.Sprintf("This device:\n"))
	builder.WriteString(selfToOutputString(peers.Self) + "\n")
	if condition != externalFilter {
		builder.WriteString(boldCol.Sprintf("Local Peers:\n"))
		if len(peers.Local) == 0 {
			builder.WriteString("[no peers]\n")
		}
		for _, p := range peers.Local {
			builder.WriteString(peerToOutputString(p) + "\n")
		}
		builder.WriteString("\n")
	}
	if condition != internalFilter {
		builder.WriteString(boldCol.Sprintf("External Peers: \n"))
		if len(peers.External) == 0 {
			builder.WriteString("[no peers]\n")
		}
		for _, p := range peers.External {
			builder.WriteString(peerToOutputString(p) + "\n")
		}
	}
	return builder.String()
}

func selfToOutputString(peer *pb.Peer) string {
	kvs := []keyval{
		{Key: "IP", Value: peer.Ip},
		{Key: "Public Key", Value: peer.Pubkey},
		{Key: "OS", Value: peer.Os},
		{Key: "Distribution", Value: peer.Distro},
	}
	return titledKeyvalListToColoredString(keyval{
		Key: "Hostname", Value: peer.Hostname,
	}, color.FgGreen, kvs)
}

func peerToOutputString(peer *pb.Peer) string {
	kvs := []keyval{
		{Key: "Status", Value: strings.ToLower(peer.Status.String())},
		{Key: "IP", Value: peer.Ip},
		{Key: "Public Key", Value: peer.Pubkey},
		{Key: "OS", Value: peer.Os},
		{Key: "Distribution", Value: peer.Distro},
		{Key: "Allow Incoming Traffic", Value: nstrings.GetBoolLabel(peer.DoIAllowInbound)},
		{Key: "Allow Routing", Value: nstrings.GetBoolLabel(peer.DoIAllowRouting)},
		{Key: "Allow Local Network Access", Value: nstrings.GetBoolLabel(peer.DoIAllowLocalNetwork)},
		{Key: "Allow Sending Files", Value: nstrings.GetBoolLabel(peer.DoIAllowFileshare)},
		{Key: "Allows Incoming Traffic", Value: nstrings.GetBoolLabel(peer.IsInboundAllowed)},
		{Key: "Allows Routing", Value: nstrings.GetBoolLabel(peer.IsRoutable)},
		{Key: "Allows Local Network Access", Value: nstrings.GetBoolLabel(peer.IsLocalNetworkAllowed)},
		{Key: "Allows Sending Files", Value: nstrings.GetBoolLabel(peer.IsFileshareAllowed)},
	}
	return titledKeyvalListToColoredString(keyval{
		Key: "Hostname", Value: peer.Hostname,
	}, color.FgYellow, kvs)
}

func titledKeyvalListToColoredString(
	title keyval,
	titleAttr color.Attribute,
	kvs []keyval,
) string {
	return title.colored(titleAttr) +
		"\n" +
		keyvalListToColoredString(kvs)
}
func keyvalListToColoredString(kvs []keyval) string {
	builder := strings.Builder{}
	for _, kv := range kvs {
		builder.WriteString(kv.colored(color.Reset) + "\n")
	}
	return builder.String()
}

func (kv keyval) colored(attr color.Attribute) string {
	boldCol := color.New(attr, color.Bold)
	if attr == color.Reset {
		boldCol = color.New(color.Bold)
	}
	return fmt.Sprintf(
		"%s: %s",
		boldCol.Sprintf("%s", kv.Key),
		color.New(attr).Sprintf(kv.Value),
	)
}

// MeshPeerAllowRouting sends the routing allow request to the meshnet
// service
func (c *cmd) MeshPeerAllowRouting(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}

	resp, err := c.meshClient.AllowRouting(
		context.Background(),
		&pb.UpdatePeerRequest{
			Identifier: peer.Identifier,
		},
	)
	if err != nil {
		return formatError(err)
	}

	if err := allowRoutingResponseToError(
		resp,
		peer.Hostname,
	); err != nil {
		return formatError(err)
	}

	color.Green(MsgMeshnetPeerRoutingAllowSuccess, peer.Hostname)
	return nil
}

// MeshPeerDenyRouting sends the routing deny request to the meshnet
// service
func (c *cmd) MeshPeerDenyRouting(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}

	resp, err := c.meshClient.DenyRouting(
		context.Background(),
		&pb.UpdatePeerRequest{
			Identifier: peer.Identifier,
		},
	)

	if err := denyRoutingResponseToError(
		resp,
		peer.Hostname,
	); err != nil {
		return formatError(err)
	}

	color.Green(MsgMeshnetPeerRoutingDenySuccess, peer.Hostname)
	return nil
}

// MeshPeerAllowIncoming sends the incoming traffic allow request to
// the meshnet service
func (c *cmd) MeshPeerAllowIncoming(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}

	resp, err := c.meshClient.AllowIncoming(
		context.Background(),
		&pb.UpdatePeerRequest{
			Identifier: peer.Identifier,
		},
	)

	if err := allowIncomingResponseToError(
		resp,
		peer.Hostname,
	); err != nil {
		return formatError(err)
	}

	color.Green(MsgMeshnetPeerIncomingAllowSuccess, peer.Hostname)
	return nil
}

// MeshPeerDenyIncoming sends the incoming traffic allow request to
// the meshnet service
func (c *cmd) MeshPeerDenyIncoming(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}

	resp, err := c.meshClient.DenyIncoming(
		context.Background(),
		&pb.UpdatePeerRequest{
			Identifier: peer.Identifier,
		},
	)

	if err := denyIncomingResponseToError(
		resp,
		peer.Hostname,
	); err != nil {
		return formatError(err)
	}

	color.Green(MsgMeshnetPeerIncomingDenySuccess, peer.Hostname)
	return nil
}

// MeshPeerAllowLocalNetwork sends the allow local network request to the meshnet service
func (c *cmd) MeshPeerAllowLocalNetwork(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}

	resp, err := c.meshClient.AllowLocalNetwork(
		context.Background(),
		&pb.UpdatePeerRequest{
			Identifier: peer.Identifier,
		},
	)

	if err := allowLocalNetworkResponseToError(
		resp,
		peer.Hostname,
	); err != nil {
		return err
	}

	color.Green(MsgMeshnetPeerLocalNetworkAllowSuccess, peer.Hostname)
	return nil
}

// MeshPeerDenyRouting sends the local network deny request to the meshnet service
func (c *cmd) MeshPeerDenyLocalNetwork(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}

	resp, err := c.meshClient.DenyLocalNetwork(
		context.Background(),
		&pb.UpdatePeerRequest{
			Identifier: peer.Identifier,
		},
	)

	if err := denyLocalNetworkResponseToError(
		resp,
		peer.Hostname,
	); err != nil {
		return err
	}

	color.Green(MsgMeshnetPeerLocalNetworkDenySuccess, peer.Hostname)
	return nil
}

// MeshPeerAllowFileshare sends the allow send request to the meshnet service
func (c *cmd) MeshPeerAllowFileshare(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}

	resp, err := c.meshClient.AllowFileshare(
		context.Background(),
		&pb.UpdatePeerRequest{
			Identifier: peer.Identifier,
		},
	)

	if err != nil {
		return errors.New(AccountInternalError)
	}

	if err := allowFileshareResponseToError(
		resp,
		peer.Hostname,
	); err != nil {
		return err
	}

	color.Green(MsgMeshnetPeerFileshareAllowSuccess, peer.Hostname)
	return nil
}

// MeshPeerDenyFileshare sends the deny send request to the meshnet service
func (c *cmd) MeshPeerDenyFileshare(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}

	resp, err := c.meshClient.DenyFileshare(
		context.Background(),
		&pb.UpdatePeerRequest{
			Identifier: peer.Identifier,
		},
	)

	if err != nil {
		return errors.New(AccountInternalError)
	}

	if err := denyFileshareResponseToError(
		resp,
		peer.Hostname,
	); err != nil {
		return err
	}

	color.Green(MsgMeshnetPeerFileshareDenySuccess, peer.Hostname)
	return nil
}

// MeshPeerRemove retrieves the peer form the service and sends a
// removal request to the meshnet service
func (c *cmd) MeshPeerRemove(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}
	// Send a removal request to the service
	removeResp, err := c.meshClient.RemovePeer(context.Background(), &pb.UpdatePeerRequest{
		Identifier: peer.Identifier,
	})
	if err != nil {
		return formatError(err)
	}
	if err := removePeerResponseToError(
		removeResp,
		peer.Hostname,
	); err != nil {
		return formatError(err)
	}

	color.Green(MsgMeshnetPeerRemoveSuccess, peer.Hostname)
	return nil
}

// MeshPeerConnect retrieves the peer form the service and sends a
// connect request to the meshnet service
func (c *cmd) MeshPeerConnect(ctx *cli.Context) error {
	peer, err := c.retrievePeerFromArgs(ctx)
	if err != nil {
		return formatError(err)
	}
	// Send a removal request to the service
	removeResp, err := c.meshClient.Connect(context.Background(), &pb.UpdatePeerRequest{
		Identifier: peer.Identifier,
	})
	if err != nil {
		return formatError(err)
	}
	if err := connectResponseToError(
		removeResp,
		peer.Hostname,
	); err != nil {
		return formatError(err)
	}

	color.Green(MsgMeshnetPeerConnectSuccess, peer.Hostname)
	return nil
}

// retrievePeerFromArgs queries the peer list from the meshnet service,
// then tries to find a peer by the given identifier, which is either
// public key, hostname or an IP address
func (c *cmd) retrievePeerFromArgs(
	ctx *cli.Context,
) (*pb.Peer, error) {
	identifier := ctx.Args().First()
	if identifier == "" {
		return nil, argsCountError(ctx)
	}

	// Get peer list to search by the identifier
	peersResp, err := c.meshClient.GetPeers(
		context.Background(),
		&pb.Empty{},
	)
	if err != nil {
		return nil, formatError(err)
	}
	peers, err := getPeersResponseToPeerList(peersResp)
	if err != nil {
		return nil, formatError(err)
	}

	// Find the real identifier (the one used in API) by the given
	// one
	peerList := []*pb.Peer{}
	peerList = append(peerList, peers.Local...)
	peerList = append(peerList, peers.External...)

	index := slices.IndexFunc(peerList, peerByIdentifier(identifier))
	if index == -1 {
		return nil, fmt.Errorf(
			MsgMeshnetPeerUnknown,
			identifier,
		)
	}

	return peerList[index], nil
}

// MeshPeerAutoComplete queries the peer list from the meshnet service, and
// displays it to stdout
func (c *cmd) MeshPeerAutoComplete(ctx *cli.Context) {
	resp, err := c.meshClient.GetPeers(
		context.Background(),
		&pb.Empty{},
	)
	if err != nil {
		return
	}
	peers, err := getPeersResponseToPeerList(resp)
	if err != nil {
		return
	}

	for _, peer := range peers.Local {
		fmt.Println(peer.GetHostname())
	}
	for _, peer := range peers.External {
		fmt.Println(peer.GetHostname())
	}
}

func peerByIdentifier(id string) func(*pb.Peer) bool {
	return func(peer *pb.Peer) bool {
		return peer.GetIp() == id || peer.GetHostname() == id || peer.GetPubkey() == id
	}
}

// allowRoutingResponseToError determines whether the allow routing
// response is an error and returns a human readable form of it.
// Otherwise, returns nil
func allowRoutingResponseToError(
	resp *pb.AllowRoutingResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.AllowRoutingResponse_Empty:
		return nil
	case *pb.AllowRoutingResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.AllowRoutingResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.AllowRoutingResponse_AllowRoutingErrorCode:
		return allowRoutingErrorCodeToError(
			resp.AllowRoutingErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// denyRoutingResponseToError determines whether the deny routing
// response is an error and returns a human readable form of it.
// Otherwise, returns nil
func denyRoutingResponseToError(
	resp *pb.DenyRoutingResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.DenyRoutingResponse_Empty:
		return nil
	case *pb.DenyRoutingResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.DenyRoutingResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.DenyRoutingResponse_DenyRoutingErrorCode:
		return denyRoutingErrorCodeToError(
			resp.DenyRoutingErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// allowIncomingResponseToError determines whether the allow incoming
// traffic response is an error and returns a human readable form of
// it. Otherwise, returns nil
func allowIncomingResponseToError(
	resp *pb.AllowIncomingResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.AllowIncomingResponse_Empty:
		return nil
	case *pb.AllowIncomingResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.AllowIncomingResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.AllowIncomingResponse_AllowIncomingErrorCode:
		return allowIncomingErrorCodeToError(
			resp.AllowIncomingErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// denyIncomingResponseToError determines whether the deny incoming
// traffic response is an error and returns a human readable form of
// it. Otherwise, returns nil
func denyIncomingResponseToError(
	resp *pb.DenyIncomingResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.DenyIncomingResponse_Empty:
		return nil
	case *pb.DenyIncomingResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.DenyIncomingResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.DenyIncomingResponse_DenyIncomingErrorCode:
		return denyIncomingErrorCodeToError(
			resp.DenyIncomingErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// allowRoutingResponseToError determines whether the allow local network
// response is an error and returns a human readable form of it.
// Otherwise, returns nil
func allowLocalNetworkResponseToError(
	resp *pb.AllowLocalNetworkResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.AllowLocalNetworkResponse_Empty:
		return nil
	case *pb.AllowLocalNetworkResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.AllowLocalNetworkResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.AllowLocalNetworkResponse_AllowLocalNetworkErrorCode:
		return allowLocalNetworkErrorCodeToError(
			resp.AllowLocalNetworkErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// denyLocalNetworkResponseToError determines whether the deny local network
// response is an error and returns a human readable form of it.
// Otherwise, returns nil
func denyLocalNetworkResponseToError(
	resp *pb.DenyLocalNetworkResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.DenyLocalNetworkResponse_Empty:
		return nil
	case *pb.DenyLocalNetworkResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.DenyLocalNetworkResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.DenyLocalNetworkResponse_DenyLocalNetworkErrorCode:
		return denyLocalNetworkErrorCodeToError(
			resp.DenyLocalNetworkErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// allowFileshareResponseToError determines whether the allow send response is an error and returns a
// human readable form of it. Otherwise, returns nil
func allowFileshareResponseToError(
	resp *pb.AllowFileshareResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.AllowFileshareResponse_Empty:
		return nil
	case *pb.AllowFileshareResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.AllowFileshareResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.AllowFileshareResponse_AllowSendErrorCode:
		return allowFileshareErrorCodeToError(
			resp.AllowSendErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// denyFileshareResponseToError determines whether the deny send response is an error and returns a
// human readable form of it. Otherwise, returns nil
func denyFileshareResponseToError(
	resp *pb.DenyFileshareResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}

	switch resp := resp.Response.(type) {
	case *pb.DenyFileshareResponse_Empty:
		return nil
	case *pb.DenyFileshareResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.DenyFileshareResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.DenyFileshareResponse_DenySendErrorCode:
		return denyFileshareErrorCodeToError(
			resp.DenySendErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// removePeerResponseToError determines whether the remove peer
// response is an error and returns a human readable form of it.
// Otherwise, returns nil
func removePeerResponseToError(
	resp *pb.RemovePeerResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}

	switch resp := resp.Response.(type) {
	case *pb.RemovePeerResponse_Empty:
		return nil
	case *pb.RemovePeerResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.RemovePeerResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// connectResponseToError determines whether the connect response is an
// returns a human readable form of it. Otherwise, returns nil
func connectResponseToError(
	resp *pb.ConnectResponse,
	identifier string,
) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.ConnectResponse_Empty:
		return nil
	case *pb.ConnectResponse_ServiceErrorCode:
		return serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.ConnectResponse_UpdatePeerErrorCode:
		return updatePeerErrorCodeToError(
			resp.UpdatePeerErrorCode,
			identifier,
		)
	case *pb.ConnectResponse_ConnectErrorCode:
		return connectErrorCodeToError(
			resp.ConnectErrorCode,
			identifier,
		)
	case *pb.ConnectResponse_MeshnetErrorCode:
		return meshnetErrorToError(resp.MeshnetErrorCode)
	default:
		return errors.New(AccountInternalError)
	}
}

// updatePeerErrorCodeToError converts update peer error code to a
// human readable error
func updatePeerErrorCodeToError(
	code pb.UpdatePeerErrorCode,
	identifier string,
) error {
	switch code {
	case pb.UpdatePeerErrorCode_PEER_NOT_FOUND:
		return fmt.Errorf(MsgMeshnetPeerUnknown, identifier)
	default:
		return errors.New(AccountInternalError)
	}
}

// allowRoutingErrorCodeToError converts allow routing error code to a
// human readable error
func allowRoutingErrorCodeToError(
	code pb.AllowRoutingErrorCode,
	identifier string,
) error {
	switch code {
	case pb.AllowRoutingErrorCode_ROUTING_ALREADY_ALLOWED:
		return fmt.Errorf(
			MsgMeshnetPeerRoutingAlreadyAllowed,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// denyRoutingErrorCodeToError converts deny routing error code to a
// human readable error
func denyRoutingErrorCodeToError(
	code pb.DenyRoutingErrorCode,
	identifier string,
) error {
	switch code {
	case pb.DenyRoutingErrorCode_ROUTING_ALREADY_DENIED:
		return fmt.Errorf(
			MsgMeshnetPeerRoutingAlreadyDenied,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// allowIncomingErrorCodeToError converts allow incoming traffic
// error code to a human readable error
func allowIncomingErrorCodeToError(
	code pb.AllowIncomingErrorCode,
	identifier string,
) error {
	switch code {
	case pb.AllowIncomingErrorCode_INCOMING_ALREADY_ALLOWED:
		return fmt.Errorf(
			MsgMeshnetPeerIncomingAlreadyAllowed,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// denyIncomingErrorCodeToError converts allow incoming traffic
// error code to a human readable error
func denyIncomingErrorCodeToError(
	code pb.DenyIncomingErrorCode,
	identifier string,
) error {
	switch code {
	case pb.DenyIncomingErrorCode_INCOMING_ALREADY_DENIED:
		return fmt.Errorf(
			MsgMeshnetPeerIncomingAlreadyDenied,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// allowLocalNetworkErrorCodeToError converts allow local network error code to a
// human readable error
func allowLocalNetworkErrorCodeToError(
	code pb.AllowLocalNetworkErrorCode,
	identifier string,
) error {
	switch code {
	case pb.AllowLocalNetworkErrorCode_LOCAL_NETWORK_ALREADY_ALLOWED:
		return fmt.Errorf(
			MsgMeshnetPeerLocalNetworkAlreadyAllowed,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// denyLocalNetworkErrorCodeToError converts allow local network error code to a
// human readable error
func denyLocalNetworkErrorCodeToError(
	code pb.DenyLocalNetworkErrorCode,
	identifier string,
) error {
	switch code {
	case pb.DenyLocalNetworkErrorCode_LOCAL_NETWORK_ALREADY_DENIED:
		return fmt.Errorf(
			MsgMeshnetPeerLocalNetworkAlreadyDenied,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// allowFileshareErrorCodeToError converts allow filesahre error code to a human readable error
func allowFileshareErrorCodeToError(
	code pb.AllowFileshareErrorCode,
	identifier string,
) error {
	switch code {
	case pb.AllowFileshareErrorCode_SEND_ALREADY_ALLOWED:
		return fmt.Errorf(
			MsgMeshnetPeerFileshareAlreadyAllowed,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// denyFileshareErrorCodeToError converts deny fileshare error code to human readable error
func denyFileshareErrorCodeToError(
	code pb.DenyFileshareErrorCode,
	identifier string,
) error {
	switch code {
	case pb.DenyFileshareErrorCode_SEND_ALREADY_DENIED:
		return fmt.Errorf(
			MsgMeshnetPeerFileshareAlreadyDenied,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// connectErrorCodeToError converts allow incoming traffic
// error code to a human readable error
func connectErrorCodeToError(
	code pb.ConnectErrorCode,
	identifier string,
) error {
	switch code {
	case pb.ConnectErrorCode_ALREADY_CONNECTED:
		return fmt.Errorf(
			MsgMeshnetPeerAlreadyConnected,
		)
	case pb.ConnectErrorCode_PEER_DOES_NOT_ALLOW_ROUTING:
		return fmt.Errorf(
			MsgMeshnetPeerDoesNotAllowRouting,
			identifier,
		)
	case pb.ConnectErrorCode_CONNECT_FAILED:
		return fmt.Errorf(
			MsgMeshnetPeerConnectFailed,
			identifier,
		)
	default:
		return errors.New(AccountInternalError)
	}
}

// getPeersResponseToPeerList determines whether the peers response is
// an error and returns a human readable form of it. If this is a valid
// invite list, it returns that.
func getPeersResponseToPeerList(
	resp *pb.GetPeersResponse,
) (*pb.PeerList, error) {
	if resp == nil {
		return nil, errors.New(AccountInternalError)
	}
	switch resp := resp.Response.(type) {
	case *pb.GetPeersResponse_Peers:
		return resp.Peers, nil
	case *pb.GetPeersResponse_ServiceErrorCode:
		return nil, serviceErrorCodeToError(resp.ServiceErrorCode)
	case *pb.GetPeersResponse_MeshnetErrorCode:
		return nil, meshnetErrorToError(resp.MeshnetErrorCode)
	default:
		return nil, errors.New(AccountInternalError)
	}
}

func getMeshnetResponseToError(resp *pb.MeshnetResponse) error {
	if resp == nil {
		return errors.New(AccountInternalError)
	}

	switch resp := resp.Response.(type) {
	case *pb.MeshnetResponse_Empty:
		return nil
	case *pb.MeshnetResponse_ServiceError:
		return serviceErrorCodeToError(resp.ServiceError)
	case *pb.MeshnetResponse_MeshnetError:
		return meshnetErrorToError(resp.MeshnetError)
	default:
		return errors.New(AccountInternalError)
	}
}
